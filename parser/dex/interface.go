package dex

import "github.com/dezswap/cosmwasm-etl/parser"

type TargetApp interface {
	parser.TargetApp[ParsedTx]
}

type Dex interface {
	Run() error
	TargetApp
	insert(height uint64, txs []ParsedTx, pools []PoolInfo) error
	checkRemoteHeight(srcHeight uint64) error
}

type PairRepo interface {
	GetPairs() (map[string]Pair, error)
}

type Repo interface {
	parser.Repo[ParsedTx]
	PairRepo
	ParsedPoolsInfo(from, to uint64) ([]PoolInfo, error)
}

type SourceDataStore interface {
	parser.SourceDataStore
	GetPoolInfos(height uint64) ([]PoolInfo, error)
}
