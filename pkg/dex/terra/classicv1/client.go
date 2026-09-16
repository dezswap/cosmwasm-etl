package classicv1

import (
	"strconv"

	"github.com/dezswap/cosmwasm-etl/pkg/terra/lcd"

	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/col4"
	"github.com/pkg/errors"
)

type queryClient struct {
	lcd lcd.Lcd[col4.LcdTxRes]
}

func NewClient(lcd lcd.Lcd[col4.LcdTxRes]) terra.QueryClient {
	return &queryClient{lcd}
}

func (c *queryClient) QueryPool(pairAddr string, height ...uint64) (*dex.PoolInfoRes, error) {
	res, err := col4.QueryContractState[dex.PoolInfoRes](c.lcd, pairAddr, dex.PAIR_QUERY_POOL_STRING, height...)
	if err != nil {
		return nil, errors.Wrap(err, "queryClient.QueryPool")
	}

	if len(height) > 0 {
		if err := c.heightCheck(res.Height, height[0]); err != nil {
			return nil, errors.Wrap(err, "queryClient.QueryPool")
		}
	}

	return &res.Result, nil
}

func (c *queryClient) QueryPairs(factoryAddr string, startAfter []dex.AssetInfo, height ...uint64) (*dex.FactoryPairsRes, error) {
	pairsReq := dex.FactoryPairsReq{}
	if startAfter != nil {
		pairsReq.Pairs.StartAfter = (*[2]dex.AssetInfo)(startAfter)
	}

	req, err := dex.QueryToJsonStr(pairsReq)
	if err != nil {
		return nil, errors.Wrap(err, "queryClient.QueryPairs")
	}

	res, err := col4.QueryContractState[dex.FactoryPairsRes](c.lcd, factoryAddr, req, height...)
	if err != nil {
		return nil, errors.Wrap(err, "queryClient.QueryPairs")
	}

	if len(height) > 0 {
		if err := c.heightCheck(res.Height, height[0]); err != nil {
			return nil, errors.Wrap(err, "queryClient.QueryPairs")
		}
	}

	return &res.Result, nil
}

func (c *queryClient) heightCheck(actualStr string, expected uint64) error {
	resHeight := "0"
	if actualStr != "" {
		resHeight = actualStr
	}

	actual, err := strconv.ParseUint(resHeight, 10, 64)
	if err != nil {
		return errors.Wrap(err, "queryClient.heightCheck")
	}

	if actual != expected {
		return dex.ErrQueryDifferentHeight
	}

	return nil
}
