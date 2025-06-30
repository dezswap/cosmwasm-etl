package columbusv1

import (
	"github.com/dezswap/cosmwasm-etl/pkg/terra/lcd"
	"strconv"

	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/col4"
	"github.com/pkg/errors"
)

type col4QueryClient struct {
	lcd lcd.Lcd[col4.LcdTxRes]
}

func NewCol4Client(lcd lcd.Lcd[col4.LcdTxRes]) terraswap.QueryClient {
	return &col4QueryClient{lcd}
}

// QueryPool implements Col4QueryClient.
func (c *col4QueryClient) QueryPool(pairAddr string, height ...uint64) (*dex.PoolInfoRes, error) {
	res, err := col4.QueryContractState[dex.PoolInfoRes](c.lcd, pairAddr, dex.PAIR_QUERY_POOL_STRING, height...)
	if err != nil {
		return nil, errors.Wrap(err, "col4QueryClient.QueryPool")
	}

	if len(height) > 0 {
		if err := c.heightCheck(res.Height, height[0]); err != nil {
			return nil, errors.Wrap(err, "col4QueryClient.QueryPool")
		}
	}

	return &res.Result, nil
}

// QueryPairs implements Col4QueryClient.
func (c *col4QueryClient) QueryPairs(factoryAddr string, startAfter []dex.AssetInfo, height ...uint64) (*dex.FactoryPairsRes, error) {
	pairsReq := dex.FactoryPairsReq{}
	if startAfter != nil {
		pairsReq.Pairs.StartAfter = (*[2]dex.AssetInfo)(startAfter)
	}

	req, err := dex.QueryToJsonStr[dex.FactoryPairsReq](pairsReq)
	if err != nil {
		return nil, errors.Wrap(err, "col4QueryClient.QueryPairs")
	}

	res, err := col4.QueryContractState[dex.FactoryPairsRes](c.lcd, factoryAddr, req, height...)
	if err != nil {
		return nil, errors.Wrap(err, "col4QueryClient.QueryPairs")
	}

	if len(height) > 0 {
		if err := c.heightCheck(res.Height, height[0]); err != nil {
			return nil, errors.Wrap(err, "col4QueryClient.QueryPairs")
		}
	}

	return &res.Result, nil
}

func (c *col4QueryClient) heightCheck(actualStr string, expected uint64) error {
	resHeight := "0"
	if actualStr != "" {
		resHeight = actualStr
	}

	actual, err := strconv.ParseUint(resHeight, 10, 64)
	if err != nil {
		return errors.Wrap(err, "col4QueryClient.heightCheck")
	}

	if actual != expected {
		return dex.ErrQueryDifferentHeight
	}

	return nil
}
