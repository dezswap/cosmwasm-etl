package dex

import (
	"github.com/dezswap/cosmwasm-etl/parser"
)

const (
	QuarantineStatusPending  = "pending"
	QuarantineStatusResolved = "resolved"
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
