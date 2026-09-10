package collector

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/nodeerr"
	"github.com/stretchr/testify/require"
)

type sourceRepoMock struct {
	syncedHeight uint64
	syncedErr    error
	saved        []savedHeight
	saveErr      error
}

type savedHeight struct {
	chainID          string
	height           uint64
	txs              parser.RawTxs
	poolInfos        []dex.PoolInfo
	savePoolSnapshot bool
}

func (m *sourceRepoMock) GetSyncedHeight(string) (uint64, error) {
	if len(m.saved) > 0 && (errors.Is(m.syncedErr, repo.ErrNotFound) || errors.Is(m.syncedErr, repo.ErrUnavailable)) {
		return m.syncedHeight, nil
	}
	return m.syncedHeight, m.syncedErr
}

func (m *sourceRepoMock) GetBlockTxs(string, uint64) (parser.RawTxs, time.Time, error) {
	return nil, time.Time{}, nil
}

func (m *sourceRepoMock) GetPoolInfos(string, uint64) ([]dex.PoolInfo, error) {
	return nil, nil
}

func (m *sourceRepoMock) SaveHeight(chainID string, height uint64, _ time.Time, txs parser.RawTxs, poolInfos []dex.PoolInfo, savePoolSnapshot bool) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, savedHeight{
		chainID:          chainID,
		height:           height,
		txs:              txs,
		poolInfos:        poolInfos,
		savePoolSnapshot: savePoolSnapshot,
	})
	m.syncedHeight = height
	return nil
}

type sourceStoreMock struct {
	syncedHeight   uint64
	syncedErr      error
	syncedFailures int
	txsFailures    int
	poolFailures   int
	transientErr   error
	txs            map[uint64]parser.RawTxs
	txsErr         error
	poolInfos      map[uint64][]dex.PoolInfo
	poolInfoErr    error
}

func (m *sourceStoreMock) GetSourceSyncedHeight() (uint64, error) {
	if m.syncedFailures > 0 {
		m.syncedFailures--
		return 0, m.transientErr
	}
	return m.syncedHeight, m.syncedErr
}

func (m *sourceStoreMock) GetSourceTxs(height uint64) (parser.RawTxs, error) {
	if m.txsFailures > 0 {
		m.txsFailures--
		return nil, m.transientErr
	}
	if m.txsErr != nil {
		return nil, m.txsErr
	}
	return m.txs[height], nil
}

func (m *sourceStoreMock) GetPoolInfos(height uint64) ([]dex.PoolInfo, error) {
	if m.poolFailures > 0 {
		m.poolFailures--
		return nil, m.transientErr
	}
	if m.poolInfoErr != nil {
		return nil, m.poolInfoErr
	}
	return m.poolInfos[height], nil
}

func TestDoCollectSourceCollectsFromStartHeightToUntilHeight(t *testing.T) {
	repo := &sourceRepoMock{syncedErr: repo.ErrNotFound}
	source := &sourceStoreMock{
		syncedHeight: 10,
		txs: map[uint64]parser.RawTxs{
			5: {{Hash: "tx5", Timestamp: time.Date(2026, 5, 19, 0, 0, 5, 0, time.UTC)}},
			6: {{Hash: "tx6", Timestamp: time.Date(2026, 5, 19, 0, 0, 6, 0, time.UTC)}},
		},
		poolInfos: map[uint64][]dex.PoolInfo{
			6: {{ContractAddr: "pair6"}},
		},
	}

	err := DoCollect(
		repo,
		source,
		configs.CollectorConfig{ChainId: "chain", StartHeight: 5, UntilHeight: 6, PoolSnapshotInterval: 2},
		logging.Discard,
	)

	require.NoError(t, err)
	require.Len(t, repo.saved, 2)
	require.Equal(t, uint64(5), repo.saved[0].height)
	require.False(t, repo.saved[0].savePoolSnapshot)
	require.Equal(t, parser.RawTxs{{Hash: "tx5", Timestamp: time.Date(2026, 5, 19, 0, 0, 5, 0, time.UTC)}}, repo.saved[0].txs)
	require.Equal(t, uint64(6), repo.saved[1].height)
	require.True(t, repo.saved[1].savePoolSnapshot)
	require.Equal(t, []dex.PoolInfo{{ContractAddr: "pair6"}}, repo.saved[1].poolInfos)
}

func TestDoCollectSourceUsesConfiguredChainAndSnapshotInterval(t *testing.T) {
	repo := &sourceRepoMock{syncedErr: repo.ErrUnavailable}
	source := &sourceStoreMock{
		syncedHeight: 3,
		txs: map[uint64]parser.RawTxs{
			1: {{Hash: "tx1"}},
			2: {{Hash: "tx2"}},
			3: {{Hash: "tx3"}},
		},
		poolInfos: map[uint64][]dex.PoolInfo{
			2: {{ContractAddr: "pair2"}},
		},
	}

	err := DoCollect(
		repo,
		source,
		configs.CollectorConfig{ChainId: "chain", StartHeight: 1, UntilHeight: 3, PoolSnapshotInterval: 2},
		logging.Discard,
	)

	require.NoError(t, err)
	require.Len(t, repo.saved, 3)
	require.Equal(t, "chain", repo.saved[0].chainID)
	require.False(t, repo.saved[0].savePoolSnapshot)
	require.True(t, repo.saved[1].savePoolSnapshot)
	require.False(t, repo.saved[2].savePoolSnapshot)
}

func TestDoCollectRetriesUnreadableTxsWithoutAdvancing(t *testing.T) {
	repo := &sourceRepoMock{syncedErr: repo.ErrNotFound}
	source := &sourceStoreMock{
		syncedHeight: 1,
		txsFailures:  3,
		transientErr: errors.New("tx source failed"),
		txs:          map[uint64]parser.RawTxs{1: {{Hash: "tx1"}}},
	}

	err := DoCollect(
		repo,
		source,
		configs.CollectorConfig{ChainId: "chain", StartHeight: 1, UntilHeight: 1},
		logging.Discard,
	)

	require.NoError(t, err)
	require.Len(t, repo.saved, 1)
	require.Equal(t, uint64(1), repo.saved[0].height)
}

func TestDoCollectRetriesUnreadablePoolInfo(t *testing.T) {
	source := &sourceStoreMock{
		syncedHeight: 1,
		poolFailures: 3,
		transientErr: errors.New("pool source failed"),
		txs:          map[uint64]parser.RawTxs{1: {{Hash: "tx1"}}},
		poolInfos:    map[uint64][]dex.PoolInfo{1: {{ContractAddr: "pair1"}}},
	}
	repo := &sourceRepoMock{syncedErr: repo.ErrNotFound}

	err := DoCollect(
		repo,
		source,
		configs.CollectorConfig{ChainId: "chain", StartHeight: 1, UntilHeight: 1, PoolSnapshotInterval: 1},
		logging.Discard,
	)

	require.NoError(t, err)
	require.Len(t, repo.saved, 1)
	require.True(t, repo.saved[0].savePoolSnapshot)
}

func TestDoCollectReturnsRepositoryHeightError(t *testing.T) {
	expected := errors.New("repo height failed")
	repo := &sourceRepoMock{syncedErr: expected}

	err := DoCollect(
		repo,
		&sourceStoreMock{syncedHeight: 1},
		configs.CollectorConfig{ChainId: "chain", StartHeight: 1, UntilHeight: 1},
		logging.Discard,
	)

	require.ErrorIs(t, err, expected)
	require.Empty(t, repo.saved)
}

func TestDoCollectRecoversFromSourceHeightError(t *testing.T) {
	repo := &sourceRepoMock{syncedErr: repo.ErrNotFound}
	source := &sourceStoreMock{
		syncedHeight:   1,
		syncedFailures: 3,
		transientErr:   errors.New("source height failed"),
		txs:            map[uint64]parser.RawTxs{1: {{Hash: "tx1"}}},
	}

	err := DoCollect(
		repo,
		source,
		configs.CollectorConfig{ChainId: "chain", StartHeight: 1, UntilHeight: 1},
		logging.Discard,
	)

	require.NoError(t, err)
	require.Len(t, repo.saved, 1)
}

func TestDoCollectReturnsSaveHeightError(t *testing.T) {
	expected := errors.New("save height failed")
	repo := &sourceRepoMock{syncedErr: repo.ErrNotFound, saveErr: expected}
	source := &sourceStoreMock{
		syncedHeight: 1,
		txs:          map[uint64]parser.RawTxs{1: {{Hash: "tx1"}}},
	}

	err := DoCollect(
		repo,
		source,
		configs.CollectorConfig{ChainId: "chain", StartHeight: 1, UntilHeight: 1},
		logging.Discard,
	)

	require.ErrorIs(t, err, expected)
	require.ErrorIs(t, err, errLocalStore)
	require.Empty(t, repo.saved)
}

type heightCollectorMock struct {
	localHeight uint64
	localErr    error
	// localErrAfter breaks out of the endless poll loop without an until height, which
	// would otherwise clamp the target and hide what the tip lag did
	localErrAfter int
	localCalls    int
	sourceHeight  uint64
	sourceHeights []uint64
	sourceCalls   int
	sourceErr     error
	collectErr    error
	collected     []uint64
	// failures counts how many times the matching step fails before succeeding
	sourceFailures  int
	collectFailures int
	// failHeight limits collect failures to one height; zero fails any height
	failHeight   uint64
	transientErr error
}

func (m *heightCollectorMock) LocalHeight() (uint64, error) {
	m.localCalls++
	if m.localErr != nil && m.localCalls >= m.localErrAfter {
		return 0, m.localErr
	}
	return m.localHeight, nil
}

func (m *heightCollectorMock) SourceHeight() (uint64, error) {
	if m.sourceFailures > 0 {
		m.sourceFailures--
		return 0, m.transientErr
	}
	if len(m.sourceHeights) > 0 {
		height := m.sourceHeights[m.sourceCalls]
		if m.sourceCalls < len(m.sourceHeights)-1 {
			m.sourceCalls++
		}
		return height, m.sourceErr
	}
	return m.sourceHeight, m.sourceErr
}

func (m *heightCollectorMock) CollectHeight(height uint64) error {
	if m.collectFailures > 0 && (m.failHeight == 0 || m.failHeight == height) {
		m.collectFailures--
		return m.transientErr
	}
	if m.collectErr != nil {
		return m.collectErr
	}
	m.collected = append(m.collected, height)
	m.localHeight = height
	return nil
}

func TestCollectHeightsCollectsBoundedRange(t *testing.T) {
	collector := &heightCollectorMock{
		localHeight:  3,
		sourceHeight: 10,
	}

	err := collectHeights(collector, heightCollectorConfig{
		StartHeight: 5,
		UntilHeight: 7,
	}, logging.Discard)

	require.NoError(t, err)
	require.Equal(t, []uint64{5, 6, 7}, collector.collected)
}

func TestCollectHeightsStopsWhenUntilHeightAlreadyReached(t *testing.T) {
	collector := &heightCollectorMock{
		localHeight:  7,
		sourceHeight: 10,
	}

	err := collectHeights(collector, heightCollectorConfig{
		StartHeight: 5,
		UntilHeight: 7,
	}, logging.Discard)

	require.NoError(t, err)
	require.Empty(t, collector.collected)
}

func TestCollectHeightsPollsUntilSourceAdvances(t *testing.T) {
	collector := &heightCollectorMock{
		localHeight:   1,
		sourceHeights: []uint64{1, 2},
	}

	err := collectHeights(collector, heightCollectorConfig{
		StartHeight: 1,
		UntilHeight: 2,
	}, logging.Discard)

	require.NoError(t, err)
	require.Equal(t, []uint64{2}, collector.collected)
}

func TestCollectHeightsPollsUntilConfiguredStartHeightIsAvailable(t *testing.T) {
	collector := &heightCollectorMock{
		sourceHeights: []uint64{4, 5},
	}

	err := collectHeights(collector, heightCollectorConfig{
		StartHeight: 5,
		UntilHeight: 5,
	}, logging.Discard)

	require.NoError(t, err)
	require.Equal(t, []uint64{5}, collector.collected)
}

func TestCollectHeightsRejectsUntilHeightBeforeStartHeight(t *testing.T) {
	collector := &heightCollectorMock{}

	err := collectHeights(collector, heightCollectorConfig{
		StartHeight: 5,
		UntilHeight: 4,
	}, logging.Discard)

	require.EqualError(t, err, "invalid height range: start_height=5 until_height=4")
	require.Empty(t, collector.collected)
}

// A finished backfill must not depend on the source being reachable to exit.
func TestCollectHeightsStopsAtUntilHeightWhenSourceIsDown(t *testing.T) {
	collector := &heightCollectorMock{
		localHeight: 5,
		sourceErr:   errors.New("source height failed"),
	}

	err := collectHeights(collector, heightCollectorConfig{
		UntilHeight: 5,
	}, logging.Discard)

	require.NoError(t, err)
	require.Empty(t, collector.collected)
}

func TestCollectHeightsRecoversFromSourceError(t *testing.T) {
	collector := &heightCollectorMock{
		localHeight:    0,
		sourceHeight:   1,
		sourceFailures: 3,
		transientErr:   errors.New("source height failed"),
	}

	err := collectHeights(collector, heightCollectorConfig{
		UntilHeight: 1,
	}, logging.Discard)

	require.NoError(t, err)
	require.Equal(t, []uint64{1}, collector.collected)
}

func TestCollectHeightsReturnsLocalError(t *testing.T) {
	expected := errors.New("local height failed")
	collector := &heightCollectorMock{localErr: expected}

	err := collectHeights(collector, heightCollectorConfig{
		UntilHeight: 1,
	}, logging.Discard)

	require.ErrorIs(t, err, expected)
	require.Empty(t, collector.collected)
}

// A failing height must stay unconsumed so it is retried rather than skipped.
func TestCollectHeightsRetriesFailedHeightWithoutSkipping(t *testing.T) {
	collector := &heightCollectorMock{
		localHeight:     0,
		sourceHeight:    2,
		collectFailures: 3,
		transientErr:    errors.New("collect height failed"),
	}

	err := collectHeights(collector, heightCollectorConfig{
		UntilHeight: 2,
	}, logging.Discard)

	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2}, collector.collected)
}

// A height the source cannot serve never clears, so retrying would stall forever.
func TestCollectHeightsReturnsSourceUnavailableError(t *testing.T) {
	expected := fmt.Errorf("%w: %w", errSourceUnavailable, nodeerr.ErrHeightUnavailable)
	collector := &heightCollectorMock{
		localHeight:  0,
		sourceHeight: 2,
		collectErr:   expected,
	}

	err := collectHeights(collector, heightCollectorConfig{
		UntilHeight: 2,
	}, logging.Discard)

	require.ErrorIs(t, err, errSourceUnavailable)
	require.Empty(t, collector.collected)
}

func TestDoCollectReturnsSourceUnavailableError(t *testing.T) {
	unavailable := func(op string) error {
		return fmt.Errorf("%s: %w", op, nodeerr.ErrHeightUnavailable)
	}

	for _, tc := range []struct {
		name   string
		source *sourceStoreMock
	}{
		{"txs", &sourceStoreMock{
			syncedHeight: 1,
			txsErr:       unavailable("baseRawDataStoreImpl.GetSourceTxs"),
		}},
		{"pool infos", &sourceStoreMock{
			syncedHeight: 1,
			txs:          map[uint64]parser.RawTxs{1: {{Hash: "tx1"}}},
			poolInfoErr:  unavailable("baseRawDataStoreImpl.GetPoolInfos"),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &sourceRepoMock{syncedErr: repo.ErrNotFound}

			err := DoCollect(
				repo,
				tc.source,
				configs.CollectorConfig{ChainId: "chain", StartHeight: 1, UntilHeight: 1, PoolSnapshotInterval: 1},
				logging.Discard,
			)

			require.ErrorIs(t, err, errSourceUnavailable)
			require.ErrorIs(t, err, nodeerr.ErrHeightUnavailable)
			require.Empty(t, repo.saved)
		})
	}
}

func TestCollectHeightsResumesAtFailedMidRangeHeight(t *testing.T) {
	collector := &heightCollectorMock{
		localHeight:     0,
		sourceHeight:    3,
		failHeight:      2,
		collectFailures: 2,
		transientErr:    errors.New("collect height failed"),
	}

	err := collectHeights(collector, heightCollectorConfig{
		UntilHeight: 3,
	}, logging.Discard)

	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2, 3}, collector.collected)
}

func TestBoundedTargetHeight(t *testing.T) {
	require.Equal(t, uint64(7), boundedTargetHeight(10, 7, 0))
	require.Equal(t, uint64(10), boundedTargetHeight(10, 0, 0))
	require.Equal(t, uint64(5), boundedTargetHeight(5, 7, 0))
}

func TestBoundedTargetHeightWithTipLag(t *testing.T) {
	require.Equal(t, uint64(98), boundedTargetHeight(100, 0, 2))
	require.Equal(t, uint64(95), boundedTargetHeight(100, 95, 2))
	require.Equal(t, uint64(98), boundedTargetHeight(100, 98, 2))
	require.Equal(t, uint64(1), boundedTargetHeight(3, 0, 2))

	// the lag must not underflow before the chain has produced enough blocks
	require.Equal(t, uint64(0), boundedTargetHeight(2, 0, 2))
	require.Equal(t, uint64(0), boundedTargetHeight(1, 0, 2))
}

func TestCollectHeightsStopsShortOfTipLag(t *testing.T) {
	stop := errors.New("stop polling")
	collector := &heightCollectorMock{
		localHeight:   90,
		sourceHeight:  100,
		localErr:      stop,
		localErrAfter: 2,
	}

	err := collectHeights(collector, heightCollectorConfig{TipLagBlocks: 2}, logging.Discard)

	require.ErrorIs(t, err, stop)
	require.Equal(t, []uint64{91, 92, 93, 94, 95, 96, 97, 98}, collector.collected)
}

func TestCollectHeightsCollectsHeightOnceTipClearsTheLag(t *testing.T) {
	stop := errors.New("stop polling")
	collector := &heightCollectorMock{
		localHeight:   98,
		sourceHeights: []uint64{100, 101},
		localErr:      stop,
		localErrAfter: 3,
	}

	err := collectHeights(collector, heightCollectorConfig{TipLagBlocks: 2}, logging.Discard)

	// height 99 stays untouched while the tip is 100 and is collected once it reaches 101
	require.ErrorIs(t, err, stop)
	require.Equal(t, []uint64{99}, collector.collected)
}

func TestCollectHeightsNeverCollectsInsideTipLag(t *testing.T) {
	stop := errors.New("stop polling")
	collector := &heightCollectorMock{
		localHeight:   98,
		sourceHeight:  100,
		localErr:      stop,
		localErrAfter: 3,
	}

	err := collectHeights(collector, heightCollectorConfig{TipLagBlocks: 2}, logging.Discard)

	require.ErrorIs(t, err, stop)
	require.Empty(t, collector.collected)
}

func TestReachedUntilHeight(t *testing.T) {
	require.True(t, reachedUntilHeight(7, 7))
	require.False(t, reachedUntilHeight(6, 7))
	require.False(t, reachedUntilHeight(7, 0))
}
