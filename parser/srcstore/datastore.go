package srcstore

import (
	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/pkg/errors"
)

type rawDataStoreImpl struct {
	mapper
	datastore.ReadStore
}

var _ parser.SourceDataStore = &rawDataStoreImpl{}

func New(store datastore.ReadStore) parser.SourceDataStore {
	return &rawDataStoreImpl{
		&mapperImpl{},
		store,
	}
}

// v implements parser.RawDataStore
func (r *rawDataStoreImpl) GetSourceSyncedHeight() (uint64, error) {
	height, err := r.GetLatestHeight()
	if err != nil {
		return 0, errors.Wrap(err, "rawDataStoreImpl.GetSourceSyncedHeight")
	}

	return height, nil
}

// GetPoolInfos implements parser.RawDataStore
func (r *rawDataStoreImpl) GetPoolInfos(height uint64) ([]parser.PoolInfo, error) {
	poolInfos, err := r.GetPoolStatusOfAllPairsByHeight(height)
	if err != nil {
		return nil, errors.Wrap(err, "rawDataStoreImpl.GetPoolInfos")
	}

	return r.mapper.rawPoolInfosToPoolInfos(poolInfos), nil
}

// GetSourceTxs implements parser.RawDataStore
func (r *rawDataStoreImpl) GetSourceTxs(height uint64) (parser.RawTxs, error) {
	block, err := r.GetBlockByHeight(height)
	if err != nil {
		return nil, errors.Wrap(err, "rawDataStoreImpl.GetSourceTxs")
	}

	return r.mapper.blockToRawTxs(block), nil
}
