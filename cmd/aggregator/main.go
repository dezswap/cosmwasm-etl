package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/davecgh/go-spew/spew"
	"github.com/dezswap/cosmwasm-etl/aggregator"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

const (
	app = "aggregator"
)

func main() {
	c := configs.New()
	logger := logging.New("aggregator", c.Log)
	if c.Sentry.DSN != "" {
		sentryEnv := fmt.Sprintf("%s-%s", c.Aggregator.ChainId, app)
		logging.ConfigureReporter(logger, c.Sentry.DSN, sentryEnv, map[string]string{
			"x-chain_id": c.Aggregator.ChainId,
			"x-app":      "aggregator",
			"x-env":      c.Log.Environment,
		})
	}
	defer catch(logger)

	app := aggregator.New(c, logger)
	if err := app.Run(); err != nil {
		logger.Panic(err)
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
