package aggregator

import (
	"math"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
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
	Execute(start time.Time, end time.Time) error
	LastProcessedHeight() uint64
}

type predeterminedTimeTask interface {
	task
	StartTimestamp(startTs time.Time) (time.Time, error)
}

type taskImpl struct {
	chainId     string
	destDb      repo.Repo
	parentTasks []task
	logger      logging.Logger

	// State
	lastProcessedHeight uint64
}

func (t *taskImpl) LastProcessedHeight() uint64 {
	return t.lastProcessedHeight
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

	priceToken string
	srcDb      parser.ReadRepository
}

type pairStatsRecentUpdateTask struct {
	taskImpl

	priceToken string
	srcDb      parser.ReadRepository
	timeRange  time.Duration
}

type accountStatsUpdateTask struct {
	taskImpl

	srcDb parser.ReadRepository
}

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

func (t *lpHistoryTask) Execute(_ time.Time, _ time.Time) error {
	lastHistories, err := t.destDb.LastLpHistory(uint64(math.MaxInt64))
	if err != nil {
		return err
	}

	latestLpMap := make(map[uint64][]string)
	for _, h := range lastHistories {
		latestLpMap[h.PairId] = []string{h.Liquidity0, h.Liquidity1}
		if h.Height > t.lastProcessedHeight {
			t.lastProcessedHeight = h.Height
		}
	}

	for {
		txs, err := t.srcDb.GetParsedTxsWithLimit(t.lastProcessedHeight+1, LpHistoryUpdateLimit)
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
		err = t.destDb.UpdateLpHistory(history)
		if err != nil {
			return err
		}
		t.lastProcessedHeight = history[len(history)-1].Height
	}

	t.logger.Infof("Complete lp history update.")

	return nil
}

func (t lpHistoryTask) generateHistory(latestLpMap map[uint64][]string, txs []schemas.ParsedTxWithPrice) ([]schemas.LpHistory, error) {
	history := []schemas.LpHistory{}

	pairIdLpHistoryMap := make(map[uint64][2]types.Dec)

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

		volume0, err := types.NewDecFromStr(tx.Asset0Amount)
		if err != nil {
			return nil, errors.Wrap(err, "lpHistoryTask.generateHistory")
		}
		volume1, err := types.NewDecFromStr(tx.Asset1Amount)
		if err != nil {
			return nil, errors.Wrap(err, "lpHistoryTask.generateHistory")
		}

		lp, ok := pairIdLpHistoryMap[tx.PairId]
		if !ok {
			// initialize with the latest lp
			if latestLp, ok := latestLpMap[tx.PairId]; ok {
				lp[repo.Liquidity0], err = types.NewDecFromStr(latestLp[repo.Liquidity0])
				if err != nil {
					return nil, errors.Wrap(err, "lpHistoryTask.generateHistory")
				}
				lp[repo.Liquidity1], err = types.NewDecFromStr(latestLp[repo.Liquidity1])
				if err != nil {
					return nil, errors.Wrap(err, "lpHistoryTask.generateHistory")
				}
			} else {
				lp[repo.Liquidity0] = types.ZeroDec()
				lp[repo.Liquidity1] = types.ZeroDec()
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

func newRouterTask(config configs.AggregatorConfig, logger logging.Logger) task {
	repo := router.NewSrcRepo(config.ChainId, config.DestDb)

	return &routerTask{
		taskImpl: taskImpl{
			chainId: config.ChainId,
			logger:  logger,
		},
		router: router.New(repo, config.Router, logger),
		db:     repo,
	}
}

func (t *routerTask) Execute(_ time.Time, _ time.Time) error {
	pairs, err := t.db.Pairs()
	if err != nil {
		return err
	}

	if len(pairs) > t.pairCnt {
		t.pairCnt = len(pairs)
		return t.router.Update()
	}

	return nil
}

func newPriceTask(config configs.AggregatorConfig, destRepo repo.Repo, logger logging.Logger, parentTasks []task) (task, error) {
	pt, err := price.New(price.NewRepo(config.ChainId, config.SrcDb), config.PriceToken, logger)
	if err != nil {
		return nil, err
	}

	return &priceTask{
		taskImpl: taskImpl{
			chainId:     config.ChainId,
			destDb:      destRepo,
			parentTasks: parentTasks,
			logger:      logger,
		},
		priceTracker: pt,
	}, nil
}

func (t *priceTask) Execute(_ time.Time, _ time.Time) error {
	height := uint64(0)

	for {
		nextHeight, err := t.priceTracker.NextHeight(height)
		if err != nil {
			return err
		}
		if nextHeight == price.NaValue {
			currHeight, err := t.priceTracker.CurrHeight()
			if err != nil {
				return err
			}
			t.lastProcessedHeight = uint64(currHeight) // update the current height for the child tasks

			return nil
		}

		height = uint64(nextHeight)
		waitUntilReachingHeight(&t.parentTasks, height)

		err = t.priceTracker.Run(height)
		if err != nil {
			return err
		}
		t.lastProcessedHeight = height
	}
}

func newPairStatsRecentUpdateTask(config configs.AggregatorConfig, srcRepo parser.ReadRepository, destRepo repo.Repo, logger logging.Logger, parentTasks []task) task {
	return &pairStatsRecentUpdateTask{
		taskImpl: taskImpl{
			chainId:     config.ChainId,
			destDb:      destRepo,
			parentTasks: parentTasks,
			logger:      logger,
		},
		priceToken: config.PriceToken,
		srcDb:      srcRepo,
		timeRange:  48 * time.Hour,
	}
}

func (t *pairStatsRecentUpdateTask) Execute(_ time.Time, end time.Time) error {
	if t.lastProcessedHeight == 0 {
		var err error
		t.lastProcessedHeight, err = t.destDb.LastHeightOfPairStatsRecent()
		if err != nil {
			return err
		}
	}
	startTs := end.Add(-1 * t.timeRange)
	startHeight, err := t.srcDb.HeightOnTimestamp(util.ToEpoch(startTs))
	if err != nil {
		return err
	}

	endHeight, err := t.srcDb.HeightOnTimestamp(util.ToEpoch(end))
	if err != nil {
		return err
	}

	var stats []schemas.PairStatsRecent
	if endHeight > t.lastProcessedHeight {
		waitUntilReachingHeight(&t.parentTasks, endHeight)

		if startHeight <= t.lastProcessedHeight {
			startHeight = t.lastProcessedHeight + 1
		}
		txs, err := t.srcDb.GetRecentParsedTxs(startHeight, endHeight)
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

			priceMap, err := t.srcDb.RecentPrices(startHeight, endHeight, tokenIds, t.priceToken)
			if err != nil {
				return err
			}

			stats, err = t.generateStats(txs, priceMap)
			if err != nil {
				return err
			}
		}
	}

	dbTx, err := t.destDb.BeginTx()
	if err != nil {
		return err
	}

	if len(stats) > 0 {
		err = t.destDb.UpdatePairStatsRecent(dbTx, stats)
		if err != nil {
			return err
		}
	}

	err = t.destDb.DeletePairStatsRecent(dbTx, startTs)
	if err != nil {
		return err
	}
	if dbTx = dbTx.Commit(); dbTx.Error != nil {
		return errors.Wrap(dbTx.Error, "pairStatsRecentUpdateTask.Execute")
	}

	t.lastProcessedHeight = endHeight

	t.logger.Infof("Complete pair stats recent update.")

	return nil
}

func (t pairStatsRecentUpdateTask) generateStats(txs []schemas.ParsedTxWithPrice, priceMap map[uint64][]schemas.Price) ([]schemas.PairStatsRecent, error) {
	type pairStat struct {
		PairId             uint64
		ChainId            string
		Volume0            types.Dec
		Volume1            types.Dec
		Volume0InPrice     types.Dec
		Volume1InPrice     types.Dec
		Liquidity0         types.Dec
		Liquidity1         types.Dec
		Liquidity0InPrice  types.Dec
		Liquidity1InPrice  types.Dec
		Commission0        types.Dec
		Commission1        types.Dec
		Commission0InPrice types.Dec
		Commission1InPrice types.Dec
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

		decimal0 := types.NewDec(10).Power(uint64(tx.Decimals0))
		decimal1 := types.NewDec(10).Power(uint64(tx.Decimals1))

		volume0, err := types.NewDecFromStr(tx.Asset0Amount)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}
		volume0InPrice := volume0.Quo(decimal0).Mul(price0)

		volume1, err := types.NewDecFromStr(tx.Asset1Amount)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}
		volume1InPrice := volume1.Quo(decimal1).Mul(price1)

		liquidity0, err := types.NewDecFromStr(tx.Asset0Liquidity)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}

		liquidity1, err := types.NewDecFromStr(tx.Asset1Liquidity)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}

		commission0, err := types.NewDecFromStr(tx.Commission0Amount)
		if err != nil {
			return nil, errors.Wrap(err, "pairStatsRecentUpdateTask.generateStats")
		}
		commission0InPrice := commission0.Quo(decimal0).Mul(price0)

		commission1, err := types.NewDecFromStr(tx.Commission1Amount)
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

func (t pairStatsRecentUpdateTask) searchPrice(tokenIdStr string, targetHeight uint64, priceMap map[uint64][]schemas.Price) (types.Dec, error) {
	tokenId, err := strconv.ParseUint(tokenIdStr, 10, 64)
	if err != nil {
		return types.ZeroDec(), err
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
	price, err := types.NewDecFromStr(priceStr)
	if err != nil {
		return types.ZeroDec(), errors.Wrap(err, "pairStatsRecentUpdateTask.searchPrice")
	}

	return price, nil
}

func newPairStatsUpdateTask(config configs.AggregatorConfig, srcRepo parser.ReadRepository, destRepo repo.Repo, logger logging.Logger, parentTasks []task) predeterminedTimeTask {
	return &pairStatsUpdateTask{
		taskImpl: taskImpl{
			chainId:     config.ChainId,
			destDb:      destRepo,
			parentTasks: parentTasks,
			logger:      logger,
		},
		priceToken: config.PriceToken,
		srcDb:      srcRepo,
	}
}

func (t pairStatsUpdateTask) StartTimestamp(startTs time.Time) (time.Time, error) {
	if !startTs.IsZero() {
		return startTs, nil
	}

	destTsF, err := t.destDb.LatestTimestamp(schemas.PairStats30m{}.TableName())
	if err != nil {
		return time.Time{}, err
	}
	srcTsF, err := t.srcDb.OldestTxTimestamp()
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

func (t *pairStatsUpdateTask) Execute(start time.Time, end time.Time) error {
	lastHeight, err := t.srcDb.HeightOnTimestamp(util.ToEpoch(end))
	if err != nil {
		return err
	}
	waitUntilReachingHeight(&t.parentTasks, lastHeight)

	startTs := util.ToEpoch(start)
	endTs := util.ToEpoch(end)
	stats, err := t.srcDb.PairStats(startTs, endTs, t.priceToken)
	if err != nil {
		return err
	}
	lpMap, err := t.srcDb.LiquiditiesOfPairStats(startTs, endTs, t.priceToken)
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
	}

	if len(stats) > 0 {
		if err := t.destDb.UpdatePairStats(stats); err != nil {
			return err
		}
	}
	t.lastProcessedHeight = lastHeight

	t.logger.Infof("Complete pair stats update for the timeframe '%s - %s'.", start.String(), end.String())

	return nil
}

func newAccountStatsUpdateTask(config configs.AggregatorConfig, srcRepo parser.ReadRepository, destRepo repo.Repo, logger logging.Logger) predeterminedTimeTask {
	return &accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: config.ChainId,
			destDb:  destRepo,
			logger:  logger,
		},
		srcDb: srcRepo,
	}
}

func (t *accountStatsUpdateTask) StartTimestamp(startTs time.Time) (time.Time, error) {
	if !startTs.IsZero() {
		return startTs, nil
	}

	destTsF, err := t.destDb.LatestTimestamp(schemas.AccountStats30m{}.TableName())
	if err != nil {
		return time.Time{}, err
	}
	srcTsF, err := t.srcDb.OldestTxTimestamp()
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

func (t *accountStatsUpdateTask) Execute(start time.Time, end time.Time) error {
	startEpoch, endEpoch := util.ToEpoch(start), util.ToEpoch(end)

	stats, err := t.srcDb.AccountStats(startEpoch, endEpoch)
	if err != nil {
		return err
	}

	if len(stats) > 0 {
		for i, s := range stats {
			s.YearUtc = end.Year()
			s.MonthUtc = int(end.Month())
			s.DayUtc = end.Day()
			s.HourUtc = end.Hour()
			s.MinuteUtc = end.Minute()
			s.Timestamp = endEpoch
			s.ChainId = t.chainId
			stats[i] = s
		}

		err = t.destDb.UpdateAccountStats(stats)
		if err != nil {
			return err
		}
	}

	t.lastProcessedHeight, err = t.srcDb.HeightOnTimestamp(util.ToEpoch(end))
	if err != nil {
		return err
	}

	t.logger.Infof("Complete account stats update for the timeframe '%s - %s'.", start.String(), end.String())

	return nil
}

func waitUntilReachingHeight(parentTasks *[]task, targetHeight uint64) {
	for _, t := range *parentTasks {
		for {
			if t.LastProcessedHeight() < targetHeight {
				time.Sleep(WaitPeriod)
			} else {
				break
			}
		}
	}
}
