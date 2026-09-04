package router

import (
	"context"
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
	UpdateRoutes(ctx context.Context, indexToAsset map[int]string, routesMap map[int]map[int][][]int) error
	Close() error
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

func (r *srcRepoImpl) Pairs(ctx context.Context) ([]Pair, error) {
	pairs := []schemas.Pair{}
	tx := r.db.WithContext(ctx).Where(schemas.Pair{ChainId: r.chainId}).Find(&pairs)
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
// its missing routes went unseen.
func (r *srcRepoImpl) PairStatus(ctx context.Context) (int, bool, error) {
	query := `
select count(*) as total,
	coalesce(bool_or(not exists (
		select 1
		from route r
		where r.chain_id = p.chain_id
			and r.asset0 = p.asset0
			and r.asset1 = p.asset1
			and r.hop_count = 0)), false) as unrouted
from pair p
where p.chain_id = ?
`
	var res struct {
		Total    int
		Unrouted bool
	}
	if tx := r.db.WithContext(ctx).Raw(query, r.chainId).Find(&res); tx.Error != nil {
		return 0, false, errors.Wrap(tx.Error, "repo.PairStatus")
	}

	return res.Total, res.Unrouted, nil
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
	tx := r.db.WithContext(ctx).Model(schemas.Route{}).Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "chain_id"}, {Name: "asset0"}, {Name: "asset1"}, {Name: "route"}},
			DoNothing: true,
		}).CreateInBatches(dbRoutes, batchSize)
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "repo.UpdateRoutes")
	}

	return nil
}
