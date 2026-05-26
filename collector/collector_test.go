package collector

import (
	"errors"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
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
	syncedHeight uint64
	syncedErr    error
	txs          map[uint64]parser.RawTxs
	txsErr       error
	poolInfos    map[uint64][]dex.PoolInfo
	poolInfoErr  error
}

func (m *sourceStoreMock) GetSourceSyncedHeight() (uint64, error) {
	return m.syncedHeight, m.syncedErr
}

func (m *sourceStoreMock) GetSourceTxs(height uint64) (parser.RawTxs, error) {
	if m.txsErr != nil {
		return nil, m.txsErr
	}
	return m.txs[height], nil
}

func (m *sourceStoreMock) GetPoolInfos(height uint64) ([]dex.PoolInfo, error) {
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

func TestDoCollectSourceReturnsSourceTxError(t *testing.T) {
	expected := errors.New("tx source failed")
	repo := &sourceRepoMock{syncedErr: repo.ErrNotFound}
	source := &sourceStoreMock{syncedHeight: 1, txsErr: expected}

	err := DoCollect(
		repo,
		source,
		configs.CollectorConfig{ChainId: "chain", StartHeight: 1, UntilHeight: 1},
		logging.Discard,
	)

	require.ErrorIs(t, err, expected)
	require.Empty(t, repo.saved)
}

func TestDoCollectSourceReturnsPoolInfoError(t *testing.T) {
	expected := errors.New("pool source failed")
	repo := &sourceRepoMock{syncedErr: repo.ErrNotFound}
	source := &sourceStoreMock{
		syncedHeight: 1,
		txs:          map[uint64]parser.RawTxs{1: {{Hash: "tx1"}}},
		poolInfoErr:  expected,
	}

	err := DoCollect(
		repo,
		source,
		configs.CollectorConfig{ChainId: "chain", StartHeight: 1, UntilHeight: 1, PoolSnapshotInterval: 1},
		logging.Discard,
	)

	require.ErrorIs(t, err, expected)
	require.Empty(t, repo.saved)
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

func TestDoCollectReturnsSourceHeightError(t *testing.T) {
	expected := errors.New("source height failed")
	repo := &sourceRepoMock{syncedErr: repo.ErrNotFound}

	err := DoCollect(
		repo,
		&sourceStoreMock{syncedErr: expected},
		configs.CollectorConfig{ChainId: "chain", StartHeight: 1, UntilHeight: 1},
		logging.Discard,
	)

	require.ErrorIs(t, err, expected)
	require.Empty(t, repo.saved)
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
	require.Empty(t, repo.saved)
}

type heightCollectorMock struct {
	localHeight   uint64
	localErr      error
	sourceHeight  uint64
	sourceHeights []uint64
	sourceCalls   int
	sourceErr     error
	collectErr    error
	collected     []uint64
}

func (m *heightCollectorMock) LocalHeight() (uint64, error) {
	return m.localHeight, m.localErr
}

func (m *heightCollectorMock) SourceHeight() (uint64, error) {
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

func TestCollectHeightsReturnsSourceError(t *testing.T) {
	expected := errors.New("source height failed")
	collector := &heightCollectorMock{
		localHeight: 0,
		sourceErr:   expected,
	}

	err := collectHeights(collector, heightCollectorConfig{
		UntilHeight: 1,
	}, logging.Discard)

	require.ErrorIs(t, err, expected)
}

func TestCollectHeightsReturnsLocalError(t *testing.T) {
	expected := errors.New("local height failed")
	collector := &heightCollectorMock{localErr: expected}

	err := collectHeights(collector, heightCollectorConfig{
		UntilHeight: 1,
	}, logging.Discard)

	require.ErrorIs(t, err, expected)
}

func TestCollectHeightsReturnsCollectError(t *testing.T) {
	expected := errors.New("collect height failed")
	collector := &heightCollectorMock{
		localHeight:  0,
		sourceHeight: 1,
		collectErr:   expected,
	}

	err := collectHeights(collector, heightCollectorConfig{
		UntilHeight: 1,
	}, logging.Discard)

	require.ErrorIs(t, err, expected)
	require.Empty(t, collector.collected)
}

func TestBoundedTargetHeight(t *testing.T) {
	require.Equal(t, uint64(7), boundedTargetHeight(10, 7))
	require.Equal(t, uint64(10), boundedTargetHeight(10, 0))
	require.Equal(t, uint64(5), boundedTargetHeight(5, 7))
}

func TestReachedUntilHeight(t *testing.T) {
	require.True(t, reachedUntilHeight(7, 7))
	require.False(t, reachedUntilHeight(6, 7))
	require.False(t, reachedUntilHeight(7, 0))
}
