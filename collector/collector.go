package collector

import (
	"errors"
	"time"

	"github.com/dezswap/cosmwasm-etl/collector/repo"
	collectorrepo "github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

// DoCollect persists normalized parser source data into the collector DB.
//
// It consumes any dex SourceDataStore implementation and stores per-height txs,
// optional pool snapshots, and synced height in PostgreSQL. That keeps the loop
// reusable for future DEX apps as long as they expose the same parser source
// interface.
func DoCollect(repo repo.Repository, source dex.SourceDataStore, collectorConfig configs.CollectorConfig, logger logging.Logger) error {
	return collectHeights(&sourceHeightCollector{
		repo:                 repo,
		source:               source,
		chainID:              collectorConfig.ChainId,
		startHeight:          collectorConfig.StartHeight,
		poolSnapshotInterval: collectorConfig.PoolSnapshotInterval,
	}, heightCollectorConfig{
		StartHeight:  collectorConfig.StartHeight,
		UntilHeight:  collectorConfig.UntilHeight,
		PollInterval: time.Duration(collectorConfig.PollIntervalSec) * time.Second,
	}, logger)
}

type sourceHeightCollector struct {
	repo                 collectorrepo.Repository
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
	if errors.Is(err, collectorrepo.ErrNotFound) || errors.Is(err, collectorrepo.ErrUnavailable) {
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
