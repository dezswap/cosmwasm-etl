package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"runtime/debug"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	collectorrepo "github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/configs"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	pds "github.com/dezswap/cosmwasm-etl/parser/dex/dezswap"
	"github.com/dezswap/cosmwasm-etl/parser/dex/repo"
	"github.com/dezswap/cosmwasm-etl/parser/dex/srcstore"
	ts_srcstore "github.com/dezswap/cosmwasm-etl/parser/dex/srcstore/terraswap"
	psf "github.com/dezswap/cosmwasm-etl/parser/dex/starfleit"
	pts "github.com/dezswap/cosmwasm-etl/parser/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/httpclient"

	"github.com/dezswap/cosmwasm-etl/pkg/grpc"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/s3client"
)

const (
	app = "parser"
)

var version = "dev" // overridden via -ldflags "-X main.version=v1.2.3"

func getDexCollectorReadStore(c configs.Config, dc configs.ParserDexConfig) datastore.ReadStore {
	nodeConf := dc.NodeConfig
	if nodeConf.GrpcConfig.Host != "" {
		nodeConf := dc.NodeConfig
		serviceDesc := grpc.GetServiceDesc("collector", nodeConf.GrpcConfig)

		store, err := datastore.New(c, serviceDesc, nil)
		if err != nil {
			panic(err)
		}
		if nodeConf.FailoverLcdHost != "" {
			store, _ = datastore.New(c, serviceDesc, datastore.NewLcdClient(nodeConf.FailoverLcdHost, httpclient.New(dc.NodeConfig.HttpClientConfig)))
		}

		return datastore.NewReadStoreWithGrpc(dc.ChainId, store)
	}

	s3Client, err := s3client.NewClient(c.S3)
	if err != nil {
		panic(err)
	}
	return datastore.NewReadStore(dc.ChainId, s3Client)
}

func dex_main(c configs.ParserDexConfig, logc configs.LogConfig, sentryc configs.SentryConfig, rdbc configs.RdbConfig, readStore datastore.ReadStore) {
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
	switch c.TargetApp {
	case dex.Terraswap:
		app, err = pts.New(repo, logger, c)
	case dex.Dezswap:
		app, err = pds.New(repo, logger, c, c.ChainId)
	case dex.Starfleit:
		app, err = psf.New(repo, logger, c, c.ChainId)
	default:
		panic("unknown target app: " + c.TargetApp)
	}

	if err != nil {
		panic(err)
	}

	var rawDataStore p_dex.SourceDataStore
	if c.TargetApp == dex.Terraswap {
		fallback, err := ts_srcstore.NewFromConfig(c.NodeConfig, c.FactoryAddress)
		if err != nil {
			panic(err)
		}
		rawDataStore = srcstore.NewCollectorFallback(c.ChainId, collectorrepo.New(rdbc), fallback, logger)
	} else {
		rawDataStore = srcstore.New(readStore)
	}

	runner := p_dex.NewDexApp(app, rawDataStore, repo, logger, c)

	const BLOCK_SECONDS = 5 * time.Second
	for errCount := uint(0); errCount <= c.ErrTolerance; {
		if err := runner.Run(); err != nil {
			if errors.Is(err, p_dex.ErrNoNewHeight) {
				logger.Infof("no new block yet: %s", err)
			} else {
				errCount++
				logger.Errorf("errCount: %d, err: %s", errCount, err)
			}
		} else {
			errCount = 0
		}
		wait := BLOCK_SECONDS * time.Duration(math.Pow(2, float64(errCount)))
		time.Sleep(wait)
	}
}

func main() {
	c := configs.New()

	grpc.SetLogConfig(c.Log)
	logger := logging.New("parser", c.Log)
	logger.WithField("version", version).Info("starting parser")

	defer catch(logger)
	if err := c.Parser.DexConfig.Validate(); err != nil {
		panic(fmt.Errorf("dex config is nil: %w", err))
	}

	dc := c.Parser.DexConfig
	var readstore datastore.ReadStore
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
