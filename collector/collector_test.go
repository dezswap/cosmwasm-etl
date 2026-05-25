package collector

import (
	"errors"
	"testing"
	"time"

	collectorrepo "github.com/dezswap/cosmwasm-etl/collector/repo"
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
	if len(m.saved) > 0 && (errors.Is(m.syncedErr, collectorrepo.ErrNotFound) || errors.Is(m.syncedErr, collectorrepo.ErrUnavailable)) {
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
	repo := &sourceRepoMock{syncedErr: collectorrepo.ErrNotFound}
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

func TestDoCollectSourceUsesChainAndSnapshotIntervalFallbacks(t *testing.T) {
	repo := &sourceRepoMock{syncedErr: collectorrepo.ErrUnavailable}
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
		configs.CollectorConfig{ChainId: "chain", UntilHeight: 3, PoolSnapshotInterval: 2},
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
	repo := &sourceRepoMock{syncedErr: collectorrepo.ErrNotFound}
	source := &sourceStoreMock{syncedHeight: 1, txsErr: expected}

	err := DoCollect(
		repo,
		source,
		configs.CollectorConfig{ChainId: "chain", UntilHeight: 1},
		logging.Discard,
	)

	require.ErrorIs(t, err, expected)
	require.Empty(t, repo.saved)
}

func TestDoCollectSourceReturnsPoolInfoError(t *testing.T) {
	expected := errors.New("pool source failed")
	repo := &sourceRepoMock{syncedErr: collectorrepo.ErrNotFound}
	source := &sourceStoreMock{
		syncedHeight: 1,
		txs:          map[uint64]parser.RawTxs{1: {{Hash: "tx1"}}},
		poolInfoErr:  expected,
	}

	err := DoCollect(
		repo,
		source,
		configs.CollectorConfig{ChainId: "chain", UntilHeight: 1, PoolSnapshotInterval: 1},
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
		configs.CollectorConfig{ChainId: "chain", UntilHeight: 1},
		logging.Discard,
	)

	require.ErrorIs(t, err, expected)
	require.Empty(t, repo.saved)
}

func TestDoCollectReturnsSourceHeightError(t *testing.T) {
	expected := errors.New("source height failed")
	repo := &sourceRepoMock{syncedErr: collectorrepo.ErrNotFound}

	err := DoCollect(
		repo,
		&sourceStoreMock{syncedErr: expected},
		configs.CollectorConfig{ChainId: "chain", UntilHeight: 1},
		logging.Discard,
	)

	require.ErrorIs(t, err, expected)
	require.Empty(t, repo.saved)
}

func TestDoCollectReturnsSaveHeightError(t *testing.T) {
	expected := errors.New("save height failed")
	repo := &sourceRepoMock{syncedErr: collectorrepo.ErrNotFound, saveErr: expected}
	source := &sourceStoreMock{
		syncedHeight: 1,
		txs:          map[uint64]parser.RawTxs{1: {{Hash: "tx1"}}},
	}

	err := DoCollect(
		repo,
		source,
		configs.CollectorConfig{ChainId: "chain", UntilHeight: 1},
		logging.Discard,
	)

	require.ErrorIs(t, err, expected)
	require.Empty(t, repo.saved)
}

func TestCollectorStartHeight(t *testing.T) {
	require.Equal(t, uint64(1), collectorStartHeight(configs.CollectorConfig{}))
	require.Equal(t, uint64(42), collectorStartHeight(configs.CollectorConfig{StartHeight: 42}))
}

func TestCollectorPoolSnapshotInterval(t *testing.T) {
	require.Equal(t, uint(7), collectorPoolSnapshotInterval(configs.CollectorConfig{PoolSnapshotInterval: 7}))
	require.Equal(t, uint(configs.PARSER_POOL_SNAPSHOT_INTERVAL), collectorPoolSnapshotInterval(configs.CollectorConfig{}))
}
