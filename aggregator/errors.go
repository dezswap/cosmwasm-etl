package aggregator

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Operations carried by RuntimeError.Operation. ErrorEvent classifies on these, so
// every new operation belongs here and in that switch rather than as a literal.
const (
	OpMarshalConfig       = "marshal_config"
	OpOpenSourceRepo      = "open_source_repository"
	OpOpenDestinationRepo = "open_destination_repository"
	OpOpenPriceRepo       = "open_price_repository"
	OpOpenRouterRepo      = "open_router_repository"
	OpInitializeTasks     = "initialize_tasks"
	OpCleanDuplicates     = "clean_duplicates"
	OpInitializeSchedule  = "initialize_schedule"
	OpExecute             = "execute"
	OpClose               = "close"
)

// Events emitted as the "event" log field.
const (
	EventInitializationFailed = "aggregator.initialization_failed"
	EventCleanupFailed        = "aggregator.cleanup_failed"
	EventTaskFailed           = "aggregator.task_failed"
)

// ErrParentBehind reports that a parent task had not reached the height a round
// needed. It is a "not yet", not a failure: schedulers retry the round instead of
// failing the whole run, since cold starts make this the normal state for a while and
// treating it as fatal turns them into a restart loop.
//
// A task may only report it if it waits on its parents before its first write that a
// rerun could not repeat safely
var ErrParentBehind = errors.New("parent task has not reached the target height")

// RuntimeError describes where an aggregator lifecycle or task failure occurred.
type RuntimeError struct {
	Operation   string
	Task        string
	WindowStart time.Time
	WindowEnd   time.Time
	Err         error
}

func (e *RuntimeError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Task != "" {
		return fmt.Sprintf("aggregator %s task=%s: %s", e.Operation, e.Task, singleLine(e.Err))
	}
	return fmt.Sprintf("aggregator %s: %s", e.Operation, singleLine(e.Err))
}

// singleLine renders an error on one line. errors.Join separates its branches with
// newlines, and these messages end up in structured log fields that must not break
// across lines, so every error rendered into a field goes through here.
func singleLine(err error) string {
	if err == nil {
		return "<nil>"
	}

	return strings.ReplaceAll(err.Error(), "\n", "; ")
}

func (e *RuntimeError) Unwrap() error { return e.Err }

// isShutdown reports whether an error seen now is fallout from ctx being canceled.
// Drivers report cancellation with their own values (pq: "canceling statement due
// to user request"), so errors.Is(err, context.Canceled) misses them and the
// context state is the only reliable signal.
func isShutdown(ctx context.Context) bool { return ctx.Err() != nil }

// ErrorFields returns stable, machine-readable fields for the root error log.
func ErrorFields(err error) logrus.Fields {
	// No is_canceled field: drivers report cancellation with their own values (see
	// isShutdown), so errors.Is(err, context.Canceled) would be false for exactly the
	// cases that matter and the field would mislead more than it informs.
	fields := logrus.Fields{
		"component":   "aggregator",
		"error":       singleLine(err),
		"error_chain": errorChain(err),
		"is_timeout":  errors.Is(err, context.DeadlineExceeded),
	}

	var runtimeErr *RuntimeError
	if errors.As(err, &runtimeErr) {
		fields["operation"] = runtimeErr.Operation
		if runtimeErr.Task != "" {
			fields["task"] = runtimeErr.Task
		}
		if !runtimeErr.WindowStart.IsZero() {
			fields["window_start"] = runtimeErr.WindowStart.UTC().Format(time.RFC3339)
		}
		if !runtimeErr.WindowEnd.IsZero() {
			fields["window_end"] = runtimeErr.WindowEnd.UTC().Format(time.RFC3339)
		}
	}

	root := rootCause(err)
	fields["error_type"] = reflect.TypeOf(root).String()
	return fields
}

// ErrorEvent classifies an aggregator error for stable root-level logging.
func ErrorEvent(err error) string {
	var runtimeErr *RuntimeError
	if !errors.As(err, &runtimeErr) {
		return EventTaskFailed
	}
	switch runtimeErr.Operation {
	case OpClose:
		return EventCleanupFailed
	case OpMarshalConfig, OpOpenSourceRepo, OpOpenDestinationRepo, OpOpenPriceRepo,
		OpOpenRouterRepo, OpInitializeTasks, OpCleanDuplicates:
		return EventInitializationFailed
	case OpExecute, OpInitializeSchedule:
		return EventTaskFailed
	default:
		return EventTaskFailed
	}
}

func errorChain(err error) []string {
	chain := make([]string, 0, 4)
	appendErrorChain(err, &chain)
	return chain
}

// appendErrorChain handles both ordinary wrapped errors and errors.Join trees. A
// join node contributes nothing itself: its Error() is just its children's messages
// concatenated with newlines, which would duplicate every branch and break the log
// field across lines. Only its children are recorded.
func appendErrorChain(err error, chain *[]string) {
	if err == nil {
		return
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range joined.Unwrap() {
			appendErrorChain(child, chain)
		}
		return
	}
	*chain = append(*chain, err.Error())
	appendErrorChain(errors.Unwrap(err), chain)
}

// rootCause follows the primary branch down to the innermost error for a stable
// type field; all branches remain available separately in error_chain. Joins are
// unwrapped wherever they appear, not just at the top: a RuntimeError wrapping a
// joined error is the normal shape of a cleanup failure.
func rootCause(err error) error {
	for {
		if joined, ok := err.(interface{ Unwrap() []error }); ok {
			children := joined.Unwrap()
			if len(children) == 0 {
				return err
			}
			err = children[0]
			continue
		}
		next := errors.Unwrap(err)
		if next == nil {
			return err
		}
		err = next
	}
}
