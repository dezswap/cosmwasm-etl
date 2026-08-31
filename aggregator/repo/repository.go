package repo

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm/clause"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
	"gorm.io/gorm"
)

const (
	Liquidity0 = 0 + iota
	Liquidity1
	Liquidity0InPrice
	Liquidity1InPrice
	TupleLength
)

// InsertBatchSize caps every slice INSERT here, keeping one statement under postgres'
// 65535 bind parameter limit and bounding what GORM builds in memory per round trip.
const InsertBatchSize = 1000

type Repo interface {
	LatestTimestamp(ctx context.Context, tableName string) (float64, error)
	LastHeightOfPairStatsRecent(ctx context.Context) (uint64, error)
	LastLpHistory(ctx context.Context, height uint64) ([]schemas.LpHistory, error)
	LastLiquidity(ctx context.Context, pairId uint64, timestamp float64) ([TupleLength]string, error)
	UpdatePairStatsRecent(ctx context.Context, stats []schemas.PairStatsRecent) error
	UpdateLpHistory(ctx context.Context, history []schemas.LpHistory) error
	DeletePairStatsRecent(ctx context.Context, deleteBefore time.Time) error

	DeleteDuplicates(ctx context.Context, end time.Time) error
	UpdatePairStats(ctx context.Context, stats []schemas.PairStats30m) error
	UpdateAccountStats(ctx context.Context, stats []schemas.AccountStats30m) error
	CreateAccounts(ctx context.Context, addresses []string) error
	AccountIds(ctx context.Context, addresses []string) (map[string]uint64, error)
	HoldingPairIds(ctx context.Context, accountId uint64) ([]uint64, error)
	Accounts(ctx context.Context, endTs float64) (map[uint64]string, error)

	WithinTx(ctx context.Context, fn func(Repo) error) error
	Close() error
}

type repoImpl struct {
	db      *gorm.DB
	chainId string
}

var _ Repo = &repoImpl{}

func New(chainId string, dbConfig configs.RdbConfig) (Repo, error) {
	gormDB, err := db.OpenGormPostgres(dbConfig)
	if err != nil {
		return nil, err
	}

	return &repoImpl{
		db:      gormDB,
		chainId: chainId,
	}, nil
}

// conn binds the repository handle to ctx so every query is cancellable.
func (r *repoImpl) conn(ctx context.Context) *gorm.DB { return r.db.WithContext(ctx) }

// WithinTx gives the callback a Repo bound to one transaction; GORM rolls back
// when the callback fails or the context is canceled.
func (r *repoImpl) WithinTx(ctx context.Context, fn func(Repo) error) error {
	return r.conn(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&repoImpl{db: tx, chainId: r.chainId})
	})
}

func (r *repoImpl) Close() error {
	db, err := r.db.DB()
	if err != nil {
		return err
	}

	return db.Close()
}

func (r *repoImpl) LatestTimestamp(ctx context.Context, tableName string) (float64, error) {
	row := r.conn(ctx).Table(tableName).Where("chain_id = ?", r.chainId).Select("coalesce(max(timestamp), 0)").Row()
	if err := row.Err(); err != nil {
		return 0, err
	}

	var ts float64
	if err := row.Scan(&ts); err != nil {
		return 0, err
	}

	return ts, nil
}

func (r *repoImpl) LastHeightOfPairStatsRecent(ctx context.Context) (uint64, error) {
	row := r.conn(ctx).Model(schemas.PairStatsRecent{}).Where("chain_id = ?", r.chainId).Select("coalesce(max(height), 0)").Row()
	if err := row.Err(); err != nil {
		return 0, err
	}

	var height uint64
	if err := row.Scan(&height); err != nil {
		return 0, err
	}

	return height, nil
}

func (r *repoImpl) LastLpHistory(ctx context.Context, height uint64) ([]schemas.LpHistory, error) {
	query := `
select lh.height,
       lh.pair_id,
       lh.liquidity0,
       lh.liquidity1
from lp_history lh
	 join (select pair_id, max(height) height
	       from lp_history
	       where chain_id = ? and height <= ?
	       group by pair_id) t on lh.height = t.height and lh.pair_id = t.pair_id
where chain_id = ?
order by lh.height asc
`
	history := []schemas.LpHistory{}
	if tx := r.conn(ctx).Raw(query, r.chainId, height, r.chainId).Scan(&history); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.LastLpHistory")
	}

	return history, nil
}

func (r *repoImpl) LastLiquidity(ctx context.Context, pairId uint64, timestamp float64) ([TupleLength]string, error) {
	type result struct {
		Liquidity0 string
		Liquidity1 string
	}

	res := result{}
	if tx := r.conn(ctx).Model(schemas.PairStats30m{}).Where(
		"pair_id = ? and timestamp = (select max(timestamp) from pair_stats_30m where pair_id = ? and timestamp <= ?)", pairId, pairId, timestamp).Select(
		"liquidity0, liquidity1, liquidity0_in_price, liquidity1_in_price").Find(&res); tx.Error != nil {
		return [TupleLength]string{}, errors.Wrap(tx.Error, "LastLiquidity")
	}

	pairLiquidity := [TupleLength]string{"0", "0", "0", "0"}
	if len(res.Liquidity0) > 0 {
		pairLiquidity[Liquidity0] = res.Liquidity0
	}
	if len(res.Liquidity1) > 0 {
		pairLiquidity[Liquidity1] = res.Liquidity1
	}

	return pairLiquidity, nil
}

func (r *repoImpl) UpdatePairStatsRecent(ctx context.Context, stats []schemas.PairStatsRecent) error {
	if len(stats) == 0 {
		return nil
	}

	tx := r.conn(ctx).Model(schemas.PairStatsRecent{}).CreateInBatches(stats, InsertBatchSize)
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "repo.UpdatePairStatsRecent")
	}

	return nil
}

func (r *repoImpl) UpdateLpHistory(ctx context.Context, history []schemas.LpHistory) error {
	if len(history) == 0 {
		return nil
	}

	if tx := r.conn(ctx).Model(schemas.LpHistory{}).CreateInBatches(&history, InsertBatchSize); tx.Error != nil {
		return tx.Error
	}

	return nil
}

// DeletePairStatsRecent prunes rows that fell out of the trailing window.
func (r *repoImpl) DeletePairStatsRecent(ctx context.Context, deleteBefore time.Time) error {
	tx := r.conn(ctx).Where(
		"timestamp < ? and chain_id = ?", util.ToEpoch(deleteBefore), r.chainId).Delete(
		&schemas.PairStatsRecent{})
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "repo.DeletePairStatsRecent")
	}

	return nil
}

// DeleteDuplicates removes every derived record at or after ts, spanning its own
// transaction so a failure or cancellation cannot leave only some of the five
// aggregate tables purged. Callers need no wrapper; running it inside WithinTx is
// still safe, since gorm nests the inner transaction as a savepoint.
func (r *repoImpl) DeleteDuplicates(ctx context.Context, ts time.Time) error {
	end := ts.Truncate(30 * time.Minute).Add(30 * time.Minute).UTC()

	return r.conn(ctx).Transaction(func(tx *gorm.DB) error {
		if result := tx.Where("timestamp >= ? and chain_id = ?", util.ToEpoch(ts), r.chainId).Delete(&schemas.LpHistory{}); result.Error != nil {
			return result.Error
		}
		if result := tx.Where("height >= (select min(height) from parsed_tx where timestamp >= ?) and chain_id = ?", util.ToEpoch(ts), r.chainId).Delete(&schemas.Price{}); result.Error != nil {
			return result.Error
		}
		if result := tx.Where("timestamp >= ? and chain_id = ?", util.ToEpoch(ts), r.chainId).Delete(&schemas.PairStatsRecent{}); result.Error != nil {
			return result.Error
		}
		if result := tx.Where("timestamp >= ? and chain_id = ?", util.ToEpoch(end), r.chainId).Delete(&schemas.PairStats30m{}); result.Error != nil {
			return result.Error
		}
		if result := tx.Where("timestamp >= ? and chain_id = ?", util.ToEpoch(end), r.chainId).Delete(&schemas.AccountStats30m{}); result.Error != nil {
			return result.Error
		}

		return nil
	})
}

func (r *repoImpl) UpdatePairStats(ctx context.Context, stats []schemas.PairStats30m) error {
	if len(stats) == 0 {
		return nil
	}

	if tx := r.conn(ctx).Omit("Id", "CreatedAt").CreateInBatches(&stats, InsertBatchSize); tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repoImpl) UpdateAccountStats(ctx context.Context, stats []schemas.AccountStats30m) error {
	if len(stats) == 0 {
		return nil
	}

	tx := r.conn(ctx).Omit("Id", "CreatedAt").Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chain_id"},
			{Name: "timestamp"},
			{Name: "account_id"},
			{Name: "pair_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"year_utc":                gorm.Expr("excluded.year_utc"),
			"month_utc":               gorm.Expr("excluded.month_utc"),
			"day_utc":                 gorm.Expr("excluded.day_utc"),
			"hour_utc":                gorm.Expr("excluded.hour_utc"),
			"minute_utc":              gorm.Expr("excluded.minute_utc"),
			"address":                 gorm.Expr("excluded.address"),
			"tx_cnt":                  gorm.Expr("excluded.tx_cnt"),
			"swap_tx_cnt":             gorm.Expr("excluded.swap_tx_cnt"),
			"provide_tx_cnt":          gorm.Expr("excluded.provide_tx_cnt"),
			"withdraw_tx_cnt":         gorm.Expr("excluded.withdraw_tx_cnt"),
			"swap_volume_in_price":    gorm.Expr("excluded.swap_volume_in_price"),
			"provide_value_in_price":  gorm.Expr("excluded.provide_value_in_price"),
			"withdraw_value_in_price": gorm.Expr("excluded.withdraw_value_in_price"),
			"net_flow_in_price":       gorm.Expr("excluded.net_flow_in_price"),
			"price_token":             gorm.Expr("excluded.price_token"),
			"net_asset0_amount":       gorm.Expr("excluded.net_asset0_amount"),
			"net_asset1_amount":       gorm.Expr("excluded.net_asset1_amount"),
			"net_lp_amount":           gorm.Expr("excluded.net_lp_amount"),
			"modified_at":             gorm.Expr("date_part('epoch'::text, now())"),
		}),
	}).CreateInBatches(&stats, InsertBatchSize)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// CreateAccounts inserts through r.db so it joins an enclosing WithinTx instead of
// running on a separate connection. Ids and created_at are left to the database,
// so callers must resolve ids with AccountIds rather than reading them back.
func (r *repoImpl) CreateAccounts(ctx context.Context, addresses []string) error {
	if len(addresses) == 0 {
		return nil
	}

	accounts := make([]schemas.Account, 0, len(addresses))
	for _, address := range addresses {
		accounts = append(accounts, schemas.Account{Address: address})
	}

	if tx := r.conn(ctx).Omit("Id", "CreatedAt").Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(accounts, InsertBatchSize); tx.Error != nil {
		return errors.Wrap(tx.Error, "repo.CreateAccounts")
	}

	return nil
}

func (r *repoImpl) AccountIds(ctx context.Context, addresses []string) (map[string]uint64, error) {
	if len(addresses) == 0 {
		return map[string]uint64{}, nil
	}

	var accounts []schemas.Account
	if tx := r.conn(ctx).Model(schemas.Account{}).Where("address in ?", addresses).Find(&accounts); tx.Error != nil {
		return nil, tx.Error
	}

	accountIds := make(map[string]uint64, len(accounts))
	for _, account := range accounts {
		accountIds[account.Address] = account.Id
	}

	return accountIds, nil
}

func (r *repoImpl) HoldingPairIds(ctx context.Context, accountId uint64) ([]uint64, error) {
	query := `
SELECT pair_id
FROM (
    SELECT pair_id, SUM(net_lp_amount) stla
    FROM account_stats_30m
    WHERE chain_id = ?
      AND account_id = ?
    GROUP BY pair_id) t
WHERE stla > 0
`
	pairIds := []uint64{}
	if tx := r.conn(ctx).Raw(query, r.chainId, accountId).Scan(&pairIds); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.HoldingPairIds")
	}

	return pairIds, nil
}

func (r *repoImpl) Accounts(ctx context.Context, endTs float64) (map[uint64]string, error) {
	query := `
SELECT id, address
FROM account
WHERE id IN (
    SELECT t.account_id
    FROM (SELECT account_id, SUM(net_lp_amount) tla_sum
    	  FROM account_stats_30m
          WHERE chain_id = ?
          GROUP BY account_id) t
    WHERE t.tla_sum > 0
    )
  OR created_at >= ?
`
	rows := []schemas.Account{}
	if tx := r.conn(ctx).Raw(query, r.chainId, endTs).Scan(&rows); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.Accounts")
	}

	accounts := make(map[uint64]string, len(rows))
	for _, account := range rows {
		accounts[account.Id] = account.Address
	}

	return accounts, nil
}
