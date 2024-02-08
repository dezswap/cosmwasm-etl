package terraswap

import (
	ts "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
)

type TerraswapType string

const (
	Mainnet     TerraswapType = "phoenix"
	ClassicV1   TerraswapType = "columbus_v1"
	ClassicV2   TerraswapType = "columbus_v2"
	Pisco       TerraswapType = "pisco"
	InvalidType TerraswapType = "invalid"
)

var ParsableRules = map[string]bool{
	string(eventlog.TransferType): true,
	string(eventlog.FromContract): true,
	string(eventlog.WasmType):     true,
}
var FactoryAddress = map[TerraswapType]string{
	Mainnet:   ts.PHOENIX_FACTORY,
	Pisco:     ts.PISCO_FACTORY,
	ClassicV1: ts.COLUMBUS_V1_FACTORY,
	ClassicV2: ts.COLUMBUS_V2_FACTORY,
}
