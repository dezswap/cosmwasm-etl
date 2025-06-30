package columbusv2

import (
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/lcd"
	"github.com/pkg/errors"
)

type queryClient struct {
	lcd lcd.Lcd[cosmos45.LcdTxRes]
}

func NewColumbusV2Client(lcd lcd.Lcd[cosmos45.LcdTxRes]) terraswap.QueryClient {
	return &queryClient{lcd}
}

// QueryPool implements Col4QueryClient.
func (c *queryClient) QueryPool(pairAddr string, height ...uint64) (*dex.PoolInfoRes, error) {
	res, err := cosmos45.QueryContractState[dex.PoolInfoRes](c.lcd, pairAddr, dex.PAIR_QUERY_POOL_BASE64_STRING, height...)
	if err != nil {
		return nil, errors.Wrap(err, "queryClient.QueryPool")
	}

	return &res.Data, nil
}

// QueryPairs implements Col4QueryClient.
func (c *queryClient) QueryPairs(factoryAddr string, startAfter []dex.AssetInfo, height ...uint64) (*dex.FactoryPairsRes, error) {
	pairsReq := dex.FactoryPairsReq{}
	if startAfter != nil {
		pairsReq.Pairs.StartAfter = (*[2]dex.AssetInfo)(startAfter)
	}

	req, err := dex.QueryToBase64Str[dex.FactoryPairsReq](pairsReq)
	if err != nil {
		return nil, errors.Wrap(err, "queryClient.QueryPairs")
	}

	res, err := cosmos45.QueryContractState[dex.FactoryPairsRes](c.lcd, factoryAddr, req, height...)
	if err != nil {
		return nil, errors.Wrap(err, "queryClient.QueryPairs")
	}

	return &res.Data, nil
}
