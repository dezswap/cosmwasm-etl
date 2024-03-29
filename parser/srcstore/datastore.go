package srcstore

import (
	"strings"

	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/pkg/errors"
)

type rawDataStoreImpl struct {
	Mapper
	datastore.ReadStore
}

var _ parser.SourceDataStore = &rawDataStoreImpl{}

func New(store datastore.ReadStore) parser.SourceDataStore {
	return &rawDataStoreImpl{
		&mapperImpl{},
		store,
	}
}

// v implements p_dex.RawDataStore
func (r *rawDataStoreImpl) GetSourceSyncedHeight() (uint64, error) {
	height, err := r.GetLatestHeight()
	if err != nil {
		return 0, errors.Wrap(err, "rawDataStoreImpl.GetSourceSyncedHeight")
	}

	return height, nil
}

// GetSourceTxs implements p_dex.RawDataStore
func (r *rawDataStoreImpl) GetSourceTxs(height uint64) (parser.RawTxs, error) {
	retryCount := 3
	var block *datastore.BlockTxsDTO
	var err error
	for ; retryCount > 0; retryCount-- {
		block, err = r.GetBlockByHeight(height)
		if err != nil {
			if strings.Contains(err.Error(), "height must not be less than 1 or greater than the current height") {
				continue
			}
			return nil, errors.Wrap(err, "rawDataStoreImpl.GetSourceTxs")
		}
		return r.BlockToRawTxs(block), nil
	}
	return nil, err
}
