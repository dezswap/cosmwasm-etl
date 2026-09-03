package price

import (
	"context"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// SrcRepo reads and writes the price source database. Every query takes a context
// so that a shutdown cancels work already in flight. WithinTx hands the callback a
// SrcRepo bound to one transaction.
type SrcRepo interface {
	FirstHeight(ctx context.Context, priceToken string) (int64, error)
	SrcHeight(ctx context.Context) (int64, error)
	NextHeight(ctx context.Context, minHeight uint64) (int64, error)
	Txs(ctx context.Context, height uint64) ([]schemas.ParsedTx, error)
	Decimals(ctx context.Context, asset string) (int64, error)
	LatestRouteUpdateTimestamp(ctx context.Context) (float64, error)
	Route(ctx context.Context, endToken string) (map[string][][]string, error)
	Liquidity(ctx context.Context, height uint64, token string, priceToken string) (string, string, error)
	UpdateDirectPrice(ctx context.Context, height uint64, txId uint64, token string, price string, priceToken string, isReverse bool) error
	UpdateRoutePrice(ctx context.Context, height uint64, txId uint64, token string, price string, priceToken string, route []string) error

	WithinTx(ctx context.Context, fn func(SrcRepo) error) error
	Close() error
}

var _ SrcRepo = &srcRepoImpl{}

type srcRepoImpl struct {
	db      *gorm.DB
	chainId string
}

func NewRepo(chainId string, dbConfig configs.RdbConfig) (SrcRepo, error) {
	gormDB, err := db.OpenGormPostgres(dbConfig)
	if err != nil {
		return nil, err
	}

	return &srcRepoImpl{
		db:      gormDB,
		chainId: chainId,
	}, nil
}

// conn binds the repository handle to ctx so every query is cancellable.
func (r *srcRepoImpl) conn(ctx context.Context) *gorm.DB { return r.db.WithContext(ctx) }

// WithinTx scopes every read and write in the callback to one transaction so a
// partially calculated height rolls back cleanly.
func (r *srcRepoImpl) WithinTx(ctx context.Context, fn func(SrcRepo) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&srcRepoImpl{db: tx, chainId: r.chainId})
	})
}

func (r *srcRepoImpl) Close() error {
	db, err := r.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (r *srcRepoImpl) FirstHeight(ctx context.Context, priceToken string) (int64, error) {
	height := NaValue
	tx := r.conn(ctx).Model(schemas.ParsedTx{}).Where(
		"chain_id = ? and (asset0 = ? or asset1 = ?)", r.chainId, priceToken, priceToken).Select(
		"coalesce(min(height), ?)", NaValue).Find(&height)
	if tx.Error != nil {
		return 0, errors.Wrap(tx.Error, "srcRepoImpl.FirstHeight")
	}

	return height, nil
}

func (r *srcRepoImpl) SrcHeight(ctx context.Context) (int64, error) {
	query := `
select coalesce(max(height), 0) from parsed_tx where chain_id = ?
`
	height := int64(0)
	if tx := r.conn(ctx).Raw(query, r.chainId).Find(&height); tx.Error != nil {
		return 0, errors.Wrap(tx.Error, "srcRepoImpl.SrcHeight")
	}

	return height, nil
}

func (r *srcRepoImpl) NextHeight(ctx context.Context, minHeight uint64) (int64, error) {
	query := `
select coalesce(min(pt.height), ?)
from parsed_tx pt
	left join ( -- include first provision
		select contract, min(height) height
		from parsed_tx
		where type = 'provide' and chain_id = ?
		group by contract) t on pt.contract = t.contract and pt.height = t.height
where pt.chain_id = ?
	and (pt.type = 'swap' or t.height is not null)
	and pt.height > (select coalesce(max(height), 0) from price where chain_id = ?)
	and pt.height > ?
`
	height := NaValue
	tx := r.conn(ctx).Raw(query, NaValue, r.chainId, r.chainId, r.chainId, minHeight).Find(&height)
	if tx.Error != nil {
		return 0, errors.Wrap(tx.Error, "srcRepoImpl.NextHeight")
	}

	return height, nil
}

func (r *srcRepoImpl) Txs(ctx context.Context, height uint64) ([]schemas.ParsedTx, error) {
	var res []schemas.ParsedTx
	tx := r.conn(ctx).Model(
		schemas.ParsedTx{}).Joins(
		"left join (select contract, min(height) height from parsed_tx where type = 'provide' and chain_id = ? group by contract) t "+ // include first provision
			"on parsed_tx.contract = t.contract and parsed_tx.height = t.height and parsed_tx.type = 'provide'", r.chainId).Where(
		"parsed_tx.chain_id = ? and parsed_tx.height = ? and (type = 'swap' or t.height is not null)",
		r.chainId, height).Order("parsed_tx.id asc").Find(&res)
	if tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "srcRepoImpl.Txs")
	}

	return res, nil
}

func (r *srcRepoImpl) Decimals(ctx context.Context, asset string) (int64, error) {
	var res int64
	tx := r.conn(ctx).Table("tokens").Select(
		"decimals").Where(
		"chain_id = ? and address = ?", r.chainId, asset).Find(&res)
	if tx.Error != nil {
		return 0, errors.Wrap(tx.Error, "srcRepoImpl.Decimals")
	}

	return res, nil
}

func (r *srcRepoImpl) LatestRouteUpdateTimestamp(ctx context.Context) (float64, error) {
	var ts float64
	// max, not min: routes are inserted with ON CONFLICT DO NOTHING, so only the
	// newest row says whether the router has published anything since the last reload
	if tx := r.conn(ctx).Model(schemas.Route{}).Where(
		"chain_id = ?", r.chainId).Select(
		"coalesce(max(created_at), 0)").Find(&ts); tx.Error != nil {
		return 0, errors.Wrap(tx.Error, "srcRepoImpl.LatestRouteUpdateTimestamp")
	}

	return ts, nil
}

func (r *srcRepoImpl) Route(ctx context.Context, endToken string) (map[string][][]string, error) {
	type result struct {
		Asset0 string
		Route  pq.StringArray `gorm:"column:route;type:text[]"`
	}
	var res []result
	tx := r.conn(ctx).Model(schemas.Route{}).Select(
		"asset0, route").Where(
		"chain_id = ? and asset1 = ?", r.chainId, endToken).Order(
		"hop_count asc").Find( // hop_count ordering is essential for routes comparison, refer to priceImpl.selectRoute(...)
		&res)
	if tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "srcRepoImpl.Route")
	}

	convertedRes := make(map[string][][]string)
	for _, r := range res {
		if convertedRes[r.Asset0] == nil {
			convertedRes[r.Asset0] = make([][]string, 0)
		}

		convertedRes[r.Asset0] = append(convertedRes[r.Asset0], r.Route)
	}

	return convertedRes, nil
}

func (r *srcRepoImpl) Liquidity(ctx context.Context, height uint64, token string, priceToken string) (string, string, error) {
	type result struct {
		Asset0     string
		Liquidity0 string
		Asset1     string
		Liquidity1 string
	}

	var res result
	tx := r.conn(ctx).Model(schemas.LpHistory{}).Joins(
		"join (select p.id pair_id, p.asset0, p.asset1, max(lh.height) height from lp_history lh join pair p on p.id = lh.pair_id "+
			"where lh.chain_id = ? and lh.height <= ? and ((p.asset0 = ? and p.asset1 = ?) or (p.asset0 = ? and p.asset1 = ?)) group by p.id) t "+
			"on lp_history.height = t.height and lp_history.pair_id = t.pair_id",
		r.chainId, height, token, priceToken, priceToken, token).Select("asset0, liquidity0, asset1, liquidity1").Find(&res)
	if tx.Error != nil {
		return "", "", errors.Wrap(tx.Error, "srcRepoImpl.Liquidity")
	}
	if res.Asset0 == token {
		return res.Liquidity0, res.Liquidity1, nil
	} else if res.Asset1 == token {
		return res.Liquidity1, res.Liquidity0, nil
	}

	return "0", "0", nil
}

func (r *srcRepoImpl) UpdateDirectPrice(ctx context.Context, height uint64, txId uint64, token string, price string, priceToken string, isReverse bool) error {
	type result struct {
		TokenId      uint64
		PriceTokenId uint64
		RouteId      uint64
	}

	var res result
	var tx *gorm.DB
	if isReverse {
		tx = r.conn(ctx).Table(
			"tokens").Select(
			"tokens.id token_id, tr.token_id price_token_id, tr.route_id").Joins(
			"join (select t.id token_id, t.chain_id, r.asset1, r.id route_id "+
				"from tokens t join route r on t.chain_id = r.chain_id and t.address = r.asset0 "+
				"where t.chain_id = ? and t.address = ? and r.hop_count = 0) tr "+
				"on tr.chain_id = tokens.chain_id and tr.asset1 = tokens.address", r.chainId, priceToken).Where(
			"tokens.address = ?", token).Find(&res)
	} else {
		tx = r.conn(ctx).Table(
			"tokens").Select(
			"tr.token_id, tokens.id price_token_id, tr.route_id").Joins(
			"join (select t.id token_id, t.chain_id, r.asset1, r.id route_id "+
				"from tokens t join route r on t.chain_id = r.chain_id and t.address = r.asset0 "+
				"where t.chain_id = ? and t.address = ? and r.hop_count = 0) tr "+
				"on tr.chain_id = tokens.chain_id and tr.asset1 = tokens.address", r.chainId, token).Where(
			"tokens.address = ?", priceToken).Find(&res)
	}
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "srcRepoImpl.UpdateDirectPrice")
	}

	tx = r.conn(ctx).Model(schemas.Price{}).Create(
		&schemas.Price{
			Height:       height,
			ChainId:      r.chainId,
			TokenId:      res.TokenId,
			Price:        price,
			PriceTokenId: res.PriceTokenId,
			RouteId:      res.RouteId,
			TxId:         txId})
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "srcRepoImpl.UpdateDirectPrice")
	}

	return nil
}

func (r *srcRepoImpl) UpdateRoutePrice(ctx context.Context, height uint64, txId uint64, token string, price string, priceToken string, route []string) error {
	type result struct {
		TokenId      uint64
		PriceTokenId uint64
		RouteId      uint64
	}

	var res result
	var tx *gorm.DB
	tx = r.conn(ctx).Table(
		"tokens").Select(
		"tr.token_id, tokens.id price_token_id, tr.route_id").Joins(
		"join (select t.id token_id, t.chain_id, r.asset1, r.id route_id "+
			"from tokens t join route r on t.chain_id = r.chain_id and t.address = r.asset0 "+
			"where t.chain_id = ? and t.address = ? and r.route = ?) tr "+
			"on tr.chain_id = tokens.chain_id and tr.asset1 = tokens.address", r.chainId, token, pq.StringArray(route)).Where(
		"tokens.address = ?", priceToken).Find(&res)

	if tx.Error != nil {
		return errors.Wrap(tx.Error, "srcRepoImpl.UpdateRoutePrice")
	}

	tx = r.conn(ctx).Model(schemas.Price{}).Create(
		&schemas.Price{
			Height:       height,
			ChainId:      r.chainId,
			TokenId:      res.TokenId,
			Price:        price,
			PriceTokenId: res.PriceTokenId,
			RouteId:      res.RouteId,
			TxId:         txId})
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "srcRepoImpl.UpdateRoutePrice")
	}

	return nil
}
