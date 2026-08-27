package aggregator

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type counterTask struct {
	counter atomic.Int64
	err     error
	invoked chan struct{}
}

func (t *counterTask) Name() string { return "counter" }

func (t *counterTask) Execute(_ context.Context, _ time.Time, _ time.Time) error {
	t.counter.Add(1)
	if t.invoked != nil {
		select {
		case t.invoked <- struct{}{}:
		default:
		}
	}

	return t.err
}
func (t *counterTask) LastProcessedHeight() uint64 {
	return 0
}
func (t *counterTask) StartTimestamp(_ context.Context, startTs time.Time) (time.Time, error) {
	return startTs, nil
}

func TestIntervalSchedule(t *testing.T) {
	assert := assert.New(t)

	task := counterTask{invoked: make(chan struct{}, 10)}
	scheduler := intervalScheduler{
		task:     &task,
		interval: 5 * time.Millisecond,
		logger:   logging.Discard,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func(tk intervalScheduler) { done <- tk.Schedule(ctx) }(scheduler)
	for i := 0; i < 3; i++ {
		select {
		case <-task.invoked:
		case <-time.After(time.Second):
			t.Fatal("scheduler did not invoke task")
		}
	}
	cancel()
	assert.NoError(<-done)
}

func TestIntervalScheduleReturnsTaskError(t *testing.T) {
	assert := assert.New(t)

	expectedErr := errors.New("task failed")
	task := counterTask{err: expectedErr}
	scheduler := intervalScheduler{
		task:     &task,
		interval: time.Hour,
		logger:   logging.Discard,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- scheduler.Schedule(ctx)
	}()

	select {
	case err := <-done:
		assert.ErrorIs(err, expectedErr)
	case <-time.After(time.Second):
		assert.Fail("scheduler did not return task error")
	}
}

func TestPredeterminedTimeScheduleStopsWhileWaiting(t *testing.T) {
	task := counterTask{invoked: make(chan struct{}, 1)}
	scheduler := predeterminedTimeScheduler{
		predeterminedTimeTask: &task,
		interval:              time.Hour,
		startTs:               time.Now(),
		logger:                logging.Discard,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func(tk predeterminedTimeScheduler) { done <- tk.Schedule(ctx) }(scheduler)
	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop after cancellation")
	}
}

func TestPredeterminedTimeScheduleReturnsCatchUpTaskError(t *testing.T) {
	assert := assert.New(t)

	expectedErr := errors.New("task failed")
	task := counterTask{err: expectedErr}
	scheduler := predeterminedTimeScheduler{
		predeterminedTimeTask: &task,
		interval:              time.Hour,
		startTs:               time.Now().Add(-2 * time.Hour),
		logger:                logging.Discard,
	}

	err := scheduler.Schedule(context.Background())

	assert.ErrorIs(err, expectedErr)
	assert.Equal(int64(1), task.counter.Load())
}

// shutdownTask reproduces the race the schedulers guard against: shutdown begins
// while a query is in flight, and the driver reports its own error value rather
// than context.Canceled, so only the context state reveals what happened.
type shutdownTask struct {
	cancel     context.CancelFunc
	execErr    error
	startTsErr error
	executed   atomic.Int64
}

func (*shutdownTask) Name() string                { return "shutdown" }
func (*shutdownTask) LastProcessedHeight() uint64 { return 0 }

func (t *shutdownTask) stop() {
	if t.cancel != nil {
		t.cancel()
	}
}

func (t *shutdownTask) StartTimestamp(_ context.Context, startTs time.Time) (time.Time, error) {
	if t.startTsErr != nil {
		t.stop()
		return time.Time{}, t.startTsErr
	}

	return startTs, nil
}

func (t *shutdownTask) Execute(_ context.Context, _ time.Time, _ time.Time) error {
	t.executed.Add(1)
	t.stop()

	return t.execErr
}

// driverCancelErr is what pq reports for a statement killed by shutdown; it does
// not satisfy errors.Is(err, context.Canceled).
func driverCancelErr() error {
	return errors.New("pq: canceling statement due to user request")
}

func TestIntervalScheduleTreatsTaskErrorDuringShutdownAsGraceful(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	task := &shutdownTask{cancel: cancel, execErr: driverCancelErr()}
	scheduler := intervalScheduler{task: task, interval: time.Hour, logger: logging.Discard}

	require.NoError(t, scheduler.Schedule(ctx))
	require.Equal(t, int64(1), task.executed.Load())
}

func TestPredeterminedTimeScheduleTreatsStartTimestampErrorDuringShutdownAsGraceful(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	task := &shutdownTask{cancel: cancel, startTsErr: driverCancelErr()}
	scheduler := predeterminedTimeScheduler{
		predeterminedTimeTask: task,
		interval:              time.Hour,
		startTs:               time.Now(),
		logger:                logging.Discard,
	}

	require.NoError(t, scheduler.Schedule(ctx))
	require.Zero(t, task.executed.Load())
}

func TestPredeterminedTimeScheduleTreatsCatchUpErrorDuringShutdownAsGraceful(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	task := &shutdownTask{cancel: cancel, execErr: driverCancelErr()}
	scheduler := predeterminedTimeScheduler{
		predeterminedTimeTask: task,
		interval:              time.Hour,
		startTs:               time.Now().Add(-3 * time.Hour),
		logger:                logging.Discard,
	}

	require.NoError(t, scheduler.Schedule(ctx))
	require.Equal(t, int64(1), task.executed.Load())
}

// The catch-up loop must re-check the context between windows, not only when a
// window fails. Three windows are pending, so a second Execute means it did not.
func TestPredeterminedTimeScheduleStopsCatchUpBetweenWindows(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	task := &shutdownTask{cancel: cancel}
	scheduler := predeterminedTimeScheduler{
		predeterminedTimeTask: task,
		interval:              time.Hour,
		startTs:               time.Now().Add(-3 * time.Hour),
		logger:                logging.Discard,
	}

	require.NoError(t, scheduler.Schedule(ctx))
	require.Equal(t, int64(1), task.executed.Load())
}

// startTs on the current interval boundary leaves nothing to catch up, so the first
// Execute happens in the steady-state loop.
func TestPredeterminedTimeScheduleTreatsSteadyStateErrorDuringShutdownAsGraceful(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	task := &shutdownTask{cancel: cancel, execErr: driverCancelErr()}
	scheduler := predeterminedTimeScheduler{
		predeterminedTimeTask: task,
		interval:              20 * time.Millisecond,
		startTs:               time.Now(),
		logger:                logging.Discard,
	}

	require.NoError(t, scheduler.Schedule(ctx))
	require.Equal(t, int64(1), task.executed.Load())
}

func TestIntervalScheduleReportsTaskAndWindowOnFailure(t *testing.T) {
	expectedErr := errors.New("task failed")
	task := counterTask{err: expectedErr}
	scheduler := intervalScheduler{task: &task, interval: time.Hour, logger: logging.Discard}

	before := time.Now()
	err := scheduler.Schedule(context.Background())
	after := time.Now()

	var runtimeErr *RuntimeError
	require.ErrorAs(t, err, &runtimeErr)
	require.Equal(t, OpExecute, runtimeErr.Operation)
	require.Equal(t, task.Name(), runtimeErr.Task)
	require.True(t, runtimeErr.WindowStart.IsZero(), "interval scheduler has no window start")
	require.False(t, runtimeErr.WindowEnd.Before(before))
	require.False(t, runtimeErr.WindowEnd.After(after))
	require.Equal(t, EventTaskFailed, ErrorEvent(err))
	require.Equal(t, task.Name(), ErrorFields(err)["task"])
}

func TestPredeterminedTimeScheduleReportsTaskAndWindowOnFailure(t *testing.T) {
	expectedErr := errors.New("task failed")
	task := counterTask{err: expectedErr}
	startTs := time.Now().Add(-2 * time.Hour)
	scheduler := predeterminedTimeScheduler{
		predeterminedTimeTask: &task,
		interval:              time.Hour,
		startTs:               startTs,
		logger:                logging.Discard,
	}

	err := scheduler.Schedule(context.Background())

	expectedStart, expectedEnd := timeframe(startTs, time.Hour)
	var runtimeErr *RuntimeError
	require.ErrorAs(t, err, &runtimeErr)
	require.Equal(t, OpExecute, runtimeErr.Operation)
	require.Equal(t, task.Name(), runtimeErr.Task)
	require.Equal(t, expectedStart, runtimeErr.WindowStart)
	require.Equal(t, expectedEnd, runtimeErr.WindowEnd)
}

func TestPredeterminedTimeScheduleReportsInitializeScheduleOnStartTimestampFailure(t *testing.T) {
	expectedErr := errors.New("start timestamp failed")
	task := &shutdownTask{startTsErr: expectedErr}
	scheduler := predeterminedTimeScheduler{
		predeterminedTimeTask: task,
		interval:              time.Hour,
		startTs:               time.Now(),
		logger:                logging.Discard,
	}

	err := scheduler.Schedule(context.Background())

	var runtimeErr *RuntimeError
	require.ErrorAs(t, err, &runtimeErr)
	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, OpInitializeSchedule, runtimeErr.Operation)
	require.Equal(t, task.Name(), runtimeErr.Task)
	require.Equal(t, EventTaskFailed, ErrorEvent(err))
}

func TestTimeframe_0_30(t *testing.T) {
	assert := assert.New(t)

	expectedStart := time.Date(2022, 10, 25, 7, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2022, 10, 25, 7, 30, 0, 0, time.UTC)

	ts := time.Unix(1666681350, 0).UTC() // 2022-10-25 07:02:30 UTC
	actualStart, actualEnd := timeframe(ts, 30*time.Minute)

	assert.Equal(expectedStart, actualStart)
	assert.Equal(expectedEnd, actualEnd)
}

func TestTimeframe_30_0(t *testing.T) {
	assert := assert.New(t)

	expectedStart := time.Date(2022, 10, 25, 7, 30, 0, 0, time.UTC)
	expectedEnd := time.Date(2022, 10, 25, 8, 0, 0, 0, time.UTC)

	ts := time.Unix(1666684022, 0).UTC() // 2022-10-25 07:47:02 UTC
	actualStart, actualEnd := timeframe(ts, 30*time.Minute)

	assert.Equal(expectedStart, actualStart)
	assert.Equal(expectedEnd, actualEnd)
}

func TestTimeframe_InclusiveStart(t *testing.T) {
	assert := assert.New(t)

	expectedStart := time.Date(2022, 10, 25, 7, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2022, 10, 25, 7, 30, 0, 0, time.UTC)

	ts := time.Unix(1666681200, 0).UTC() // 2022-10-25 07:00:00 UTC
	actualStart, actualEnd := timeframe(ts, 30*time.Minute)

	assert.Equal(expectedStart, actualStart)
	assert.Equal(expectedEnd, actualEnd)
}

func TestTimeframe_ExclusiveEnd(t *testing.T) {
	assert := assert.New(t)

	expectedStart := time.Date(2022, 10, 25, 7, 30, 0, 0, time.UTC)
	expectedEnd := time.Date(2022, 10, 25, 8, 0, 0, 0, time.UTC)

	ts := time.Unix(1666683000, 0).UTC() // 2022-10-25 07:30:00 UTC
	actualStart, actualEnd := timeframe(ts, 30*time.Minute)

	assert.Equal(expectedStart, actualStart)
	assert.Equal(expectedEnd, actualEnd)
}
