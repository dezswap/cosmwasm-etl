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
	tasks      []task

	logger logging.Logger
}

var (
	_       Aggregator = &aggregatorImpl{}
	errChan chan error
)

func New(c configs.Config, logger logging.Logger) Aggregator {
	repo.Logger = logger

	if c.Log.Level == logrus.DebugLevel {
		a, err := json.Marshal(c.Aggregator)
		if err != nil {
			panic(err)
		}
		logger.Debug(string(a))
	}

	srcRepo := parser.NewReadRepo(c.Aggregator.ChainId, c.Aggregator.SrcDb)
	destRepo := repo.New(c.Aggregator.ChainId, c.Aggregator.DestDb)

	// init tasks
	rt, err := NewRouterTask(c.Aggregator, logger)
	if err != nil {
		panic(err)
	}
	lht := NewLpHistoryTask(c.Aggregator, srcRepo, destRepo, logger)
	pt, err := NewPriceTask(c.Aggregator, destRepo, logger, []task{lht})
	if err != nil {
		panic(err)
	}
	tasks := []task{
		rt, lht, pt,
		NewPairStatsIn24hUpdateTask(c.Aggregator, srcRepo, destRepo, logger, []task{pt}),
		NewPairStatsUpdateTask(c.Aggregator, srcRepo, destRepo, logger, []task{pt}),
	}

	return &aggregatorImpl{
		chainId:    c.Aggregator.ChainId,
		startTs:    c.Aggregator.StartTs,
		cleanDups:  c.Aggregator.CleanDups,
		srcDbConn:  srcRepo,
		destDbConn: destRepo,
		tasks:      tasks,
		logger:     logger,
	}
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
		go func(tk task) {
			defer wg.Done()
			if err := tk.Schedule(ctx, a.startTs); err != nil {
				errChan <- err
			}
		}(t)
	}

	wg.Wait()
}
