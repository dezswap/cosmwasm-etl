package dex

import "github.com/dezswap/cosmwasm-etl/parser"

type DexParserApp interface {
	Run() error
	TargetApp
	SourceDataStore
	Repo
}

type TargetApp interface {
	parser.TargetApp[ParsedTx]
}

type Repo interface {
	parser.Repo[ParsedTx]
	PairRepo
	ParsedPoolsInfo(from, to uint64) ([]PoolInfo, error)
	ValidationExceptionList() ([]string, error)
}

type SourceDataStore interface {
	parser.SourceDataStore
	GetPoolInfos(height uint64) ([]PoolInfo, error)
}

type PairRepo interface {
	GetPairs() (map[string]Pair, error)
}
