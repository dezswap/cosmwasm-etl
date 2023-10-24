package parser

import (
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/stretchr/testify/mock"
)

type ParserMock struct{ mock.Mock }
type RepoMock struct{ mock.Mock }
type RawStoreMock struct{ mock.Mock }

var _ Repo = &RepoMock{}

var _ Parser = &ParserMock{}
var _ SourceDataStore = &RawStoreMock{}
var _ Repo = &RepoMock{}

// matchedToParsedTx implements parser
func (p *ParserMock) MatchedToParsedTx(result eventlog.MatchedResult, optional ...interface{}) (*ParsedTx, error) {
	args := p.Mock.MethodCalled("matchedToParsedTx", result)
	return args.Get(0).(*ParsedTx), args.Error(1)
}

// parse implements parser
func (p *ParserMock) Parse(hash string, timestamp time.Time, raws eventlog.LogResults, optionals ...interface{}) ([]*ParsedTx, error) {
	args := p.Mock.MethodCalled("parse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	return args.Get(0).([]*ParsedTx), args.Error(1)
}

// GetPoolInfos implements RawDataStore
func (r *RawStoreMock) GetPoolInfos(height uint64) ([]PoolInfo, error) {
	args := r.Mock.MethodCalled("GetPoolInfos", height)
	return args.Get(0).([]PoolInfo), args.Error(1)
}

// GetSourceSyncedHeight implements RawDataStore
func (r *RawStoreMock) GetSourceSyncedHeight() (uint64, error) {
	args := r.Mock.MethodCalled("GetSourceSyncedHeight")
	return args.Get(0).(uint64), args.Error(1)
}

// GetSourceTxs implements RawDataStore
func (r *RawStoreMock) GetSourceTxs(height uint64) (RawTxs, error) {
	args := r.Mock.MethodCalled("GetSourceTxs", height)
	return args.Get(0).(RawTxs), args.Error(1)
}

// GetPairs implements Repo
func (m *RepoMock) GetPairs() (map[string]Pair, error) {
	args := m.Mock.MethodCalled("GetPairs")
	return args.Get(0).(map[string]Pair), args.Error(1)
}

// GetSyncedHeight implements Repo
func (m *RepoMock) GetSyncedHeight() (uint64, error) {
	args := m.Mock.MethodCalled("GetSyncedHeight")
	return args.Get(0).(uint64), args.Error(1)
}

// Insert implements Repo
func (m *RepoMock) Insert(height uint64, txs []ParsedTx, pools []PoolInfo, pairDto []Pair) error {
	args := m.Mock.MethodCalled("Insert", height, txs, pools, pairDto)
	return args.Error(0)
}
