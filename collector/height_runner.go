package collector

import (
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/logging"
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

// collectHeights runs the common contiguous-height collection loop.
// Implementations own source reads and persistence for a single height, while
// the runner handles local/source progress, until-height bounds, and polling.
func collectHeights(collector heightCollector, config heightCollectorConfig, logger logging.Logger) error {
func collectHeights(collector heightCollector, config heightCollectorConfig, logger logging.Logger) error {
	startHeight := normalizeStartHeight(config.StartHeight)
	if config.UntilHeight > 0 && config.UntilHeight < startHeight {
		return fmt.Errorf("invalid height range: start_height=%d until_height=%d", startHeight, config.UntilHeight)
	}

	pollInterval := config.PollInterval
	if pollInterval == 0 {
		pollInterval = collectorPollInterval
	}
	// ... rest of function
}
	if pollInterval == 0 {
		pollInterval = collectorPollInterval
	}

	for {
		localHeight, err := collector.LocalHeight()
		if err != nil {
			return err
		}

		srcHeight, err := collector.SourceHeight()
		if err != nil {
			return err
		}

		targetHeight := boundedTargetHeight(srcHeight, config.UntilHeight)
		if localHeight >= targetHeight {
			if reachedUntilHeight(localHeight, config.UntilHeight) {
				logger.Infof("collector reached until height %d", config.UntilHeight)
				return nil
			}
			logger.Infof("no new collector source height: local=%d source=%d", localHeight, srcHeight)
			time.Sleep(pollInterval)
			continue
		}

		for height := localHeight + 1; height <= targetHeight; height++ {
			if height < startHeight {
				continue
			}
			if err := collector.CollectHeight(height); err != nil {
				return err
			}
			logger.Infof("collected source height %d", height)
		}
	}
}

func normalizeStartHeight(height uint64) uint64 {
	if height > 0 {
		return height
	}
	return 1
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
