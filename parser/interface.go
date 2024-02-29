package parser

type ParserApp[T any] interface {
	Run() error
	SourceDataStore
	TargetApp[T]
	Repo[T]
}

type TargetApp[T any] interface {
	ParseTxs(tx RawTx, height uint64) ([]T, error)
}

type Repo[T any] interface {
	Insert(height uint64, txs []T, arg ...interface{}) error
	GetSyncedHeight() (uint64, error)
}

type SourceDataStore interface {
	GetSourceSyncedHeight() (uint64, error)
	GetSourceTxs(height uint64) (RawTxs, error)
}
