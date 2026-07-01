package dex

import (
	"errors"
	"strings"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
)

const (
	QuarantineStatusPending  = "pending"
	QuarantineStatusResolved = "resolved"

	PartialQuarantineStagePrefix = "partial_"
)

type ParseQuarantine struct {
	ID       uint64
	Height   uint64
	Hash     string
	Stage    string
	Contract string
	Action   string
	Error    string
	RawTx    parser.RawTx
}

type PartialParseQuarantineError struct {
	ParsedTxs  []ParsedTx
	Quarantine ParseQuarantine
	Err        error
}

func (e *PartialParseQuarantineError) Error() string {
	if e.Err == nil {
		return "partial parse quarantine"
	}
	return e.Err.Error()
}

func (e *PartialParseQuarantineError) Unwrap() error {
	return e.Err
}

func IsPartialQuarantineStage(stage string) bool {
	return strings.HasPrefix(stage, PartialQuarantineStagePrefix)
}

type PartialQuarantineRecorder struct {
	tx                 parser.RawTx
	height             uint64
	containsCreatePair bool
	quarantine         *ParseQuarantine
	err                error
}

func NewPartialQuarantineRecorder(tx parser.RawTx, height uint64) PartialQuarantineRecorder {
	return PartialQuarantineRecorder{
		tx:                 tx,
		height:             height,
		containsCreatePair: RawTxContainsCreatePair(tx),
	}
}

func (r *PartialQuarantineRecorder) Record(stage string, err error) bool {
	var ambiguity *eventlog.AmbiguousEventError
	if !errors.As(err, &ambiguity) || r.containsCreatePair {
		return false
	}
	if r.quarantine == nil {
		r.quarantine = &ParseQuarantine{
			Height:   r.height,
			Hash:     r.tx.Hash,
			Stage:    PartialQuarantineStagePrefix + stage,
			Contract: ambiguity.Contract,
			Action:   ambiguity.Action,
			Error:    err.Error(),
			RawTx:    r.tx,
		}
		r.err = err
	}
	return true
}

func (r *PartialQuarantineRecorder) Err(parsedTxs []ParsedTx) error {
	if r.quarantine == nil {
		return nil
	}
	return &PartialParseQuarantineError{
		ParsedTxs:  parsedTxs,
		Quarantine: *r.quarantine,
		Err:        r.err,
	}
}

func RawTxContainsCreatePair(tx parser.RawTx) bool {
	for _, log := range tx.LogResults {
		for _, attr := range log.Attributes {
			if attr.Key == "action" && attr.Value == string(CreatePair) {
				return true
			}
		}
	}
	return false
}
