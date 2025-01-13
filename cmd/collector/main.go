package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dezswap/cosmwasm-etl/collector"
	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	coldata "github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/grpc"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

const (
	HTTP_REDIRECT_LIMIT = 15
	app                 = "collector"
)

func main() {
	c := configs.New()
	nodeConf := c.Collector.NodeConfig
	logger := logging.New("main", c.Log)

	if c.Sentry.DSN != "" {
		sentryEnv := fmt.Sprintf("%s-%s", c.Collector.ChainId, app)
		logging.ConfigureReporter(logger, c.Sentry.DSN, sentryEnv, map[string]string{
			"x-chain_id": c.Collector.ChainId,
			"x-app":      "collector",
			"x-env":      c.Log.Environment,
		})
	}
	if nodeConf.GrpcConfig.BackoffDelay.Duration == time.Duration(0) {
		panic("invalid back off delay")
	}

	defer catch(logger)

	grpc.SetLogConfig(c.Log)
	serviceDesc := grpc.GetServiceDesc("collector", nodeConf.GrpcConfig)
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:      10,               // Maximum idle connections to keep open
			IdleConnTimeout:   30 * time.Second, // Time to keep idle connections open
			DisableKeepAlives: false,            // Use HTTP Keep-Alive
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= HTTP_REDIRECT_LIMIT { // Increase the maximum number of redirects allowed here
				return fmt.Errorf("stopped after %d redirects", HTTP_REDIRECT_LIMIT)
			}
			return nil
		},
	}

	failoverClient := datastore.NewLcdClient(nodeConf.FailoverLcdHost, httpClient)
	col, err := coldata.New(c, serviceDesc, failoverClient)
	if err != nil {
		logger.Panic(err)
	}

	err = collector.DoCollect(col, logger)
	if err != nil {
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
