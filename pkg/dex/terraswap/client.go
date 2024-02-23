package terraswap

import (
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
)

type QueryClient interface {
	QueryPool(pairAddr string, height ...uint64) (*dex.PoolInfoRes, error)
	QueryPairs(factoryAddr string, startAfter []dex.AssetInfo, height ...uint64) (*dex.FactoryPairsRes, error)
}
