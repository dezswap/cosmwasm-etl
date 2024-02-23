package phoenix

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
	CreatePairMatchedLen   = FactoryLpAddrIdx + 1
	PairSwapMatchedLen     = PairSwapCommissionAmountIdx + 1
	PairProvideMatchedLen  = PairProvideShareIdx + 1
	PairWithdrawMatchedLen = PairWithdrawRefundAssetsIdx + 1
	WasmTransferMatchedLen = WasmTransferAmountIdx + 1
	TransferMatchedLen     = TransferAmountIdx + 1
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
	PairSwapSenderIdx = iota + PairActionIdx + 1
	PairSwapReceiverIdx
	PairSwapOfferAssetIdx
	PairSwapAskAssetIdx
	PairSwapOfferAmountIdx
	PairSwapReturnAmountIdx
	PairSwapSpreadAmountIdx
	PairSwapCommissionAmountIdx
)

const (
	PairProvideSenderIdx = iota + PairActionIdx + 1
	PairProvideReceiverIdx
	PairProvideAssetsIdx
	PairProvideShareIdx
)

const (
	PairWithdrawSenderIdx = iota + PairActionIdx + 1
	PairWithdrawWithdrawShareIdx
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
