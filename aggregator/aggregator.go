package aggregator

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dezswap/cosmwasm-etl/aggregator/repo"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/db/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/sirupsen/logrus"
)

type Aggregator interface {
	Run() error
}

type aggregatorImpl struct {
	chainId    string
	startTs    time.Time
	cleanDups  bool
	srcDbConn  parser.ReadRepository
	destDbConn repo.Repo
	tasks      []scheduler

	logger logging.Logger
}

var (
	_       Aggregator = &aggregatorImpl{}
	errChan chan error
)

func New(c configs.Config, logger logging.Logger) Aggregator {
	repo.Logger = logger

	if c.Log.Level == logrus.DebugLevel {
		a, err := json.Marshal(c.Redacted().Aggregator)
		if err != nil {
			panic(err)
		}
		logger.Debug(string(a))
	}

	srcRepo := parser.NewReadRepo(c.Aggregator.ChainId, c.Aggregator.SrcDb)
	destRepo := repo.New(c.Aggregator.ChainId, c.Aggregator.DestDb)

	taskSchedulers, err := initTaskSchedulers(c.Aggregator, srcRepo, destRepo, logger)
	if err != nil {
		panic(err)
	}

	return &aggregatorImpl{
		chainId:    c.Aggregator.ChainId,
		startTs:    c.Aggregator.StartTs,
		cleanDups:  c.Aggregator.CleanDups,
		srcDbConn:  srcRepo,
		destDbConn: destRepo,
		tasks:      taskSchedulers,
		logger:     logger,
	}
}

func initTaskSchedulers(config configs.AggregatorConfig, srcRepo parser.ReadRepository, destRepo repo.Repo, logger logging.Logger) ([]scheduler, error) {
	lht := newLpHistoryTask(config, srcRepo, destRepo, logger)
	pt, err := newPriceTask(config, destRepo, logger, []task{lht})
	if err != nil {
		return nil, err
	}

	return []scheduler{
		newIntervalScheduler(newRouterTask(config, logger), logger),
		newIntervalScheduler(lht, logger),
		newIntervalScheduler(pt, logger),
		newIntervalScheduler(newPairStatsRecentUpdateTask(config, srcRepo, destRepo, logger, []task{pt}), logger),
		newPredeterminedTimeScheduler(newPairStatsUpdateTask(config, srcRepo, destRepo, logger, []task{pt}), config.StartTs, logger),
		newPredeterminedTimeScheduler(newAccountStatsUpdateTask(config, srcRepo, destRepo, logger), config.StartTs, logger),
	}, nil
}

func (a aggregatorImpl) Run() error {
	a.logger.Info("Aggregator has been started.")

	defer a.srcDbConn.Close()
	defer a.destDbConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		a.logger.Infof("Signal has been caught: %s", sig.String())
		cancel()
	}()

	errChan = make(chan error)

	go func() {
		err := <-errChan
		a.logger.Errorf("Error has been caught: %s", err.Error())
		cancel()
	}()

	a.runTasks(ctx)

	a.logger.Info("cosmwasm-etc aggregator has been stopped.")

	return nil
}

func (a aggregatorImpl) runTasks(ctx context.Context) {
	wg := sync.WaitGroup{}

	if a.cleanDups {
		if err := a.destDbConn.DeleteDuplicates(a.startTs); err != nil {
			errChan <- err
			return
		}
		a.logger.Infof("Stats data since %s has been deleted for new update.", a.startTs.String())
	}

	for _, t := range a.tasks {
		wg.Add(1)
		go func(tk scheduler) {
			defer wg.Done()
			if err := tk.Schedule(ctx); err != nil {
				errChan <- err
			}
		}(t)
	}

	wg.Wait()
}
