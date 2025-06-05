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

// GetSyncedHeight implements Repo
func (m *RepoMock) GetSyncedHeight() (uint64, error) {
	args := m.MethodCalled("GetSyncedHeight")
	return args.Get(0).(uint64), args.Error(1)
}

// Insert implements Repo
func (m *RepoMock) Insert(height uint64, txs []ParsedTx, arg ...interface{}) error {
	if len(arg) != 2 {
		errMsg := fmt.Sprintf("invalid others(%v)", arg)
		return errors.New(errMsg)
	}

	pools, ok := arg[0].([]PoolInfo)
	if !ok {
		errMsg := fmt.Sprintf("invalid pools(%v)", arg[0])
		return errors.New(errMsg)
	}
	pairs, ok := arg[1].([]Pair)
	if !ok {
		errMsg := fmt.Sprintf("invalid pairs(%v)", arg[1])
		return errors.New(errMsg)
	}
	args := m.MethodCalled("Insert", height, txs, pools, pairs)
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
