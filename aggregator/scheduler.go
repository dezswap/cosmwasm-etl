package aggregator

import (
	"context"
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
	startTs  time.Time
	interval time.Duration
	logger   logging.Logger
}

func (s *intervalScheduler) Schedule(ctx context.Context) error {
	endTs := time.Now()

	for {
		if !waitFor(ctx, time.Until(endTs)) {
			return nil
		}
		if err := s.Execute(ctx, time.Time{}, endTs); err != nil {
			if isShutdown(ctx) {
				return nil
			}
			return &RuntimeError{Operation: OpExecute, Task: s.Name(), WindowEnd: endTs, Err: err}
		}
		s.logger.Infof("%s(%s) has been finished", s.Name(), endTs.UTC().Format(time.RFC1123Z))

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
		if err := (s.predeterminedTimeTask).Execute(ctx, start, end); err != nil {
			if isShutdown(ctx) {
				return nil
			}
			return &RuntimeError{Operation: OpExecute, Task: s.Name(), WindowStart: start, WindowEnd: end, Err: err}
		}
		start = end
		end = end.Add(s.interval)
	}

	for {
		if !waitFor(ctx, time.Until(end)) {
			return nil
		}
		if err := (s.predeterminedTimeTask).Execute(ctx, start, end); err != nil {
			if isShutdown(ctx) {
				return nil
			}
			return &RuntimeError{Operation: OpExecute, Task: s.Name(), WindowStart: start, WindowEnd: end, Err: err}
		}
		s.logger.Infof("%s(%s-%s) has been finished", s.Name(), start.UTC().Format(time.RFC1123Z), end.UTC().Format(time.RFC1123Z))

		start = end
		end = end.Add(s.interval)
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
		interval:              30 * time.Minute,
		logger:                logger,
	}
}
