package terraswap

import (
	"fmt"

	"github.com/dezswap/cosmwasm-etl/configs"
	pdex "github.com/dezswap/cosmwasm-etl/parser/dex"
	dts "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	dts_colv1 "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/columbusv1"
	dts_colv2 "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/columbusv2"
	dts_phoenix "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/phoenix"
	"github.com/dezswap/cosmwasm-etl/pkg/httpclient"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/col4"
	terra_cosmos45 "github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
	"github.com/pkg/errors"
)

func NewFromConfig(c configs.NodeConfig, factoryAddress string) (pdex.SourceDataStore, error) {
	httpClient := httpclient.New(c.HttpClientConfig)
	r := rpc.New(c.RestClientConfig.RpcHost, httpClient)

	switch dts.TerraswapFactory(factoryAddress) {
	case dts.MAINNET_FACTORY:
		lcd := terra_cosmos45.NewLcd(c.RestClientConfig.LcdHost, httpClient)
		terraswapQueryClient := dts_phoenix.NewPhoenixClient(lcd)
		return NewPhoenixStore(factoryAddress, r, lcd, terraswapQueryClient), nil
	case dts.CLASSIC_V2_FACTORY:
		lcd := terra_cosmos45.NewLcd(c.RestClientConfig.LcdHost, httpClient)
		terraswapQueryClient := dts_colv2.NewColumbusV2Client(lcd)
		return NewCol5Store(factoryAddress, r, lcd, terraswapQueryClient), nil
	case dts.PISCO_FACTORY:
		return nil, errors.New("not implemented yet")
	case dts.CLASSIC_V1_FACTORY:
		lcd := col4.NewLcd(c.RestClientConfig.LcdHost, httpClient)
		terraswapQueryClient := dts_colv1.NewCol4Client(lcd)
		return NewCol4Store(factoryAddress, r, lcd, terraswapQueryClient), nil
	default:
		return nil, fmt.Errorf("invalid factory address: %s", factoryAddress)
	}
}
