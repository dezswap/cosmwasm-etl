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
	"github.com/dezswap/cosmwasm-etl/configs"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/parser/dex/dexwiring"
	"github.com/dezswap/cosmwasm-etl/parser/dex/repo"
	"github.com/sirupsen/logrus"

	"github.com/dezswap/cosmwasm-etl/pkg/grpc"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

const (
	app = "parser"
)

var version = "dev" // overridden via -ldflags "-X main.version=v1.2.3"

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
	app, err := dexwiring.NewTargetApp(repo, logger, c)
	if err != nil {
		panic(err)
	}

	rawDataStore, err := dexwiring.NewSourceDataStore(c, rdbc, readStore, logger)
	if err != nil {
		panic(err)
	}

	runner := p_dex.NewDexApp(app, rawDataStore, repo, logger, c)

	const BLOCK_SECONDS = 5 * time.Second
	for errCount := uint(0); errCount <= c.ErrTolerance; {
		if err := runner.Run(); err != nil {
			if errors.Is(err, p_dex.ErrNoNewHeight) {
				logger.WithFields(logrus.Fields{
					"event":     "parser.waiting_for_new_height",
					"operation": "parser.run",
					"chain_id":  c.ChainId,
					"err":       logging.NewErrorField(err),
				}).Info("parser waiting for new height")
			} else {
				errCount++
				logger.WithFields(logrus.Fields{
					"event":       "parser.run_failed",
					"operation":   "parser.run",
					"chain_id":    c.ChainId,
					"retry_count": errCount,
					"err":         logging.NewErrorField(err),
				}).Error("parser run failed")
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
	readstore, err := dexwiring.NewTargetReadStore(c, dc)
	if err != nil {
		panic(err)
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
