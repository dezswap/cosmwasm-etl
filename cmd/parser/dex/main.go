package main

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	collector_store "github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/dezswap/cosmwasm-etl/configs"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/pkg/errors"

	pds "github.com/dezswap/cosmwasm-etl/parser/dex/dezswap"
	"github.com/dezswap/cosmwasm-etl/parser/dex/repo"
	"github.com/dezswap/cosmwasm-etl/parser/dex/srcstore"
	ts_srcstore "github.com/dezswap/cosmwasm-etl/parser/dex/srcstore/terraswap"
	psf "github.com/dezswap/cosmwasm-etl/parser/dex/starfleit"
	pts "github.com/dezswap/cosmwasm-etl/parser/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	dts "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	dts_colv1 "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/columbus_v1"
	dts_phoenix "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/phoenix"

	"github.com/dezswap/cosmwasm-etl/pkg/grpc"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/s3client"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/col4"
	terra_phoenix "github.com/dezswap/cosmwasm-etl/pkg/terra/phoenix"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
)

const (
	app = "parser"
)

func getDexCollectorReadStore(c configs.Config, dc configs.ParserDexConfig) collector_store.ReadStore {
	nodeConf := dc.NodeConfig
	if nodeConf.GrpcConfig.Host != "" {
		nodeConf := dc.NodeConfig
		serviceDesc := grpc.GetServiceDesc("collector", nodeConf.GrpcConfig)

		store, err := collector_store.New(c, serviceDesc, nil)
		if err != nil {
			panic(err)
		}
		if nodeConf.FailoverLcdHost != "" {
			httpClient := &http.Client{
				Transport: &http.Transport{
					MaxIdleConns:      10,               // Maximum idle connections to keep open
					IdleConnTimeout:   30 * time.Second, // Time to keep idle connections open
					DisableKeepAlives: false,            // Use HTTP Keep-Alive
				},
			}
			store, _ = collector_store.New(c, serviceDesc, datastore.NewLcdClient(nodeConf.FailoverLcdHost, httpClient))
		}

		return collector_store.NewReadStoreWithGrpc(dc.ChainId, store)
	}

	s3Client, err := s3client.NewClient()
	if err != nil {
		panic(err)
	}
	return collector_store.NewReadStore(dc.ChainId, s3Client)
}

func dex_main(c configs.ParserDexConfig, logc configs.LogConfig, sentryc configs.SentryConfig, rdbc configs.RdbConfig, readStore collector_store.ReadStore) {
	logger := logging.New("main", logc)
	if sentryc.DSN != "" {
		sentryEnv := fmt.Sprintf("%s-%s", c.ChainId, app)
		logging.ConfigureReporter(logger, sentryc.DSN, sentryEnv, map[string]string{
			"x-chain_id": c.ChainId,
			"x-app":      "parser",
			"x-env":      logc.Environment,
		})
	}
	defer catch(logger)

	repo := repo.New(c.ChainId, rdbc)
	var app p_dex.TargetApp
	var err error
	if c.TargetApp == dex.Terraswap {
		app, err = pts.New(repo, logger, c)
	} else if c.TargetApp == dex.Dezswap {
		app, err = pds.New(repo, logger, c, c.ChainId)
	} else if c.TargetApp == dex.Starfleit {
		app, err = psf.New(repo, logger, c, c.ChainId)
	} else {
		panic("unknown target app: " + c.TargetApp)
	}

	if err != nil {
		panic(err)
	}

	var rawDataStore p_dex.SourceDataStore
	if c.TargetApp == dex.Terraswap {
		r := rpc.New(c.NodeConfig.RestClientConfig.RpcHost, &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:      10,               // Maximum idle connections to keep open
				IdleConnTimeout:   30 * time.Second, // Time to keep idle connections open
				DisableKeepAlives: false,            // Use HTTP Keep-Alive
			},
		})

		switch dts.TerraswapFactory(c.FactoryAddress) {
		case dts.MAINNET_FACTORY:
			lcd := terra_phoenix.NewLcd(c.NodeConfig.RestClientConfig.LcdHost, &http.Client{
				Transport: &http.Transport{
					MaxIdleConns:      10,               // Maximum idle connections to keep open
					IdleConnTimeout:   30 * time.Second, // Time to keep idle connections open
					DisableKeepAlives: false,            // Use HTTP Keep-Alive
				},
			})
			terraswapQueryClient := dts_phoenix.NewPhoenixClient(lcd)
			rawDataStore = ts_srcstore.NewPhoenixStore(c.FactoryAddress, r, lcd, terraswapQueryClient)
		case dts.CLASSIC_V2_FACTORY, dts.PISCO_FACTORY:
			panic(errors.New("not implemented yet"))
		case dts.CLASSIC_V1_FACTORY:
			lcd := col4.NewLcd(c.NodeConfig.RestClientConfig.LcdHost, &http.Client{
				Transport: &http.Transport{
					MaxIdleConns:      10,               // Maximum idle connections to keep open
					IdleConnTimeout:   30 * time.Second, // Time to keep idle connections open
					DisableKeepAlives: false,            // Use HTTP Keep-Alive
				},
			})
			terraswapQueryClient := dts_colv1.NewCol4Client(lcd)
			rawDataStore = ts_srcstore.NewCol4Store(c.FactoryAddress, r, lcd, terraswapQueryClient)
		default:
			panic(errors.Errorf("invalid factory address: %s", c.FactoryAddress))
		}
	} else {
		rawDataStore = srcstore.New(readStore)
	}

	runner := p_dex.NewDexApp(app, rawDataStore, repo, logger, c)

	const BLOCK_SECONDS = 5 * time.Second
	for errCount := uint(0); errCount <= c.ErrTolerance; {
		if err := runner.Run(); err != nil {
			errCount++
			logger.Errorf("errCount: %d, err: %s", errCount, err)
		} else {
			errCount = 0
		}
		wait := BLOCK_SECONDS * time.Duration(math.Pow(2, float64(errCount)))
		time.Sleep(wait)
	}

}

func main() {
	c := configs.New()
	logger := logging.New("parser", c.Log)
	defer catch(logger)
	if c.Parser.DexConfig == nil {
		panic("dex config is nil")
	}

	dc := *c.Parser.DexConfig
	var readstore collector_store.ReadStore
	switch dc.TargetApp {
	case dex.Terraswap:
	case dex.Dezswap, dex.Starfleit:
		readstore = getDexCollectorReadStore(c, dc)
	}
	dex_main(dc, c.Log, c.Sentry, c.Rdb, readstore)

}

func catch(logger logging.Logger) {
	recovered := recover()

	if recovered != nil {
		defer os.Exit(1)

		err, ok := recovered.(error)
		if !ok {
			logger.Errorf("could not convert recovered error into error: %s\n", spew.Sdump(recovered))
			return
		}

		stack := string(debug.Stack())
		logger.WithField("err", logging.NewErrorField(err)).WithField("stack", stack).Errorf("panic caught")
	}
}
