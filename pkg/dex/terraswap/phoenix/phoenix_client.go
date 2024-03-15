package phoenix

import (
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/phoenix"
	"github.com/pkg/errors"
)

type phoenixQueryClient struct {
	lcd phoenix.Lcd
}

func NewPhoenixClient(lcd phoenix.Lcd) terraswap.QueryClient {
	return &phoenixQueryClient{lcd}
}

// QueryPool implements Col4QueryClient.
func (c *phoenixQueryClient) QueryPool(pairAddr string, height ...uint64) (*dex.PoolInfoRes, error) {
	res, err := phoenix.QueryContractState[dex.PoolInfoRes](c.lcd, pairAddr, dex.PAIR_QUERY_POOL_BASE64_STRING, height...)
	if err != nil {
		return nil, errors.Wrap(err, "phoenixQueryClient.QueryPool")
	}

	return &res.Data, nil
}

// QueryPairs implements Col4QueryClient.
func (c *phoenixQueryClient) QueryPairs(factoryAddr string, startAfter []dex.AssetInfo, height ...uint64) (*dex.FactoryPairsRes, error) {
	pairsReq := dex.FactoryPairsReq{}
	if startAfter != nil {
		pairsReq.Pairs.StartAfter = (*[2]dex.AssetInfo)(startAfter)
	}

	req, err := dex.QueryToBase64Str[dex.FactoryPairsReq](pairsReq)
	if err != nil {
		return nil, errors.Wrap(err, "phoenixQueryClient.QueryPairs")
	}

	res, err := phoenix.QueryContractState[dex.FactoryPairsRes](c.lcd, factoryAddr, req, height...)
	if err != nil {
		return nil, errors.Wrap(err, "phoenixQueryClient.QueryPairs")
	}

	return &res.Data, nil
}
