package repo

import (
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm/logger"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	Liquidity0 = 0 + iota
	Liquidity1
	Liquidity0InPrice
	Liquidity1InPrice
	TupleLength
)

var Logger logging.Logger

type Repo interface {
	LatestTimestamp(tableName string) (float64, error)
	LastHeightOfPairStatsRecent() (uint64, error)
	LastLpHistory(height uint64) ([]schemas.LpHistory, error)
	LastLiquidity(pairId uint64, timestamp float64) ([TupleLength]string, error)
	BeginTx() (*gorm.DB, error)
	UpdatePairStatsRecent(tx *gorm.DB, stats []schemas.PairStatsRecent) error
	UpdateLpHistory(history []schemas.LpHistory) error
	DeletePairStatsRecent(tx *gorm.DB, deleteBefore time.Time) error

	DeleteDuplicates(end time.Time) error
	UpdatePairStats(stats []schemas.PairStats30m) error
	UpdateAccountStats(stats []schemas.AccountStats30m) error
	CreateAccounts(addresses []string) error
	HoldingPairIds(accountId uint64) ([]uint64, error)
	Accounts(endTs float64) (map[uint64]string, error)
	Close() error
}

type repoImpl struct {
	db      *gorm.DB
	chainId string
}

func New(chainId string, dbConfig configs.RdbConfig) Repo {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Error, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,        // Disable color
		},
	)

	pq := db.PostgresDb{}
	err := pq.Init(dbConfig)
	if err != nil {
		panic(err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: pq.Db,
	}), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		panic(err)
	}

	Logger.Infof("Successfully connected to the database %s:%d/%s.", dbConfig.Host, dbConfig.Port, dbConfig.Database)

	return &repoImpl{
		db:      gormDB,
		chainId: chainId,
	}
}

func (r *repoImpl) Close() error {
	db, err := r.db.DB()
	if err != nil {
		return err
	}

	return db.Close()
}

func (r *repoImpl) LatestTimestamp(tableName string) (float64, error) {
	row := r.db.Table(tableName).Where("chain_id = ?", r.chainId).Select("coalesce(max(timestamp), 0)").Row()
	if err := row.Err(); err != nil {
		return 0, err
	}

	var ts float64
	if err := row.Scan(&ts); err != nil {
		return 0, err
	}

	return ts, nil
}

func (r *repoImpl) LastHeightOfPairStatsRecent() (uint64, error) {
	row := r.db.Model(schemas.PairStatsRecent{}).Where("chain_id = ?", r.chainId).Select("coalesce(max(height), 0)").Row()
	if err := row.Err(); err != nil {
		return 0, err
	}

	var height uint64
	if err := row.Scan(&height); err != nil {
		return 0, err
	}

	return height, nil
}

func (r *repoImpl) LastLpHistory(height uint64) ([]schemas.LpHistory, error) {
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
	if tx := r.db.Raw(query, r.chainId, height, r.chainId).Scan(&history); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.LastLpHistory")
	}

	return history, nil
}

func (r *repoImpl) LastLiquidity(pairId uint64, timestamp float64) ([TupleLength]string, error) {
	type result struct {
		Liquidity0 string
		Liquidity1 string
	}

	res := result{}
	if tx := r.db.Model(schemas.PairStats30m{}).Where(
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

func (r *repoImpl) BeginTx() (*gorm.DB, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.BeginTx")
	}

	return tx, nil
}

func (r *repoImpl) UpdatePairStatsRecent(tx *gorm.DB, stats []schemas.PairStatsRecent) error {
	tx = tx.Model(schemas.PairStatsRecent{}).CreateInBatches(stats, len(stats))
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "repo.UpdatePairStatsRecent")
	}

	return nil
}

func (r *repoImpl) UpdateLpHistory(history []schemas.LpHistory) error {
	if tx := r.db.Model(schemas.LpHistory{}).CreateInBatches(&history, len(history)); tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repoImpl) DeletePairStatsRecent(tx *gorm.DB, deleteBefore time.Time) error {
	tx = tx.Where(
		"timestamp < ?", deleteBefore.Unix()).Delete(
		&schemas.PairStatsRecent{})
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "repo.DeletePairStatsRecent")
	}

	return nil
}

func (r *repoImpl) DeleteDuplicates(ts time.Time) error {
	if tx := r.db.Where("timestamp >= ? and chain_id = ?", util.ToEpoch(ts), r.chainId).Delete(&schemas.LpHistory{}); tx.Error != nil {
		return tx.Error
	}
	if tx := r.db.Where("height >= (select min(height) from parsed_tx where timestamp >= ?) and chain_id = ?", util.ToEpoch(ts), r.chainId).Delete(&schemas.Price{}); tx.Error != nil {
		return tx.Error
	}
	if tx := r.db.Where("timestamp >= ? and chain_id = ?", util.ToEpoch(ts), r.chainId).Delete(&schemas.PairStatsRecent{}); tx.Error != nil {
		return tx.Error
	}
	end := ts.Truncate(30 * time.Minute).Add(30 * time.Minute).UTC()
	if tx := r.db.Where("timestamp >= ? and chain_id = ?", util.ToEpoch(end), r.chainId).Delete(&schemas.PairStats30m{}); tx.Error != nil {
		return tx.Error
	}
	if tx := r.db.Where("timestamp >= ? and chain_id = ?", util.ToEpoch(end), r.chainId).Delete(&schemas.AccountStats30m{}); tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repoImpl) UpdatePairStats(stats []schemas.PairStats30m) error {
	if tx := r.db.Omit("Id", "CreatedAt").Create(&stats); tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repoImpl) UpdateAccountStats(stats []schemas.AccountStats30m) error {
	if tx := r.db.Omit("Id", "CreatedAt").Create(&stats); tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repoImpl) CreateAccounts(addresses []string) error {
	db, err := r.db.DB()
	if err != nil {
		return err
	}

	sql := `
INSERT INTO account(address) VALUES($1) ON CONFLICT DO NOTHING
`
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, address := range addresses {
		if _, err := tx.Exec(sql, address); err != nil {
			if e := tx.Rollback(); e != nil { // lint issue, usually do not check this return
				return e
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (r *repoImpl) HoldingPairIds(accountId uint64) ([]uint64, error) {
	query := `
SELECT pair_id
FROM (
    SELECT pair_id, SUM(total_lp_amount) stla
    FROM h_account_stats_30m
    WHERE chain_id = $1
      AND account_id = $2
    GROUP BY pair_id) t
WHERE stla > 0
`
	db, err := r.db.DB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(query, r.chainId, accountId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pairIds := []uint64{}
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		pairIds = append(pairIds, id)
	}

	return pairIds, nil
}

func (r *repoImpl) Accounts(endTs float64) (map[uint64]string, error) {
	query := `
SELECT id, address
FROM account
WHERE id IN (
    SELECT t.account_id
    FROM (SELECT account_id, SUM(total_lp_amount) tla_sum
    	  FROM h_account_stats_30m
          WHERE chain_id = $1
          GROUP BY account_id) t
    WHERE t.tla_sum > 0
    )
  OR created_at >= $2
`
	db, err := r.db.DB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(query, r.chainId, endTs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make(map[uint64]string)
	for rows.Next() {
		var id uint64
		var address string
		if err := rows.Scan(&id, &address); err != nil {
			return nil, err
		}

		accounts[id] = address
	}

	return accounts, nil
}
