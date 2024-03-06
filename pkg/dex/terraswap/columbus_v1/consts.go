package columbus_v1

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
	PairSwapMatchedLen     = PairSwapCommissionAmountIdx + 1
	PairProvideMatchedLen  = PairProvideShareIdx + 1
	PairWithdrawMatchedLen = PairWithdrawRefundAssetsIdx + 1
	WasmTransferMatchedLen = WasmTransferAmountIdx + 1
	TransferMatchedLen     = TransferAmountIdx + 1
)

const (
	FactoryAddrIdx = iota
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
