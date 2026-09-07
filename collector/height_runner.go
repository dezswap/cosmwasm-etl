package collector

import (
	"errors"
	"fmt"
	"time"

	"github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
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
		if c.startHeight == 0 {
			return 0, nil
		}
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
		return classifySourceErr(err)
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
			return classifySourceErr(err)
		}
	}

	if err := c.repo.SaveHeight(c.chainID, height, blockTime, txs, poolInfos, savePoolSnapshot); err != nil {
		return fmt.Errorf("%w: %w", errLocalStore, err)
	}
	return nil
}

// classifySourceErr keeps retrying the default, since only the rpc client carries
// a verdict: pool queries reach the chain over LCD and arrive unclassified.
func classifySourceErr(err error) error {
	if errors.Is(err, rpc.ErrHeightUnavailable) {
		return fmt.Errorf("%w: %w", errSourceUnavailable, err)
	}
	return err
}
