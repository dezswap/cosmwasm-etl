package terra

import (
	"fmt"

	"github.com/dezswap/cosmwasm-etl/configs"
	pdex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra/classicv1"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra/classicv2"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra/terra2"
	"github.com/dezswap/cosmwasm-etl/pkg/httpclient"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/col4"
	terra_cosmos45 "github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
	"github.com/pkg/errors"
)

func NewFromConfig(c configs.NodeConfig, factoryAddress string) (pdex.SourceDataStore, error) {
	httpClient := httpclient.New(c.HttpClientConfig)
	r := rpc.New(c.RestClientConfig.RpcHost, httpClient)

	switch terra.Factory(factoryAddress) {
	case terra.MainnetFactory:
		lcd := terra_cosmos45.NewLcd(c.RestClientConfig.LcdHost, httpClient)
		queryClient := terra2.NewClient(lcd)
		return NewTerra2Store(factoryAddress, r, lcd, queryClient), nil
	case terra.ClassicV2Factory:
		lcd := terra_cosmos45.NewLcd(c.RestClientConfig.LcdHost, httpClient)
		queryClient := classicv2.NewClient(lcd)
		return NewClassicV2Store(factoryAddress, r, lcd, queryClient), nil
	case terra.PiscoFactory:
		return nil, errors.New("not implemented yet")
	case terra.ClassicV1Factory:
		lcd := col4.NewLcd(c.RestClientConfig.LcdHost, httpClient)
		queryClient := classicv1.NewClient(lcd)
		return NewClassicV1Store(factoryAddress, r, lcd, queryClient), nil
	default:
		return nil, fmt.Errorf("invalid factory address: %s", factoryAddress)
	}
}
