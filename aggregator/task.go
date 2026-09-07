package aggregator

import (
	"context"
	"math"
	"strconv"
	"sync/atomic"
	"time"

	cmath "cosmossdk.io/math"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/price"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/router"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"

	"github.com/dezswap/cosmwasm-etl/aggregator/repo"
	"github.com/dezswap/cosmwasm-etl/pkg/db/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
)

const (
	LpHistoryUpdateLimit      = 100
	PairStatsRecentTimeRange  = 48 * time.Hour
	PairStatsRecentHeightSpan = 1000
)

const WaitPeriod = 10 * time.Second

type task interface {
	Name() string
	Execute(ctx context.Context, start time.Time, end time.Time) error
	LastProcessedHeight() uint64
}

type predeterminedTimeTask interface {
	task
	StartTimestamp(ctx context.Context, startTs time.Time) (time.Time, error)
}

type taskImpl struct {
	chainId         string
	destDb          repo.Repo
	parentTasks     []task
	taskWaitTimeout time.Duration
	logger          logging.Logger

	// State
	lastProcessedHeight atomic.Uint64
}

func (t *taskImpl) LastProcessedHeight() uint64 {
	return t.lastProcessedHeight.Load()
}

// advanceHeight publishes progress and never moves it backwards: child tasks gate on
// this value, and a lower reading would re-block work whose inputs are already complete.
func (t *taskImpl) advanceHeight(height uint64) {
	for {
		curr := t.lastProcessedHeight.Load()
		if height <= curr || t.lastProcessedHeight.CompareAndSwap(curr, height) {
			return
		}
	}
}

type lpHistoryTask struct {
	taskImpl

	srcDb parser.ReadRepository
}

type routerTask struct {
	taskImpl

	router  router.Router
	srcDb   parser.ReadRepository
	db      router.SrcRepo
	pairCnt int

	reportedMissingRoutes bool
}

type priceTask struct {
	taskImpl

	priceTracker price.Price
}

type pairStatsUpdateTask struct {
	taskImpl

	priceToken  string
	srcDb       parser.ReadRepository
	prevStatMap map[uint64]schemas.PairStats30m
}

type pairStatsRecentUpdateTask struct {
	taskImpl

	priceToken string
	srcDb      parser.ReadRepository
	timeRange  time.Duration
}

type accountStatsUpdateTask struct {
	taskImpl

	priceToken string
	srcDb      parser.ReadRepository
}

func (*lpHistoryTask) Name() string             { return "lp_history" }
func (*routerTask) Name() string                { return "router" }
func (*priceTask) Name() string                 { return "price" }
func (*pairStatsRecentUpdateTask) Name() string { return "pair_stats_recent" }
func (*pairStatsUpdateTask) Name() string       { return "pair_stats_30m" }
func (*accountStatsUpdateTask) Name() string    { return "account_stats_30m" }

func newLpHistoryTask(config configs.AggregatorConfig, srcRepo parser.ReadRepository, destRepo repo.Repo, logger logging.Logger) task {
	return &lpHistoryTask{
		taskImpl: taskImpl{
			chainId: config.ChainId,
			destDb:  destRepo,
			logger:  logger,
		},
		srcDb: srcRepo,
	}
}

func (t *lpHistoryTask) Execute(ctx context.Context, _ time.Time, _ time.Time) error {
	lastHistories, err := t.destDb.LastLpHistory(ctx, uint64(math.MaxInt64))
	if err != nil {
		return err
	}

	latestLpMap := make(map[uint64][]string)
	for _, h := range lastHistories {
		latestLpMap[h.PairId] = []string{h.Liquidity0, h.Liquidity1}
		t.advanceHeight(h.Height)
	}

	for {
		txs, err := t.srcDb.GetParsedTxsWithLimit(ctx, t.lastProcessedHeight.Load()+1, LpHistoryUpdateLimit)
		if err != nil {
			return err
		}
		if len(txs) < 1 {
			break
		}

		history, err := t.generateHistory(latestLpMap, txs)
		if err != nil {
			return err
		}
		err = t.destDb.UpdateLpHistory(ctx, history)
		if err != nil {
			return err
		}
		t.advanceHeight(history[len(history)-1].Height)
	}

	t.logger.Infof("Complete lp history update.")

	return nil
}

func (t *lpHistoryTask) generateHistory(latestLpMap map[uint64][]string, txs []schemas.ParsedTxWithPrice) ([]schemas.LpHistory, error) {
	history := []schemas.LpHistory{}

	pairIdLpHistoryMap := make(map[uint64][2]cmath.LegacyDec)

	var currLpHistory schemas.LpHistory
	currHeight := uint64(0)
	currPairId := uint64(0)
	for _, tx := range txs {
		if currHeight != tx.Height || currPairId != tx.PairId {
			if currHeight > 0 {
				lp := pairIdLpHistoryMap[currPairId]
				currLpHistory.Liquidity0 = lp[repo.Liquidity0].String()
				currLpHistory.Liquidity1 = lp[repo.Liquidity1].String()
				history = append(history, currLpHistory)
			}

			currHeight = tx.Height
			currPairId = tx.PairId
			currLpHistory = schemas.LpHistory{
				Height:    tx.Height,
				PairId:    tx.PairId,
				ChainId:   tx.ChainId,
				Timestamp: tx.Timestamp,
			}
		}

		volume0, err := cmath.LegacyNewDecFromStr(tx.Asset0Amount)
		if err != nil {
			return nil, errors.Wrap(err, "lpHistoryTask.generateHistory")
		}
		volume1, err := cmath.LegacyNewDecFromStr(tx.Asset1Amount)
		if err != nil {
			return nil, errors.Wrap(err, "lpHistoryTask.generateHistory")
		}

		lp, ok := pairIdLpHistoryMap[tx.PairId]
		if !ok {
			// initialize with the latest lp
			if latestLp, ok := latestLpMap[tx.PairId]; ok {
				lp[repo.Liquidity0], err = cmath.LegacyNewDecFromStr(latestLp[repo.Liquidity0])
				if err != nil {
					return nil, errors.Wrap(err, "lpHistoryTask.generateHistory")
				}
				lp[repo.Liquidity1], err = cmath.LegacyNewDecFromStr(latestLp[repo.Liquidity1])
				if err != nil {
					return nil, errors.Wrap(err, "lpHistoryTask.generateHistory")
				}
			} else {
				lp[repo.Liquidity0] = cmath.LegacyZeroDec()
				lp[repo.Liquidity1] = cmath.LegacyZeroDec()
			}
		}

		lp[repo.Liquidity0] = lp[repo.Liquidity0].Add(volume0)
		lp[repo.Liquidity1] = lp[repo.Liquidity1].Add(volume1)
		pairIdLpHistoryMap[tx.PairId] = lp
	}

	// append the last lp history
	lp := pairIdLpHistoryMap[currPairId]
	currLpHistory.Liquidity0 = lp[repo.Liquidity0].String()
	currLpHistory.Liquidity1 = lp[repo.Liquidity1].String()
	history = append(history, currLpHistory)

	for id, lp := range pairIdLpHistoryMap {
		if _, ok := latestLpMap[id]; ok {
			latestLpMap[id][repo.Liquidity0] = lp[repo.Liquidity0].String()
			latestLpMap[id][repo.Liquidity1] = lp[repo.Liquidity1].String()
		} else {
			lpHistory := []string{lp[repo.Liquidity0].String(), lp[repo.Liquidity1].String()}
			latestLpMap[id] = lpHistory
		}
	}

	return history, nil
}

func newRouterTask(config configs.AggregatorConfig, srcRepo parser.ReadRepository, repo router.SrcRepo, logger logging.Logger) task {
	return &routerTask{
		taskImpl: taskImpl{
			chainId: config.ChainId,
			logger:  logger,
		},
		router: router.New(repo, config.Router, logger),
		srcDb:  srcRepo,
		db:     repo,
	}
}

// Execute publishes the height whose pairs the route table covers, which the price task
// gates on. The height is read before the pair set, or a pair created between the two
// reads would be claimed as covered. A round that rebuilds nothing still publishes, or
// the gate never opens again.
func (t *routerTask) Execute(ctx context.Context, _ time.Time, _ time.Time) error {
	syncedHeight, err := t.srcDb.GetSyncedHeight(ctx)
	if err != nil {
		return err
	}

	pairCount, unrouted, err := t.db.PairStatus(ctx)
	if err != nil {
		return err
	}

	// pairCnt starts at 0, so without this a restart rebuilds a route table that is
	// already complete. Coverage is the hop_count 0 rows, which a raised max_hop_count
	// leaves in place: clear the chain's route rows to make that setting take effect.
	if t.pairCnt == 0 && !unrouted {
		t.pairCnt = pairCount
	}

	// Checked before the rebuild below, whose condition this negates: a pair with no
	// route row and no growth to rebuild for leaves the height claiming coverage the
	// route table does not have, and price drops those prices for good. Report the edge,
	// not every round.
	missingRoutes := unrouted && pairCount <= t.pairCnt
	if missingRoutes && !t.reportedMissingRoutes {
		t.logger.Warnf("chain(%s) has a pair with no route row; prices routed through it "+
			"are dropped until a new pair forces a rebuild", t.chainId)
	}
	t.reportedMissingRoutes = missingRoutes

	if pairCount > t.pairCnt {
		// only remember the new count once the rebuild succeeded, otherwise a failed
		// update would be skipped on every later run
		if err := t.router.Update(ctx); err != nil {
			return err
		}
		t.pairCnt = pairCount
	}

	t.advanceHeight(syncedHeight)

	return nil
}

func newPriceTask(ctx context.Context, config configs.AggregatorConfig, destRepo repo.Repo, priceRepo price.SrcRepo, logger logging.Logger, parentTasks []task) (task, error) {
	pt, err := price.New(ctx, priceRepo, config.PriceToken, logger)
	if err != nil {
		return nil, err
	}

	return &priceTask{
		taskImpl: taskImpl{
			chainId:         config.ChainId,
			destDb:          destRepo,
			parentTasks:     parentTasks,
			taskWaitTimeout: taskWaitTimeout(config),
			logger:          logger,
		},
		priceTracker: pt,
	}, nil
}

// Execute prices every candidate height up to the source tip, then reports the tip
// rather than the last height it wrote. Prices only exist for swaps and first
// provisions, so the last price row lags the source; children gate on the source
// height and would wait for a height this task never claims.
//
// The tip is read before the loop, so heights arriving while it runs are not claimed.
func (t *priceTask) Execute(ctx context.Context, _ time.Time, _ time.Time) error {
	tip, err := t.priceTracker.SrcHeight(ctx)
	if err != nil {
		return err
	}

	height := uint64(0)
	for {
		nextHeight, err := t.priceTracker.NextHeight(ctx, height)
		if err != nil {
			return err
		}
		if nextHeight == price.NaValue || nextHeight > tip {
			break
		}

		height = uint64(nextHeight)
		if err := waitUntilReachingHeight(ctx, t.parentTasks, height, t.taskWaitTimeout); err != nil {
			return err
		}

		if err := t.priceTracker.Run(ctx, height); err != nil {
			return err
		}
		t.advanceHeight(height)
	}

	// everything at or below the tip has been considered, priced or not. A non-positive
	// tip would convert to MaxUint64 and open every child's gate forever.
	if tip > 0 {
		t.advanceHeight(uint64(tip))
	}

	return nil
}

func newPairStatsRecentUpdateTask(config configs.AggregatorConfig, srcRepo parser.ReadRepository, destRepo repo.Repo, logger logging.Logger, parentTasks []task) task {
	return &pairStatsRecentUpdateTask{
		taskImpl: taskImpl{
			chainId:         config.ChainId,
			destDb:          destRepo,
			parentTasks:     parentTasks,
			taskWaitTimeout: taskWaitTimeout(config),
			logger:          logger,
		},
		priceToken: config.PriceToken,
		srcDb:      srcRepo,
		timeRange:  PairStatsRecentTimeRange,
	}
}

func (t *pairStatsRecentUpdateTask) Execute(ctx context.Context, _ time.Time, end time.Time) error {
	lastProcessedHeight := t.lastProcessedHeight.Load()
	if lastProcessedHeight == 0 {
		loadedHeight, err := t.destDb.LastHeightOfPairStatsRecent(ctx)
		if err != nil {
			return err
		}
		t.advanceHeight(loadedHeight)
		lastProcessedHeight = loadedHeight
	}
	startTs := end.Add(-1 * t.timeRange)
	startHeight, err := t.srcDb.HeightOnTimestamp(ctx, util.ToEpoch(startTs))
	if err != nil {
		return err
	}

	endHeight, err := t.srcDb.HeightOnTimestamp(ctx, util.ToEpoch(end))
	if err != nil {
		return err
	}

	// Rows fall out of the trailing window regardless of upstream, so the prune runs
	// before anything that can bail out; a round given up on a parent would otherwise
	// leave them for a whole cold start. The prune in the transaction below has its own job.
	if err := t.destDb.DeletePairStatsRecent(ctx, startTs); err != nil {
		return err
	}

	if endHeight <= lastProcessedHeight {
		t.logger.Infof("Complete pair stats recent update.")

		return nil
	}

	if err := waitUntilReachingHeight(ctx, t.parentTasks, endHeight, t.taskWaitTimeout); err != nil {
		return err
	}

	if startHeight <= lastProcessedHeight {
		startHeight = lastProcessedHeight + 1
	}

	// The span loop reads inside this transaction on purpose, which keeps it open for a
	// whole cold start: pair_stats_recent has no unique constraint (its _uidx indexes
	// are not unique), so a span committed on its own becomes duplicate rows once a
	// later span fails and the round retries. To shorten the transaction, add that
	// unique index first and commit spans separately.
	if err := t.destDb.WithinTx(ctx, func(txRepo repo.Repo) error {
		if err := t.updateStatsBySpan(ctx, txRepo, startHeight, endHeight); err != nil {
			return err
		}

		return txRepo.DeletePairStatsRecent(ctx, startTs)
	}); err != nil {
		return err
	}

	t.advanceHeight(endHeight)

	t.logger.Infof("Complete pair stats recent update.")

	return nil
}

// updateStatsBySpan walks [startHeight, endHeight] in spans, writing each before
// reading the next, so a cold start never holds the whole window in memory.
//
// Splitting matches a single pass only while both of these hold:
//
//   - a stats row covers one pair at one height, so no group straddles a boundary;
//   - PricesForHeightRange seeds each token from its last price at or before the start height,
//     so a span prices a transaction as the full window would. Without the seed, a span
//     whose prices were all set earlier values its transactions at zero.
func (t *pairStatsRecentUpdateTask) updateStatsBySpan(ctx context.Context, txRepo repo.Repo, startHeight, endHeight uint64) error {
	for spanStart := startHeight; spanStart <= endHeight; spanStart += PairStatsRecentHeightSpan {
		spanEnd := min(spanStart+PairStatsRecentHeightSpan-1, endHeight)

		stats, err := t.statsOfSpan(ctx, spanStart, spanEnd)
		if err != nil {
			return err
		}
		if len(stats) == 0 {
			// a cold start walks over long stretches with no activity at all
			continue
		}
		if err := txRepo.UpdatePairStatsRecent(ctx, stats); err != nil {
			return err
		}
	}

	return nil
}

// statsOfSpan derives one span's stats; the transactions and prices it reads stay
// local, so only the derived rows outlive the call.
func (t *pairStatsRecentUpdateTask) statsOfSpan(ctx context.Context, startHeight, endHeight uint64) ([]schemas.PairStatsRecent, error) {
	txs, err := t.srcDb.GetParsedTxsInHeightRange(ctx, startHeight, endHeight)
	if err != nil {
		return nil, err
	}
	if len(txs) == 0 {
		return nil, nil
	}

	priceMap, err := t.srcDb.PricesForHeightRange(ctx, startHeight, endHeight, uniquePriceTokenIds(txs), t.priceToken)
	if err != nil {
		return nil, err
	}

	return t.generateStats(txs, priceMap)
}

func uniquePriceTokenIds(txs []schemas.ParsedTxWithPrice) []string {
	tokenIdMap := make(map[string]bool)
	for _, tx := range txs {
		tokenIdMap[tx.Price0] = true
		tokenIdMap[tx.Price1] = true
	}

	tokenIds := make([]string, 0, len(tokenIdMap))
	for key := range tokenIdMap {
		tokenIds = append(tokenIds, key)
	}

	return tokenIds
}

func (t *pairStatsRecentUpdateTask) generateStats(txs []schemas.ParsedTxWithPrice, priceMap map[uint64][]schemas.Price) ([]schemas.PairStatsRecent, error) {
	type pairStat struct {
		PairId             uint64
		ChainId            string
		Volume0            cmath.LegacyDec
		Volume1            cmath.LegacyDec
		Volume0InPrice     cmath.LegacyDec
		Volume1InPrice     cmath.LegacyDec
		Liquidity0         cmath.LegacyDec
		Liquidity1         cmath.LegacyDec
		Liquidity0InPrice  cmath.LegacyDec
		Liquidity1InPrice  cmath.LegacyDec
		Commission0        cmath.LegacyDec
		Commission1        cmath.LegacyDec
		Commission0InPrice cmath.LegacyDec
		Commission1InPrice cmath.LegacyDec
		Height             uint64
		Timestamp          float64
	}

	stats := make([]schemas.PairStatsRecent, 0)
	currStatMap := make(map[uint64]pairStat, 0)
	for _, tx := range txs {
		price0, err := t.searchPrice(tx.Price0, tx.Height, priceMap)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}

		price1, err := t.searchPrice(tx.Price1, tx.Height, priceMap)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}

		decimal0 := cmath.LegacyNewDec(10).Power(uint64(tx.Decimals0))
		decimal1 := cmath.LegacyNewDec(10).Power(uint64(tx.Decimals1))

		volume0, err := cmath.LegacyNewDecFromStr(tx.Asset0Amount)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}
		volume0InPrice := volume0.Quo(decimal0).Mul(price0)

		volume1, err := cmath.LegacyNewDecFromStr(tx.Asset1Amount)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}
		volume1InPrice := volume1.Quo(decimal1).Mul(price1)

		liquidity0, err := cmath.LegacyNewDecFromStr(tx.Asset0Liquidity)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}

		liquidity1, err := cmath.LegacyNewDecFromStr(tx.Asset1Liquidity)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}

		commission0, err := cmath.LegacyNewDecFromStr(tx.Commission0Amount)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}
		commission0InPrice := commission0.Quo(decimal0).Mul(price0)

		commission1, err := cmath.LegacyNewDecFromStr(tx.Commission1Amount)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}
		commission1InPrice := commission1.Quo(decimal1).Mul(price1)

		stat := currStatMap[tx.PairId]
		if stat.Height != tx.Height {
			if stat.Height > 0 { // is not first
				stats = append(stats, schemas.PairStatsRecent{
					PairId:             stat.PairId,
					ChainId:            t.chainId,
					Volume0:            stat.Volume0.String(),
					Volume1:            stat.Volume1.String(),
					Volume0InPrice:     stat.Volume0InPrice.String(),
					Volume1InPrice:     stat.Volume1InPrice.String(),
					Liquidity0:         stat.Liquidity0.String(),
					Liquidity1:         stat.Liquidity1.String(),
					Liquidity0InPrice:  stat.Liquidity0InPrice.String(),
					Liquidity1InPrice:  stat.Liquidity1InPrice.String(),
					Commission0:        stat.Commission0.String(),
					Commission1:        stat.Commission1.String(),
					Commission0InPrice: stat.Commission0InPrice.String(),
					Commission1InPrice: stat.Commission1InPrice.String(),
					PriceToken:         t.priceToken,
					Height:             stat.Height,
					Timestamp:          stat.Timestamp,
				})
			}

			currStatMap[tx.PairId] = pairStat{
				PairId:             tx.PairId,
				ChainId:            t.chainId,
				Volume0:            volume0.Abs(),
				Volume1:            volume1.Abs(),
				Volume0InPrice:     volume0InPrice.Abs(),
				Volume1InPrice:     volume1InPrice.Abs(),
				Liquidity0:         liquidity0,
				Liquidity1:         liquidity1,
				Liquidity0InPrice:  liquidity0.Quo(decimal0).Mul(price0),
				Liquidity1InPrice:  liquidity1.Quo(decimal1).Mul(price1),
				Commission0:        commission0.Abs(),
				Commission1:        commission1.Abs(),
				Commission0InPrice: commission0InPrice.Abs(),
				Commission1InPrice: commission1InPrice.Abs(),
				Height:             tx.Height,
				Timestamp:          tx.Timestamp,
			}
		} else {
			currStatMap[tx.PairId] = pairStat{
				PairId:             tx.PairId,
				ChainId:            t.chainId,
				Volume0:            stat.Volume0.Add(volume0.Abs()),
				Volume1:            stat.Volume1.Add(volume1.Abs()),
				Volume0InPrice:     stat.Volume0InPrice.Add(volume0InPrice.Abs()),
				Volume1InPrice:     stat.Volume1InPrice.Add(volume1InPrice.Abs()),
				Liquidity0:         liquidity0,
				Liquidity1:         liquidity1,
				Liquidity0InPrice:  stat.Liquidity0.Quo(decimal0).Mul(price0),
				Liquidity1InPrice:  stat.Liquidity1.Quo(decimal1).Mul(price1),
				Commission0:        stat.Commission0.Add(commission0.Abs()),
				Commission1:        stat.Commission1.Add(commission1.Abs()),
				Commission0InPrice: stat.Commission0InPrice.Add(commission0InPrice.Abs()),
				Commission1InPrice: stat.Commission1InPrice.Add(commission1InPrice.Abs()),
				Height:             tx.Height,
				Timestamp:          tx.Timestamp,
			}
		}
	}

	// flush remains
	for _, s := range currStatMap {
		if s.Height > 0 {
			stats = append(stats, schemas.PairStatsRecent{
				PairId:             s.PairId,
				ChainId:            t.chainId,
				Volume0:            s.Volume0.String(),
				Volume1:            s.Volume1.String(),
				Volume0InPrice:     s.Volume0InPrice.String(),
				Volume1InPrice:     s.Volume1InPrice.String(),
				Liquidity0:         s.Liquidity0.String(),
				Liquidity1:         s.Liquidity1.String(),
				Liquidity0InPrice:  s.Liquidity0InPrice.String(),
				Liquidity1InPrice:  s.Liquidity1InPrice.String(),
				Commission0:        s.Commission0.String(),
				Commission1:        s.Commission1.String(),
				Commission0InPrice: s.Commission0InPrice.String(),
				Commission1InPrice: s.Commission1InPrice.String(),
				PriceToken:         t.priceToken,
				Height:             s.Height,
				Timestamp:          s.Timestamp,
			})
		}
	}

	return stats, nil
}

func (t *pairStatsRecentUpdateTask) searchPrice(tokenIdStr string, targetHeight uint64, priceMap map[uint64][]schemas.Price) (cmath.LegacyDec, error) {
	tokenId, err := strconv.ParseUint(tokenIdStr, 10, 64)
	if err != nil {
		return cmath.LegacyZeroDec(), err
	}
	priceStr := "0"
	if ps, ok := priceMap[tokenId]; ok {
		for _, p := range ps {
			if p.Height > targetHeight {
				break
			}
			priceStr = p.Price
		}
	}
	price, err := cmath.LegacyNewDecFromStr(priceStr)
	if err != nil {
		return cmath.LegacyZeroDec(), errors.Wrap(err, "pairStatsRecentUpdateTask.searchPrice")
	}

	return price, nil
}

func newPairStatsUpdateTask(config configs.AggregatorConfig, srcRepo parser.ReadRepository, destRepo repo.Repo, logger logging.Logger, parentTasks []task) predeterminedTimeTask {
	return &pairStatsUpdateTask{
		taskImpl: taskImpl{
			chainId:         config.ChainId,
			destDb:          destRepo,
			parentTasks:     parentTasks,
			taskWaitTimeout: taskWaitTimeout(config),
			logger:          logger,
		},
		priceToken:  config.PriceToken,
		srcDb:       srcRepo,
		prevStatMap: make(map[uint64]schemas.PairStats30m),
	}
}

func (t *pairStatsUpdateTask) StartTimestamp(ctx context.Context, startTs time.Time) (time.Time, error) {
	if !startTs.IsZero() {
		return startTs, nil
	}

	destTsF, err := t.destDb.LatestTimestamp(ctx, schemas.PairStats30m{}.TableName())
	if err != nil {
		return time.Time{}, err
	}
	srcTsF, err := t.srcDb.OldestTxTimestamp(ctx)
	if err != nil {
		return time.Time{}, err
	}

	destTs := util.ToTime(destTsF)
	srcTs := util.ToTime(srcTsF)
	if destTs.Before(srcTs) {
		return srcTs, nil
	}

	return destTs, nil
}

func (t *pairStatsUpdateTask) Execute(ctx context.Context, start time.Time, end time.Time) error {
	lastHeight, err := t.srcDb.HeightOnTimestamp(ctx, util.ToEpoch(end))
	if err != nil {
		return err
	}
	if err := waitUntilReachingHeight(ctx, t.parentTasks, lastHeight, t.taskWaitTimeout); err != nil {
		return err
	}

	startTs := util.ToEpoch(start)
	endTs := util.ToEpoch(end)
	stats, err := t.srcDb.PairStats(ctx, startTs, endTs, t.priceToken, t.prevStatMap)
	if err != nil {
		return err
	}
	if len(stats) == 0 {
		// no stats, skip remaining steps
		return nil
	}

	lpMap, err := t.srcDb.LiquiditiesOfPairStats(ctx, startTs, endTs, t.priceToken)
	if err != nil {
		return err
	}

	// one gap usually spans every pair of the window, so the pairs are collected and
	// reported once instead of a warning per pair
	carriedPairs, zeroedPairs := []uint64{}, []uint64{}
	for i, s := range stats {
		lp, ok := lpMap[s.PairId]
		if !ok {
			// a pair with transactions in the window has an lp history too, so a gap
			// means the lp history task left one. the liquidity is unknown, not drained:
			// carry the last known one over instead of reporting the pair as empty
			last, carried, err := t.lastKnownLiquidity(ctx, s.PairId, endTs)
			if err != nil {
				return err
			}
			if carried {
				carriedPairs = append(carriedPairs, s.PairId)
			} else {
				zeroedPairs = append(zeroedPairs, s.PairId)
			}
			lp = last
		}

		s.Liquidity0 = lp.Liquidity0
		s.Liquidity0InPrice = lp.Liquidity0InPrice
		s.Liquidity1 = lp.Liquidity1
		s.Liquidity1InPrice = lp.Liquidity1InPrice
		stats[i] = s

		t.prevStatMap[s.PairId] = s
	}

	if len(carriedPairs) > 0 || len(zeroedPairs) > 0 {
		t.logger.Warnf(
			"No lp history in the timeframe '%s - %s': carried the previous liquidity over for pairs %v, zeroed pairs %v with none.",
			start.String(), end.String(), carriedPairs, zeroedPairs)
	}

	if len(stats) > 0 {
		if err := t.destDb.UpdatePairStats(ctx, stats); err != nil {
			return err
		}
	}
	t.advanceHeight(lastHeight)

	t.logger.Infof("Complete pair stats update for the timeframe '%s - %s'.", start.String(), end.String())

	return nil
}

// lastKnownLiquidity answers what a pair held before the window ending at endTs: the
// stats of the previous window, else the newest row already written before it. It
// reports whether it found any; a pair with no history at all still needs zeros, since
// the numeric columns of pair_stats_30m reject the empty string.
//
// Both sources stay behind endTs. A rerun of an older window runs with later windows
// already written, and copying one of those back would date future liquidity to a
// historical row.
func (t *pairStatsUpdateTask) lastKnownLiquidity(ctx context.Context, pairId uint64, endTs float64) (schemas.PairStats30m, bool, error) {
	if prev, ok := t.prevStatMap[pairId]; ok && prev.Timestamp < endTs {
		return prev, true, nil
	}

	// prevStatMap only remembers the windows this process aggregated, so a restart has
	// to read the previous one back from the database
	prev, ok, err := t.destDb.LatestPairStat(ctx, pairId, endTs)
	if err != nil {
		return schemas.PairStats30m{}, false, err
	}
	if ok {
		return prev, true, nil
	}

	return schemas.PairStats30m{
		Liquidity0:        "0",
		Liquidity1:        "0",
		Liquidity0InPrice: "0",
		Liquidity1InPrice: "0",
	}, false, nil
}

func newAccountStatsUpdateTask(config configs.AggregatorConfig, srcRepo parser.ReadRepository, destRepo repo.Repo, logger logging.Logger, parentTasks []task) predeterminedTimeTask {
	return &accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId:         config.ChainId,
			destDb:          destRepo,
			parentTasks:     parentTasks,
			taskWaitTimeout: taskWaitTimeout(config),
			logger:          logger,
		},
		priceToken: config.PriceToken,
		srcDb:      srcRepo,
	}
}

func (t *accountStatsUpdateTask) StartTimestamp(ctx context.Context, startTs time.Time) (time.Time, error) {
	if !startTs.IsZero() {
		return startTs, nil
	}

	destTsF, err := t.destDb.LatestTimestamp(ctx, schemas.AccountStats30m{}.TableName())
	if err != nil {
		return time.Time{}, err
	}
	srcTsF, err := t.srcDb.OldestTxTimestamp(ctx)
	if err != nil {
		return time.Time{}, err
	}

	destTs := util.ToTime(destTsF)
	srcTs := util.ToTime(srcTsF)
	if destTs.Before(srcTs) {
		return srcTs, nil
	}

	return destTs, nil
}

func (t *accountStatsUpdateTask) Execute(ctx context.Context, start time.Time, end time.Time) error {
	startEpoch, endEpoch := util.ToEpoch(start), util.ToEpoch(end)

	endHeight, err := t.srcDb.HeightOnTimestamp(ctx, endEpoch)
	if err != nil {
		return err
	}
	if err := waitUntilReachingHeight(ctx, t.parentTasks, endHeight, t.taskWaitTimeout); err != nil {
		return err
	}

	stats, err := t.srcDb.AccountStats(ctx, startEpoch, endEpoch, t.priceToken)
	if err != nil {
		return err
	}

	if len(stats) > 0 {
		addresses := uniqueAccountAddresses(stats)
		if err := t.destDb.CreateAccounts(ctx, addresses); err != nil {
			return err
		}

		accountIds, err := t.destDb.AccountIds(ctx, addresses)
		if err != nil {
			return err
		}

		for i, s := range stats {
			accountId, ok := accountIds[s.Address]
			if !ok {
				return errors.Errorf("accountStatsUpdateTask.Execute: account id not found for address %s", s.Address)
			}

			s.YearUtc = end.Year()
			s.MonthUtc = int(end.Month())
			s.DayUtc = end.Day()
			s.HourUtc = end.Hour()
			s.MinuteUtc = end.Minute()
			s.Timestamp = endEpoch
			s.ChainId = t.chainId
			s.AccountId = accountId
			stats[i] = s
		}

		err = t.destDb.UpdateAccountStats(ctx, stats)
		if err != nil {
			return err
		}
	}

	t.advanceHeight(endHeight)

	t.logger.Infof("Complete account stats update for the timeframe '%s - %s'.", start.String(), end.String())

	return nil
}

func uniqueAccountAddresses(stats []schemas.AccountStats30m) []string {
	addressMap := make(map[string]bool, len(stats))
	addresses := make([]string, 0, len(stats))
	for _, stat := range stats {
		if addressMap[stat.Address] {
			continue
		}
		addressMap[stat.Address] = true
		addresses = append(addresses, stat.Address)
	}

	return addresses
}

// taskWaitTimeout resolves the configured wait timeout, falling back to the
// default when the config omits it or sets a non-positive value.
func taskWaitTimeout(config configs.AggregatorConfig) time.Duration {
	if config.TaskWaitTimeout <= 0 {
		return configs.DefaultTaskWaitTimeout
	}

	return config.TaskWaitTimeout
}

// waitUntilReachingHeight blocks until every parent task reaches targetHeight
// or the context/timeout ends the wait.
func waitUntilReachingHeight(ctx context.Context, parentTasks []task, targetHeight uint64, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = configs.DefaultTaskWaitTimeout
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for _, parent := range parentTasks {
		for {
			currentHeight := parent.LastProcessedHeight()
			if currentHeight >= targetHeight {
				break
			}

			if !waitFor(waitCtx, WaitPeriod) {
				if isShutdown(ctx) {
					return errors.Wrapf(ctx.Err(), "waitUntilReachingHeight: shutting down while waiting for task %s to reach target height %d; current height=%d", parent.Name(), targetHeight, currentHeight)
				}

				return errors.Wrapf(ErrParentBehind, "waitUntilReachingHeight: parent task %s did not reach target height %d; current height=%d timeout=%s", parent.Name(), targetHeight, currentHeight, timeout)
			}
		}
	}

	return nil
}
