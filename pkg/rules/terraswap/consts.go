package terraswap

import (
	ts "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
)

const (
	MainnetKey   = "phoenix"
	ClassicV1Key = "columbus_v1"
	ClassicV2Key = "columbus_v2"
	TestnetKey   = "pisco"
)

var FactoryAddress = map[string]string{
	MainnetKey:   ts.PHOENIX_FACTORY,
	TestnetKey:   ts.PISCO_FACTORY,
	ClassicV1Key: ts.COLUMBUS_V1_FACTORY,
	ClassicV2Key: ts.COLUMBUS_V2_FACTORY,
}

type PairAction string

const (
	SwapAction     = PairAction("swap")
	ProvideAction  = PairAction("provide_liquidity")
	WithdrawAction = PairAction("withdraw_liquidity")
)
const WasmTransferAction = "transfer"

const (
	CreatePairMatchedLen         = FactoryLpAddrIdx + 1
	PairCommonMatchedLen         = PairSenderIdx + 1
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
	PairSenderIdx
)

const (
	PairSwapReceiverIdx = iota + 3
	PairSwapOfferAssetIdx
	PairSwapAskAssetIdx
	PairSwapOfferAmountIdx
	PairSwapReturnAmountIdx
	PairSwapSpreadAmountIdx
	PairSwapCommissionAmountIdx
)

const (
	PairProvideReceiverIdx = iota + 3
	PairProvideAssetsIdx
	PairProvideShareIdx
)

const (
	PairWithdrawWithdrawShareIdx = iota + 3
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
