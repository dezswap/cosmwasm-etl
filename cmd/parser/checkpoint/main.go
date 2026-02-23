// Heights:
// - targetHeight: block height to create checkpoint
// - dbHeight: last checkpoint height in DB
// - sourceHeight: current synced height of node
//
// Flow:
// 1. input targetHeight
// 2. validate heights (dbHeight < sourceHeight, dbHeight < targetHeight, targetHeight <= sourceHeight)
// 3. read pool states at targetHeight
// 4. save checkpoint to DB
package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser/checkpoint"
	pdex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/parser/dex/repo"
	"github.com/dezswap/cosmwasm-etl/parser/dex/srcstore"
	pts "github.com/dezswap/cosmwasm-etl/parser/dex/srcstore/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/columbusv1"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/columbusv2"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/phoenix"
	"github.com/dezswap/cosmwasm-etl/pkg/grpc"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/col4"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
	"github.com/pkg/errors"
)

func main() {
	c := configs.New()
	grpc.SetLogConfig(c.Log)

	var targetHeight uint64
	flag.Uint64Var(&targetHeight, "height", 0, "target block height")
	flag.Parse()

	if err := run(c, targetHeight); err != nil {
		panic(err)
	}
}

func run(c configs.Config, targetHeight uint64) error {
	r := repo.New(c.Parser.DexConfig.ChainId, c.Rdb)
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:      10,
			IdleConnTimeout:   30 * time.Second,
			DisableKeepAlives: false,
		},
	}
	ds := NewSourceDataStore(c, httpClient)

	builder := checkpoint.NewBuilder(r, ds)
	return builder.Build(targetHeight)
}

func NewSourceDataStore(c configs.Config, httpClient *http.Client) pdex.SourceDataStore {
	dc := *c.Parser.DexConfig

	if dc.TargetApp == dex.Terraswap {
		r := rpc.New(dc.NodeConfig.RestClientConfig.RpcHost, httpClient)

		switch terraswap.TerraswapFactory(dc.FactoryAddress) {
		case terraswap.CLASSIC_V1_FACTORY:
			lcd := col4.NewLcd(dc.NodeConfig.RestClientConfig.LcdHost, httpClient)
			queryClient := columbusv1.NewCol4Client(lcd)
			return pts.NewCol4Store(dc.FactoryAddress, r, lcd, queryClient)
		case terraswap.CLASSIC_V2_FACTORY:
			lcd := cosmos45.NewLcd(dc.NodeConfig.RestClientConfig.LcdHost, httpClient)
			queryClient := columbusv2.NewColumbusV2Client(lcd)
			return pts.NewCol5Store(dc.FactoryAddress, r, lcd, queryClient)
		case terraswap.MAINNET_FACTORY:
			lcd := cosmos45.NewLcd(dc.NodeConfig.RestClientConfig.LcdHost, httpClient)
			queryClient := phoenix.NewPhoenixClient(lcd)
			return pts.NewCol5Store(dc.FactoryAddress, r, lcd, queryClient)
		case terraswap.PISCO_FACTORY:
			panic(errors.New("not implemented yet"))
		default:
			panic(errors.Errorf("invalid factory address: %s", dc.FactoryAddress))
		}
	}

	readStore := NewReadStore(c, dc.ChainId, httpClient)
	return srcstore.New(readStore)
}

func NewReadStore(c configs.Config, chainId string, httpClient *http.Client) datastore.ReadStore {
	nc := c.Parser.DexConfig.NodeConfig
	serviceDesc := grpc.GetServiceDesc("checkpoint", nc.GrpcConfig)

	var lcdClient datastore.LcdClient
	if nc.FailoverLcdHost != "" {
		lcdClient = datastore.NewLcdClient(nc.FailoverLcdHost, httpClient)
	}
	isXplaChain := util.NetworkNameByChainID(c.Collector.ChainId) == util.NetworkXpla
	store, err := datastore.New(c, serviceDesc, lcdClient, isXplaChain)
	if err != nil {
		panic(err)
	}

	return datastore.NewReadStoreWithGrpc(chainId, store)
}
