package collector

import (
	"errors"
	"time"

	"github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
)

type sourceHeightCollector struct {
	repo                 repo.Repository
	source               dex.SourceDataStore
	chainID              string
	startHeight          uint64
	poolSnapshotInterval uint
}

func (c *sourceHeightCollector) LocalHeight() (uint64, error) {
	localHeight, err := c.repo.GetSyncedHeight(c.chainID)
	if err == nil {
		return localHeight, nil
	}
	if errors.Is(err, repo.ErrNotFound) || errors.Is(err, repo.ErrUnavailable) {
		return c.startHeight - 1, nil
	}
	return 0, err
}

func (c *sourceHeightCollector) SourceHeight() (uint64, error) {
	return c.source.GetSourceSyncedHeight()
}

func (c *sourceHeightCollector) CollectHeight(height uint64) error {
	txs, err := c.source.GetSourceTxs(height)
	if err != nil {
		return err
	}

	blockTime := time.Time{}
	if len(txs) > 0 {
		blockTime = txs[0].Timestamp
	}

	savePoolSnapshot := c.poolSnapshotInterval > 0 && height%uint64(c.poolSnapshotInterval) == 0
	poolInfos := []dex.PoolInfo{}
	if savePoolSnapshot {
		poolInfos, err = c.source.GetPoolInfos(height)
		if err != nil {
			return err
		}
	}

	return c.repo.SaveHeight(c.chainID, height, blockTime, txs, poolInfos, savePoolSnapshot)
}
