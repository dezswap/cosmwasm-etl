package collector

import (
	"errors"
	"fmt"
	"time"

	collectorrepo "github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

var (
	// errLocalStore marks a persistence failure. CollectHeight mixes source reads
	// with a local write, so implementations wrap the write error to tell the runner
	// which half failed.
	errLocalStore = errors.New("local store")
	// errSourceUnavailable marks a height the source can never serve. Implementations
	// translate their own verdict into it so the runner stays source agnostic.
	errSourceUnavailable = errors.New("source cannot serve height")
)

type heightCollector interface {
	LocalHeight() (uint64, error)
	SourceHeight() (uint64, error)
	CollectHeight(height uint64) error
}

type heightCollectorConfig struct {
	StartHeight  uint64
	UntilHeight  uint64
	PollInterval time.Duration
}

// DoCollect persists normalized parser source data into the collector DB.
//
// It consumes any dex SourceDataStore implementation and stores per-height txs,
// optional pool snapshots, and synced height in PostgreSQL. That keeps the loop
// reusable for future DEX apps as long as they expose the same parser source
// interface.
func DoCollect(repo collectorrepo.Repository, source dex.SourceDataStore, collectorConfig configs.CollectorConfig, logger logging.Logger) error {
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

// collectHeights runs the contiguous-height loop. Implementations own per-height
// reads and persistence; the runner owns progress, until-height bounds, and polling.
//
// Source failures retry indefinitely, leaving the height unconsumed for the next
// poll. A local store failure or a height the source cannot serve exits instead:
// neither clears on its own, and retrying would stall on the same height forever.
func collectHeights(collector heightCollector, config heightCollectorConfig, logger logging.Logger) error {
	startHeight := config.StartHeight
	if config.UntilHeight > 0 && config.UntilHeight < startHeight {
		return fmt.Errorf("invalid height range: start_height=%d until_height=%d", startHeight, config.UntilHeight)
	}

	pollInterval := config.PollInterval

	for {
		localHeight, err := collector.LocalHeight()
		if err != nil {
			return err
		}

		// Checked before the source read so a finished backfill exits even when
		// the node is down.
		if reachedUntilHeight(localHeight, config.UntilHeight) {
			logger.Infof("collector reached until height %d", config.UntilHeight)
			return nil
		}

		srcHeight, err := collector.SourceHeight()
		if err != nil {
			logger.Warnf("collector could not read source height, retrying: %s", err)
			time.Sleep(pollInterval)
			continue
		}

		targetHeight := boundedTargetHeight(srcHeight, config.UntilHeight)
		if localHeight >= targetHeight {
			logger.Infof("no new collector source height: local=%d source=%d", localHeight, srcHeight)
			time.Sleep(pollInterval)
			continue
		}

		nextHeight := localHeight + 1
		if nextHeight < startHeight {
			nextHeight = startHeight
		}
		if nextHeight > targetHeight {
			logger.Infof("no collectible height yet: local=%d source=%d start=%d", localHeight, srcHeight, startHeight)
			time.Sleep(pollInterval)
			continue
		}

		for height := nextHeight; height <= targetHeight; height++ {
			if err := collector.CollectHeight(height); err != nil {
				if errors.Is(err, errLocalStore) || errors.Is(err, errSourceUnavailable) {
					return err
				}
				logger.Warnf("collector could not collect height %d, retrying: %s", height, err)
				time.Sleep(pollInterval)
				break
			}
			logger.Infof("collected source height %d", height)
		}
	}
}

func boundedTargetHeight(sourceHeight, untilHeight uint64) uint64 {
	if untilHeight > 0 && untilHeight < sourceHeight {
		return untilHeight
	}
	return sourceHeight
}

func reachedUntilHeight(localHeight, untilHeight uint64) bool {
	return untilHeight > 0 && localHeight >= untilHeight
}
