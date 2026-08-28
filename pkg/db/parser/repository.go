package parser

import (
	"context"
	"database/sql"
	"strings"

	"github.com/dezswap/cosmwasm-etl/pkg/util"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	"gorm.io/gorm"
)

type ReadRepository interface {
	GetSyncedHeight(ctx context.Context) (uint64, error)
	GetPairs(ctx context.Context) ([]schemas.Pair, error)
	GetPoolInfosByHeight(ctx context.Context, height uint64) ([]schemas.PoolInfo, error)
	GetParsedTxs(ctx context.Context, height uint64) ([]schemas.ParsedTx, error)
	GetParsedTxsOfPair(ctx context.Context, height uint64, pair string) ([]schemas.ParsedTx, error)

	// aggregator
	HeightOnTimestamp(ctx context.Context, timestamp float64) (uint64, error)
	LastHeightOfPrice(ctx context.Context) (uint64, error)
	GetParsedTxsWithLimit(ctx context.Context, startHeight uint64, limit int) ([]schemas.ParsedTxWithPrice, error)
	GetParsedTxsInHeightRange(ctx context.Context, startHeight uint64, endHeight uint64) ([]schemas.ParsedTxWithPrice, error)
	PricesForHeightRange(ctx context.Context, startHeight uint64, endHeight uint64, targetTokens []string, priceToken string) (map[uint64][]schemas.Price, error)
	GetParsedTxsWithPriceOfPair(ctx context.Context, pairId uint64, priceToken string, startTs float64, endTs float64) ([]schemas.ParsedTxWithPrice, error)
	PairStats(ctx context.Context, startTs float64, endTs float64, priceToken string, prevStatsMap map[uint64]schemas.PairStats30m) ([]schemas.PairStats30m, error)
	AccountStats(ctx context.Context, startTs float64, endTs float64, priceToken string) ([]schemas.AccountStats30m, error)
	LiquiditiesOfPairStats(ctx context.Context, startTs float64, endTs float64, priceToken string) (map[uint64]schemas.PairStats30m, error)
	OldestTxTimestamp(ctx context.Context) (float64, error)
	LatestTxTimestamp(ctx context.Context) (float64, error)
	PairIds(ctx context.Context) ([]uint64, error)
	NewPairIds(ctx context.Context, account string, startTs float64, endTs float64) ([]uint64, error)
	NewAccounts(ctx context.Context, startTs float64, endTs float64) ([]string, error)
	ProviderCount(ctx context.Context, pairId uint64, startTs float64, endTs float64) (uint64, error)
	TxCountOfAccount(ctx context.Context, account string, pairId uint64, startTs float64, endTs float64) (uint64, error)
	AssetAmountInPair(ctx context.Context, pairId uint64, startTs float64, endTs float64) (string, string, string, error)
	AssetAmountInPairOfAccount(ctx context.Context, account string, pairId uint64, startTs float64, endTs float64) (string, string, string, error)
	CommissionAmountInPair(ctx context.Context, pairId uint64, startTs float64, endTs float64) (string, string, error)

	Close() error
}

type readRepoImpl struct {
	db      *gorm.DB
	chainId string
}

var _ ReadRepository = &readRepoImpl{}

func NewReadRepo(chainId string, dbConfig configs.RdbConfig) (ReadRepository, error) {
	gormDB, err := db.OpenGormPostgres(dbConfig)
	if err != nil {
		return nil, err
	}

	return &readRepoImpl{
		db:      gormDB,
		chainId: chainId,
	}, nil
}

// conn binds the repository handle to ctx so every query is cancellable.
func (r *readRepoImpl) conn(ctx context.Context) *gorm.DB { return r.db.WithContext(ctx) }

// GetSyncedHeight implements parser.Repo
func (r *readRepoImpl) GetSyncedHeight(ctx context.Context) (uint64, error) {
	syncedHeight := schemas.SyncedHeight{}
	tx := r.conn(ctx).Select("height").Where("chain_id = ?", r.chainId).First(&syncedHeight)

	if tx.Error != nil {
		if !strings.Contains(tx.Error.Error(), "not found") {
			return 0, errors.Wrap(tx.Error, "repo.GetSyncedHeight")
		}

		if err := r.conn(ctx).Model(&schemas.SyncedHeight{}).Create(&schemas.SyncedHeight{ChainId: r.chainId, Height: 0}); err.Error != nil {
			return 0, errors.Wrap(err.Error, "repo.GetSyncedHeight")
		}
	}
	return syncedHeight.Height, nil
}

// GetPairs implements parser.Repo
func (r *readRepoImpl) GetPairs(ctx context.Context) ([]schemas.Pair, error) {
	pairs := []schemas.Pair{}
	tx := r.conn(ctx).Where(schemas.Pair{ChainId: r.chainId}).Find(&pairs)
	if tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetPairs")
	}
	return pairs, nil
}

// GetPairs implements parser.Repo
func (r *readRepoImpl) GetPoolInfosByHeight(ctx context.Context, height uint64) ([]schemas.PoolInfo, error) {
	poolInfo := []schemas.PoolInfo{}
	if tx := r.conn(ctx).Where(schemas.PoolInfo{ChainId: r.chainId, Height: height}).Find(&poolInfo); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetPairs")
	}
	return poolInfo, nil
}

func (r *readRepoImpl) GetParsedTxs(ctx context.Context, height uint64) ([]schemas.ParsedTx, error) {
	parsedTxs := []schemas.ParsedTx{}
	if tx := r.conn(ctx).Where(schemas.ParsedTx{ChainId: r.chainId, Height: height}).Find(&parsedTxs); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetParsedTxs")
	}
	return parsedTxs, nil
}

func (r *readRepoImpl) GetParsedTxsOfPair(ctx context.Context, height uint64, pair string) ([]schemas.ParsedTx, error) {
	parsedTxs := []schemas.ParsedTx{}
	if tx := r.conn(ctx).Where(schemas.PoolInfo{ChainId: r.chainId, Height: height, Contract: pair}).Find(&parsedTxs); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetParsedTxs")
	}
	return parsedTxs, nil
}

func (r *readRepoImpl) HeightOnTimestamp(ctx context.Context, timestamp float64) (uint64, error) {
	var height uint64
	if tx := r.conn(ctx).Model(schemas.ParsedTx{}).Where(
		"chain_id = ? and timestamp <= ?", r.chainId, timestamp).Select("coalesce(max(height), 0)").Find(&height); tx.Error != nil {
		return 0, errors.Wrap(tx.Error, "repo.HeightOnTimestamp")
	}

	return height, nil
}

func (r *readRepoImpl) LastHeightOfPrice(ctx context.Context) (uint64, error) {
	row := r.conn(ctx).Model(schemas.Price{}).Where("chain_id = ?", r.chainId).Select("coalesce(max(height), 0)").Row()
	if err := row.Err(); err != nil {
		return 0, err
	}

	var height uint64
	if err := row.Scan(&height); err != nil {
		return 0, err
	}

	return height, nil
}

func (r *readRepoImpl) GetParsedTxsWithLimit(ctx context.Context, startHeight uint64, limit int) ([]schemas.ParsedTxWithPrice, error) {
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
	if tx := r.conn(ctx).Raw(query, r.chainId, startHeight, r.chainId, startHeight, limit).Scan(&res); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetParsedTxsWithLimit")
	}

	return res, nil
}

// GetParsedTxsInHeightRange returns rows ordered by ascending height. Callers close a pair's
// group when the height changes, so the ordering is required, not incidental.
func (r *readRepoImpl) GetParsedTxsInHeightRange(ctx context.Context, startHeight uint64, endHeight uint64) ([]schemas.ParsedTxWithPrice, error) {
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
     join lp_history lh on pt.chain_id = lh.chain_id and p.id = lh.pair_id and pt.height = lh.height
     join tokens t0 on pt.chain_id = t0.chain_id and pt.asset0 = t0.address
     join tokens t1 on pt.chain_id = t1.chain_id and pt.asset1 = t1.address
where pt.chain_id = ?
  and pt.height >= ?
  and pt.height <= ?
  and pt.type in ('swap', 'provide', 'withdraw')
order by pt.height asc, p.id asc
`
	res := []schemas.ParsedTxWithPrice{}
	if tx := r.conn(ctx).Raw(query, r.chainId, startHeight, endHeight).Scan(&res); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetParsedTxsInHeightRange")
	}

	return res, nil
}

// PricesForHeightRange returns prices ordered by token and height because searchPrice scans each token's slice in order.
//
// Each token is seeded with its last price at or before startHeight, so a narrow range
// prices its heights exactly as a wider one would. Callers read span by span and rely
// on it: without the seed, a span whose prices were set earlier resolves them to zero.
func (r *readRepoImpl) PricesForHeightRange(ctx context.Context, startHeight uint64, endHeight uint64, targetTokens []string, priceToken string) (map[uint64][]schemas.Price, error) {
	query := `
with price_token as (
	select id
	from tokens
	where chain_id = ? and address = ?
),
target_tokens as (
	select t.id
	from tokens t
	    join unnest(?::bigint[]) target(id) on t.id = target.id
	where t.chain_id = ?
),
seed_price as (
	select 0 height,
	       tt.id token_id,
	       1 price
	from target_tokens tt
	    join price_token pt on tt.id = pt.id
),
start_height as (
	select tt.id token_id, coalesce(max(p.height), ?) height
	from target_tokens tt
	    left join price p on p.token_id = tt.id
	        and p.price_token_id = (select id from price_token)
	        and p.chain_id = ?
			and p.height <= ?
	group by tt.id
)
select height,
       token_id,
       price
from seed_price
union all
select p.height,
       p.token_id,
       p.price
from price p
    join start_height sh on p.token_id = sh.token_id and p.height >= sh.height
    join price_token pt on p.price_token_id = pt.id
where p.chain_id = ?
  and p.height <= ?
order by token_id, height
	`
	var res []schemas.Price
	if tx := r.conn(ctx).Raw(query, r.chainId, priceToken, pq.Array(targetTokens), r.chainId, startHeight, r.chainId, startHeight, r.chainId, endHeight).Scan(&res); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.PricesForHeightRange")
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

func (r *readRepoImpl) GetParsedTxsWithPriceOfPair(ctx context.Context, pairId uint64, priceToken string, startTs float64, endTs float64) ([]schemas.ParsedTxWithPrice, error) {
	res := []schemas.ParsedTxWithPrice{}

	if tx := r.conn(ctx).Model(schemas.ParsedTx{}).Joins(
		"join pair p on parsed_tx.chain_id = p.chain_id and parsed_tx.contract = p.contract "+
			"join tokens t0 on parsed_tx.chain_id = t0.chain_id and parsed_tx.asset0 = t0.address "+
			"join tokens t1 on parsed_tx.chain_id = t1.chain_id and parsed_tx.asset1 = t1.address "+
			"join tokens price_token on parsed_tx.chain_id = price_token.chain_id and price_token.address = ? "+
			"left outer join price p0 on t0.id = p0.token_id and p0.price_token_id = price_token.id and p0.chain_id = parsed_tx.chain_id and p0.height <= parsed_tx.height "+
			"left outer join price p1 on t1.id = p1.token_id and p1.price_token_id = price_token.id and p1.chain_id = parsed_tx.chain_id and p1.height <= parsed_tx.height",
		priceToken).Where(
		"parsed_tx.chain_id = ? and p.id = ? and parsed_tx.timestamp >= ? and parsed_tx.timestamp < ? and type in ('swap', 'provide', 'withdraw')", r.chainId, pairId, startTs, endTs).Order(
		"parsed_tx.height, p0.height desc, p1.height desc").Select(
		"distinct on (parsed_tx.height) parsed_tx.asset0_amount, parsed_tx.asset1_amount," +
			"parsed_tx.commission0_amount, parsed_tx.commission1_amount," +
			"CASE WHEN t0.id = price_token.id THEN '1' ELSE coalesce(p0.price, '0') END price0, CASE WHEN t1.id = price_token.id THEN '1' ELSE coalesce(p1.price, '0') END price1," +
			"t0.decimals decimals0, t1.decimals decimals1").Find(&res); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetParsedTxsWithPriceOfPair")
	}

	return res, nil
}

func (r *readRepoImpl) PairStats(ctx context.Context, startTs float64, endTs float64, priceToken string, prevStatsMap map[uint64]schemas.PairStats30m) (stats []schemas.PairStats30m, err error) {
	query := `
select -- asset0's stats by pairs
    pair_id,
    coalesce(sum(volume) filter (where type = 'swap'),0) as volume0,
    (coalesce(sum(volume_in_price) filter (where type = 'swap'),0))::numeric as volume0_in_price,
    (avg(last_volume))::numeric as last_swap_price,
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
                case when t.id = price_token.id then 1 else coalesce(pr.price, 0) end as price,
                t.decimals
            from parsed_tx pt
                join pair p on pt.chain_id = p.chain_id and pt.contract = p.contract
                join tokens t on pt.chain_id = t.chain_id and pt.asset0 = t.address
                join tokens price_token on pt.chain_id = price_token.chain_id and price_token.address = ?
                left join lateral (
                    select price from price
                    where token_id = t.id and price_token_id = price_token.id and height <= pt.height and chain_id = ?
                    order by height desc limit 1
                ) pr on true
            where pt.chain_id = ?
              and pt.timestamp >= ?
              and pt.timestamp < ?
              and type in ('swap', 'provide', 'withdraw')) t) t
group by pair_id
`
	var asset0Stats []schemas.PairStats30m
	if tx := r.conn(ctx).Raw(query, priceToken, r.chainId, r.chainId, startTs, endTs).Scan(&asset0Stats); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "readRepoImpl.PairStats")
	}

	query = `
select -- asset1's stats by pairs
    pair_id,
    coalesce(sum(volume) filter (where type = 'swap'),0) as volume1,
    (coalesce(sum(volume_in_price) filter (where type = 'swap'),0))::numeric as volume1_in_price,
    (avg(last_volume))::numeric as last_swap_price,
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
                coalesce(pr.price, case when t.id = price_token.id then 1 else 0 end) as price,
                t.decimals
            from parsed_tx pt
                join pair p on pt.chain_id = p.chain_id and pt.contract = p.contract
                join tokens t on pt.chain_id = t.chain_id and pt.asset1 = t.address
                join tokens price_token on pt.chain_id = price_token.chain_id and price_token.address = ?
                left join lateral (
                    select price from price
                    where token_id = t.id and price_token_id = price_token.id and height <= pt.height and chain_id = ?
                    order by height desc limit 1
                ) pr on true
            where pt.chain_id = ?
              and pt.timestamp >= ?
              and pt.timestamp < ?
              and type in ('swap', 'provide', 'withdraw')) t) t
group by pair_id
`
	var asset1Stats []schemas.PairStats30m
	if tx := r.conn(ctx).Raw(query, priceToken, r.chainId, r.chainId, startTs, endTs).Scan(&asset1Stats); tx.Error != nil {
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
					lps, err := r.latestPairStat(ctx, asset0.PairId)
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

func (r *readRepoImpl) latestPairStat(ctx context.Context, pairId uint64) (schemas.PairStats30m, error) {
	var stat schemas.PairStats30m

	if tx := r.conn(ctx).Model(schemas.PairStats30m{}).Where("chain_id = ? and pair_id = ?", r.chainId, pairId).Order(
		"timestamp desc").Limit(1).Find(&stat); tx.Error != nil {
		if errors.Is(tx.Error, sql.ErrNoRows) {
			return schemas.PairStats30m{}, nil
		}
		return schemas.PairStats30m{}, tx.Error
	}

	return stat, nil
}

func (r *readRepoImpl) AccountStats(ctx context.Context, startTs float64, endTs float64, priceToken string) ([]schemas.AccountStats30m, error) {
	query := `
WITH tx_values AS (
    SELECT
        pt.sender address,
        p.id pair_id,
        pt.hash,
        pt.type,
        pt.asset0_amount::numeric asset0_amount,
        pt.asset1_amount::numeric asset1_amount,
        CASE
            WHEN pt.type = 'withdraw' THEN -abs(pt.lp_amount::numeric)
            ELSE pt.lp_amount::numeric
        END lp_amount,
        abs(pt.asset0_amount::numeric) * coalesce(pr0.price, CASE WHEN t0.address = ? THEN 1 ELSE 0 END) / (10::numeric ^ t0.decimals) asset0_value_in_price,
        abs(pt.asset1_amount::numeric) * coalesce(pr1.price, CASE WHEN t1.address = ? THEN 1 ELSE 0 END) / (10::numeric ^ t1.decimals) asset1_value_in_price,
        pt.asset0_amount::numeric * coalesce(pr0.price, CASE WHEN t0.address = ? THEN 1 ELSE 0 END) / (10::numeric ^ t0.decimals) net_asset0_value_in_price,
        pt.asset1_amount::numeric * coalesce(pr1.price, CASE WHEN t1.address = ? THEN 1 ELSE 0 END) / (10::numeric ^ t1.decimals) net_asset1_value_in_price
    FROM parsed_tx pt
    JOIN pair p ON p.chain_id = pt.chain_id AND p.contract = pt.contract
    JOIN tokens t0 ON pt.chain_id = t0.chain_id AND pt.asset0 = t0.address
    JOIN tokens t1 ON pt.chain_id = t1.chain_id AND pt.asset1 = t1.address
    JOIN tokens price_token ON pt.chain_id = price_token.chain_id AND price_token.address = ?
    LEFT JOIN LATERAL (
        SELECT price
        FROM price
        WHERE token_id = t0.id
          AND price_token_id = price_token.id
          AND height <= pt.height
          AND chain_id = ?
        ORDER BY height DESC, id DESC
        LIMIT 1
    ) pr0 ON true
    LEFT JOIN LATERAL (
        SELECT price
        FROM price
        WHERE token_id = t1.id
          AND price_token_id = price_token.id
          AND height <= pt.height
          AND chain_id = ?
        ORDER BY height DESC, id DESC
        LIMIT 1
    ) pr1 ON true
WHERE pt.chain_id = ?
  and pt.timestamp >= ?
  and pt.timestamp < ?
  and pt.type in ('swap', 'provide', 'withdraw')
)
SELECT
    address,
    pair_id,
    count(distinct hash) tx_cnt,
    count(distinct hash) filter (where type = 'swap') swap_tx_cnt,
    count(distinct hash) filter (where type = 'provide') provide_tx_cnt,
    count(distinct hash) filter (where type = 'withdraw') withdraw_tx_cnt,
    coalesce(sum(
	    CASE
			WHEN asset0_amount < 0 THEN asset0_value_in_price
			WHEN asset1_amount < 0 THEN asset1_value_in_price
			ELSE greatest(asset0_value_in_price, asset1_value_in_price)
		END
	) filter (where type = 'swap'), 0) swap_volume_in_price,
    coalesce(sum(asset0_value_in_price + asset1_value_in_price) filter (where type = 'provide'), 0) provide_value_in_price,
    coalesce(sum(asset0_value_in_price + asset1_value_in_price) filter (where type = 'withdraw'), 0) withdraw_value_in_price,
    coalesce(sum(net_asset0_value_in_price + net_asset1_value_in_price), 0) net_flow_in_price,
    ? price_token,
    coalesce(sum(asset0_amount), 0) net_asset0_amount,
    coalesce(sum(asset1_amount), 0) net_asset1_amount,
    coalesce(sum(lp_amount), 0) net_lp_amount
FROM tx_values
GROUP BY address, pair_id;
`
	res := []schemas.AccountStats30m{}
	if tx := r.conn(ctx).Raw(query, priceToken, priceToken, priceToken, priceToken, priceToken, r.chainId, r.chainId, r.chainId, startTs, endTs, priceToken).Scan(&res); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.AccountStats")
	}

	return res, nil
}

func (r *readRepoImpl) LiquiditiesOfPairStats(ctx context.Context, startTs float64, endTs float64, priceToken string) (pairIdLpMap map[uint64]schemas.PairStats30m, err error) {
	query := `
WITH price_token AS (
    SELECT id
    FROM tokens
    WHERE chain_id = ? AND address = ?
),
latest_lp AS (
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
    SELECT
        CASE
            WHEN rht.token_address = ? THEN 1::numeric
            ELSE COALESCE(pr.price, 0::numeric)
        END AS price,
        rht.token_address,
        rht.token_decimals,
        rht.height
    FROM required_height_by_token rht
    CROSS JOIN price_token
    LEFT JOIN LATERAL (
        SELECT p.price
        FROM price p
        WHERE p.token_id = rht.token_id
          AND p.price_token_id = price_token.id
          AND p.chain_id = ?
          AND p.height <= rht.height
        ORDER BY p.height DESC, p.id DESC
        LIMIT 1
    ) pr ON true
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
	if tx := r.conn(ctx).Raw(query, r.chainId, priceToken, r.chainId, startTs, endTs, priceToken, r.chainId).Scan(&pairLps); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.LiquiditiesOfPairStats")
	}

	pairIdLpMap = make(map[uint64]schemas.PairStats30m)
	for _, lp := range pairLps {
		pairIdLpMap[lp.PairId] = lp
	}

	return
}

func (r *readRepoImpl) TxHeightToSync(ctx context.Context, syncedHeight int64, condition ...string) (int64, error) {
	where := "chain_id = ? and height > ?"
	if len(condition) > 0 {
		for _, c := range condition {
			where = where + " and " + c
		}
	}

	var height int64
	tx := r.conn(ctx).Model(schemas.ParsedTx{}).Where(
		where, r.chainId, syncedHeight).Select(
		"coalesce(min(height), -1)").Find(&height)
	if tx.Error != nil {
		return -1, errors.Wrap(tx.Error, "")
	}

	return height, nil
}

func (r *readRepoImpl) OldestTxTimestamp(ctx context.Context) (float64, error) {
	row := r.conn(ctx).Table("parsed_tx").Where("chain_id = ?", r.chainId).Select("coalesce(min(timestamp), 0)").Row()
	if err := row.Err(); err != nil {
		return 0, err
	}

	var ts float64
	if err := row.Scan(&ts); err != nil {
		return 0, err
	}

	return ts, nil
}

func (r *readRepoImpl) LatestTxTimestamp(ctx context.Context) (float64, error) {
	row := r.conn(ctx).Table("parsed_tx").Where("chain_id = ?", r.chainId).Select("MAX(timestamp)").Row()
	if err := row.Err(); err != nil {
		return 0, err
	}

	var ts float64
	if err := row.Scan(&ts); err != nil {
		return 0, err
	}

	return ts, nil
}

func (r *readRepoImpl) PairIds(ctx context.Context) ([]uint64, error) {
	rows, err := r.conn(ctx).Table("pair").Where("chain_id = ?", r.chainId).Select("id").Rows()
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

func (r *readRepoImpl) NewPairIds(ctx context.Context, account string, startTs float64, endTs float64) ([]uint64, error) {
	query := `
SELECT DISTINCT p.id
FROM parsed_tx pt JOIN pair p ON pt.contract = p.contract AND pt.chain_id = p.chain_id
WHERE pt.chain_id = ?
  AND pt.sender = ?
  AND pt.timestamp >= ?
  AND pt.timestamp < ?
  AND pt.type IN ('provide', 'withdraw')
`
	pairIds := []uint64{}
	if tx := r.conn(ctx).Raw(query, r.chainId, account, startTs, endTs).Scan(&pairIds); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "readRepoImpl.NewPairIds")
	}

	return pairIds, nil
}

func (r *readRepoImpl) NewAccounts(ctx context.Context, startTs float64, endTs float64) ([]string, error) {
	query := `
SELECT sender FROM (
    SELECT DISTINCT ON(sender) sender, timestamp
    FROM parsed_tx
    WHERE chain_id = ?
          AND type IN ('provide', 'withdraw')
    ORDER BY sender ASC, timestamp ASC) t
WHERE timestamp >= ?
  AND timestamp < ?
`
	accounts := []string{}
	if tx := r.conn(ctx).Raw(query, r.chainId, startTs, endTs).Scan(&accounts); tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "readRepoImpl.NewAccounts")
	}

	return accounts, nil
}

func (r *readRepoImpl) ProviderCount(ctx context.Context, pairId uint64, startTs float64, endTs float64) (uint64, error) {
	query := `
SELECT COUNT(*)
FROM (SELECT pt.sender
      FROM parsed_tx pt JOIN pair p ON pt.contract = p.contract AND pt.chain_id = p.chain_id
      WHERE pt.chain_id = ?
        AND p.id = ?
        AND pt.timestamp >= ?
        AND pt.timestamp < ?
        AND pt.type = 'provide'
      GROUP BY pt.sender) t
`
	var cnt uint64
	if err := r.conn(ctx).Raw(query, r.chainId, pairId, startTs, endTs).Row().Scan(&cnt); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return cnt, err
	}

	return cnt, nil
}

func (r *readRepoImpl) TxCountOfAccount(ctx context.Context, account string, pairId uint64, startTs float64, endTs float64) (uint64, error) {
	query := `
SELECT COUNT(*)
FROM parsed_tx pt JOIN pair p ON pt.contract = p.contract AND pt.chain_id = p.chain_id
WHERE pt.chain_id = ?
  AND pt.sender = ?
  AND p.id = ?
  AND pt.timestamp >= ?
  AND pt.timestamp < ?
  AND pt.type IN ('provide', 'withdraw')
`
	var cnt uint64
	if err := r.conn(ctx).Raw(query, r.chainId, account, pairId, startTs, endTs).Row().Scan(&cnt); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	return cnt, nil
}

func (r *readRepoImpl) AssetAmountInPair(ctx context.Context, pairId uint64, startTs float64, endTs float64) (string, string, string, error) {
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
WHERE pt.chain_id = ?
  AND p.id = ?
  AND pt.timestamp >= ?
  AND pt.timestamp < ?
  AND pt.type IN ('swap', 'provide', 'withdraw')
`
	var asset0Amount string
	var asset1Amount string
	var lpAmount string
	if err := r.conn(ctx).Raw(query, r.chainId, pairId, startTs, endTs).Row().Scan(&asset0Amount, &asset1Amount, &lpAmount); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return asset0Amount, asset1Amount, lpAmount, err
	}

	return asset0Amount, asset1Amount, lpAmount, nil
}

func (r *readRepoImpl) AssetAmountInPairOfAccount(ctx context.Context, account string, pairId uint64, startTs float64, endTs float64) (string, string, string, error) {
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
WHERE pt.chain_id = ?
  AND pt.sender = ?
  AND p.id = ?
  AND pt.timestamp >= ?
  AND pt.timestamp < ?
  AND pt.type IN ('provide', 'withdraw')
`
	var asset0Amount string
	var asset1Amount string
	var lpAmount string
	if err := r.conn(ctx).Raw(query, r.chainId, account, pairId, startTs, endTs).Row().Scan(&asset0Amount, &asset1Amount, &lpAmount); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", "", "", err
	}

	return asset0Amount, asset1Amount, lpAmount, nil
}

func (r *readRepoImpl) CommissionAmountInPair(ctx context.Context, pairId uint64, startTs float64, endTs float64) (string, string, error) {
	query := `
WITH t AS (
    SELECT commission_amount, asset0_amount, asset1_amount
    FROM parsed_tx
    WHERE chain_id = ?
      AND contract IN (SELECT contract FROM pair WHERE id = ?)
      AND timestamp >= ?
      AND timestamp < ?
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
	var asset0Commission string
	var asset1Commission string
	if err := r.conn(ctx).Raw(query, r.chainId, pairId, startTs, endTs).Row().Scan(&asset0Commission, &asset1Commission); err != nil && !errors.Is(err, sql.ErrNoRows) {
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
