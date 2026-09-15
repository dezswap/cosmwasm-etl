package terra

import (
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/lcd"
	"github.com/pkg/errors"
)

type QueryClient interface {
	QueryPool(pairAddr string, height ...uint64) (*dex.PoolInfoRes, error)
	QueryPairs(factoryAddr string, startAfter []dex.AssetInfo, height ...uint64) (*dex.FactoryPairsRes, error)
}

type cosmos45QueryClient struct {
	lcd lcd.Lcd[cosmos45.LcdTxRes]
}

// NewCosmos45Client serves every chain whose contracts answer base64 encoded queries over
// a Cosmos SDK 0.45 LCD, which today is terra classic's v2 factory and terra 2.0 alike.
func NewCosmos45Client(lcd lcd.Lcd[cosmos45.LcdTxRes]) QueryClient {
	return &cosmos45QueryClient{lcd}
}

func (c *cosmos45QueryClient) QueryPool(pairAddr string, height ...uint64) (*dex.PoolInfoRes, error) {
	res, err := cosmos45.QueryContractState[dex.PoolInfoRes](c.lcd, pairAddr, dex.PAIR_QUERY_POOL_BASE64_STRING, height...)
	if err != nil {
		return nil, errors.Wrap(err, "cosmos45QueryClient.QueryPool")
	}

	return &res.Data, nil
}

func (c *cosmos45QueryClient) QueryPairs(factoryAddr string, startAfter []dex.AssetInfo, height ...uint64) (*dex.FactoryPairsRes, error) {
	pairsReq := dex.FactoryPairsReq{}
	if startAfter != nil {
		pairsReq.Pairs.StartAfter = (*[2]dex.AssetInfo)(startAfter)
	}

	req, err := dex.QueryToBase64Str(pairsReq)
	if err != nil {
		return nil, errors.Wrap(err, "cosmos45QueryClient.QueryPairs")
	}

	res, err := cosmos45.QueryContractState[dex.FactoryPairsRes](c.lcd, factoryAddr, req, height...)
	if err != nil {
		return nil, errors.Wrap(err, "cosmos45QueryClient.QueryPairs")
	}

	return &res.Data, nil
}
