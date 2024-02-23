package terraswap

import (
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
)

var ParsableRules = map[string]bool{
	string(eventlog.TransferType): true,
	string(eventlog.FromContract): true,
	string(eventlog.WasmType):     true,
}
