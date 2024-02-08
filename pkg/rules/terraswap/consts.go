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

type PairAction string

const (
	SwapAction     = PairAction("swap")
	ProvideAction  = PairAction("provide_liquidity")
	WithdrawAction = PairAction("withdraw_liquidity")
)
const (
	WasmTransferAction     = "transfer"
	WasmTransferFromAction = "transfer_from"
)

const (
	CreatePairMatchedLen         = FactoryLpAddrIdx + 1
	PairSwapMatchedLen           = PairSwapCommissionAmountIdx + 1
	PairProvideMatchedLen        = PairProvideShareIdx + 1
	PairWithdrawMatchedLen       = PairWithdrawRefundAssetsIdx + 1
	WasmTransferMatchedLen       = WasmTransferAmountIdx + 1
	TransferMatchedLen           = TransferAmountIdx + 1
	PairInitialProvideMatchedLen = PairInitialProvideToIdx + 1
)

const (
	FactoryAddrIdx = iota
	FactoryActionIdx
	FactoryPairIdx
	FactoryPairAddrIdx
	FactoryLpAddrIdx
)

const (
	PairAddrIdx = iota
	PairActionIdx
)

const (
	PairSwapOfferAssetIdx = iota + 2
	PairSwapAskAssetIdx
	PairSwapOfferAmountIdx
	PairSwapReturnAmountIdx
	PairSwapTaxAmountIdx
	PairSwapSpreadAmountIdx
	PairSwapCommissionAmountIdx
)

const (
	PairProvideAssetsIdx = iota + 2
	PairProvideShareIdx
)

const (
	PairWithdrawWithdrawShareIdx = iota + 2
	PairWithdrawRefundAssetsIdx
)

const (
	WasmTransferCw20AddrIdx = iota
	WasmTransferActionIdx
	WasmTransferFromIdx
	WasmTransferToIdx
	WasmTransferAmountIdx
)

const (
	TransferRecipientIdx = iota
	TransferSenderIdx
	TransferAmountIdx
)

const (
	PairInitialProvideAddrIdx = iota
	PairInitialProvideActionIdx
	PairInitialProvideAmountIdx
	PairInitialProvideToIdx
)
