package dex

import (
	"fmt"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
)

type ParserMock struct{ mock.Mock }
type RepoMock struct{ mock.Mock }

type RawStoreMock struct{ mock.Mock }

var _ Repo = &RepoMock{}

var _ parser.Parser[ParsedTx] = &ParserMock{}
var _ SourceDataStore = &RawStoreMock{}

// matchedToParsedTx implements parser
func (p *ParserMock) MatchedToParsedTx(result eventlog.MatchedResult, optional ...interface{}) ([]*ParsedTx, error) {
	args := p.MethodCalled("matchedToParsedTx", result)
	return args.Get(0).([]*ParsedTx), args.Error(1)
}

// parse implements parser
func (p *ParserMock) Parse(raws eventlog.LogResults, defaultValue parser.Overrider[ParsedTx], optionals ...interface{}) ([]*ParsedTx, error) {
	args := p.MethodCalled("parse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	return args.Get(0).([]*ParsedTx), args.Error(1)
}

// GetPoolInfos implements RawDataStore
func (r *RawStoreMock) GetPoolInfos(height uint64) ([]PoolInfo, error) {
	args := r.MethodCalled("GetPoolInfos", height)
	return args.Get(0).([]PoolInfo), args.Error(1)
}

// GetSourceSyncedHeight implements RawDataStore
func (r *RawStoreMock) GetSourceSyncedHeight() (uint64, error) {
	args := r.MethodCalled("GetSourceSyncedHeight")
	return args.Get(0).(uint64), args.Error(1)
}

// GetSourceTxs implements RawDataStore
func (r *RawStoreMock) GetSourceTxs(height uint64) (parser.RawTxs, error) {
	args := r.MethodCalled("GetSourceTxs", height)
	return args.Get(0).(parser.RawTxs), args.Error(1)
}

// GetPairs implements Repo
func (m *RepoMock) GetPairs() (map[string]Pair, error) {
	args := m.MethodCalled("GetPairs")
	return args.Get(0).(map[string]Pair), args.Error(1)
}

// GetTokenExceptions implements Repo
func (m *RepoMock) GetTokenExceptions() (map[string]bool, error) {
	args := m.MethodCalled("GetTokenExceptions")
	return args.Get(0).(map[string]bool), args.Error(1)
}

// GetSyncedHeight implements Repo
func (m *RepoMock) GetSyncedHeight() (uint64, error) {
	args := m.MethodCalled("GetSyncedHeight")
	return args.Get(0).(uint64), args.Error(1)
}

// Insert implements Repo
func (m *RepoMock) Insert(srcHeight uint64, targetHeight uint64, txs []ParsedTx, arg ...interface{}) error {
	if len(arg) != InsertArgCount {
		errMsg := fmt.Sprintf("invalid others(%v)", arg)
		return errors.New(errMsg)
	}

	pools, ok := arg[InsertArgPoolsIndex].([]PoolInfo)
	if !ok {
		errMsg := fmt.Sprintf("invalid pools(%v)", arg[InsertArgPoolsIndex])
		return errors.New(errMsg)
	}
	pairs, ok := arg[InsertArgPairsIndex].([]Pair)
	if !ok {
		errMsg := fmt.Sprintf("invalid pairs(%v)", arg[InsertArgPairsIndex])
		return errors.New(errMsg)
	}
	quarantines, ok := arg[InsertArgParseQuarantinesIndex].([]ParseQuarantine)
	if !ok {
		errMsg := fmt.Sprintf("invalid quarantines(%v)", arg[InsertArgParseQuarantinesIndex])
		return errors.New(errMsg)
	}
	args := m.MethodCalled("Insert", srcHeight, targetHeight, txs, pools, pairs, quarantines)
	return args.Error(0)
}

func (m *RepoMock) InsertPairValidationException(chainID string, contractAddress string) error {
	args := m.MethodCalled("InsertPairValidationException")
	return args.Error(0)
}

// ParsedPoolInfo implements Repo.
func (m *RepoMock) ParsedPoolsInfo(from, to uint64) ([]PoolInfo, error) {
	args := m.MethodCalled("ParsedPoolsInfo", from, to)
	return args.Get(0).([]PoolInfo), args.Error(1)
}

// ValidationExceptionList implements Repo.
func (m *RepoMock) ValidationExceptionList() ([]string, error) {
	args := m.MethodCalled("ValidationExceptionList")
	return args.Get(0).([]string), args.Error(1)
}

// GetValidationHeight implements Repo.
func (m *RepoMock) GetValidationHeight() (uint64, error) {
	return 0, nil
}

// SetValidationHeight implements Repo.
func (m *RepoMock) SetValidationHeight(_ uint64) error {
	return nil
}

// ClearValidationHeight implements Repo.
func (m *RepoMock) ClearValidationHeight() error {
	return nil
}

func (m *RepoMock) PendingParseQuarantines() ([]ParseQuarantine, error) {
	args := m.MethodCalled("PendingParseQuarantines")
	return args.Get(0).([]ParseQuarantine), args.Error(1)
}

func (m *RepoMock) ResolveParseQuarantine(id uint64, height uint64, txs []ParsedTx) error {
	args := m.MethodCalled("ResolveParseQuarantine", id, height, txs)
	return args.Error(0)
}
