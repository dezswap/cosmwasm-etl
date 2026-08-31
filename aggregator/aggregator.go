package aggregator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/dezswap/cosmwasm-etl/aggregator/repo"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/db/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/price"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/router"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Aggregator interface {
	Run(context.Context) error
	Close() error
}

type aggregatorImpl struct {
	startTs   time.Time
	cleanDups bool
	destDb    repo.Repo
	tasks     []scheduler
	closers   []namedCloser
	closeOnce sync.Once
	closeErr  error

	logger logging.Logger
}

var _ Aggregator = &aggregatorImpl{}

type namedCloser struct {
	name   string
	closer io.Closer
}

// New owns every repository it opens; a failure partway through closes the ones
// already opened before returning.
func New(ctx context.Context, c configs.Config, logger logging.Logger) (Aggregator, error) {
	if c.Log.ParsedLevel() == logrus.DebugLevel {
		a, err := json.Marshal(c.Redacted().Aggregator)
		if err != nil {
			return nil, &RuntimeError{Operation: OpMarshalConfig, Err: err}
		}
		logger.Debug(string(a))
	}

	srcRepo, err := parser.NewReadRepo(c.Aggregator.ChainId, c.Aggregator.SrcDb)
	if err != nil {
		return nil, &RuntimeError{Operation: OpOpenSourceRepo, Err: err}
	}
	closers := []namedCloser{{name: "source_repository", closer: srcRepo}}
	closeOnError := func(initErr error) (Aggregator, error) {
		return nil, errors.Join(initErr, closeAll(closers))
	}

	destRepo, err := repo.New(c.Aggregator.ChainId, c.Aggregator.DestDb)
	if err != nil {
		return closeOnError(&RuntimeError{Operation: OpOpenDestinationRepo, Err: err})
	}
	closers = append(closers, namedCloser{name: "destination_repository", closer: destRepo})

	priceRepo, err := price.NewRepo(c.Aggregator.ChainId, c.Aggregator.SrcDb)
	if err != nil {
		return closeOnError(&RuntimeError{Operation: OpOpenPriceRepo, Err: err})
	}
	closers = append(closers, namedCloser{name: "price_repository", closer: priceRepo})

	routerRepo, err := router.NewSrcRepo(c.Aggregator.ChainId, c.Aggregator.DestDb)
	if err != nil {
		return closeOnError(&RuntimeError{Operation: OpOpenRouterRepo, Err: err})
	}
	closers = append(closers, namedCloser{name: "router_repository", closer: routerRepo})

	taskSchedulers, err := initTaskSchedulers(ctx, c.Aggregator, srcRepo, destRepo, priceRepo, routerRepo, logger)
	if err != nil {
		return closeOnError(&RuntimeError{Operation: OpInitializeTasks, Err: err})
	}

	return &aggregatorImpl{
		startTs:   c.Aggregator.StartTs,
		cleanDups: c.Aggregator.CleanDups,
		destDb:    destRepo,
		tasks:     taskSchedulers,
		closers:   closers,
		logger:    logger,
	}, nil
}

// initTaskSchedulers wires tasks in processing order: liquidity feeds price, and each
// statistics task waits on whatever its own queries read.
func initTaskSchedulers(ctx context.Context, config configs.AggregatorConfig, srcRepo parser.ReadRepository, destRepo repo.Repo, priceRepo price.SrcRepo, routerRepo router.SrcRepo, logger logging.Logger) ([]scheduler, error) {
	lht := newLpHistoryTask(config, srcRepo, destRepo, logger)
	pt, err := newPriceTask(ctx, config, destRepo, priceRepo, logger, []task{lht})
	if err != nil {
		return nil, err
	}

	return []scheduler{
		newIntervalScheduler(newRouterTask(config, routerRepo, logger), logger),
		newIntervalScheduler(lht, logger),
		newIntervalScheduler(pt, logger),
		newIntervalScheduler(newPairStatsRecentUpdateTask(config, srcRepo, destRepo, logger, []task{lht, pt}), logger),
		newPredeterminedTimeScheduler(newPairStatsUpdateTask(config, srcRepo, destRepo, logger, []task{lht, pt}), config.StartTs, logger),
		newPredeterminedTimeScheduler(newAccountStatsUpdateTask(config, srcRepo, destRepo, logger, []task{pt}), config.StartTs, logger),
	}, nil
}

func (a *aggregatorImpl) Run(ctx context.Context) error {
	a.logger.Info("Aggregator has been started.")

	if err := a.runTasks(ctx); err != nil {
		return err
	}

	a.logger.Info("cosmwasm-etc aggregator has been stopped.")

	return nil
}

// runTasks returns the first scheduler failure; the errgroup context stops the rest.
func (a *aggregatorImpl) runTasks(ctx context.Context) error {
	if a.cleanDups {
		if err := a.destDb.DeleteDuplicates(ctx, a.startTs); err != nil {
			if isShutdown(ctx) {
				return nil
			}
			return &RuntimeError{Operation: OpCleanDuplicates, Err: err}
		}
		a.logger.Infof("Stats data since %s has been deleted for new update.", a.startTs.String())
	}

	group, groupCtx := errgroup.WithContext(ctx)
	for _, scheduler := range a.tasks {
		group.Go(func() error {
			return scheduler.Schedule(groupCtx)
		})
	}
	return group.Wait()
}

// Close releases all owned repositories once and preserves every close error.
func (a *aggregatorImpl) Close() error {
	a.closeOnce.Do(func() {
		a.closeErr = closeAll(a.closers)
	})
	return a.closeErr
}

// closeAll closes in reverse acquisition order, joining failures so one bad close
// does not skip the rest.
func closeAll(closers []namedCloser) error {
	var result error
	for i := len(closers) - 1; i >= 0; i-- {
		if err := closers[i].closer.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("close %s: %w", closers[i].name, err))
		}
	}
	return result
}
