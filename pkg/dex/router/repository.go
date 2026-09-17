package router

import (
	"context"
	"fmt"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SrcRepo interface {
	Pairs(ctx context.Context) ([]Pair, error)
	PairStatus(ctx context.Context) (count int, unrouted bool, err error)
	HiddenTokens(ctx context.Context) ([]string, error)
	SyncHiddenRoutes(ctx context.Context) error
	UpdateRoutes(ctx context.Context, indexToAsset map[int]string, routesMap map[int]map[int][][]int) error
	Close() error
}

// hiddenPairFilter drops pairs holding a hidden token.
func hiddenPairFilter(pairAlias string) string {
	return fmt.Sprintf(`not exists (
	select 1
	from token_exception te
	where te.chain_id = %[1]s.chain_id
		and te.hidden
		and (te.contract = %[1]s.asset0 or te.contract = %[1]s.asset1))`, pairAlias)
}

var _ SrcRepo = &srcRepoImpl{}

type srcRepoImpl struct {
	db      *gorm.DB
	chainId string
}

func NewSrcRepo(chainId string, dbConfig configs.RdbConfig) (SrcRepo, error) {
	gormDB, err := db.OpenGormPostgres(dbConfig)
	if err != nil {
		return nil, err
	}

	return &srcRepoImpl{
		db:      gormDB,
		chainId: chainId,
	}, nil
}

func (r *srcRepoImpl) Close() error {
	db, err := r.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

// Pairs leaves out pairs holding a hidden token, so no route through one is ever built.
func (r *srcRepoImpl) Pairs(ctx context.Context) ([]Pair, error) {
	pairs := []schemas.Pair{}
	tx := r.db.WithContext(ctx).Where(schemas.Pair{ChainId: r.chainId}).Where(hiddenPairFilter("pair")).Find(&pairs)
	if tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.Pairs")
	}

	newPairs := []Pair{}
	for _, p := range pairs {
		assetInfos := []string{p.Asset0, p.Asset1}
		newPairs = append(newPairs, Pair{Contract: p.Contract, AssetInfos: assetInfos})
	}

	return newPairs, nil
}

// PairStatus measures the chain's pair table against its route table:
//
//   - count is how many pairs the chain has.
//   - unrouted is whether any of them is missing from the route table. A rebuild writes
//     every route in one transaction and always leaves a hop_count 0 row per pair, so
//     that row marks coverage exactly.
//
// Both come from one query, or a pair created between two reads would be counted while
// its missing routes went unseen. Hidden pairs are left out: they get no route row by
// design, so counting them would report the table as permanently unrouted.
func (r *srcRepoImpl) PairStatus(ctx context.Context) (int, bool, error) {
	query := fmt.Sprintf(`
select count(*) as total,
	coalesce(bool_or(not exists (
		select 1
		from route r
		where r.chain_id = p.chain_id
			and r.asset0 = p.asset0
			and r.asset1 = p.asset1
			and r.hop_count = 0
			and r.deleted_at is null)), false) as unrouted
from pair p
where p.chain_id = ?
	and %s
`, hiddenPairFilter("p"))
	var res struct {
		Total    int
		Unrouted bool
	}
	if tx := r.db.WithContext(ctx).Raw(query, r.chainId).Find(&res); tx.Error != nil {
		return 0, false, errors.Wrap(tx.Error, "repo.PairStatus")
	}

	return res.Total, res.Unrouted, nil
}

// HiddenTokens is sorted, so callers can compare two reads for equality.
func (r *srcRepoImpl) HiddenTokens(ctx context.Context) ([]string, error) {
	tokens := []string{}
	tx := r.db.WithContext(ctx).Model(&schemas.TokenException{}).Where(
		"chain_id = ? and hidden", r.chainId).Order("contract").Pluck("contract", &tokens)
	if tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.HiddenTokens")
	}

	return tokens, nil
}

// hiddenRouteMatch matches the hop array too, or a route merely passing through the
// hidden token goes unnoticed.
const hiddenRouteMatch = `exists (
		select 1
		from token_exception te
		where te.chain_id = r.chain_id
			and te.hidden
			and (te.contract = r.asset0 or te.contract = r.asset1 or te.contract = any(r.route)))`

// SyncHiddenRoutes retires the routes that reach a hidden token and restores the ones
// that no longer do. Retired rather than deleted: price rows carry a route id and are
// never rewritten, so deleting would leave them pointing at nothing.
func (r *srcRepoImpl) SyncHiddenRoutes(ctx context.Context) error {
	retire := `
update route r
set deleted_at = extract(epoch from now()), modified_at = extract(epoch from now())
where r.chain_id = ?
	and r.deleted_at is null
	and ` + hiddenRouteMatch

	restore := `
update route r
set deleted_at = null, modified_at = extract(epoch from now())
where r.chain_id = ?
	and r.deleted_at is not null
	and not ` + hiddenRouteMatch

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(retire, r.chainId).Error; err != nil {
			return err
		}

		return tx.Exec(restore, r.chainId).Error
	})
	if err != nil {
		return errors.Wrap(err, "repo.SyncHiddenRoutes")
	}

	return nil
}

func (r *srcRepoImpl) UpdateRoutes(ctx context.Context, indexToAsset map[int]string, routesMap map[int]map[int][][]int) error {
	dbRoutes := make([]schemas.Route, 0) // nolint: prealloc
	for a0, a1Routes := range routesMap {
		for a1, routes := range a1Routes {
			for _, route := range routes {
				assetRoute := make(pq.StringArray, len(route))
				for i, a := range route {
					assetRoute[i] = indexToAsset[a]
				}

				dbRoutes = append(dbRoutes, schemas.Route{
					ChainId:  r.chainId,
					Asset0:   indexToAsset[a0],
					Asset1:   indexToAsset[a1],
					HopCount: len(route) - 1,
					Route:    assetRoute,
				})
			}
		}
	}

	// batchSize limits inserts to avoid PostgreSQL's 65,535 parameter limit
	const batchSize = 10000
	// deleted_at belongs to SyncHiddenRoutes, and omitting it keeps the parameter budget
	tx := r.db.WithContext(ctx).Model(schemas.Route{}).Omit("DeletedAt").Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "chain_id"}, {Name: "asset0"}, {Name: "asset1"}, {Name: "route"}},
			DoNothing: true,
		}).CreateInBatches(dbRoutes, batchSize)
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "repo.UpdateRoutes")
	}

	return nil
}
