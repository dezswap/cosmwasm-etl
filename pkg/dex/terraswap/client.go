package terraswap

import (
	"strconv"

	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/col4"
	"github.com/pkg/errors"
)

type Col4QueryClient interface {
	QueryPool(pairAddr string, height ...uint64) (*dex.PoolInfoRes, error)
	QueryPairs(factoryAddr string, startAfter []dex.AssetInfo, height ...uint64) (*dex.FactoryPairsRes, error)
}

type col4QueryClient struct {
	lcd col4.Lcd
}

func NewCol4Client(lcd col4.Lcd) Col4QueryClient {
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
		return dex.QUERY_DIFFERENT_HEIGHT_ERROR
	}

	return nil
}
