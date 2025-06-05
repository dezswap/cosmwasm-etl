package srcstore

import (
	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	psrcstore "github.com/dezswap/cosmwasm-etl/parser/srcstore"
	"github.com/pkg/errors"
)

type rawDataStoreImpl struct {
	parser.SourceDataStore
	mapper
	datastore.ReadStore
}

var _ dex.SourceDataStore = &rawDataStoreImpl{}

func New(store datastore.ReadStore) dex.SourceDataStore {
	pstore := psrcstore.New(store)
	pm := psrcstore.NewMapper()
	m := newMapper(pm)

	return &rawDataStoreImpl{
		pstore,
		m,
		store,
	}
}

// GetPoolInfos implements p_dex.RawDataStore
func (r *rawDataStoreImpl) GetPoolInfos(height uint64) ([]dex.PoolInfo, error) {
	poolInfos, err := r.GetPoolStatusOfAllPairsByHeight(height)
	if err != nil {
		return nil, errors.Wrap(err, "rawDataStoreImpl.GetPoolInfos")
	}

	return r.rawPoolInfosToPoolInfos(poolInfos), nil
}
