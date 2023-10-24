package aggregator

import (
	"context"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/price"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/router"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
	"math"
	"strconv"
	"time"

	"github.com/dezswap/cosmwasm-etl/aggregator/repo"
	"github.com/dezswap/cosmwasm-etl/pkg/db/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
)

const LP_HISTORY_UPDATE_LIMIT = 100
const WAIT_PERIOD_SEC = 10 * time.Second

type task interface {
	Schedule(ctx context.Context, startTs time.Time) error
	Execute(start time.Time, end time.Time) error
	LastProcessedHeight() uint64
}

type taskImpl struct {
	chainId     string
	destDb      repo.Repo
	interval    time.Duration
	parentTasks []task
	logger      logging.Logger

	// State
	lastProcessedHeight uint64
}

type lpHistoryTask struct {
	taskImpl

	srcDb parser.ReadRepository
}

type routerTask struct {
	taskImpl

	router router.Router
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

type pairStatsIn24hUpdateTask struct {
	taskImpl

	priceToken string
	srcDb      parser.ReadRepository
}

type accountStatsUpdateTask struct {
	taskImpl

	srcDb parser.ReadRepository
}

var _ task = &accountStatsUpdateTask{}

func NewLpHistoryTask(config configs.AggregatorConfig, srcRepo parser.ReadRepository, destRepo repo.Repo, logger logging.Logger) *lpHistoryTask {
	return &lpHistoryTask{
		taskImpl: taskImpl{
			chainId:  config.ChainId,
			destDb:   destRepo,
			interval: 5 * time.Minute,
			logger:   logger,
		},
		srcDb: srcRepo,
	}
}

func (t *lpHistoryTask) Schedule(ctx context.Context, _ time.Time) error {
	startTs := time.Now()
	done := false
	for {
		select {
		case <-ctx.Done():
			done = true
			break
		case <-time.After(time.Until(startTs)):
			if err := t.Execute(startTs, time.Time{}); err != nil {
				errChan <- err
			}
			next := startTs.Truncate(t.interval).Add(t.interval)
			if next.Before(time.Now()) {
				startTs = time.Now().Truncate(t.interval).Add(t.interval)
			} else {
				startTs = next
			}
			t.logger.Debugf("The task for lp history has been finished in goroutine.")
		}
		if done {
			break
		}
	}

	return nil
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
		txs, err := t.srcDb.GetParsedTxsWithLimit(t.lastProcessedHeight+1, LP_HISTORY_UPDATE_LIMIT)
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

func (t *lpHistoryTask) LastProcessedHeight() uint64 {
	return t.lastProcessedHeight
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
			return nil, err
		}
		volume1, err := types.NewDecFromStr(tx.Asset1Amount)
		if err != nil {
			return nil, err
		}

		lp, ok := pairIdLpHistoryMap[tx.PairId]
		if !ok {
			// initialize with the latest lp
			if latestLp, ok := latestLpMap[tx.PairId]; ok {
				lp[repo.Liquidity0], err = types.NewDecFromStr(latestLp[repo.Liquidity0])
				if err != nil {
					return nil, err
				}
				lp[repo.Liquidity1], err = types.NewDecFromStr(latestLp[repo.Liquidity1])
				if err != nil {
					return nil, err
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

func NewRouterTask(config configs.AggregatorConfig, logger logging.Logger) (*routerTask, error) {
	srcRepo := router.NewSrcRepo(config.ChainId, config.SrcDb)
	r, err := router.New(srcRepo, config.Router, logger)
	if err != nil {
		return &routerTask{}, err
	}

	return &routerTask{
		taskImpl: taskImpl{
			chainId:  config.ChainId,
			interval: 5 * time.Minute,
			logger:   logger,
		},
		router: r,
	}, nil
}

func (t *routerTask) Schedule(ctx context.Context, _ time.Time) error {
	startTs := time.Now()
	done := false
	for {
		select {
		case <-ctx.Done():
			done = true
			break
		case <-time.After(time.Until(startTs)):
			if err := t.Execute(startTs, time.Time{}); err != nil {
				errChan <- err
			}
			next := startTs.Truncate(t.interval).Add(t.interval)
			if next.Before(time.Now()) {
				startTs = time.Now().Truncate(t.interval).Add(t.interval)
			} else {
				startTs = next
			}
			t.logger.Debugf("The task for route has been finished in goroutine.")
		}
		if done {
			break
		}
	}

	return nil
}

func (t *routerTask) Execute(_ time.Time, _ time.Time) error {
	return t.router.Update()
}

func (t *routerTask) LastProcessedHeight() uint64 {
	return t.lastProcessedHeight
}

func NewPriceTask(config configs.AggregatorConfig, destRepo repo.Repo, logger logging.Logger, parentTasks []task) (*priceTask, error) {
	srcRepo := price.NewRepo(config.ChainId, config.SrcDb)
	pt, err := price.New(srcRepo, config.PriceToken, logger)
	if err != nil {
		return &priceTask{}, err
	}

	return &priceTask{
		taskImpl: taskImpl{
			chainId:     config.ChainId,
			destDb:      destRepo,
			interval:    5 * time.Minute,
			parentTasks: parentTasks,
			logger:      logger,
		},
		priceTracker: pt,
	}, nil
}

func (t *priceTask) Schedule(ctx context.Context, _ time.Time) error {
	startTs := time.Now()
	done := false
	for {
		select {
		case <-ctx.Done():
			done = true
			break
		case <-time.After(time.Until(startTs)):
			if err := t.Execute(startTs, time.Time{}); err != nil {
				errChan <- err
			}
			next := startTs.Truncate(t.interval).Add(t.interval)
			if next.Before(time.Now()) {
				startTs = time.Now().Truncate(t.interval).Add(t.interval)
			} else {
				startTs = next
			}
			t.logger.Debugf("The task for price has been finished in goroutine.")
		}
		if done {
			break
		}
	}

	return nil
}

func (t *priceTask) Execute(_ time.Time, _ time.Time) error {
	height := uint64(0)

	for {
		nextHeight, err := t.priceTracker.NextHeight(height)
		if err != nil {
			return err
		}
		if nextHeight == price.NA_VALUE {
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

func (t *priceTask) LastProcessedHeight() uint64 {
	return t.lastProcessedHeight
}

func NewPairStatsIn24hUpdateTask(config configs.AggregatorConfig, srcRepo parser.ReadRepository, destRepo repo.Repo, logger logging.Logger, parentTasks []task) *pairStatsIn24hUpdateTask {
	return &pairStatsIn24hUpdateTask{
		taskImpl: taskImpl{
			chainId:     config.ChainId,
			destDb:      destRepo,
			interval:    5 * time.Minute,
			parentTasks: parentTasks,
			logger:      logger,
		},
		priceToken: config.PriceToken,
		srcDb:      srcRepo,
	}
}

func (t *pairStatsIn24hUpdateTask) Schedule(ctx context.Context, _ time.Time) error {
	ts := time.Now()
	done := false
	for {
		select {
		case <-ctx.Done():
			done = true
			break
		case <-time.After(time.Until(ts)):
			if err := t.Execute(ts.Add(-24*time.Hour), ts); err != nil {
				errChan <- err
			}
			ts = ts.Truncate(t.interval).Add(t.interval)
			t.logger.Debugf("The task for pair stats in 24h has been finished in goroutine.")
		}
		if done {
			break
		}
	}

	return nil
}

func (t *pairStatsIn24hUpdateTask) Execute(start time.Time, end time.Time) error {
	if t.lastProcessedHeight == 0 {
		var err error
		t.lastProcessedHeight, err = t.destDb.LastHeightOfPairStatsIn24h()
		if err != nil {
			return err
		}
	}
	startHeight, err := t.srcDb.HeightOnTimestamp(util.ToEpoch(start))
	if err != nil {
		return err
	}

	endHeight, err := t.srcDb.HeightOnTimestamp(util.ToEpoch(end))
	if err != nil {
		return err
	}

	var stats []schemas.PairStatsIn24h
	if endHeight > t.lastProcessedHeight {
		waitUntilReachingHeight(&t.parentTasks, endHeight)

		if startHeight <= t.lastProcessedHeight {
			startHeight = t.lastProcessedHeight + 1
		}
		txs, err := t.srcDb.GetParsedTxsInRecent24h(startHeight, endHeight)
		if err != nil {
			return err
		}

		tokenIdMap := make(map[string]bool)
		for _, tx := range txs {
			tokenIdMap[tx.Price0] = true
			tokenIdMap[tx.Price1] = true
		}

		var tokenIds []string
		for key := range tokenIdMap {
			tokenIds = append(tokenIds, key)
		}

		priceMap, err := t.srcDb.PriceInRecent24h(startHeight, endHeight, tokenIds, t.priceToken)
		if err != nil {
			return err
		}

		stats, err = t.generateStats(txs, priceMap)
		if err != nil {
			return err
		}
	}

	dbTx, err := t.destDb.BeginTx()
	if err != nil {
		return err
	}

	if len(stats) > 0 {
		err = t.destDb.UpdatePairStatsIn24h(dbTx, stats)
		if err != nil {
			return err
		}
	}

	err = t.destDb.DeletePairStatsIn24h(dbTx, start)
	if err != nil {
		return err
	}
	if dbTx = dbTx.Commit(); dbTx.Error != nil {
		return errors.Wrap(dbTx.Error, "pairStatsIn24hUpdateTask.Execute")
	}

	t.lastProcessedHeight = endHeight

	t.logger.Infof("Complete pair stats in 24h update.")

	return nil
}

func (t pairStatsIn24hUpdateTask) generateStats(txs []schemas.ParsedTxWithPrice, priceMap map[uint64][]schemas.Price) ([]schemas.PairStatsIn24h, error) {
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

	stats := make([]schemas.PairStatsIn24h, 0)
	currStatMap := make(map[uint64]pairStat, 0)
	for _, tx := range txs {
		price0, err := t.searchPrice(tx.Price0, tx.Height, priceMap)
		if err != nil {
			return nil, err
		}

		price1, err := t.searchPrice(tx.Price1, tx.Height, priceMap)
		if err != nil {
			return nil, err
		}

		decimal0 := types.NewDec(10).Power(uint64(tx.Decimals0))
		decimal1 := types.NewDec(10).Power(uint64(tx.Decimals1))

		volume0, err := types.NewDecFromStr(tx.Asset0Amount)
		if err != nil {
			return nil, err
		}
		volume0InPrice := volume0.Quo(decimal0).Mul(price0)

		volume1, err := types.NewDecFromStr(tx.Asset1Amount)
		if err != nil {
			return nil, err
		}
		volume1InPrice := volume1.Quo(decimal1).Mul(price1)

		liquidity0, err := types.NewDecFromStr(tx.Asset0Liquidity)
		if err != nil {
			return nil, err
		}

		liquidity1, err := types.NewDecFromStr(tx.Asset1Liquidity)
		if err != nil {
			return nil, err
		}

		commission0, err := types.NewDecFromStr(tx.Commission0Amount)
		if err != nil {
			return nil, err
		}
		commission0InPrice := commission0.Quo(decimal0).Mul(price0)

		commission1, err := types.NewDecFromStr(tx.Commission1Amount)
		if err != nil {
			return nil, err
		}
		commission1InPrice := commission1.Quo(decimal1).Mul(price1)

		stat := currStatMap[tx.PairId]
		if stat.Height != tx.Height {
			if stat.Height > 0 { // is not first
				stats = append(stats, schemas.PairStatsIn24h{
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
			stats = append(stats, schemas.PairStatsIn24h{
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

func (t pairStatsIn24hUpdateTask) searchPrice(tokenIdStr string, targetHeight uint64, priceMap map[uint64][]schemas.Price) (types.Dec, error) {
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
		return types.ZeroDec(), err
	}

	return price, nil
}

func (t *pairStatsIn24hUpdateTask) LastProcessedHeight() uint64 {
	return t.lastProcessedHeight
}

func NewPairStatsUpdateTask(config configs.AggregatorConfig, srcRepo parser.ReadRepository, destRepo repo.Repo, logger logging.Logger, parentTasks []task) *pairStatsUpdateTask {
	return &pairStatsUpdateTask{
		taskImpl: taskImpl{
			chainId:     config.ChainId,
			destDb:      destRepo,
			interval:    30 * time.Minute,
			parentTasks: parentTasks,
			logger:      logger,
		},
		priceToken: config.PriceToken,
		srcDb:      srcRepo,
	}
}

func (t *pairStatsUpdateTask) Schedule(ctx context.Context, startTs time.Time) error {
	startTs, err := t.startTimestamp(startTs)
	if err != nil {
		return err
	}

	start, end := t.timeframe(startTs)
	done := false

	for end.Before(time.Now()) {
		if done {
			return nil
		}
		if err := t.Execute(start, end); err != nil {
			errChan <- err
		}
		start = end
		end = end.Add(t.interval)
	}

	for {
		select {
		case <-ctx.Done():
			done = true
			break
		case <-time.After(time.Until(end)):
			if err := t.Execute(start, end); err != nil {
				errChan <- err
			}
			start = end
			end = end.Add(t.interval)
			t.logger.Debugf("The task for pair stats by 30m has been finished in goroutine.")
		}
		if done {
			break
		}
	}

	return nil
}

func (t pairStatsUpdateTask) startTimestamp(startTs time.Time) (time.Time, error) {
	if !startTs.IsZero() {
		return startTs, nil
	}

	destTsF, err := t.destDb.LatestTimestampOfPairStats()
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

func (pairStatsUpdateTask) timeframe(ts time.Time) (time.Time, time.Time) {
	start := ts.Truncate(30 * time.Minute).UTC()
	end := start.Add(30 * time.Minute).UTC()

	return start, end
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

func (t *pairStatsUpdateTask) LastProcessedHeight() uint64 {
	return t.lastProcessedHeight
}

func (t *accountStatsUpdateTask) Schedule(_ context.Context, _ time.Time) error {
	// TODO: implement
	return t.Execute(time.Time{}, time.Time{})
}

func (t *accountStatsUpdateTask) Execute(start time.Time, end time.Time) error {
	startEpoch, endEpoch := util.ToEpoch(start), util.ToEpoch(end)

	if err := t.updateAccounts(startEpoch, endEpoch); err != nil {
		return err
	}

	accounts, err := t.destDb.Accounts(endEpoch)
	if err != nil {
		return err
	}

	for id, address := range accounts {
		pairIds, err := t.srcDb.NewPairIds(address, startEpoch, endEpoch)
		if err != nil {
			return err
		}
		if pis, err := t.destDb.HoldingPairIds(id); err != nil {
			return err
		} else {
			pairIds = append(pairIds, pis...)
		}

		for _, pairId := range pairIds {
			stats := schemas.NewUserStat30min(t.chainId, end, pairId, id)

			stats.TxCnt, err = t.srcDb.TxCountOfAccount(address, pairId, startEpoch, endEpoch)
			if err != nil {
				return err
			}

			stats.Asset0Amount, stats.Asset1Amount, stats.TotalLpAmount, err = t.srcDb.AssetAmountInPairOfAccount(address, pairId, startEpoch, endEpoch)
			if err != nil {
				return err
			}

			if err := t.destDb.UpdateAccountStats(stats); err != nil {
				return err
			}
		}
	}

	t.logger.Infof("Complete account stats update.")

	return nil
}

func (t *accountStatsUpdateTask) LastProcessedHeight() uint64 {
	return t.lastProcessedHeight
}

func (t accountStatsUpdateTask) updateAccounts(startEpoch float64, endEpoch float64) error {
	accounts, err := t.srcDb.NewAccounts(startEpoch, endEpoch)
	if err != nil {
		return err
	}

	if len(accounts) > 0 {
		if err := t.destDb.CreateAccounts(accounts); err != nil {
			return err
		}
	}

	return nil
}

func waitUntilReachingHeight(parentTasks *[]task, targetHeight uint64) {
	for _, t := range *parentTasks {
		for {
			if t.LastProcessedHeight() < targetHeight {
				time.Sleep(WAIT_PERIOD_SEC)
			} else {
				break
			}
		}
	}
}
