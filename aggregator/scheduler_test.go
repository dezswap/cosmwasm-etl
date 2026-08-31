package aggregator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
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

// parentBehindTask reports ErrParentBehind for its first failures rounds, then
// succeeds, recording the window it was handed each time.
type parentBehindTask struct {
	mutex    sync.Mutex
	windows  [][2]time.Time
	failures int
	invoked  chan struct{}
}

func (*parentBehindTask) Name() string                { return "parent_behind" }
func (*parentBehindTask) LastProcessedHeight() uint64 { return 0 }

func (t *parentBehindTask) StartTimestamp(_ context.Context, startTs time.Time) (time.Time, error) {
	return startTs, nil
}

func (t *parentBehindTask) Execute(_ context.Context, start, end time.Time) error {
	t.mutex.Lock()
	t.windows = append(t.windows, [2]time.Time{start, end})
	behind := t.failures > 0
	if behind {
		t.failures--
	}
	t.mutex.Unlock()

	if t.invoked != nil {
		select {
		case t.invoked <- struct{}{}:
		default:
		}
	}
	if behind {
		return fmt.Errorf("price: %w", ErrParentBehind)
	}

	return nil
}

func (t *parentBehindTask) recordedWindows() [][2]time.Time {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return append([][2]time.Time(nil), t.windows...)
}

// An upstream task that has not caught up is a normal cold-start state, not a failure:
// returning it would take every other task down through the errgroup.
func TestIntervalScheduleSkipsRoundWhenParentIsBehind(t *testing.T) {
	task := &parentBehindTask{failures: 1000, invoked: make(chan struct{}, 10)}
	scheduler := intervalScheduler{task: task, interval: 5 * time.Millisecond, logger: logging.Discard}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func(s intervalScheduler) { done <- s.Schedule(ctx) }(scheduler)

	for range 3 {
		select {
		case <-task.invoked:
		case <-time.After(time.Second):
			t.Fatal("scheduler stopped scheduling after the parent fell behind")
		}
	}
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop after cancellation")
	}
}

// A fixed window has nothing that recomputes it later, so falling behind must retry
// the same window. Advancing past it would leave a permanent hole in the stats.
func TestPredeterminedTimeScheduleRetriesSameWindowWhenParentIsBehind(t *testing.T) {
	task := &parentBehindTask{failures: 1, invoked: make(chan struct{}, 10)}
	scheduler := predeterminedTimeScheduler{
		predeterminedTimeTask: task,
		interval:              time.Hour,
		startTs:               time.Now().Add(-3 * time.Hour),
		retryWait:             time.Millisecond,
		logger:                logging.Discard,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func(s predeterminedTimeScheduler) { done <- s.Schedule(ctx) }(scheduler)

	for range 3 {
		select {
		case <-task.invoked:
		case <-time.After(time.Second):
			t.Fatal("scheduler did not keep working through the retry")
		}
	}
	cancel()
	require.NoError(t, <-done)

	windows := task.recordedWindows()
	require.GreaterOrEqual(t, len(windows), 3)
	require.Equal(t, windows[0], windows[1], "the window the parent was behind on must be retried, not skipped")
	require.NotEqual(t, windows[1], windows[2], "the window must advance once it succeeded")
}

// The Sentry hook reports warn and above, so a cold start's rounds stay at info. The
// threshold is wall clock because the round rate follows the configurable taskWaitTimeout.
func TestParentBehindTrackerEscalatesOnlyOnceTheStreakOutlastsTheThreshold(t *testing.T) {
	behind := errors.New("parent is behind")
	logger, hook := test.NewNullLogger()

	tracker := parentBehindTracker{}
	tracker.report(logger, "task(window) skipped", behind)
	require.Equal(t, logrus.InfoLevel, hook.LastEntry().Level, "a streak that just started is a routine cold start")

	// many rounds, but not long enough yet: a short taskWaitTimeout must not page
	for range 100 {
		tracker.report(logger, "task(window) skipped", behind)
	}
	require.Equal(t, logrus.InfoLevel, hook.LastEntry().Level, "the round count alone must not escalate")
	require.Equal(t, 101, tracker.streak)

	tracker.since = time.Now().Add(-parentBehindEscalation)
	tracker.report(logger, "task(window) skipped", behind)
	require.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level, "a streak that outlasts the threshold must surface on its own")
	require.Contains(t, hook.LastEntry().Message, "102 in a row")
}

// Past the threshold every round would page, so the escalation repeats per interval.
func TestParentBehindTrackerRepeatsEscalationOnlyOncePerInterval(t *testing.T) {
	behind := errors.New("parent is behind")
	logger, hook := test.NewNullLogger()

	tracker := parentBehindTracker{since: time.Now().Add(-parentBehindEscalation)}
	tracker.report(logger, "task(window) skipped", behind)
	require.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)

	for range 10 {
		tracker.report(logger, "task(window) skipped", behind)
		require.Equal(t, logrus.InfoLevel, hook.LastEntry().Level, "the rounds after an escalation must not page again")
	}

	tracker.reportedAt = time.Now().Add(-parentBehindEscalation)
	tracker.report(logger, "task(window) skipped", behind)
	require.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level, "a streak still stuck an interval later has to surface again")
}

// Restamping the streak start each round would keep the report at info forever.
func TestPredeterminedTimeExecuteKeepsTheStreakStartUntilTheWindowLands(t *testing.T) {
	logger, hook := test.NewNullLogger()
	startedAt := time.Now().Add(-parentBehindEscalation)
	scheduler := predeterminedTimeScheduler{
		predeterminedTimeTask: &parentBehindTask{failures: 1},
		retryWait:             time.Millisecond,
		parentBehind:          parentBehindTracker{streak: 7, since: startedAt},
		logger:                logger,
	}

	retry, err := scheduler.execute(context.Background(), time.Time{}, time.Time{})
	require.NoError(t, err)
	require.True(t, retry)
	require.Equal(t, startedAt, scheduler.parentBehind.since, "the streak start must not be restamped")
	require.Equal(t, 8, scheduler.parentBehind.streak)
	require.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
}

// A window that lands clears the streak; otherwise skips accumulate over a long run
// and escalate on an ordinary hiccup.
func TestPredeterminedTimeExecuteClearsStreakOnceTheWindowLands(t *testing.T) {
	scheduler := predeterminedTimeScheduler{
		predeterminedTimeTask: &parentBehindTask{failures: 1},
		retryWait:             time.Millisecond,
		logger:                logging.Discard,
	}

	retry, err := scheduler.execute(context.Background(), time.Time{}, time.Time{})
	require.NoError(t, err)
	require.True(t, retry)
	require.Equal(t, 1, scheduler.parentBehind.streak)
	require.False(t, scheduler.parentBehind.since.IsZero(), "the first round given up starts the streak")

	retry, err = scheduler.execute(context.Background(), time.Time{}, time.Time{})
	require.NoError(t, err)
	require.False(t, retry)
	require.Equal(t, parentBehindTracker{}, scheduler.parentBehind, "a landed window clears the whole streak")
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
