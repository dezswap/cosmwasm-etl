package repo

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm/clause"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
	"gorm.io/gorm"
)

// Indices into the liquidity pair the lp history task carries per pair.
const (
	Liquidity0 = 0 + iota
	Liquidity1
)

// InsertBatchSize caps every slice INSERT here, keeping one statement under postgres'
// 65535 bind parameter limit and bounding what GORM builds in memory per round trip.
const InsertBatchSize = 1000

type Repo interface {
	LatestTimestamp(ctx context.Context, tableName string) (float64, error)
	LastHeightOfPairStatsRecent(ctx context.Context) (uint64, error)
	LastLpHistory(ctx context.Context, height uint64) ([]schemas.LpHistory, error)
	UpdatePairStatsRecent(ctx context.Context, stats []schemas.PairStatsRecent) error
	UpdateLpHistory(ctx context.Context, history []schemas.LpHistory) error
	DeletePairStatsRecent(ctx context.Context, deleteBefore time.Time) error

	DeleteDuplicates(ctx context.Context, end time.Time) error
	// LatestPairStat returns the newest row written for a pair on this chain strictly
	// before the given timestamp. The bound keeps a rerun of an older window from
	// reading back a row that belongs to a later one. The bool reports whether such a
	// row exists at all; on false the row is a zero value whose amounts are empty
	// strings, which the numeric columns reject, so the caller has to supply its own
	// defaults rather than pass the row on.
	LatestPairStat(ctx context.Context, pairId uint64, before float64) (schemas.PairStats30m, bool, error)
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

func (r *repoImpl) LatestPairStat(ctx context.Context, pairId uint64, before float64) (schemas.PairStats30m, bool, error) {
	stat := schemas.PairStats30m{}
	// a window now holds at most one row per pair, but rows written before the unique
	// index existed may still be doubled up, and only the id tells the newest apart
	tx := r.conn(ctx).Model(schemas.PairStats30m{}).Where(
		"chain_id = ? and pair_id = ? and timestamp < ?", r.chainId, pairId, before).Order(
		"timestamp desc, id desc").Limit(1).Find(&stat)
	if tx.Error != nil {
		return schemas.PairStats30m{}, false, errors.Wrap(tx.Error, "repo.LatestPairStat")
	}

	return stat, tx.RowsAffected > 0, nil
}

func (r *repoImpl) UpdatePairStats(ctx context.Context, stats []schemas.PairStats30m) error {
	if len(stats) == 0 {
		return nil
	}

	if err := r.validatePairStats(stats); err != nil {
		return err
	}

	tx := r.conn(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chain_id"},
			{Name: "timestamp"},
			{Name: "pair_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"year_utc":             gorm.Expr("excluded.year_utc"),
			"month_utc":            gorm.Expr("excluded.month_utc"),
			"day_utc":              gorm.Expr("excluded.day_utc"),
			"hour_utc":             gorm.Expr("excluded.hour_utc"),
			"minute_utc":           gorm.Expr("excluded.minute_utc"),
			"volume0":              gorm.Expr("excluded.volume0"),
			"volume1":              gorm.Expr("excluded.volume1"),
			"volume0_in_price":     gorm.Expr("excluded.volume0_in_price"),
			"volume1_in_price":     gorm.Expr("excluded.volume1_in_price"),
			"last_swap_price":      gorm.Expr("excluded.last_swap_price"),
			"liquidity0":           gorm.Expr("excluded.liquidity0"),
			"liquidity1":           gorm.Expr("excluded.liquidity1"),
			"liquidity0_in_price":  gorm.Expr("excluded.liquidity0_in_price"),
			"liquidity1_in_price":  gorm.Expr("excluded.liquidity1_in_price"),
			"commission0":          gorm.Expr("excluded.commission0"),
			"commission1":          gorm.Expr("excluded.commission1"),
			"commission0_in_price": gorm.Expr("excluded.commission0_in_price"),
			"commission1_in_price": gorm.Expr("excluded.commission1_in_price"),
			"price_token":          gorm.Expr("excluded.price_token"),
			"tx_cnt":               gorm.Expr("excluded.tx_cnt"),
			"provider_cnt":         gorm.Expr("excluded.provider_cnt"),
			"modified_at":          gorm.Expr("date_part('epoch'::text, now())"),
		}),
	}).CreateInBatches(&stats, InsertBatchSize)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// pairStatAmounts pairs every numeric column of pair_stats_30m with the field filling
// it. Kept package level so the check below allocates nothing per row.
var pairStatAmounts = []struct {
	column string
	of     func(*schemas.PairStats30m) string
}{
	{"volume0", func(s *schemas.PairStats30m) string { return s.Volume0 }},
	{"volume1", func(s *schemas.PairStats30m) string { return s.Volume1 }},
	{"volume0_in_price", func(s *schemas.PairStats30m) string { return s.Volume0InPrice }},
	{"volume1_in_price", func(s *schemas.PairStats30m) string { return s.Volume1InPrice }},
	{"last_swap_price", func(s *schemas.PairStats30m) string { return s.LastSwapPrice }},
	{"liquidity0", func(s *schemas.PairStats30m) string { return s.Liquidity0 }},
	{"liquidity1", func(s *schemas.PairStats30m) string { return s.Liquidity1 }},
	{"liquidity0_in_price", func(s *schemas.PairStats30m) string { return s.Liquidity0InPrice }},
	{"liquidity1_in_price", func(s *schemas.PairStats30m) string { return s.Liquidity1InPrice }},
	{"commission0", func(s *schemas.PairStats30m) string { return s.Commission0 }},
	{"commission1", func(s *schemas.PairStats30m) string { return s.Commission1 }},
	{"commission0_in_price", func(s *schemas.PairStats30m) string { return s.Commission0InPrice }},
	{"commission1_in_price", func(s *schemas.PairStats30m) string { return s.Commission1InPrice }},
}

// pairStatWindow keys a row the way pair_stats_30m's unique index does.
type pairStatWindow struct {
	pairId    uint64
	timestamp float64
}

// validatePairStats rejects rows the writer must not send. The insert files a row under
// the chain id the row carries, not the one this repo is scoped to, so a foreign chain
// id would land as a perfectly valid row under the wrong key and postgres would never
// complain. An empty amount and a repeated window it does reject, but without naming the
// pair or the column.
func (r *repoImpl) validatePairStats(stats []schemas.PairStats30m) error {
	seen := make(map[pairStatWindow]struct{}, len(stats))

	for i := range stats {
		stat := &stats[i]
		if stat.ChainId != r.chainId {
			return errors.Errorf(
				"repo.UpdatePairStats: pair %d on timestamp %.0f belongs to %s, not %s",
				stat.PairId, stat.Timestamp, stat.ChainId, r.chainId)
		}

		for _, amount := range pairStatAmounts {
			if strings.TrimSpace(amount.of(stat)) == "" {
				return errors.Errorf(
					"repo.UpdatePairStats: empty %s for pair %d of %s on timestamp %.0f",
					amount.column, stat.PairId, stat.ChainId, stat.Timestamp)
			}
		}

		window := pairStatWindow{stat.PairId, stat.Timestamp}
		if _, duplicated := seen[window]; duplicated {
			return errors.Errorf(
				"repo.UpdatePairStats: duplicate row for pair %d of %s on timestamp %.0f",
				stat.PairId, stat.ChainId, stat.Timestamp)
		}
		seen[window] = struct{}{}
	}

	return nil
}

func (r *repoImpl) UpdateAccountStats(ctx context.Context, stats []schemas.AccountStats30m) error {
	if len(stats) == 0 {
		return nil
	}

	tx := r.conn(ctx).Clauses(clause.OnConflict{
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
