package srcstore

import (
	"errors"
	"sync"

	collectorrepo "github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

type collectorFallbackStore struct {
	chainID  string
	repo     collectorrepo.Repository
	fallback dex.SourceDataStore
	logger   logging.Logger

	logUnavailableOnce sync.Once
}

var _ dex.SourceDataStore = (*collectorFallbackStore)(nil)

// NewCollectorFallback reads parser source data from collector DB first and
// delegates to fallback when collector data is not available yet. Corrupt data
// or real DB failures are returned as errors so they are not hidden by fallback.
func NewCollectorFallback(chainID string, repo collectorrepo.Repository, fallback dex.SourceDataStore, logger logging.Logger) dex.SourceDataStore {
	return &collectorFallbackStore{
		chainID:  chainID,
		repo:     repo,
		fallback: fallback,
		logger:   logger,
	}
}

func (s *collectorFallbackStore) GetSourceSyncedHeight() (uint64, error) {
	height, err := s.repo.GetSyncedHeight(s.chainID)
	if err == nil {
		return height, nil
	}
	if shouldFallbackCollector(err) {
		s.logCollectorFallback("synced height", err)
		return s.fallback.GetSourceSyncedHeight()
	}
	return 0, err
}

func (s *collectorFallbackStore) GetSourceTxs(height uint64) (parser.RawTxs, error) {
	txs, _, err := s.repo.GetBlockTxs(s.chainID, height)
	if err == nil {
		return txs, nil
	}
	if shouldFallbackCollector(err) {
		s.logCollectorFallback("block txs", err)
		return s.fallback.GetSourceTxs(height)
	}
	return nil, err
}

func (s *collectorFallbackStore) GetPoolInfos(height uint64) ([]dex.PoolInfo, error) {
	poolInfos, err := s.repo.GetPoolInfos(s.chainID, height)
	if err == nil {
		return poolInfos, nil
	}
	if shouldFallbackCollector(err) {
		s.logCollectorFallback("pool snapshot", err)
		return s.fallback.GetPoolInfos(height)
	}
	return nil, err
}

func shouldFallbackCollector(err error) bool {
	return errors.Is(err, collectorrepo.ErrNotFound) || errors.Is(err, collectorrepo.ErrUnavailable)
}

func (s *collectorFallbackStore) logCollectorFallback(target string, err error) {
	if errors.Is(err, collectorrepo.ErrUnavailable) {
		s.logUnavailableOnce.Do(func() {
			s.logger.Warnf("collector source tables are unavailable; using source fallback")
		})
		return
	}
	s.logger.Debugf("collector %s not found; using source fallback", target)
}
