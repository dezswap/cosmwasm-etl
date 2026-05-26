package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/davecgh/go-spew/spew"
	"github.com/dezswap/cosmwasm-etl/collector"
	"github.com/dezswap/cosmwasm-etl/collector/repo"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser/dex/srcstore/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/grpc"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

const app = "collector"

var version = "dev" // overridden via -ldflags "-X main.version=v1.2.3"

func main() {
	c := configs.New()
	logger := logging.New("main", c.Log)
	if c.Sentry.DSN != "" {
		sentryEnv := fmt.Sprintf("%s-%s", c.Collector.ChainId, app)
		logging.ConfigureReporter(logger, c.Sentry.DSN, sentryEnv, map[string]string{
			"x-chain_id": c.Collector.ChainId,
			"x-app":      "collector",
			"x-env":      c.Log.Environment,
		})
	}
	defer catch(logger)

	logger.WithField("version", version).Info("starting collector")
	grpc.SetLogConfig(c.Log)

	if err := c.Collector.Validate(); err != nil {
		panic(fmt.Errorf("dex config is nil: %w", err))
	}

	source, err := terraswap.NewFromConfig(c.Collector.NodeConfig, c.Collector.PairFactoryContractAddress)
	if err != nil {
		panic(err)
	}

	if err := collector.DoCollect(repo.New(c.Rdb), source, c.Collector, logger); err != nil {
		panic(err)
	}
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
