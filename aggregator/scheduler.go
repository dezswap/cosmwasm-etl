package aggregator

import (
	"context"
	"errors"
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

type scheduler interface {
	Schedule(ctx context.Context) error
}

type intervalScheduler struct {
	task
	interval time.Duration
	logger   logging.Logger
}

type predeterminedTimeScheduler struct {
	predeterminedTimeTask
	startTs time.Time
	// retryWait paces retries of a window an upstream task is not ready for; the wait
	// inside the task absorbs most of the delay, this only prevents a hot loop.
	retryWait time.Duration
	interval  time.Duration
	logger    logging.Logger
}

func (s *intervalScheduler) Schedule(ctx context.Context) error {
	endTs := time.Now()

	for {
		if !waitFor(ctx, time.Until(endTs)) {
			return nil
		}
		switch err := s.Execute(ctx, time.Time{}, endTs); {
		case err == nil:
			s.logger.Infof("%s(%s) has been finished", s.Name(), endTs.UTC().Format(time.RFC1123Z))
		case isShutdown(ctx):
			return nil
		case errors.Is(err, ErrParentBehind):
			// the next round recomputes its window from scratch, so nothing is lost
			s.logger.Warnf("%s(%s) skipped, upstream is behind: %s", s.Name(), endTs.UTC().Format(time.RFC1123Z), err)
		default:
			return &RuntimeError{Operation: OpExecute, Task: s.Name(), WindowEnd: endTs, Err: err}
		}

		next := endTs.Truncate(s.interval).Add(s.interval)
		if next.Before(time.Now()) {
			endTs = time.Now().Truncate(s.interval).Add(s.interval)
		} else {
			endTs = next
		}
	}
}

func (s *predeterminedTimeScheduler) Schedule(ctx context.Context) error {
	optimizedStartTs, err := (s.predeterminedTimeTask).StartTimestamp(ctx, s.startTs)
	if err != nil {
		if isShutdown(ctx) {
			return nil
		}
		return &RuntimeError{Operation: OpInitializeSchedule, Task: s.Name(), Err: err}
	}

	start, end := timeframe(optimizedStartTs, s.interval)
	for end.Before(time.Now()) {
		if isShutdown(ctx) {
			return nil
		}
		retry, err := s.execute(ctx, start, end)
		if err != nil {
			return err
		}
		if retry {
			continue
		}
		start = end
		end = end.Add(s.interval)
	}

	for {
		if !waitFor(ctx, time.Until(end)) {
			return nil
		}
		retry, err := s.execute(ctx, start, end)
		if err != nil {
			return err
		}
		if isShutdown(ctx) {
			return nil
		}
		if retry {
			continue
		}
		s.logger.Infof("%s(%s-%s) has been finished", s.Name(), start.UTC().Format(time.RFC1123Z), end.UTC().Format(time.RFC1123Z))

		start = end
		end = end.Add(s.interval)
	}
}

// execute runs one fixed window. Nothing later recomputes it, so a parent that is
// still behind makes the caller retry the same window instead of advancing past it.
// The retry replays the window in full, so a task scheduled here must wait on its
// parents before its first write.
//
//	retry  the window has to be run again; the caller must not advance
//	err    already wrapped for the caller to return as-is; nil once shutdown began
func (s *predeterminedTimeScheduler) execute(ctx context.Context, start, end time.Time) (bool, error) {
	switch err := (s.predeterminedTimeTask).Execute(ctx, start, end); {
	case err == nil:
		return false, nil
	case isShutdown(ctx):
		return false, nil
	case errors.Is(err, ErrParentBehind):
		s.logger.Warnf("%s(%s-%s) retrying, upstream is behind: %s", s.Name(), start.UTC().Format(time.RFC1123Z), end.UTC().Format(time.RFC1123Z), err)
		if !waitFor(ctx, s.retryWait) {
			return false, nil
		}
		return true, nil
	default:
		return false, &RuntimeError{Operation: OpExecute, Task: s.Name(), WindowStart: start, WindowEnd: end, Err: err}
	}
}

// waitFor blocks until delay elapses or ctx ends, whichever comes first.
//
//	true  the delay elapsed normally; go ahead and run the next round
//	false ctx ended; stop the loop and return
//
// false is the result that must not be ignored: treating it as "keep going" leaves
// the scheduler running after shutdown has begun.
func waitFor(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return !isShutdown(ctx)
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func timeframe(ts time.Time, interval time.Duration) (time.Time, time.Time) {
	start := ts.Truncate(interval).UTC()

	return start, start.Add(interval).UTC()
}

func newIntervalScheduler(task task, logger logging.Logger) scheduler {
	return &intervalScheduler{
		task:     task,
		interval: 5 * time.Minute,
		logger:   logger,
	}
}

func newPredeterminedTimeScheduler(task predeterminedTimeTask, startTs time.Time, logger logging.Logger) scheduler {
	return &predeterminedTimeScheduler{
		predeterminedTimeTask: task,
		startTs:               startTs,
		retryWait:             WaitPeriod,
		interval:              30 * time.Minute,
		logger:                logger,
	}
}
