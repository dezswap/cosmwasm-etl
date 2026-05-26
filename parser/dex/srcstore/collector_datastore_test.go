package srcstore

import (
	"errors"
	"testing"
	"time"

	collectorrepo "github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/stretchr/testify/require"
)

type fakeCollectorRepo struct {
	height      uint64
	heightErr   error
	txs         parser.RawTxs
	txsErr      error
	poolInfos   []dex.PoolInfo
	poolInfoErr error
}

func (f *fakeCollectorRepo) GetSyncedHeight(string) (uint64, error) {
	return f.height, f.heightErr
}

func (f *fakeCollectorRepo) GetBlockTxs(string, uint64) (parser.RawTxs, time.Time, error) {
	return f.txs, time.Time{}, f.txsErr
}

func (f *fakeCollectorRepo) GetPoolInfos(string, uint64) ([]dex.PoolInfo, error) {
	return f.poolInfos, f.poolInfoErr
}

func (f *fakeCollectorRepo) SaveHeight(string, uint64, time.Time, parser.RawTxs, []dex.PoolInfo, bool) error {
	return nil
}

type fakeCollectorFallback struct {
	height    uint64
	txs       parser.RawTxs
	poolInfos []dex.PoolInfo
	called    bool
}

func (f *fakeCollectorFallback) GetSourceSyncedHeight() (uint64, error) {
	f.called = true
	return f.height, nil
}

func (f *fakeCollectorFallback) GetSourceTxs(uint64) (parser.RawTxs, error) {
	f.called = true
	return f.txs, nil
}

func (f *fakeCollectorFallback) GetPoolInfos(uint64) ([]dex.PoolInfo, error) {
	f.called = true
	return f.poolInfos, nil
}

func TestCollectorFallbackStore_GetSourceTxsUsesCollectorDB(t *testing.T) {
	repo := &fakeCollectorRepo{txs: parser.RawTxs{{Hash: "db"}}}
	fallback := &fakeCollectorFallback{txs: parser.RawTxs{{Hash: "fallback"}}}
	store := NewCollectorFallback("chain", repo, fallback, logging.Discard)

	actual, err := store.GetSourceTxs(10)

	require.NoError(t, err)
	require.Equal(t, parser.RawTxs{{Hash: "db"}}, actual)
	require.False(t, fallback.called)
}

func TestCollectorFallbackStore_GetSourceSyncedHeightUsesCollectorDB(t *testing.T) {
	repo := &fakeCollectorRepo{height: 15}
	fallback := &fakeCollectorFallback{height: 99}
	store := NewCollectorFallback("chain", repo, fallback, logging.Discard)

	actual, err := store.GetSourceSyncedHeight()

	require.NoError(t, err)
	require.Equal(t, uint64(15), actual)
	require.False(t, fallback.called)
}

func TestCollectorFallbackStore_GetSourceSyncedHeightFallbackOnUnavailableCollectorTables(t *testing.T) {
	repo := &fakeCollectorRepo{heightErr: collectorrepo.ErrUnavailable}
	fallback := &fakeCollectorFallback{height: 99}
	store := NewCollectorFallback("chain", repo, fallback, logging.Discard)

	actual, err := store.GetSourceSyncedHeight()

	require.NoError(t, err)
	require.Equal(t, uint64(99), actual)
	require.True(t, fallback.called)
}

func TestCollectorFallbackStore_GetSourceSyncedHeightDoesNotFallbackOnHardError(t *testing.T) {
	hardErr := errors.New("db failed")
	repo := &fakeCollectorRepo{heightErr: hardErr}
	fallback := &fakeCollectorFallback{height: 99}
	store := NewCollectorFallback("chain", repo, fallback, logging.Discard)

	_, err := store.GetSourceSyncedHeight()

	require.ErrorIs(t, err, hardErr)
	require.False(t, fallback.called)
}

func TestCollectorFallbackStore_GetSourceTxsFallbackOnMissingCollectorData(t *testing.T) {
	repo := &fakeCollectorRepo{txsErr: collectorrepo.ErrNotFound}
	fallback := &fakeCollectorFallback{txs: parser.RawTxs{{Hash: "fallback"}}}
	store := NewCollectorFallback("chain", repo, fallback, logging.Discard)

	actual, err := store.GetSourceTxs(10)

	require.NoError(t, err)
	require.Equal(t, fallback.txs, actual)
	require.True(t, fallback.called)
}

func TestCollectorFallbackStore_GetSourceTxsDoesNotFallbackOnHardError(t *testing.T) {
	hardErr := errors.New("bad json")
	repo := &fakeCollectorRepo{txsErr: hardErr}
	fallback := &fakeCollectorFallback{txs: parser.RawTxs{{Hash: "fallback"}}}
	store := NewCollectorFallback("chain", repo, fallback, logging.Discard)

	_, err := store.GetSourceTxs(10)

	require.ErrorIs(t, err, hardErr)
	require.False(t, fallback.called)
}

func TestCollectorFallbackStore_GetPoolInfosFallbackOnUnavailableCollectorTables(t *testing.T) {
	repo := &fakeCollectorRepo{poolInfoErr: collectorrepo.ErrUnavailable}
	fallback := &fakeCollectorFallback{poolInfos: []dex.PoolInfo{{ContractAddr: "fallback"}}}
	store := NewCollectorFallback("chain", repo, fallback, logging.Discard)

	actual, err := store.GetPoolInfos(10)

	require.NoError(t, err)
	require.Equal(t, fallback.poolInfos, actual)
	require.True(t, fallback.called)
}

func TestCollectorFallbackStore_GetPoolInfosUsesCollectorDB(t *testing.T) {
	repo := &fakeCollectorRepo{poolInfos: []dex.PoolInfo{{ContractAddr: "db"}}}
	fallback := &fakeCollectorFallback{poolInfos: []dex.PoolInfo{{ContractAddr: "fallback"}}}
	store := NewCollectorFallback("chain", repo, fallback, logging.Discard)

	actual, err := store.GetPoolInfos(10)

	require.NoError(t, err)
	require.Equal(t, repo.poolInfos, actual)
	require.False(t, fallback.called)
}

func TestCollectorFallbackStore_GetPoolInfosFallbackOnMissingCollectorData(t *testing.T) {
	repo := &fakeCollectorRepo{poolInfoErr: collectorrepo.ErrNotFound}
	fallback := &fakeCollectorFallback{poolInfos: []dex.PoolInfo{{ContractAddr: "fallback"}}}
	store := NewCollectorFallback("chain", repo, fallback, logging.Discard)

	actual, err := store.GetPoolInfos(10)

	require.NoError(t, err)
	require.Equal(t, fallback.poolInfos, actual)
	require.True(t, fallback.called)
}

func TestCollectorFallbackStore_GetPoolInfosDoesNotFallbackOnHardError(t *testing.T) {
	hardErr := errors.New("bad json")
	repo := &fakeCollectorRepo{poolInfoErr: hardErr}
	fallback := &fakeCollectorFallback{poolInfos: []dex.PoolInfo{{ContractAddr: "fallback"}}}
	store := NewCollectorFallback("chain", repo, fallback, logging.Discard)

	_, err := store.GetPoolInfos(10)

	require.ErrorIs(t, err, hardErr)
	require.False(t, fallback.called)
}
