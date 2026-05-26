package collector

import (
	"errors"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/stretchr/testify/require"
)

func TestSourceHeightCollectorLocalHeightUsesPreviousStartHeightWhenRepositoryHasNoProgress(t *testing.T) {
	for _, err := range []error{repo.ErrNotFound, repo.ErrUnavailable} {
		collector := &sourceHeightCollector{
			repo:        &sourceRepoMock{syncedErr: err},
			chainID:     "chain",
			startHeight: 5,
		}

		height, actualErr := collector.LocalHeight()

		require.NoError(t, actualErr)
		require.Equal(t, uint64(4), height)
	}
}

func TestSourceHeightCollectorLocalHeightReturnsRepositoryError(t *testing.T) {
	expected := errors.New("height failed")
	collector := &sourceHeightCollector{
		repo:    &sourceRepoMock{syncedErr: expected},
		chainID: "chain",
	}

	_, err := collector.LocalHeight()

	require.ErrorIs(t, err, expected)
}

func TestSourceHeightCollectorCollectHeightSavesScheduledPoolSnapshot(t *testing.T) {
	repository := &sourceRepoMock{}
	source := &sourceStoreMock{
		txs: map[uint64]parser.RawTxs{
			10: {{Hash: "tx", Timestamp: time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)}},
		},
		poolInfos: map[uint64][]dex.PoolInfo{
			10: {{ContractAddr: "pair"}},
		},
	}
	collector := &sourceHeightCollector{
		repo:                 repository,
		source:               source,
		chainID:              "chain",
		poolSnapshotInterval: 10,
	}

	err := collector.CollectHeight(10)

	require.NoError(t, err)
	require.Len(t, repository.saved, 1)
	require.True(t, repository.saved[0].savePoolSnapshot)
	require.Equal(t, []dex.PoolInfo{{ContractAddr: "pair"}}, repository.saved[0].poolInfos)
	require.Equal(t, parser.RawTxs{{Hash: "tx", Timestamp: time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)}}, repository.saved[0].txs)
}
