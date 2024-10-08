package parser

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/util"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/pkg/errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ReadRepository interface {
	GetSyncedHeight() (uint64, error)
	GetPairs() ([]schemas.Pair, error)
	GetPoolInfosByHeight(height uint64) ([]schemas.PoolInfo, error)
	GetParsedTxs(height uint64) ([]schemas.ParsedTx, error)
	GetParsedTxsOfPair(height uint64, pair string) ([]schemas.ParsedTx, error)

	// aggregator
	HeightOnTimestamp(timestamp float64) (uint64, error)
	LastHeightOfPrice() (uint64, error)
	GetParsedTxsWithLimit(startHeight uint64, limit int) ([]schemas.ParsedTxWithPrice, error)
	GetRecentParsedTxs(startHeight uint64, endHeight uint64) ([]schemas.ParsedTxWithPrice, error)
	RecentPrices(startHeight uint64, endHeight uint64, targetTokens []string, priceToken string) (map[uint64][]schemas.Price, error)
	GetParsedTxsWithPriceOfPair(pairId uint64, priceToken string, startTs float64, endTs float64) ([]schemas.ParsedTxWithPrice, error)
	PairStats(startTs float64, endTs float64, priceToken string, prevStatsMap map[uint64]schemas.PairStats30m) ([]schemas.PairStats30m, error)
	AccountStats(startTs float64, endTs float64) ([]schemas.AccountStats30m, error)
	LiquiditiesOfPairStats(startTs float64, endTs float64, priceToken string) (map[uint64]schemas.PairStats30m, error)
	OldestTxTimestamp() (float64, error)
	LatestTxTimestamp() (float64, error)
	PairIds() ([]uint64, error)
	NewPairIds(account string, startTs float64, endTs float64) ([]uint64, error)
	NewAccounts(startTs float64, endTs float64) ([]string, error)
	ProviderCount(pairId uint64, startTs float64, endTs float64) (uint64, error)
	TxCountOfAccount(account string, pairId uint64, startTs float64, endTs float64) (uint64, error)
	AssetAmountInPair(pairId uint64, startTs float64, endTs float64) (string, string, string, error)
	AssetAmountInPairOfAccount(account string, pairId uint64, startTs float64, endTs float64) (string, string, string, error)
	CommissionAmountInPair(pairId uint64, startTs float64, endTs float64) (string, string, error)
	Close() error
}

type readRepoImpl struct {
	db      *gorm.DB
	chainId string
}

var _ ReadRepository = &readRepoImpl{}

func NewReadRepo(chainId string, dbConfig configs.RdbConfig) ReadRepository {
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

	return &readRepoImpl{
		db:      gormDB,
		chainId: chainId,
	}
}

// GetSyncedHeight implements parser.Repo
func (r *readRepoImpl) GetSyncedHeight() (uint64, error) {
	syncedHeight := schemas.SyncedHeight{}
	tx := r.db.Select("height").Where("chain_id = ?", r.chainId).First(&syncedHeight)

	if tx.Error != nil {
		if !strings.Contains(tx.Error.Error(), "not found") {
			return 0, errors.Wrap(tx.Error, "repo.GetSyncedHeight")
		}

		if err := r.db.Model(&schemas.SyncedHeight{}).Create(&schemas.SyncedHeight{ChainId: r.chainId, Height: 0}); err.Error != nil {
			return 0, errors.Wrap(err.Error, "repo.GetSyncedHeight")
		}
	}
	return syncedHeight.Height, nil
}

// GetPairs implements parser.Repo
func (r *readRepoImpl) GetPairs() ([]schemas.Pair, error) {
	pairs := []schemas.Pair{}
	tx := r.db.Where(schemas.Pair{ChainId: r.chainId}).Find(&pairs)
	if tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetPairs")
	}
	return pairs, nil
}

// GetPairs implements parser.Repo
func (r *readRepoImpl) GetPoolInfosByHeight(height uint64) ([]schemas.PoolInfo, error) {
	poolInfo := []schemas.PoolInfo{}
	if tx := r.db.Where(schemas.PoolInfo{ChainId: r.chainId, Height: height}).Find(&poolInfo); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetPairs")
	}
	return poolInfo, nil
}

func (r *readRepoImpl) GetParsedTxs(height uint64) ([]schemas.ParsedTx, error) {
	parsedTxs := []schemas.ParsedTx{}
	if tx := r.db.Where(schemas.ParsedTx{ChainId: r.chainId, Height: height}).Find(&parsedTxs); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetParsedTxs")
	}
	return parsedTxs, nil
}

func (r *readRepoImpl) GetParsedTxsOfPair(height uint64, pair string) ([]schemas.ParsedTx, error) {
	parsedTxs := []schemas.ParsedTx{}
	if tx := r.db.Where(schemas.PoolInfo{ChainId: r.chainId, Height: height, Contract: pair}).Find(&parsedTxs); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetParsedTxs")
	}
	return parsedTxs, nil
}

func (r *readRepoImpl) HeightOnTimestamp(timestamp float64) (uint64, error) {
	var height uint64
	if tx := r.db.Model(schemas.ParsedTx{}).Where(
		"chain_id = ? and timestamp <= ?", r.chainId, timestamp).Select("coalesce(max(height), 0)").Find(&height); tx.Error != nil {
		return 0, errors.Wrap(tx.Error, "repo.HeightOnTimestamp")
	}

	return height, nil
}

func (r *readRepoImpl) LastHeightOfPrice() (uint64, error) {
	row := r.db.Model(schemas.Price{}).Where("chain_id = ?", r.chainId).Select("coalesce(max(height), 0)").Row()
	if err := row.Err(); err != nil {
		return 0, err
	}

	var height uint64
	if err := row.Scan(&height); err != nil {
		return 0, err
	}

	return height, nil
}

func (r *readRepoImpl) GetParsedTxsWithLimit(startHeight uint64, limit int) ([]schemas.ParsedTxWithPrice, error) {
	query := `
select p.id pair_id, pt.chain_id, pt.asset0_amount, pt.asset1_amount,
       pt.commission0_amount, pt.commission1_amount, pt.height, pt.timestamp
from parsed_tx pt join pair p on pt.chain_id = p.chain_id and pt.contract = p.contract
where pt.chain_id = ?
  and pt.height >= ?
  and pt.height <= (
    select max(height) from (
      select height
      from parsed_tx
      where chain_id = ? and height >= ?
      order by height limit ?) t)
order by pt.height asc, p.id asc
`
	var res []schemas.ParsedTxWithPrice
	if tx := r.db.Raw(query, r.chainId, startHeight, r.chainId, startHeight, limit).Scan(&res); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetParsedTxsWithLimit")
	}

	return res, nil
}

// GetRecentParsedTxs return value is ordered by ascending height
func (r *readRepoImpl) GetRecentParsedTxs(startHeight uint64, endHeight uint64) ([]schemas.ParsedTxWithPrice, error) {
	query := `
select p.id pair_id,
       case when pt.type = 'swap' then pt.asset0_amount else 0 end as asset0_amount,
       case when pt.type = 'swap' then pt.asset1_amount else 0 end as asset1_amount,
       lh.liquidity0 asset0_liquidity,
       lh.liquidity1 asset1_liquidity,
       pt.commission0_amount,
       pt.commission1_amount,
       t0.id price0, -- FIXME: use mismatch field
       t1.id price1,
       t0.decimals decimals0,
       t1.decimals decimals1,
       pt.height,
       pt.timestamp
from parsed_tx pt
     join pair p on pt.chain_id = p.chain_id and pt.contract = p.contract
     join lp_history lh on p.id = lh.pair_id and pt.height = lh.height
     join tokens t0 on pt.chain_id = t0.chain_id and pt.asset0 = t0.address
     join tokens t1 on pt.chain_id = t1.chain_id and pt.asset1 = t1.address
where pt.chain_id = ?
  and pt.height >= ?
  and pt.height <= ?
  and pt.type in ('swap', 'provide', 'withdraw')
`
	res := []schemas.ParsedTxWithPrice{}
	if tx := r.db.Raw(query, r.chainId, startHeight, endHeight).Scan(&res); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetRecentParsedTxs")
	}

	return res, nil
}

func (r *readRepoImpl) RecentPrices(startHeight uint64, endHeight uint64, targetTokens []string, priceToken string) (map[uint64][]schemas.Price, error) {
	query := `
select 0 height,
       id as token_id,
       1 as price
from tokens where address = ?
union
select p.height,
       p.token_id,
       p.price
from price p
    join (
    	select token_id, max(height) height
    	from price p join tokens t on t.id = p.token_id
    	where p.chain_id = ?
    	  and p.height <= ?
    	  %s
    	group by p.token_id) t on p.token_id = t.token_id and p.height >= t.height
where p.height <= ?
order by height
`
	query = fmt.Sprintf(query, "and t.id in ("+strings.Join(targetTokens, ",")+")")
	var res []schemas.Price
	if tx := r.db.Raw(query, priceToken, r.chainId, startHeight, endHeight).Scan(&res); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.RecentPrices")
	}

	priceMap := make(map[uint64][]schemas.Price)
	for _, r := range res {
		if p, ok := priceMap[r.TokenId]; ok {
			priceMap[r.TokenId] = append(p, r)
		} else {
			priceMap[r.TokenId] = []schemas.Price{r}
		}
	}

	return priceMap, nil
}

func (r *readRepoImpl) GetParsedTxsWithPriceOfPair(pairId uint64, priceToken string, startTs float64, endTs float64) ([]schemas.ParsedTxWithPrice, error) {
	res := []schemas.ParsedTxWithPrice{}

	if tx := r.db.Model(schemas.ParsedTx{}).Joins(
		"join pair p on parsed_tx.chain_id = p.chain_id and parsed_tx.contract = p.contract "+
			"join tokens t0 on parsed_tx.chain_id = t0.chain_id and parsed_tx.asset0 = t0.address "+
			"join tokens t1 on parsed_tx.chain_id = t1.chain_id and parsed_tx.asset1 = t1.address "+
			"left outer join price p0 on t0.id = p0.token_id and p0.height <= parsed_tx.height "+
			"left outer join price p1 on t1.id = p1.token_id and p1.height <= parsed_tx.height").Where(
		"parsed_tx.chain_id = ? and p.id = ? and parsed_tx.timestamp >= ? and parsed_tx.timestamp < ? and type in ('swap', 'provide', 'withdraw')", r.chainId, pairId, startTs, endTs).Order(
		"parsed_tx.height, p0.height desc, p1.height desc").Select(
		"distinct on (parsed_tx.height) parsed_tx.asset0_amount, parsed_tx.asset1_amount,"+
			"parsed_tx.commission0_amount, parsed_tx.commission1_amount,"+
			"CASE WHEN t0.address = ? THEN '1' ELSE coalesce(p0.price, '0') END price0, CASE WHEN t1.address = ? THEN '1' ELSE coalesce(p1.price, '0') END price1,"+
			"t0.decimals decimals0, t1.decimals decimals1",
		priceToken, priceToken).Find(&res); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetParsedTxsWithPriceOfPair")
	}

	return res, nil
}

func (r *readRepoImpl) PairStats(startTs float64, endTs float64, priceToken string, prevStatsMap map[uint64]schemas.PairStats30m) (stats []schemas.PairStats30m, err error) {
	query := `
select -- asset0's stats by pairs
    pair_id,
    coalesce(sum(volume) filter (where type = 'swap'),0) as volume0,
    coalesce(sum(volume_in_price) filter (where type = 'swap'),0) as volume0_in_price,
    avg(last_volume) as last_swap_price,
    sum(commission) as commission0,
    sum(commission_in_price) as commission0_in_price,
    count(distinct hash) as tx_cnt,
    count(distinct sender) filter (where type = 'provide') as provider_cnt
from (select distinct -- processed asset0 values
          height,
          pair_id,
          hash,
          sender,
          type,
          first_value(volume / pow(10, decimals)) over (partition by pair_id order by height desc) last_volume,
          abs(volume) as volume,
          abs(volume) * price / pow(10, decimals) as volume_in_price,
          commission,
          abs(commission) * price / pow(10, decimals) as commission_in_price
      from (select -- txs' asset0 values in a specific time range
                pt.height,
                p.id pair_id,
                pt.hash,
                pt.sender,
                pt.type,
                pt.asset0_amount as volume,
                pt.commission0_amount as commission,
                coalesce(first_value(pr.price) over (partition by pt.height order by pr.height desc), 0) as price,
                t.decimals
            from parsed_tx pt
                join pair p on pt.chain_id = p.chain_id and pt.contract = p.contract
                join tokens t on pt.chain_id = t.chain_id and pt.asset0 = t.address
                left join (select height, token_id, price from price -- token prices
                      union
                      select 0 height, id as token_id, 1 as price from tokens where address = ?) pr
                    on t.id = pr.token_id and pr.height <= pt.height
            where pt.chain_id = ?
              and pt.timestamp >= ?
              and pt.timestamp < ?
              and type in ('swap', 'provide', 'withdraw')) t) t
group by pair_id
`
	var asset0Stats []schemas.PairStats30m
	if tx := r.db.Raw(query, priceToken, r.chainId, startTs, endTs).Scan(&asset0Stats); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "readRepoImpl.PairStats")
	}

	query = `
select -- asset1's stats by pairs
    pair_id,
    coalesce(sum(volume) filter (where type = 'swap'),0) as volume1,
    coalesce(sum(volume_in_price) filter (where type = 'swap'),0) as volume1_in_price,
    avg(last_volume) as last_swap_price,
    sum(commission) commission1,
    sum(commission_in_price) commission1_in_price
from (select distinct -- processed asset1 values
          height,
          pair_id,
          hash,
          type,
          first_value(volume / pow(10, decimals)) over (partition by pair_id order by height desc) last_volume,
          abs(volume) as volume,
          abs(volume) * price / pow(10, decimals) as volume_in_price,
          commission,
          abs(commission) * price / pow(10, decimals) as commission_in_price
      from (select -- txs' asset1 values in a specific time range
                pt.height,
                p.id pair_id,
                pt.hash,
                type,
                pt.asset1_amount as volume,
                pt.commission1_amount as commission,
                coalesce(first_value(pr.price) over (partition by pt.height order by pr.height desc), 0) as price,
                t.decimals
            from parsed_tx pt
                join pair p on pt.chain_id = p.chain_id and pt.contract = p.contract
                join tokens t on pt.chain_id = t.chain_id and pt.asset1 = t.address
                left join (select height, token_id, price from price
                      union select 0 height, id as token_id, 1 as price from tokens where address = ?) pr
                    on t.id = pr.token_id and pr.height <= pt.height
            where pt.chain_id = ?
              and pt.timestamp >= ?
              and pt.timestamp < ?
              and type in ('swap', 'provide', 'withdraw')) t) t
group by pair_id
`
	var asset1Stats []schemas.PairStats30m
	if tx := r.db.Raw(query, priceToken, r.chainId, startTs, endTs).Scan(&asset1Stats); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "readRepoImpl.PairStats")
	}
	asset1StatsMap := make(map[uint64]schemas.PairStats30m)
	for _, s := range asset1Stats {
		asset1StatsMap[s.PairId] = s
	}

	for _, asset0 := range asset0Stats {
		if asset1, ok := asset1StatsMap[asset0.PairId]; ok {
			lastVolume0, err := util.ExponentToDecimal(asset0.LastSwapPrice)
			if err != nil {
				return nil, errors.Wrap(err, "readRepoImpl.PairStats")
			}
			lastVolume1, err := util.ExponentToDecimal(asset1.LastSwapPrice)
			if err != nil {
				return nil, errors.Wrap(err, "readRepoImpl.PairStats")
			}

			var lastSwapPrice string
			if lastVolume0.IsZero() || lastVolume1.IsZero() {
				if p, ok := prevStatsMap[asset0.PairId]; ok {
					lastSwapPrice = p.LastSwapPrice
				} else {
					lps, err := r.latestPairStat(asset0.PairId)
					if err != nil {
						return nil, errors.Wrap(err, "readRepoImpl.PairStats")
					}
					lastSwapPrice = lps.LastSwapPrice
				}
			} else {
				lastSwapPrice = lastVolume1.Quo(lastVolume0).Abs().String()
			}

			ts := util.ToTime(endTs)
			stats = append(stats, schemas.PairStats30m{
				YearUtc:        ts.Year(),
				MonthUtc:       int(ts.Month()),
				DayUtc:         ts.Day(),
				HourUtc:        ts.Hour(),
				MinuteUtc:      ts.Minute(),
				PairId:         asset0.PairId,
				ChainId:        r.chainId,
				Volume0:        asset0.Volume0,
				Volume1:        asset1.Volume1,
				Volume0InPrice: asset0.Volume0InPrice,
				Volume1InPrice: asset1.Volume1InPrice,
				LastSwapPrice:  lastSwapPrice,
				// read below fields separately by `LiquiditiesOfPairStats`
				// Liquidity0         string  `json:"liquidity0"`
				// Liquidity1         string  `json:"liquidity1"`
				// Liquidity0InPrice  string  `json:"liquidity0_in_price"`
				// Liquidity1InPrice  string  `json:"liquidity1_in_price"`
				Commission0:        asset0.Commission0,
				Commission1:        asset1.Commission1,
				Commission0InPrice: asset0.Commission0InPrice,
				Commission1InPrice: asset1.Commission1InPrice,
				PriceToken:         priceToken,
				TxCnt:              asset0.TxCnt,
				ProviderCnt:        asset0.ProviderCnt,
				Timestamp:          endTs,
			})
		}
	}

	return
}

func (r *readRepoImpl) latestPairStat(pairId uint64) (schemas.PairStats30m, error) {
	var stat schemas.PairStats30m

	if tx := r.db.Model(schemas.PairStats30m{}).Where("chain_id = ? and pair_id = ?", r.chainId, pairId).Order(
		"timestamp desc").Limit(1).Find(&stat); tx.Error != nil {
		if errors.Is(tx.Error, sql.ErrNoRows) {
			return schemas.PairStats30m{}, nil
		}
		return schemas.PairStats30m{}, tx.Error
	}

	return stat, nil
}

func (r *readRepoImpl) AccountStats(startTs float64, endTs float64) ([]schemas.AccountStats30m, error) {
	query := `
select pt.sender address, p.id pair_id, count(*) tx_cnt
from parsed_tx pt
    join pair p on p.chain_id = pt.chain_id and p.contract = pt.contract
where pt.chain_id = ?
  and pt.timestamp >= ?
  and pt.timestamp < ?
  AND pt.type IN ('swap', 'provide', 'withdraw')
group by pt.sender, p.id;
`
	res := []schemas.AccountStats30m{}
	if tx := r.db.Raw(query, r.chainId, startTs, endTs).Scan(&res); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.AccountStats")
	}

	return res, nil
}

func (r *readRepoImpl) LiquiditiesOfPairStats(startTs float64, endTs float64, priceToken string) (pairIdLpMap map[uint64]schemas.PairStats30m, err error) {
	query := `
WITH latest_lp AS (
    SELECT pair_id, MAX(height) AS height
    FROM lp_history
    WHERE chain_id = ?
        AND timestamp >= ?
        AND timestamp < ?
    GROUP BY pair_id
),
required_height_by_token AS (
    SELECT DISTINCT ll.height, t.id token_id, t.address token_address, t.decimals token_decimals
    FROM latest_lp ll
    JOIN pair p ON ll.pair_id = p.id
    JOIN tokens t ON p.chain_id = t.chain_id AND (p.asset0 = t.address OR p.asset1 = t.address)
),
token_price_by_height AS (
    SELECT DISTINCT
        COALESCE(FIRST_VALUE(p.price) OVER (PARTITION BY p.token_id ORDER BY p.height DESC), 0) price,
        rht.token_address,
        rht.token_decimals,
        rht.height
    FROM required_height_by_token rht LEFT JOIN price p ON rht.token_id = p.token_id AND p.height <= rht.height
    UNION
    SELECT 1 price,
       rht.token_address,
       rht.token_decimals,
       rht.height
    FROM required_height_by_token rht
    WHERE token_address = ?
)
SELECT lh.pair_id,
       lh.liquidity0,
       lh.liquidity1,
       lh.liquidity0 * COALESCE(t0.price, 0) / POWER(10, t0.token_decimals) as liquidity0_in_price,
       lh.liquidity1 * COALESCE(t1.price, 0) / POWER(10, t1.token_decimals) as liquidity1_in_price
FROM lp_history lh
    JOIN latest_lp ll ON lh.pair_id = ll.pair_id AND lh.height = ll.height
    JOIN pair p ON lh.pair_id = p.id
    LEFT JOIN token_price_by_height t0 ON p.asset0 = t0.token_address AND ll.height = t0.height
    LEFT JOIN token_price_by_height t1 ON p.asset1 = t1.token_address AND ll.height = t1.height
`

	var pairLps []schemas.PairStats30m
	if tx := r.db.Raw(query, r.chainId, startTs, endTs, priceToken).Scan(&pairLps); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.LiquiditiesOfPairStats")
	}

	pairIdLpMap = make(map[uint64]schemas.PairStats30m)
	for _, lp := range pairLps {
		pairIdLpMap[lp.PairId] = lp
	}

	return
}

func (r *readRepoImpl) TxHeightToSync(syncedHeight int64, condition ...string) (int64, error) {
	where := "chain_id = ? and height > ?"
	if len(condition) > 0 {
		for _, c := range condition {
			where = where + " and " + c
		}
	}

	var height int64
	tx := r.db.Model(schemas.ParsedTx{}).Where(
		where, r.chainId, syncedHeight).Select(
		"coalesce(min(height), -1)").Find(&height)
	if tx.Error != nil {
		return -1, errors.Wrap(tx.Error, "")
	}

	return height, nil
}

func (r *readRepoImpl) OldestTxTimestamp() (float64, error) {
	row := r.db.Table("parsed_tx").Where("chain_id = ?", r.chainId).Select("coalesce(min(timestamp), 0)").Row()
	if err := row.Err(); err != nil {
		return 0, err
	}

	var ts float64
	if err := row.Scan(&ts); err != nil {
		return 0, err
	}

	return ts, nil
}

func (r *readRepoImpl) LatestTxTimestamp() (float64, error) {
	row := r.db.Table("parsed_tx").Where("chain_id = ?", r.chainId).Select("MAX(timestamp)").Row()
	if err := row.Err(); err != nil {
		return 0, err
	}

	var ts float64
	if err := row.Scan(&ts); err != nil {
		return 0, err
	}

	return ts, nil
}

func (r *readRepoImpl) PairIds() ([]uint64, error) {
	rows, err := r.db.Table("pair").Where("chain_id = ?", r.chainId).Select("id").Rows()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	pairs := []uint64{}
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		pairs = append(pairs, id)
	}

	return pairs, nil
}

func (r *readRepoImpl) NewPairIds(account string, startTs float64, endTs float64) ([]uint64, error) {
	query := `
SELECT DISTINCT p.id
FROM parsed_tx pt JOIN pair p ON pt.contract = p.contract AND pt.chain_id = p.chain_id
WHERE pt.chain_id = $1
  AND pt.sender = $2
  AND pt.timestamp >= $3
  AND pt.timestamp < $4
  AND pt.type IN ('provide', 'withdraw')
`
	db, err := r.db.DB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(query, r.chainId, account, startTs, endTs)
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

func (r *readRepoImpl) NewAccounts(startTs float64, endTs float64) ([]string, error) {
	query := `
SELECT sender FROM (
    SELECT DISTINCT ON(sender) sender, timestamp
    FROM parsed_tx
    WHERE chain_id = $1
          AND type IN ('provide', 'withdraw')
    ORDER BY sender ASC, timestamp ASC) t
WHERE timestamp >= $2
  AND timestamp < $3
`
	db, err := r.db.DB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(query, r.chainId, startTs, endTs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []string{}
	for rows.Next() {
		var account string
		if err := rows.Scan(&account); err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}

func (r *readRepoImpl) ProviderCount(pairId uint64, startTs float64, endTs float64) (uint64, error) {
	query := `
SELECT COUNT(*)
FROM (SELECT pt.sender
      FROM parsed_tx pt JOIN pair p ON pt.contract = p.contract AND pt.chain_id = p.chain_id
      WHERE pt.chain_id = $1
        AND p.id = $2
        AND pt.timestamp >= $3
        AND pt.timestamp < $4
        AND pt.type = 'provide'
      GROUP BY pt.sender) t
`
	db, err := r.db.DB()
	if err != nil {
		return 0, err
	}

	var cnt uint64
	if err := db.QueryRow(query, r.chainId, pairId, startTs, endTs).Scan(&cnt); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return cnt, err
	}

	return cnt, nil
}

func (r *readRepoImpl) TxCountOfAccount(account string, pairId uint64, startTs float64, endTs float64) (uint64, error) {
	query := `
SELECT COUNT(*)
FROM parsed_tx pt JOIN pair p ON pt.contract = p.contract AND pt.chain_id = p.chain_id
WHERE pt.chain_id = $1
  AND pt.sender = $2
  AND p.id = $3
  AND pt.timestamp >= $4
  AND pt.timestamp < $5
  AND pt.type IN ('provide', 'withdraw')
`
	db, err := r.db.DB()
	if err != nil {
		return 0, err
	}

	var cnt uint64
	if err := db.QueryRow(query, r.chainId, account, pairId, startTs, endTs).Scan(&cnt); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	return cnt, nil
}

func (r *readRepoImpl) AssetAmountInPair(pairId uint64, startTs float64, endTs float64) (string, string, string, error) {
	query := `
SELECT
       coalesce(sum(asset0_amount), 0),
       coalesce(sum(asset1_amount), 0),
       coalesce(sum(
           case
               when type = 'withdraw' then lp_amount * -1
               else lp_amount
           end
       ), 0)
FROM parsed_tx pt JOIN pair p ON pt.contract = p.contract AND pt.chain_id = p.chain_id
WHERE pt.chain_id = $1
  AND p.id = $2
  AND pt.timestamp >= $3
  AND pt.timestamp < $4
  AND pt.type IN ('swap', 'provide', 'withdraw')
`
	db, err := r.db.DB()
	if err != nil {
		return "", "", "", err
	}

	var asset0Amount string
	var asset1Amount string
	var lpAmount string
	if err := db.QueryRow(query, r.chainId, pairId, startTs, endTs).Scan(&asset0Amount, &asset1Amount, &lpAmount); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return asset0Amount, asset1Amount, lpAmount, err
	}

	return asset0Amount, asset1Amount, lpAmount, nil
}

func (r *readRepoImpl) AssetAmountInPairOfAccount(account string, pairId uint64, startTs float64, endTs float64) (string, string, string, error) {
	query := `
SELECT
       coalesce(sum(pt.asset0_amount), 0),
       coalesce(sum(pt.asset1_amount), 0),
       coalesce(sum(
           case
               when type = 'withdraw' then pt.lp_amount * -1
               else pt.lp_amount
           end
       ), 0)
FROM parsed_tx pt JOIN pair p ON pt.contract = p.contract AND pt.chain_id = p.chain_id
WHERE pt.chain_id = $1
  AND pt.sender = $2
  AND p.id = $3
  AND pt.timestamp >= $4
  AND pt.timestamp < $5
  AND pt.type IN ('provide', 'withdraw')
`
	db, err := r.db.DB()
	if err != nil {
		return "", "", "", err
	}

	var asset0Amount string
	var asset1Amount string
	var lpAmount string
	if err := db.QueryRow(query, r.chainId, account, pairId, startTs, endTs).Scan(&asset0Amount, &asset1Amount, &lpAmount); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", "", "", err
	}

	return asset0Amount, asset1Amount, lpAmount, nil
}

func (r *readRepoImpl) CommissionAmountInPair(pairId uint64, startTs float64, endTs float64) (string, string, error) {
	query := `
WITH t AS (
    SELECT commission_amount, asset0_amount, asset1_amount
    FROM parsed_tx
    WHERE chain_id = $1
      AND contract IN (SELECT contract FROM pair WHERE id = $2)
      AND timestamp >= $3
      AND timestamp < $4
      AND type='swap')
SELECT
       (
           SELECT coalesce(sum(commission_amount), 0)
           FROM t
           WHERE asset0_amount < 0) asset0_commission,
       (
           SELECT coalesce(sum(commission_amount), 0)
           FROM t
           WHERE asset1_amount < 0) asset1_commission
`
	db, err := r.db.DB()
	if err != nil {
		return "", "", err
	}

	var asset0Commission string
	var asset1Commission string
	if err := db.QueryRow(query, r.chainId, pairId, startTs, endTs).Scan(&asset0Commission, &asset1Commission); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", "", err
	}

	return asset0Commission, asset1Commission, nil
}

func (r *readRepoImpl) Close() error {
	db, err := r.db.DB()
	if err != nil {
		return err
	}

	return db.Close()
}
