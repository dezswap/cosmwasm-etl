package aggregator

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/aggregator/repo"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/price"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/router"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"

	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type completedTask struct {
	height atomic.Uint64
}

func (t *completedTask) Name() string { return "completed" }

func (t *completedTask) Execute(_ context.Context, _ time.Time, _ time.Time) error {
	return nil
}

func (t *completedTask) LastProcessedHeight() uint64 {
	return t.height.Load()
}

func (t *completedTask) setLastProcessedHeight(height uint64) {
	t.height.Store(height)
}

func TestWaitUntilReachingHeightAlreadyReached(t *testing.T) {
	assert := assert.New(t)

	parent := &completedTask{}
	parent.setLastProcessedHeight(10)

	err := waitUntilReachingHeight(context.Background(), []task{parent}, 10, time.Minute)

	assert.NoError(err)
}

func TestWaitUntilReachingHeightContextCanceled(t *testing.T) {
	assert := assert.New(t)

	parent := &completedTask{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitUntilReachingHeight(ctx, []task{parent}, 10, time.Minute)

	assert.ErrorIs(err, context.Canceled)
	// shutdown is not a retryable "parent is behind"; conflating them keeps schedulers
	// retrying instead of stopping
	assert.NotErrorIs(err, ErrParentBehind)
	assert.ErrorContains(err, "target height 10")
	assert.ErrorContains(err, "current height=0")
}

// Running out of the wait timeout means the parent is merely behind, not broken;
// reporting it as a plain timeout failed the run and took every other task with it.
func TestWaitUntilReachingHeightTimeout(t *testing.T) {
	assert := assert.New(t)

	parent := &completedTask{}

	err := waitUntilReachingHeight(context.Background(), []task{parent}, 10, time.Millisecond)

	assert.ErrorIs(err, ErrParentBehind)
	assert.NotErrorIs(err, context.Canceled)
	assert.ErrorContains(err, "target height 10")
	assert.ErrorContains(err, "timeout=1ms")
}

// stubPrice keeps srcHeight independent of the candidates it prices: the input reaches
// further than the last price row whenever the newest transactions are not swaps.
type stubPrice struct {
	srcHeight  int64
	candidates []int64
	priced     []uint64
}

func (p *stubPrice) SrcHeight(context.Context) (int64, error) { return p.srcHeight, nil }

func (p *stubPrice) NextHeight(_ context.Context, minHeight uint64) (int64, error) {
	for _, candidate := range p.candidates {
		// the real query excludes heights already priced, so a rerun resumes
		if candidate > int64(minHeight) && !slices.Contains(p.priced, uint64(candidate)) {
			return candidate, nil
		}
	}

	return price.NaValue, nil
}

func (p *stubPrice) Run(_ context.Context, height uint64) error {
	p.priced = append(p.priced, height)

	return nil
}

func newPriceTaskForTest(tracker price.Price) *priceTask {
	return &priceTask{
		taskImpl:     taskImpl{chainId: "cube_47-5", logger: logging.Discard},
		priceTracker: tracker,
	}
}

// Children gate on the source height, so reporting the last written price height
// leaves them waiting for a height this task never claims.
func TestPriceTaskReportsSourceTipRatherThanLastPricedHeight(t *testing.T) {
	tracker := &stubPrice{srcHeight: 100, candidates: []int64{10, 20}}
	task := newPriceTaskForTest(tracker)

	require.NoError(t, task.Execute(context.Background(), time.Time{}, time.Time{}))

	require.Equal(t, []uint64{10, 20}, tracker.priced)
	require.Equal(t, uint64(100), task.LastProcessedHeight())
}

// The tip is read before the loop, so heights arriving while it runs belong to the
// next round rather than being vouched for unexamined.
func TestPriceTaskStopsAtTheTipItStartedWith(t *testing.T) {
	tracker := &stubPrice{srcHeight: 15, candidates: []int64{10, 20}}
	task := newPriceTaskForTest(tracker)

	require.NoError(t, task.Execute(context.Background(), time.Time{}, time.Time{}))

	require.Equal(t, []uint64{10}, tracker.priced)
	require.Equal(t, uint64(15), task.LastProcessedHeight())
}

// A round whose last heights produced no price row used to publish max(price.height),
// dropping below what the same round had already reached.
func TestPriceTaskNeverMovesHeightBackward(t *testing.T) {
	tracker := &stubPrice{srcHeight: 30, candidates: []int64{10, 20, 30}}
	task := newPriceTaskForTest(tracker)

	require.NoError(t, task.Execute(context.Background(), time.Time{}, time.Time{}))
	require.Equal(t, uint64(30), task.LastProcessedHeight())

	tracker.srcHeight = 25
	require.NoError(t, task.Execute(context.Background(), time.Time{}, time.Time{}))
	require.Equal(t, uint64(30), task.LastProcessedHeight())
	require.Equal(t, []uint64{10, 20, 30}, tracker.priced, "a rerun must not reprice a height that already has a price row")
}

func TestAdvanceHeightIgnoresLowerValues(t *testing.T) {
	impl := taskImpl{}

	impl.advanceHeight(10)
	impl.advanceHeight(4)

	require.Equal(t, uint64(10), impl.LastProcessedHeight())
}

func TestLpHistoryTaskExecute(t *testing.T) {
	assert := assert.New(t)

	history := []schemas.LpHistory{
		{
			Height:     1,
			PairId:     1,
			ChainId:    "cube_47-5",
			Liquidity0: "1000000",
			Liquidity1: "1000000",
			Timestamp:  1692939766,
		},
	}

	txs := []schemas.ParsedTxWithPrice{
		{
			PairId:            1,
			ChainId:           "cube_47-5",
			Asset0Amount:      "1000000",
			Asset1Amount:      "-500000",
			Commission0Amount: "0",
			Commission1Amount: "50000",
			Price0:            "1",
			Price1:            "1",
			Decimals0:         6,
			Decimals1:         6,
			Height:            2,
			Timestamp:         1692939767,
		},
	}

	expected := []schemas.LpHistory{
		{
			Height:     2,
			PairId:     1,
			ChainId:    "cube_47-5",
			Liquidity0: "2000000.000000000000000000",
			Liquidity1: "500000.000000000000000000",
			Timestamp:  1692939767,
		},
	}

	rp := repoMock{}
	rp.On("LastLpHistory", mock.Anything).Return(history, nil)
	rp.On("GetParsedTxsWithLimit", mock.Anything, mock.Anything).Return(txs, nil)
	rp.On("UpdateLpHistory", mock.Anything).Return(nil)

	task := lpHistoryTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		srcDb: &rp,
	}

	err := task.Execute(context.Background(), time.Time{}, time.Time{})
	assert.NoError(err)
	assert.Equal(expected[0], rp.updatedLpHistory[0])
}

func TestPairStatsRecentUpdateTaskExecute(t *testing.T) {
	assert := assert.New(t)

	height := uint64(100000)
	txs := []schemas.ParsedTxWithPrice{
		{
			PairId:            1,
			ChainId:           "",
			Asset0Amount:      "3000000",
			Asset1Amount:      "3000000",
			Asset0Liquidity:   "3000000",
			Asset1Liquidity:   "3000000",
			Commission0Amount: "1000000",
			Commission1Amount: "1000000",
			Price0:            "1",
			Price1:            "2",
			Decimals0:         6,
			Decimals1:         6,
			Height:            height,
			Timestamp:         float64(0),
		},
		{
			PairId:            1,
			ChainId:           "",
			Asset0Amount:      "3000000",
			Asset1Amount:      "4000000",
			Asset0Liquidity:   "6000000",
			Asset1Liquidity:   "7000000",
			Commission0Amount: "1000000",
			Commission1Amount: "2000000",
			Price0:            "1",
			Price1:            "2",
			Decimals0:         6,
			Decimals1:         6,
			Height:            height + 1,
			Timestamp:         float64(0),
		},
	}

	priceMap := map[uint64][]schemas.Price{
		1: {
			schemas.Price{
				Height:  height,
				TokenId: 1,
				Price:   "1",
			},
			schemas.Price{
				Height:  height + 1,
				TokenId: 1,
				Price:   "2",
			},
		},
		2: {
			schemas.Price{
				Height:  height,
				TokenId: 2,
				Price:   "1",
			},
			schemas.Price{
				Height:  height,
				TokenId: 2,
				Price:   "2",
			},
		},
	}

	expected := []schemas.PairStatsRecent{
		{
			PairId:             1,
			ChainId:            "",
			Volume0:            "3000000.000000000000000000",
			Volume1:            "4000000.000000000000000000",
			Volume0InPrice:     "6.000000000000000000",
			Volume1InPrice:     "8.000000000000000000",
			Liquidity0:         "6000000.000000000000000000",
			Liquidity1:         "7000000.000000000000000000",
			Liquidity0InPrice:  "12.000000000000000000",
			Liquidity1InPrice:  "14.000000000000000000",
			Commission0:        "1000000.000000000000000000",
			Commission1:        "2000000.000000000000000000",
			Commission0InPrice: "2.000000000000000000",
			Commission1InPrice: "4.000000000000000000",
			Height:             height + 1,
			Timestamp:          float64(0),
		},
	}

	rp := repoMock{}
	rp.On("HeightOnTimestamp").Return(txs[0].Height, nil)
	rp.On("LastHeightOfPrice").Return(txs[len(txs)-1].Height, nil)
	rp.On("GetParsedTxsInHeightRange").Return(txs, nil)
	rp.On("PricesForHeightRange").Return(priceMap, nil)

	task := pairStatsRecentUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		priceToken: "",
		srcDb:      &rp,
	}

	err := task.Execute(context.Background(), time.Time{}, time.Time{})
	assert.NoError(err)
	assert.Equal(expected[0], rp.updatedPairStatsRecent[len(rp.updatedPairStatsRecent)-1])
}

// spanRecordingRepo serves transactions by height and records the spans it was asked
// for, so a test can see how a window was split and what each split wrote.
type spanRecordingRepo struct {
	*repoMock

	heightByTs  map[float64]uint64
	txsByHeight map[uint64][]schemas.ParsedTxWithPrice
	prices      []schemas.Price
	readErr     error

	readSpans [][2]uint64
	written   []schemas.PairStatsRecent
	pruneCnt  int
	// calls records prunes and writes in order; the task prunes twice for two different
	// reasons and only the order tells them apart.
	calls []string
}

// spanTestWindowEnd is the end the span tests hand to Execute. The task derives the
// window start from it, so the stub can key on the timestamp it is asked about.
var spanTestWindowEnd = time.Time{}

// HeightOnTimestamp answers by timestamp rather than by call order: if Execute ever
// asks about a window this stub was not built for, the test fails outright instead of
// quietly receiving the other end of the range.
func (r *spanRecordingRepo) HeightOnTimestamp(_ context.Context, ts float64) (uint64, error) {
	height, ok := r.heightByTs[ts]
	if !ok {
		return 0, fmt.Errorf("spanRecordingRepo: no height stubbed for timestamp %v", ts)
	}

	return height, nil
}

// GetParsedTxsInHeightRange mirrors the repository contract: rows ordered by height, then pair.
func (r *spanRecordingRepo) GetParsedTxsInHeightRange(_ context.Context, startHeight, endHeight uint64) ([]schemas.ParsedTxWithPrice, error) {
	r.readSpans = append(r.readSpans, [2]uint64{startHeight, endHeight})
	if r.readErr != nil {
		return nil, r.readErr
	}

	var txs []schemas.ParsedTxWithPrice
	for height := startHeight; height <= endHeight; height++ {
		atHeight := slices.Clone(r.txsByHeight[height])
		slices.SortStableFunc(atHeight, func(a, b schemas.ParsedTxWithPrice) int {
			return cmp.Compare(a.PairId, b.PairId)
		})
		txs = append(txs, atHeight...)
	}

	return txs, nil
}

// PricesForHeightRange mirrors the real query's seeding: each token starts from its last price
// at or before startHeight. Returning only in-range prices instead would let a broken
// seed pass here while the aggregator valued transactions at zero.
func (r *spanRecordingRepo) PricesForHeightRange(_ context.Context, startHeight, endHeight uint64, _ []string, _ string) (map[uint64][]schemas.Price, error) {
	seed := map[uint64]uint64{}
	for _, p := range r.prices {
		if p.Height <= startHeight && p.Height >= seed[p.TokenId] {
			seed[p.TokenId] = p.Height
		}
	}

	prices := map[uint64][]schemas.Price{}
	for _, p := range r.prices {
		if p.Height >= seed[p.TokenId] && p.Height <= endHeight {
			prices[p.TokenId] = append(prices[p.TokenId], p)
		}
	}
	for _, ps := range prices {
		slices.SortStableFunc(ps, func(a, b schemas.Price) int {
			return cmp.Compare(a.Height, b.Height)
		})
	}

	return prices, nil
}

func (r *spanRecordingRepo) WithinTx(_ context.Context, fn func(repo.Repo) error) error {
	return fn(r)
}

func (r *spanRecordingRepo) UpdatePairStatsRecent(_ context.Context, stats []schemas.PairStatsRecent) error {
	r.written = append(r.written, stats...)
	r.calls = append(r.calls, "write")

	return nil
}

func (r *spanRecordingRepo) DeletePairStatsRecent(_ context.Context, _ time.Time) error {
	r.pruneCnt++
	r.calls = append(r.calls, "prune")

	return nil
}

// spanTx places one transaction of a pair at a height. Varying amount keeps a lost or
// double counted transaction from cancelling out of the totals.
type spanTx struct {
	height uint64
	pairId uint64
	amount string
}

func newSpanRecordingRepo(startHeight, endHeight uint64, txs []spanTx) *spanRecordingRepo {
	r := &spanRecordingRepo{
		repoMock: &repoMock{},
		heightByTs: map[float64]uint64{
			util.ToEpoch(spanTestWindowEnd.Add(-PairStatsRecentTimeRange)): startHeight,
			util.ToEpoch(spanTestWindowEnd):                                endHeight,
		},
		txsByHeight: map[uint64][]schemas.ParsedTxWithPrice{},
		prices: []schemas.Price{
			{Height: 0, TokenId: 1, Price: "1"},
			{Height: 0, TokenId: 2, Price: "2"},
			// repriced mid-window: later spans have to seed from this height
			{Height: 500, TokenId: 1, Price: "5"},
		},
	}
	for _, tx := range txs {
		r.txsByHeight[tx.height] = append(r.txsByHeight[tx.height], schemas.ParsedTxWithPrice{
			PairId:            tx.pairId,
			Asset0Amount:      tx.amount,
			Asset1Amount:      tx.amount,
			Asset0Liquidity:   "6000000",
			Asset1Liquidity:   "7000000",
			Commission0Amount: "1000000",
			Commission1Amount: "2000000",
			Price0:            "1",
			Price1:            "2",
			Decimals0:         6,
			Decimals1:         6,
			Height:            tx.height,
		})
	}

	return r
}

// The task splits a cold start's window into spans so it never holds all of it in
// memory. Splitting must not change what gets written.
func TestPairStatsRecentUpdateTaskReadsInHeightSpans(t *testing.T) {
	const startHeight, endHeight = 1, 2*PairStatsRecentHeightSpan + 500
	const spanBoundary = PairStatsRecentHeightSpan // last height of the first span

	txs := []spanTx{
		{height: startHeight, pairId: 1, amount: "1000000"},
		// one pair twice at one height: drives generateStats' accumulate branch
		{height: spanBoundary, pairId: 1, amount: "2000000"},
		{height: spanBoundary, pairId: 1, amount: "3000000"},
		{height: spanBoundary, pairId: 2, amount: "4000000"},
		// same pair across the boundary: its closed group must not bleed into the next
		{height: spanBoundary + 1, pairId: 1, amount: "5000000"},
		{height: spanBoundary + 1, pairId: 1, amount: "6000000"},
		{height: endHeight, pairId: 2, amount: "7000000"},
	}

	rp := newSpanRecordingRepo(startHeight, endHeight, txs)
	task := pairStatsRecentUpdateTask{
		taskImpl:   taskImpl{chainId: "test-chain", destDb: rp, logger: logging.Discard},
		priceToken: "test-price-token",
		srcDb:      rp,
		timeRange:  PairStatsRecentTimeRange,
	}

	require.NoError(t, task.Execute(context.Background(), time.Time{}, spanTestWindowEnd))

	require.Equal(t, [][2]uint64{
		{startHeight, PairStatsRecentHeightSpan},
		{PairStatsRecentHeightSpan + 1, 2 * PairStatsRecentHeightSpan},
		{2*PairStatsRecentHeightSpan + 1, endHeight},
	}, rp.readSpans, "the window must be read one span at a time")

	// Compare against a single pass over the whole window. Reading prices through the
	// same stub covers the seeding too: an unseeded span would price differently.
	ctx := context.Background()
	singlePass := newSpanRecordingRepo(startHeight, endHeight, txs)
	allTxs, err := singlePass.GetParsedTxsInHeightRange(ctx, startHeight, endHeight)
	require.NoError(t, err)
	allPrices, err := singlePass.PricesForHeightRange(ctx, startHeight, endHeight, nil, "")
	require.NoError(t, err)
	expected, err := task.generateStats(allTxs, allPrices)
	require.NoError(t, err)

	// one row per pair per height; a leaky boundary would merge or add one
	require.Len(t, expected, 5)
	// generateStats flushes its leftovers in map order, so compare as a set
	require.ElementsMatch(t, expected, rp.written)
	require.Equal(t, uint64(endHeight), task.LastProcessedHeight())

	// The comparison above cancels out anything the task stamps on every row, since both
	// sides come from the same task. Assert those separately.
	for _, s := range rp.written {
		require.Equal(t, "test-chain", s.ChainId)
		require.Equal(t, "test-price-token", s.PriceToken)
	}

	// Two prunes, each with its own job: one before the round can give up on a parent,
	// one with the spans just written. Neither is redundant, so both are pinned here.
	require.Equal(t, 2, rp.pruneCnt)
	require.Equal(t, "prune", rp.calls[0], "the trailing window must be pruned before anything that can bail out")
	require.Equal(t, "prune", rp.calls[len(rp.calls)-1], "the spans just written must be pruned in the same transaction")
	require.Contains(t, rp.calls[1:len(rp.calls)-1], "write", "the second prune must come after the spans, not before")
}

// endHeight drops when source transactions are pruned. Following it down would re-read
// an already written range, which lands as duplicate rows.
func TestPairStatsRecentUpdateTaskNeverMovesHeightBackward(t *testing.T) {
	const processed = 9999

	rp := newSpanRecordingRepo(1, processed-500, nil)
	task := pairStatsRecentUpdateTask{
		taskImpl:  taskImpl{destDb: rp, logger: logging.Discard},
		srcDb:     rp,
		timeRange: PairStatsRecentTimeRange,
	}
	task.lastProcessedHeight.Store(processed)

	require.NoError(t, task.Execute(context.Background(), time.Time{}, spanTestWindowEnd))

	require.Equal(t, uint64(processed), task.LastProcessedHeight())
	require.Empty(t, rp.readSpans, "a window already covered must not be read again")
	// Rows keep falling out of the trailing window while the source stands still, so
	// skipping the round entirely would leave pair_stats_recent holding stale rows.
	require.Equal(t, 1, rp.pruneCnt, "the prune must run even with nothing new to derive")
}

// A round given up on a parent still has to prune: the trailing window keeps sliding
// through a long cold start, and nothing else cleans up.
func TestPairStatsRecentUpdateTaskPrunesWhenParentIsBehind(t *testing.T) {
	rp := newSpanRecordingRepo(1, 500, nil)
	parent := &completedTask{}
	task := pairStatsRecentUpdateTask{
		taskImpl: taskImpl{
			destDb:          rp,
			parentTasks:     []task{parent},
			taskWaitTimeout: time.Millisecond,
			logger:          logging.Discard,
		},
		srcDb:     rp,
		timeRange: PairStatsRecentTimeRange,
	}

	err := task.Execute(context.Background(), time.Time{}, spanTestWindowEnd)

	require.ErrorIs(t, err, ErrParentBehind)
	require.Equal(t, 1, rp.pruneCnt, "the trailing window must be pruned before the round gives up")
	require.Empty(t, rp.readSpans, "nothing may be derived while the parent is behind")
	require.Zero(t, task.LastProcessedHeight())
}

// A failed span must abort the round, not commit a window with a hole and advance past it.
func TestPairStatsRecentUpdateTaskStopsOnSpanFailure(t *testing.T) {
	expectedErr := errors.New("read failed")
	rp := newSpanRecordingRepo(1, 2*PairStatsRecentHeightSpan, nil)
	rp.readErr = expectedErr

	task := pairStatsRecentUpdateTask{
		taskImpl:  taskImpl{destDb: rp, logger: logging.Discard},
		srcDb:     rp,
		timeRange: PairStatsRecentTimeRange,
	}

	err := task.Execute(context.Background(), time.Time{}, spanTestWindowEnd)

	require.ErrorIs(t, err, expectedErr)
	require.Len(t, rp.readSpans, 1, "the round must stop at the failed span")
	require.Zero(t, task.LastProcessedHeight())
}

func TestPairStatsRecentUpdateTaskDoesNotAdvanceHeightWhenTransactionFails(t *testing.T) {
	expectedErr := errors.New("transaction failed")
	rp := repoMock{withinTxErr: expectedErr}
	rp.On("HeightOnTimestamp").Return(uint64(10), nil)
	rp.On("GetParsedTxsInHeightRange").Return([]schemas.ParsedTxWithPrice{}, nil)
	task := pairStatsRecentUpdateTask{
		taskImpl:  taskImpl{destDb: &rp, logger: logging.Discard},
		srcDb:     &rp,
		timeRange: time.Hour,
	}

	err := task.Execute(context.Background(), time.Time{}, time.Now())

	require.ErrorIs(t, err, expectedErr)
	require.Zero(t, task.LastProcessedHeight())
}

// routerSrcRepoStub feeds routerTask.Execute a pair set without a database.
type routerSrcRepoStub struct {
	pairs []router.Pair
	err   error
}

func (r *routerSrcRepoStub) Pairs(context.Context) ([]router.Pair, error) {
	return r.pairs, r.err
}

func (*routerSrcRepoStub) UpdateRoutes(context.Context, map[int]string, map[int]map[int][][]int) error {
	return nil
}

func (*routerSrcRepoStub) Close() error { return nil }

// routerStub counts rebuilds and can fail them on demand.
type routerStub struct {
	router.Router
	err     error
	updates int
}

func (r *routerStub) Update(context.Context) error {
	r.updates++
	return r.err
}

func newRouterTaskForTest(db router.SrcRepo, rt router.Router) *routerTask {
	return &routerTask{
		taskImpl: taskImpl{logger: logging.Discard},
		router:   rt,
		db:       db,
	}
}

// pairCnt records how many pairs the cached graph was built from, so it must only
// advance once the rebuild actually succeeded. Advancing it first would make the
// next run see no growth and skip the retry, leaving the router permanently stale.
func TestRouterTaskRetriesUpdateAfterFailure(t *testing.T) {
	db := &routerSrcRepoStub{pairs: []router.Pair{
		{Contract: "pair0", AssetInfos: []string{"uusd", "uluna"}},
		{Contract: "pair1", AssetInfos: []string{"uluna", "ukrw"}},
	}}
	rt := &routerStub{err: errors.New("route rebuild failed")}
	task := newRouterTaskForTest(db, rt)

	require.Error(t, task.Execute(context.Background(), time.Time{}, time.Time{}))
	require.Equal(t, 1, rt.updates)
	require.Zero(t, task.pairCnt, "a failed rebuild must not be recorded as applied")

	rt.err = nil
	require.NoError(t, task.Execute(context.Background(), time.Time{}, time.Time{}))
	require.Equal(t, 2, rt.updates, "the next run must retry the rebuild")
	require.Equal(t, len(db.pairs), task.pairCnt)
}

// Once the graph matches the pair set, repeated runs must not rebuild it.
func TestRouterTaskSkipsUpdateWhenPairCountUnchanged(t *testing.T) {
	db := &routerSrcRepoStub{pairs: []router.Pair{
		{Contract: "pair0", AssetInfos: []string{"uusd", "uluna"}},
	}}
	rt := &routerStub{}
	task := newRouterTaskForTest(db, rt)

	require.NoError(t, task.Execute(context.Background(), time.Time{}, time.Time{}))
	require.NoError(t, task.Execute(context.Background(), time.Time{}, time.Time{}))

	require.Equal(t, 1, rt.updates)
}

func TestRouterTaskReturnsPairLookupError(t *testing.T) {
	expectedErr := errors.New("pairs query failed")
	rt := &routerStub{}
	task := newRouterTaskForTest(&routerSrcRepoStub{err: expectedErr}, rt)

	require.ErrorIs(t, task.Execute(context.Background(), time.Time{}, time.Time{}), expectedErr)
	require.Zero(t, rt.updates)
}

func TestPairStatsUpdateTaskExecute(t *testing.T) {
	assert := assert.New(t)

	end := time.Unix(1666765800, 0).UTC() // 2022-10-26 06:30:00 UTC
	pairId := uint64(1)
	txCnt := 1
	providerCnt := uint64(4)

	stats := []schemas.PairStats30m{
		{
			YearUtc:            end.Year(),
			MonthUtc:           int(end.Month()),
			DayUtc:             end.Day(),
			HourUtc:            end.Hour(),
			MinuteUtc:          end.Minute(),
			PairId:             pairId,
			Volume0:            "6000000.000000000000000000",
			Volume1:            "7000000.000000000000000000",
			Volume0InPrice:     "9.000000000000000000",
			Volume1InPrice:     "11.000000000000000000",
			LastSwapPrice:      "0.750000000000000000",
			Commission0:        "2000000.000000000000000000",
			Commission1:        "3000000.000000000000000000",
			Commission0InPrice: "3.000000000000000000",
			Commission1InPrice: "5.000000000000000000",
			TxCnt:              txCnt,
			ProviderCnt:        providerCnt,
			Timestamp:          float64(end.Unix()),
		},
	}

	lpMap := map[uint64]schemas.PairStats30m{
		pairId: {
			PairId:            pairId,
			Liquidity0:        "7000000.000000000000000000",
			Liquidity1:        "8000000.000000000000000000",
			Liquidity0InPrice: "14.000000000000000000",
			Liquidity1InPrice: "16.000000000000000000",
		},
	}

	expected := schemas.PairStats30m{
		YearUtc:            end.Year(),
		MonthUtc:           int(end.Month()),
		DayUtc:             end.Day(),
		HourUtc:            end.Hour(),
		MinuteUtc:          end.Minute(),
		PairId:             pairId,
		Volume0:            "6000000.000000000000000000",
		Volume1:            "7000000.000000000000000000",
		Volume0InPrice:     "9.000000000000000000",
		Volume1InPrice:     "11.000000000000000000",
		LastSwapPrice:      "0.750000000000000000",
		Liquidity0:         "7000000.000000000000000000",
		Liquidity1:         "8000000.000000000000000000",
		Liquidity0InPrice:  "14.000000000000000000",
		Liquidity1InPrice:  "16.000000000000000000",
		Commission0:        "2000000.000000000000000000",
		Commission1:        "3000000.000000000000000000",
		Commission0InPrice: "3.000000000000000000",
		Commission1InPrice: "5.000000000000000000",
		TxCnt:              txCnt,
		ProviderCnt:        providerCnt,
		Timestamp:          float64(end.Unix()),
	}

	rp := repoMock{}
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)
	rp.On("LastHeightOfPrice").Return(uint64(0), nil)
	rp.On("PairStats", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(stats, nil)
	rp.On("LiquiditiesOfPairStats", mock.Anything, mock.Anything, mock.Anything).Return(lpMap, nil)

	task := pairStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		srcDb:       &rp,
		prevStatMap: make(map[uint64]schemas.PairStats30m),
	}
	err := task.Execute(context.Background(), time.Time{}, end)

	assert.NoError(err)
	assert.Equal(expected, rp.updatedPairStats[0])
}

// A pair transacting in the window has an lp history too, so a gap is the parent task
// lagging. The liquidity is unknown then, not zero, and reporting the pair as drained
// would dent the liquidity series for that window.
func TestPairStatsUpdateTaskCarriesLiquidityWithoutLpHistory(t *testing.T) {
	end := time.Unix(1666765800, 0).UTC() // 2022-10-26 06:30:00 UTC
	pairId := uint64(1)

	stats := []schemas.PairStats30m{
		{
			PairId:             pairId,
			Volume0:            "6000000.000000000000000000",
			Volume1:            "7000000.000000000000000000",
			Volume0InPrice:     "9.000000000000000000",
			Volume1InPrice:     "11.000000000000000000",
			LastSwapPrice:      "0.750000000000000000",
			Commission0:        "2000000.000000000000000000",
			Commission1:        "3000000.000000000000000000",
			Commission0InPrice: "3.000000000000000000",
			Commission1InPrice: "5.000000000000000000",
			Timestamp:          float64(end.Unix()),
		},
	}

	carried := schemas.PairStats30m{
		PairId:            pairId,
		Liquidity0:        "7000000.000000000000000000",
		Liquidity1:        "8000000.000000000000000000",
		Liquidity0InPrice: "14.000000000000000000",
		Liquidity1InPrice: "16.000000000000000000",
	}

	for _, tc := range []struct {
		name                     string
		prevStat                 map[uint64]schemas.PairStats30m
		written                  map[uint64]schemas.PairStats30m
		liquidity, liquidityInPr string
	}{
		{
			name:      "carries the previous window over",
			prevStat:  map[uint64]schemas.PairStats30m{pairId: carried},
			liquidity: "7000000.000000000000000000", liquidityInPr: "14.000000000000000000",
		},
		{
			// prevStatMap is empty until this process aggregates a window, so a restart
			// has to fall back to the newest row in the database
			name:      "reads the previous window back after a restart",
			prevStat:  map[uint64]schemas.PairStats30m{},
			written:   map[uint64]schemas.PairStats30m{pairId: carried},
			liquidity: "7000000.000000000000000000", liquidityInPr: "14.000000000000000000",
		},
		{
			// nothing to carry over, and the numeric columns reject an empty string
			name:      "zeroes a pair seen for the first time",
			prevStat:  map[uint64]schemas.PairStats30m{},
			liquidity: "0", liquidityInPr: "0",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			rp := repoMock{latestPairStats: tc.written}
			rp.On("HeightOnTimestamp").Return(uint64(0), nil)
			rp.On("LastHeightOfPrice").Return(uint64(0), nil)
			rp.On("PairStats", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(stats, nil)
			rp.On("LiquiditiesOfPairStats", mock.Anything, mock.Anything, mock.Anything).Return(
				map[uint64]schemas.PairStats30m{}, nil)

			task := pairStatsUpdateTask{
				taskImpl: taskImpl{
					chainId: "",
					destDb:  &rp,
					logger:  logging.Discard,
				},
				srcDb:       &rp,
				prevStatMap: tc.prevStat,
			}
			err := task.Execute(context.Background(), time.Time{}, end)

			assert.NoError(err)
			actual := rp.updatedPairStats[0]
			assert.Equal(tc.liquidity, actual.Liquidity0)
			assert.Equal(tc.liquidityInPr, actual.Liquidity0InPrice)
		})
	}
}

// Reading the previous window back can fail like any other query, and a window written
// with the liquidity of a failed read would be wrong rather than merely late.
func TestPairStatsUpdateTaskFailsOnUnreadablePreviousLiquidity(t *testing.T) {
	end := time.Unix(1666765800, 0).UTC() // 2022-10-26 06:30:00 UTC
	expectedErr := errors.New("read failed")

	rp := repoMock{latestPairStatErr: expectedErr}
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)
	rp.On("LastHeightOfPrice").Return(uint64(0), nil)
	rp.On("PairStats", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		[]schemas.PairStats30m{{PairId: 1, Timestamp: float64(end.Unix())}}, nil)
	rp.On("LiquiditiesOfPairStats", mock.Anything, mock.Anything, mock.Anything).Return(
		map[uint64]schemas.PairStats30m{}, nil)

	task := pairStatsUpdateTask{
		taskImpl: taskImpl{
			destDb: &rp,
			logger: logging.Discard,
		},
		srcDb:       &rp,
		prevStatMap: make(map[uint64]schemas.PairStats30m),
	}
	err := task.Execute(context.Background(), time.Time{}, end)

	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, rp.updatedPairStats, "a window must not be written from liquidity that could not be read")
}

func TestExecuteAccountStatsUpdateTask(t *testing.T) {
	assert := assert.New(t)

	end := time.Unix(1666765800, 0).UTC() // 2022-10-26 06:30:00 UTC
	accountAddress := "terra0wal1let2"
	accountId := uint64(7)
	pairId := uint64(1)
	txCnt := uint64(5)
	priceToken := "uusd"

	expected := []schemas.AccountStats30m{
		{
			YearUtc:             end.Year(),
			MonthUtc:            int(end.Month()),
			DayUtc:              end.Day(),
			HourUtc:             end.Hour(),
			MinuteUtc:           end.Minute(),
			Timestamp:           util.ToEpoch(end),
			AccountId:           accountId,
			Address:             accountAddress,
			PairId:              pairId,
			TxCnt:               txCnt,
			SwapTxCnt:           3,
			ProvideTxCnt:        1,
			SwapVolumeInPrice:   "10",
			ProvideValueInPrice: "20",
			PriceToken:          priceToken,
			NetLpAmount:         "30",
		},
	}

	stats := []schemas.AccountStats30m{{
		Address:             accountAddress,
		PairId:              pairId,
		TxCnt:               txCnt,
		SwapTxCnt:           3,
		ProvideTxCnt:        1,
		SwapVolumeInPrice:   "10",
		ProvideValueInPrice: "20",
		PriceToken:          priceToken,
		NetLpAmount:         "30",
	}}

	rp := repoMock{}
	rp.On("AccountStats", mock.Anything, mock.Anything, priceToken).Return(stats, nil)
	rp.On("AccountIds").Return(map[string]uint64{accountAddress: accountId}, nil)
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		priceToken: priceToken,
		srcDb:      &rp,
	}
	err := task.Execute(context.Background(), time.Time{}, end)

	assert.NoError(err)
	assert.Equal(expected, rp.updatedAccountStats)
}

func TestExecuteAccountStatsUpdateTaskDeduplicatesAccountCreation(t *testing.T) {
	assert := assert.New(t)

	priceToken := "uusd"
	accountAddress := "terra0duplicated"
	stats := []schemas.AccountStats30m{
		{Address: accountAddress, PairId: 1, TxCnt: 1},
		{Address: accountAddress, PairId: 2, TxCnt: 1},
	}

	rp := repoMock{}
	rp.On("AccountStats", mock.Anything, mock.Anything, priceToken).Return(stats, nil)
	rp.On("AccountIds").Return(map[string]uint64{accountAddress: 11}, nil)
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "cube_47-5",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		priceToken: priceToken,
		srcDb:      &rp,
	}
	err := task.Execute(context.Background(), time.Time{}, time.Unix(1666765800, 0).UTC())

	assert.NoError(err)
	assert.Equal([]string{accountAddress}, rp.updatedAccounts)
	assert.Len(rp.updatedAccountStats, 2)
	assert.Equal(uint64(11), rp.updatedAccountStats[0].AccountId)
	assert.Equal(uint64(11), rp.updatedAccountStats[1].AccountId)
}

func TestExecuteAccountStatsUpdateTaskReturnsErrorWhenAccountIdMissing(t *testing.T) {
	assert := assert.New(t)

	priceToken := "uusd"
	accountAddress := "terra0missing"
	stats := []schemas.AccountStats30m{{Address: accountAddress, PairId: 1, TxCnt: 1}}

	rp := repoMock{}
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)
	rp.On("AccountStats", mock.Anything, mock.Anything, priceToken).Return(stats, nil)
	rp.On("AccountIds").Return(map[string]uint64{}, nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "cube_47-5",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		priceToken: priceToken,
		srcDb:      &rp,
	}
	err := task.Execute(context.Background(), time.Time{}, time.Unix(1666765800, 0).UTC())

	assert.ErrorContains(err, "account id not found")
	assert.Empty(rp.updatedAccountStats)
}

func TestExecuteAccountStatsUpdateTaskPropagatesCreateAccountsError(t *testing.T) {
	assert := assert.New(t)

	priceToken := "uusd"
	createErr := errors.New("create accounts failed")
	stats := []schemas.AccountStats30m{{Address: "terra0fail", PairId: 1, TxCnt: 1}}

	rp := repoMock{createAccountsErr: createErr}
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)
	rp.On("AccountStats", mock.Anything, mock.Anything, priceToken).Return(stats, nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "cube_47-5",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		priceToken: priceToken,
		srcDb:      &rp,
	}
	err := task.Execute(context.Background(), time.Time{}, time.Unix(1666765800, 0).UTC())

	assert.ErrorIs(err, createErr)
	assert.Empty(rp.updatedAccountStats)
	rp.AssertNotCalled(t, "AccountIds", mock.Anything)
}

func TestExecuteAccountStatsUpdateTaskPassesPriceToken(t *testing.T) {
	assert := assert.New(t)

	priceToken := "uusd"
	end := time.Unix(1666765800, 0).UTC()

	rp := repoMock{}
	rp.On("AccountStats", util.ToEpoch(time.Time{}), util.ToEpoch(end), priceToken).Return([]schemas.AccountStats30m{}, nil)
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "cube_47-5",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		priceToken: priceToken,
		srcDb:      &rp,
	}
	err := task.Execute(context.Background(), time.Time{}, end)

	assert.NoError(err)
	rp.AssertCalled(t, "AccountStats", util.ToEpoch(time.Time{}), util.ToEpoch(end), priceToken)
}

func TestExecuteAccountStatsUpdateTaskNoStats(t *testing.T) {
	assert := assert.New(t)

	rp := repoMock{}
	rp.On("AccountStats", mock.Anything, mock.Anything, "").Return([]schemas.AccountStats30m{}, nil)
	rp.On("HeightOnTimestamp").Return(uint64(10), nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "cube_47-5",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		srcDb: &rp,
	}
	err := task.Execute(context.Background(), time.Time{}, time.Unix(1666765800, 0).UTC())

	assert.NoError(err)
	assert.Empty(rp.updatedAccounts)
	assert.Empty(rp.updatedAccountStats)
	assert.Equal(uint64(10), task.LastProcessedHeight())
}

func TestExecuteAccountStatsUpdateTaskWaitsForParentPriceTask(t *testing.T) {
	assert := assert.New(t)

	end := time.Unix(1666765800, 0).UTC()
	endHeight := uint64(10)
	accountStatsCalled := make(chan struct{}, 1)

	rp := repoMock{}
	rp.On("HeightOnTimestamp").Return(endHeight, nil)
	rp.On("AccountStats", mock.Anything, mock.Anything, "").Run(func(_ mock.Arguments) {
		accountStatsCalled <- struct{}{}
	}).Return([]schemas.AccountStats30m{}, nil)

	parent := &completedTask{}
	parent.setLastProcessedHeight(endHeight - 1)
	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId:         "cube_47-5",
			destDb:          &rp,
			parentTasks:     []task{parent},
			taskWaitTimeout: time.Minute,
			logger:          logging.Discard,
		},
		srcDb: &rp,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- task.Execute(context.Background(), time.Time{}, end)
	}()

	select {
	case <-accountStatsCalled:
		assert.Fail("AccountStats was called before the parent price task reached the target height")
	case <-time.After(WaitPeriod / 2):
	}

	parent.setLastProcessedHeight(endHeight)

	var err error
	select {
	case err = <-errCh:
	case <-time.After(WaitPeriod * 2):
		assert.Fail("account stats update task did not complete after the parent price task reached the target height")
		return
	}

	assert.NoError(err)
	assert.Equal(endHeight, task.LastProcessedHeight())
	rp.AssertCalled(t, "AccountStats", util.ToEpoch(time.Time{}), util.ToEpoch(end), "")
}

func TestExecuteAccountStatsUpdateTaskParentWaitTimeout(t *testing.T) {
	assert := assert.New(t)

	end := time.Unix(1666765800, 0).UTC()
	endHeight := uint64(10)

	rp := repoMock{}
	rp.On("HeightOnTimestamp").Return(endHeight, nil)

	parent := &completedTask{}
	parent.setLastProcessedHeight(endHeight - 1)
	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId:         "cube_47-5",
			destDb:          &rp,
			parentTasks:     []task{parent},
			taskWaitTimeout: time.Millisecond,
			logger:          logging.Discard,
		},
		srcDb: &rp,
	}

	err := task.Execute(context.Background(), time.Time{}, end)

	assert.ErrorIs(err, ErrParentBehind)
	// the scheduler's retry is safe only because the round bailed out before writing
	assert.Empty(rp.updatedAccountStats)
	assert.Equal(uint64(0), task.LastProcessedHeight())
	rp.AssertNotCalled(t, "AccountStats", mock.Anything, mock.Anything, mock.Anything)
}
