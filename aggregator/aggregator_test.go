package aggregator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/aggregator/repo"
	dbparser "github.com/dezswap/cosmwasm-etl/pkg/db/parser"

	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type repoMock struct {
	mock.Mock

	calledGetParsedTxsWithLimit bool

	syncedHeight    uint64
	syncedHeightErr error

	updatedLpHistory       []schemas.LpHistory
	updatedPairStatsRecent []schemas.PairStatsRecent
	updatedPairStats       []schemas.PairStats30m
	updatedAccountStats    []schemas.AccountStats30m
	updatedAccounts        []string
	latestPairStats        map[uint64]schemas.PairStats30m
	latestPairStatErr      error
	createAccountsErr      error
	deleteDuplicatesErr    error
	closeErr               error
	withinTxErr            error
}

// repoMock stands in for both repositories the aggregator talks to.
var (
	_ repo.Repo               = &repoMock{}
	_ dbparser.ReadRepository = &repoMock{}
)

func (r *repoMock) WithinTx(_ context.Context, fn func(repo.Repo) error) error {
	if r.withinTxErr != nil {
		return r.withinTxErr
	}
	return fn(r)
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func (r *repoMock) LatestTimestamp(_ context.Context, _ string) (float64, error) {
	return 0, nil
}

func (r *repoMock) GetSyncedHeight(_ context.Context) (uint64, error) {
	return r.syncedHeight, r.syncedHeightErr
}

func (r *repoMock) GetPairs(_ context.Context) ([]schemas.Pair, error) {
	return nil, nil
}

func (r *repoMock) GetPoolInfosByHeight(_ context.Context, _ uint64) ([]schemas.PoolInfo, error) {
	return nil, nil
}

func (r *repoMock) GetParsedTxs(_ context.Context, _ uint64) ([]schemas.ParsedTx, error) {
	return nil, nil
}

func (r *repoMock) GetParsedTxsOfPair(_ context.Context, _ uint64, _ string) ([]schemas.ParsedTx, error) {
	return nil, nil
}

func (r *repoMock) TxHeightToSync(_ context.Context, _ int64, _ ...string) (int64, error) {
	return 0, nil
}

func (r *repoMock) HeightOnTimestamp(_ context.Context, _ float64) (uint64, error) {
	args := r.Mock.MethodCalled("HeightOnTimestamp")
	return args.Get(0).(uint64), args.Error(1)
}

func (r *repoMock) LastHeightOfPrice(_ context.Context) (uint64, error) {
	args := r.Mock.MethodCalled("LastHeightOfPrice")
	return args.Get(0).(uint64), args.Error(1)
}

func (r *repoMock) GetParsedTxsInHeightRange(_ context.Context, _ uint64, _ uint64) ([]schemas.ParsedTxWithPrice, error) {
	args := r.Mock.MethodCalled("GetParsedTxsInHeightRange")
	return args.Get(0).([]schemas.ParsedTxWithPrice), args.Error(1)
}

func (r *repoMock) PricesForHeightRange(_ context.Context, _ uint64, _ uint64, _ []string, _ string) (map[uint64][]schemas.Price, error) {
	args := r.Mock.MethodCalled("PricesForHeightRange")
	return args.Get(0).(map[uint64][]schemas.Price), args.Error(1)
}

func (r *repoMock) GetParsedTxsWithPriceOfPair(_ context.Context, _ uint64, _ string, _ float64, _ float64) ([]schemas.ParsedTxWithPrice, error) {
	args := r.Mock.MethodCalled("GetParsedTxsWithPriceOfPair")
	return args.Get(0).([]schemas.ParsedTxWithPrice), args.Error(1)
}

func (r *repoMock) OldestTxTimestamp(_ context.Context) (float64, error) {
	args := r.Mock.MethodCalled("OldestTxTimestamp")
	return args.Get(0).(float64), args.Error(1)
}

func (r *repoMock) LatestTxTimestamp(_ context.Context) (float64, error) {
	return 0, nil
}

func (r *repoMock) PairIds(_ context.Context) ([]uint64, error) {
	args := r.Mock.MethodCalled("PairIds")
	return args.Get(0).([]uint64), args.Error(1)
}

func (r *repoMock) NewPairIds(_ context.Context, _ string, _ float64, _ float64) ([]uint64, error) {
	args := r.Mock.MethodCalled("NewPairIds")
	return args.Get(0).([]uint64), args.Error(1)
}

func (r *repoMock) NewAccounts(_ context.Context, _ float64, _ float64) ([]string, error) {
	args := r.Mock.MethodCalled("NewAccounts")
	return args.Get(0).([]string), args.Error(1)
}

func (r *repoMock) ProviderCount(_ context.Context, _ uint64, _ float64, _ float64) (uint64, error) {
	args := r.Mock.MethodCalled("ProviderCount")
	return args.Get(0).(uint64), args.Error(1)
}

func (r *repoMock) TxCountOfAccount(_ context.Context, _ string, _ uint64, _ float64, _ float64) (uint64, error) {
	args := r.Mock.MethodCalled("TxCountOfAccount")
	return args.Get(0).(uint64), args.Error(1)
}

func (r *repoMock) AssetAmountInPair(_ context.Context, _ uint64, _ float64, _ float64) (string, string, string, error) {
	args := r.Mock.MethodCalled("AssetAmountInPair")
	return args.Get(0).(string), args.Get(1).(string), args.Get(2).(string), args.Error(3)
}

func (r *repoMock) AssetAmountInPairOfAccount(_ context.Context, _ string, _ uint64, _ float64, _ float64) (string, string, string, error) {
	args := r.Mock.MethodCalled("AssetAmountInPairOfAccount")
	return args.Get(0).(string), args.Get(1).(string), args.Get(2).(string), args.Error(3)
}

func (r *repoMock) CommissionAmountInPair(_ context.Context, _ uint64, _ float64, _ float64) (string, string, error) {
	args := r.Mock.MethodCalled("CommissionAmountInPair")
	return args.Get(0).(string), args.Get(1).(string), args.Error(2)
}

// aggregator/repo/repository.go interface
func (r *repoMock) LastHeightOfPairStatsRecent(_ context.Context) (uint64, error) {
	return 0, nil
}

func (r *repoMock) GetParsedTxsWithLimit(_ context.Context, _ uint64, _ int) ([]schemas.ParsedTxWithPrice, error) {
	args := r.Mock.MethodCalled("GetParsedTxsWithLimit")
	if r.calledGetParsedTxsWithLimit {
		return []schemas.ParsedTxWithPrice{}, args.Error(1)
	}

	r.calledGetParsedTxsWithLimit = true
	return args.Get(0).([]schemas.ParsedTxWithPrice), args.Error(1)
}

func (r *repoMock) LastLpHistory(_ context.Context, _ uint64) ([]schemas.LpHistory, error) {
	args := r.Mock.MethodCalled("LastLpHistory")
	return args.Get(0).([]schemas.LpHistory), args.Error(1)
}

func (r *repoMock) UpdatePairStatsRecent(_ context.Context, stats []schemas.PairStatsRecent) error {
	r.updatedPairStatsRecent = stats
	return nil
}

func (r *repoMock) PairStats(_ context.Context, _ float64, _ float64, _ string, _ map[uint64]schemas.PairStats30m) ([]schemas.PairStats30m, error) {
	args := r.Mock.MethodCalled("PairStats")
	return args.Get(0).([]schemas.PairStats30m), args.Error(1)
}

func (r *repoMock) AccountStats(_ context.Context, startTs float64, endTs float64, priceToken string) ([]schemas.AccountStats30m, error) {
	args := r.Mock.MethodCalled("AccountStats", startTs, endTs, priceToken)
	return args.Get(0).([]schemas.AccountStats30m), args.Error(1)
}

func (r *repoMock) LiquiditiesOfPairStats(_ context.Context, _ float64, _ float64, _ string) (map[uint64]schemas.PairStats30m, error) {
	args := r.Mock.MethodCalled("LiquiditiesOfPairStats")
	return args.Get(0).(map[uint64]schemas.PairStats30m), args.Error(1)
}

func (r *repoMock) UpdateLpHistory(_ context.Context, history []schemas.LpHistory) error {
	r.updatedLpHistory = history
	return nil
}

func (r *repoMock) DeletePairStatsRecent(_ context.Context, _ time.Time) error {
	return nil
}

func (r *repoMock) DeleteDuplicates(_ context.Context, _ time.Time) error {
	return r.deleteDuplicatesErr
}

// an unset latestPairStats stands for a chain whose pairs have no stats written yet.
// The timestamp bound is honoured so a test can leave a row of a later window behind.
func (r *repoMock) LatestPairStat(_ context.Context, pairId uint64, before float64) (schemas.PairStats30m, bool, error) {
	if r.latestPairStatErr != nil {
		return schemas.PairStats30m{}, false, r.latestPairStatErr
	}

	stat, ok := r.latestPairStats[pairId]
	if !ok || stat.Timestamp >= before {
		return schemas.PairStats30m{}, false, nil
	}

	return stat, true, nil
}

func (r *repoMock) UpdatePairStats(_ context.Context, stats []schemas.PairStats30m) error {
	r.updatedPairStats = stats
	return nil
}

func (r *repoMock) UpdateAccountStats(_ context.Context, stats []schemas.AccountStats30m) error {
	r.updatedAccountStats = stats
	return nil
}

func (r *repoMock) CreateAccounts(_ context.Context, addresses []string) error {
	r.updatedAccounts = addresses
	return r.createAccountsErr
}

func (r *repoMock) AccountIds(_ context.Context, _ []string) (map[string]uint64, error) {
	args := r.Mock.MethodCalled("AccountIds")
	return args.Get(0).(map[string]uint64), args.Error(1)
}

func (r *repoMock) HoldingPairIds(_ context.Context, _ uint64) ([]uint64, error) {
	args := r.Mock.MethodCalled("HoldingPairIds")
	return args.Get(0).([]uint64), args.Error(1)
}

func (r *repoMock) Accounts(_ context.Context, _ float64) (map[uint64]string, error) {
	args := r.Mock.MethodCalled("Accounts")
	return args.Get(0).(map[uint64]string), args.Error(1)
}

func (r *repoMock) Close() error { return r.closeErr }

type schedulerFunc func(context.Context) error

func (f schedulerFunc) Schedule(ctx context.Context) error { return f(ctx) }

// countingCloser records each Close call and the order calls arrived in, so a test
// can tell one run from two and check the order they ran in. A double that merely
// returns a fixed error cannot: two runs produce the same message as one.
type countingCloser struct {
	name  string
	err   error
	calls int
	order *[]string
}

func (c *countingCloser) Close() error {
	c.calls++
	*c.order = append(*c.order, c.name)

	return c.err
}

// newClosers wires the doubles into the aggregator alongside a shared order log.
func newClosers(order *[]string, errs ...error) ([]namedCloser, []*countingCloser) {
	doubles := make([]*countingCloser, 0, len(errs))
	closers := make([]namedCloser, 0, len(errs))
	for i, err := range errs {
		name := fmt.Sprintf("repo%d", i)
		double := &countingCloser{name: name, err: err, order: order}
		doubles = append(doubles, double)
		closers = append(closers, namedCloser{name: name, closer: double})
	}

	return closers, doubles
}

func TestRunReturnsTaskErrorAndCancelsSibling(t *testing.T) {
	expectedErr := errors.New("task failed")
	started := make(chan struct{})
	canceled := make(chan struct{})
	app := &aggregatorImpl{
		tasks: []scheduler{
			schedulerFunc(func(context.Context) error {
				<-started
				return expectedErr
			}),
			schedulerFunc(func(ctx context.Context) error {
				close(started)
				<-ctx.Done()
				close(canceled)
				return nil
			}),
		},
		logger: logging.Discard,
	}

	err := app.Run(context.Background())

	require.ErrorIs(t, err, expectedErr)
	select {
	case <-canceled:
	default:
		t.Fatal("sibling scheduler was not canceled")
	}
}

func TestRunTreatsExternalCancellationAsGraceful(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	app := &aggregatorImpl{
		tasks: []scheduler{schedulerFunc(func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		})},
		logger: logging.Discard,
	}
	cancel()

	require.NoError(t, app.Run(ctx))
}

// Drivers report a canceled statement with their own error value, not
// context.Canceled, so shutdown must be read from the context.
func TestRunTreatsDriverCancellationDuringCleanupAsGraceful(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	app := &aggregatorImpl{
		cleanDups: true,
		destDb:    &repoMock{deleteDuplicatesErr: errors.New("pq: canceling statement due to user request")},
		logger:    logging.Discard,
	}

	require.NoError(t, app.Run(ctx))
}

func TestRunReportsCleanupFailureWhileRunning(t *testing.T) {
	expectedErr := errors.New("delete duplicates failed")
	app := &aggregatorImpl{
		cleanDups: true,
		destDb:    &repoMock{deleteDuplicatesErr: expectedErr},
		logger:    logging.Discard,
	}

	err := app.Run(context.Background())

	require.ErrorIs(t, err, expectedErr)
	var runtimeErr *RuntimeError
	require.ErrorAs(t, err, &runtimeErr)
	require.Equal(t, "clean_duplicates", runtimeErr.Operation)
}

// One bad close must not hide the others, so every failure survives in the result.
func TestClosePreservesEveryCloseError(t *testing.T) {
	var order []string
	firstErr := errors.New("first close failed")
	secondErr := errors.New("second close failed")
	closers, doubles := newClosers(&order, firstErr, secondErr)
	app := &aggregatorImpl{closers: closers}

	err := app.Close()

	require.ErrorIs(t, err, firstErr)
	require.ErrorIs(t, err, secondErr)
	for _, double := range doubles {
		require.Equal(t, 1, double.calls, "a failing close must not stop the rest")
	}
}

// Repositories are torn down in the opposite order they were opened, so a later
// one never outlives something it was built on.
func TestCloseRunsInReverseAcquisitionOrder(t *testing.T) {
	var order []string
	closers, _ := newClosers(&order, nil, nil, nil)
	app := &aggregatorImpl{closers: closers}

	require.NoError(t, app.Close())
	require.Equal(t, []string{"repo2", "repo1", "repo0"}, order)
}

// Close is idempotent: the second call returns the first result without touching
// the repositories again.
func TestCloseRunsOnlyOnce(t *testing.T) {
	var order []string
	expectedErr := errors.New("close failed")
	closers, doubles := newClosers(&order, expectedErr)
	app := &aggregatorImpl{closers: closers}

	first := app.Close()
	second := app.Close()

	require.ErrorIs(t, first, expectedErr)
	require.Equal(t, first, second, "the memoized error must be returned verbatim")
	require.Equal(t, 1, doubles[0].calls, "the second Close must not re-run the closers")
	require.Len(t, order, 1)
}

func TestErrorFieldsExposeRuntimeContextAndCauseChain(t *testing.T) {
	cause := context.DeadlineExceeded
	err := &RuntimeError{
		Operation:   OpExecute,
		Task:        "price",
		WindowStart: time.Unix(10, 0),
		WindowEnd:   time.Unix(20, 0),
		Err:         cause,
	}

	fields := ErrorFields(err)

	require.Equal(t, "execute", fields["operation"])
	require.Equal(t, "price", fields["task"])
	require.Equal(t, true, fields["is_timeout"])
	require.Equal(t, "context.deadlineExceededError", fields["error_type"])
	require.Len(t, fields["error_chain"], 2)
	require.Equal(t, EventTaskFailed, ErrorEvent(err))
}

// closeFailure gives rootCause an unambiguous innermost type to report, so the
// assertion does not depend on the stdlib's internal error type names.
type closeFailure struct{ name string }

func (e closeFailure) Error() string { return "close " + e.name + " failed" }

// A cleanup failure is a RuntimeError wrapping errors.Join, which is the shape
// cmd/aggregator builds. rootCause has to descend through the join even though it
// sits below the top level, otherwise error_type degrades to *errors.joinError and
// says nothing about what actually broke.
func TestErrorFieldsReportsRootCauseThroughJoinedCleanupFailure(t *testing.T) {
	first := closeFailure{name: "source_repository"}
	joined := errors.Join(
		fmt.Errorf("close source_repository: %w", first),
		fmt.Errorf("close destination_repository: %w", closeFailure{name: "destination_repository"}),
	)
	err := &RuntimeError{Operation: OpClose, Err: joined}

	fields := ErrorFields(err)

	require.Equal(t, "aggregator.closeFailure", fields["error_type"])
	require.Equal(t, EventCleanupFailed, ErrorEvent(err))
	require.Equal(t, OpClose, fields["operation"])
}

// The joined branches are logged once each, on a single line: errors.Join renders
// its children newline-separated, which would both duplicate every branch and break
// the structured log field across lines.
func TestErrorFieldsFlattensJoinedCauseWithoutDuplicates(t *testing.T) {
	joined := errors.Join(closeFailure{name: "a"}, closeFailure{name: "b"})
	err := &RuntimeError{Operation: OpClose, Err: joined}

	fields := ErrorFields(err)

	require.NotContains(t, fields["error"], "\n", "log fields must stay on one line")
	require.Equal(t, []string{
		"aggregator close: close a failed; close b failed",
		"close a failed",
		"close b failed",
	}, fields["error_chain"])
}

// The join can also sit at the very top, which is the shape cmd/aggregator builds
// when a run failure and a cleanup failure arrive together. RuntimeError.Error()
// never sees that join, so the field itself has to be flattened.
func TestErrorFieldsFlattensTopLevelJoin(t *testing.T) {
	err := errors.Join(
		&RuntimeError{Operation: OpExecute, Task: "price", Err: closeFailure{name: "a"}},
		&RuntimeError{Operation: OpClose, Err: closeFailure{name: "b"}},
	)

	fields := ErrorFields(err)

	require.NotContains(t, fields["error"], "\n", "log fields must stay on one line")
	require.Equal(t,
		"aggregator execute task=price: close a failed; aggregator close: close b failed",
		fields["error"])
}

func TestErrorEventClassifiesEveryOperation(t *testing.T) {
	cause := context.DeadlineExceeded
	expected := map[string]string{
		OpMarshalConfig:       EventInitializationFailed,
		OpOpenSourceRepo:      EventInitializationFailed,
		OpOpenDestinationRepo: EventInitializationFailed,
		OpOpenPriceRepo:       EventInitializationFailed,
		OpOpenRouterRepo:      EventInitializationFailed,
		OpInitializeTasks:     EventInitializationFailed,
		OpCleanDuplicates:     EventInitializationFailed,
		OpInitializeSchedule:  EventTaskFailed,
		OpExecute:             EventTaskFailed,
		OpClose:               EventCleanupFailed,
	}

	for operation, event := range expected {
		require.Equal(t, event, ErrorEvent(&RuntimeError{Operation: operation, Err: cause}), operation)
	}
	require.Equal(t, EventTaskFailed, ErrorEvent(cause))
}
