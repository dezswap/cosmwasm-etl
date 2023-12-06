package aggregator

import (
	"context"
	"github.com/delight-labs/cosmwasm-etl/pkg/logging"
	"reflect"
	"time"
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
	done := false
	for {
		select {
		case <-ctx.Done():
			done = true
			break
		case <-time.After(time.Until(endTs)):
			if err := s.task.Execute(time.Time{}, endTs); err != nil {
				errChan <- err
			}
			s.logger.Infof("%s(%s) has been finished", reflect.TypeOf(s.task), endTs.UTC().Format(time.RFC1123Z))

			next := endTs.Truncate(s.interval).Add(s.interval)
			if next.Before(time.Now()) {
				endTs = time.Now().Truncate(s.interval).Add(s.interval)
			} else {
				endTs = next
			}
		}
		if done {
			break
		}
	}

	return nil
}

func (s *predeterminedTimeScheduler) Schedule(ctx context.Context) error {
	optimizedStartTs, err := (s.predeterminedTimeTask).StartTimestamp(s.startTs)
	if err != nil {
		return err
	}

	start, end := timeframe(optimizedStartTs, s.interval)
	for end.Before(time.Now()) {
		if err := (s.predeterminedTimeTask).Execute(start, end); err != nil {
			errChan <- err
		}
		start = end
		end = end.Add(s.interval)
	}

	done := false
	for {
		select {
		case <-ctx.Done():
			done = true
			break
		case <-time.After(time.Until(end)):
			if err := (s.predeterminedTimeTask).Execute(start, end); err != nil {
				errChan <- err
			}
			s.logger.Infof("%s(%s-%s) has been finished", reflect.TypeOf(s.predeterminedTimeTask), start.UTC().Format(time.RFC1123Z), end.UTC().Format(time.RFC1123Z))

			start = end
			end = end.Add(s.interval)
		}
		if done {
			break
		}
	}

	return nil
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
