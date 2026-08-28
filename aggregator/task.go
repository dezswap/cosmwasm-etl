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

const LpHistoryUpdateLimit = 100
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

type lpHistoryTask struct {
	taskImpl

	srcDb parser.ReadRepository
}

type routerTask struct {
	taskImpl

	router  router.Router
	db      router.SrcRepo
	pairCnt int
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
		if h.Height > t.lastProcessedHeight.Load() {
			t.lastProcessedHeight.Store(h.Height)
		}
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
		t.lastProcessedHeight.Store(history[len(history)-1].Height)
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

func newRouterTask(config configs.AggregatorConfig, repo router.SrcRepo, logger logging.Logger) task {
	return &routerTask{
		taskImpl: taskImpl{
			chainId: config.ChainId,
			logger:  logger,
		},
		router: router.New(repo, config.Router, logger),
		db:     repo,
	}
}

func (t *routerTask) Execute(ctx context.Context, _ time.Time, _ time.Time) error {
	pairs, err := t.db.Pairs(ctx)
	if err != nil {
		return err
	}

	if len(pairs) > t.pairCnt {
		// only remember the new count once the rebuild succeeded, otherwise a failed
		// update would be skipped on every later run
		if err := t.router.Update(ctx); err != nil {
			return err
		}
		t.pairCnt = len(pairs)
	}

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

func (t *priceTask) Execute(ctx context.Context, _ time.Time, _ time.Time) error {
	height := uint64(0)

	for {
		nextHeight, err := t.priceTracker.NextHeight(ctx, height)
		if err != nil {
			return err
		}
		if nextHeight == price.NaValue {
			currHeight, err := t.priceTracker.CurrHeight(ctx)
			if err != nil {
				return err
			}
			t.lastProcessedHeight.Store(uint64(currHeight)) // update the current height for the child tasks

			return nil
		}

		height = uint64(nextHeight)
		if err := waitUntilReachingHeight(ctx, t.parentTasks, height, t.taskWaitTimeout); err != nil {
			return err
		}

		err = t.priceTracker.Run(ctx, height)
		if err != nil {
			return err
		}
		t.lastProcessedHeight.Store(height)
	}
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
		timeRange:  48 * time.Hour,
	}
}

func (t *pairStatsRecentUpdateTask) Execute(ctx context.Context, _ time.Time, end time.Time) error {
	lastProcessedHeight := t.lastProcessedHeight.Load()
	if lastProcessedHeight == 0 {
		loadedHeight, err := t.destDb.LastHeightOfPairStatsRecent(ctx)
		if err != nil {
			return err
		}
		t.lastProcessedHeight.Store(loadedHeight)
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

	var stats []schemas.PairStatsRecent
	if endHeight > lastProcessedHeight {
		if err := waitUntilReachingHeight(ctx, t.parentTasks, endHeight, t.taskWaitTimeout); err != nil {
			return err
		}

		if startHeight <= lastProcessedHeight {
			startHeight = lastProcessedHeight + 1
		}
		txs, err := t.srcDb.GetRecentParsedTxs(ctx, startHeight, endHeight)
		if err != nil {
			return err
		}

		if len(txs) > 0 {
			tokenIdMap := make(map[string]bool)
			for _, tx := range txs {
				tokenIdMap[tx.Price0] = true
				tokenIdMap[tx.Price1] = true
			}

			var tokenIds []string
			for key := range tokenIdMap {
				tokenIds = append(tokenIds, key)
			}

			priceMap, err := t.srcDb.RecentPrices(ctx, startHeight, endHeight, tokenIds, t.priceToken)
			if err != nil {
				return err
			}

			stats, err = t.generateStats(txs, priceMap)
			if err != nil {
				return err
			}
		}
	}

	err = t.destDb.WithinTx(ctx, func(txRepo repo.Repo) error {
		if len(stats) > 0 {
			if err := txRepo.UpdatePairStatsRecent(ctx, stats); err != nil {
				return err
			}
		}
		return txRepo.DeletePairStatsRecent(ctx, startTs)
	})
	if err != nil {
		return err
	}

	t.lastProcessedHeight.Store(endHeight)

	t.logger.Infof("Complete pair stats recent update.")

	return nil
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

	for i, s := range stats {
		if lp, ok := lpMap[s.PairId]; ok {
			s.Liquidity0 = lp.Liquidity0
			s.Liquidity0InPrice = lp.Liquidity0InPrice
			s.Liquidity1 = lp.Liquidity1
			s.Liquidity1InPrice = lp.Liquidity1InPrice
			stats[i] = s
		}

		t.prevStatMap[s.PairId] = s
	}

	if len(stats) > 0 {
		if err := t.destDb.UpdatePairStats(ctx, stats); err != nil {
			return err
		}
	}
	t.lastProcessedHeight.Store(lastHeight)

	t.logger.Infof("Complete pair stats update for the timeframe '%s - %s'.", start.String(), end.String())

	return nil
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

	t.lastProcessedHeight.Store(endHeight)

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
				return errors.Wrapf(waitCtx.Err(), "waitUntilReachingHeight: parent task did not reach target height %d; current height=%d timeout=%s", targetHeight, currentHeight, timeout)
			}
		}
	}

	return nil
}
