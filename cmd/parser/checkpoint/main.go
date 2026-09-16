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
	srcterra "github.com/dezswap/cosmwasm-etl/parser/dex/srcstore/terra"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra/classicv1"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra/classicv2"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra/terra2"
	"github.com/dezswap/cosmwasm-etl/pkg/grpc"
	"github.com/dezswap/cosmwasm-etl/pkg/httpclient"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/col4"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
	"github.com/pkg/errors"
)

var version = "dev" // overridden via -ldflags "-X main.version=v1.2.3"

func main() {
	c := configs.New()
	grpc.SetLogConfig(c.Log)

	var targetHeight uint64
	flag.Uint64Var(&targetHeight, "height", 0, "target block height")
	flag.Parse()

	logger := logging.New("checkpoint", c.Log)
	logger.WithField("version", version).Info("starting checkpoint")

	if err := run(c, targetHeight); err != nil {
		panic(err)
	}
}

func run(c configs.Config, targetHeight uint64) error {
	if err := c.Parser.DexConfig.Validate(); err != nil {
		return errors.Wrap(err, "invalid parser dex config")
	}

	r := repo.New(c.Parser.DexConfig.ChainId, c.Rdb)
	httpClient := &http.Client{
		Timeout: httpclient.DefaultTimeout,
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
	dc := c.Parser.DexConfig

	if name := dex.ChainNameOf(dc.ChainId); name == dex.ChainNameTerraClassic || name == dex.ChainNameTerra2 {
		r := rpc.New(dc.NodeConfig.RestClientConfig.RpcHost, httpClient)

		switch terra.Factory(dc.FactoryAddress) {
		case terra.ClassicV1Factory:
			lcd := col4.NewLcd(dc.NodeConfig.RestClientConfig.LcdHost, httpClient)
			queryClient := classicv1.NewClient(lcd)
			return srcterra.NewClassicV1Store(dc.FactoryAddress, r, lcd, queryClient)
		case terra.ClassicV2Factory:
			lcd := cosmos45.NewLcd(dc.NodeConfig.RestClientConfig.LcdHost, httpClient)
			queryClient := classicv2.NewClient(lcd)
			return srcterra.NewClassicV2Store(dc.FactoryAddress, r, lcd, queryClient)
		case terra.MainnetFactory:
			lcd := cosmos45.NewLcd(dc.NodeConfig.RestClientConfig.LcdHost, httpClient)
			queryClient := terra2.NewClient(lcd)
			return srcterra.NewTerra2Store(dc.FactoryAddress, r, lcd, queryClient)
		case terra.PiscoFactory:
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

	store, err := datastore.New(c, serviceDesc, lcdClient)
	if err != nil {
		panic(err)
	}

	return datastore.NewReadStoreWithGrpc(chainId, store)
}
