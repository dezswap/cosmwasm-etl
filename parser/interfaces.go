package parser

import (
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
)

type Runner interface {
	Run() error
}

type TargetApp interface {
	ParseTxs(tx RawTx, height uint64) ([]ParsedTx, error)
}

type Dex interface {
	Runner
	TargetApp
	insert(height uint64, txs []ParsedTx, pools []PoolInfo) error
	checkRemoteHeight(srcHeight uint64) error
}

type PairRepo interface {
	GetPairs() (map[string]Pair, error)
}

type Repo interface {
	Insert(height uint64, txs []ParsedTx, pools []PoolInfo, pairDto []Pair) error
	GetSyncedHeight() (uint64, error)
	PairRepo
	ParsedPoolsInfo(from, to uint64) ([]PoolInfo, error)
}

type SourceDataStore interface {
	GetSourceSyncedHeight() (uint64, error)
	GetSourceTxs(height uint64) (RawTxs, error)
	GetPoolInfos(height uint64) ([]PoolInfo, error)
}

type Mapper interface {
	// return nil if the matched result is not for this parser
	MatchedToParsedTx(eventlog.MatchedResult, ...interface{}) (*ParsedTx, error)
}

type Parser interface {
	Parse(hash string, timestamp time.Time, raws eventlog.LogResults, optionals ...interface{}) ([]*ParsedTx, error)
	Mapper
}
