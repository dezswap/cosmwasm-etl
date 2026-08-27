package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dezswap/cosmwasm-etl/aggregator"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

const (
	app = "aggregator"
)

var version = "dev" // overridden via -ldflags "-X main.version=v1.2.3"

func main() {
	c := configs.New()
	logger := logging.New("aggregator", c.Log)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if c.Sentry.DSN != "" {
		sentryEnv := fmt.Sprintf("%s-%s", c.Aggregator.ChainId, app)
		logging.ConfigureReporter(logger, c.Sentry.DSN, sentryEnv, map[string]string{
			"x-chain_id": c.Aggregator.ChainId,
			"x-app":      "aggregator",
			"x-env":      c.Log.Environment,
		})
	}
	logger.WithField("version", version).Info("starting aggregator")

	if err := run(ctx, c, logger); err != nil {
		reportError(logger, c, err, "aggregator stopped with error")
		os.Exit(1)
	}
}

// run keeps execution and cleanup errors together so the root log can explain both.
// A cleanup failure on its own is logged but does not fail the process: Close runs
// only after Run returned, so on a clean shutdown a bad close is worth reporting
// while a non-zero exit would read as a crash to an orchestrator.
func run(ctx context.Context, c configs.Config, logger logging.Logger) error {
	app, err := aggregator.New(ctx, c, logger)
	if err != nil {
		if ctx.Err() != nil {
			// shutdown arrived before startup finished; nothing has run yet
			return nil
		}
		return err
	}

	runErr := app.Run(ctx)
	closeErr := app.Close()
	if closeErr == nil {
		return runErr
	}

	closeErr = &aggregator.RuntimeError{Operation: aggregator.OpClose, Err: closeErr}
	if runErr == nil {
		reportError(logger, c, closeErr, "aggregator shut down cleanly but failed to release its repositories")
		return nil
	}
	return errors.Join(runErr, closeErr)
}

func reportError(logger logging.Logger, c configs.Config, err error, msg string) {
	fields := aggregator.ErrorFields(err)
	fields["chain_id"] = c.Aggregator.ChainId
	fields["event"] = aggregator.ErrorEvent(err)
	logger.WithFields(fields).Error(msg)
}
