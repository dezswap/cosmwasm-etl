package starfleit

const (
	MainnetPrefix = "fetchhub"
	TestnetPrefix = "dorado"
)

const (
	MainnetV2Height = 11543129
	TestnetV2Height = 11543129
)

var FactoryAddress = map[string]string{
	"fetchhub": "fetch1slz6c85kxp4ek5ufmcakfhnscv9r2snlemxgwz6cjhklgh7v2hms8rgt5v",
	"dorado":   "fetch1kmag3937lrl6dtsv29mlfsedzngl9egv5c3apnr468q50gu04zrqea398u",
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
	PairCommonMatchedLen         = PairActionIdx + 1
	PairSwapMatchedLen           = PairSwapSpreadAmountIdx + 1
	PairProvideMatchedLen        = PairProvideShareIdx + 1
	PairV2ProvideMatchedLen      = PairV2ProvideShareIdx + 1
	PairWithdrawMatchedLen       = PairWithdrawWithdrawShareIdx + 1
	PairInitialProvideMatchedLen = PairInitialProvideToIdx + 1
	WasmCommonTransferMatchedLen = WasmCommonTransferActionIdx + 1
	WasmV1TransferMatchedLen     = WasmTransferToIdx + 1
	WasmV2TransferMatchedLen     = WasmTransferFromToIdx + 1
	TransferMatchedLen           = TransferAmountIdx + 1
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
	PairSwapAskAssetIdx = iota + PairCommonMatchedLen
	PairSwapCommissionAmountIdx
	PairSwapOfferAmountIdx
	PairSwapOfferAssetIdx
	PairSwapReceiverIdx
	PairSwapReturnAmountIdx
	PairSwapSenderIdx
	PairSwapSpreadAmountIdx
)

const (
	PairProvideAssetsIdx = iota + PairCommonMatchedLen
	PairProvideReceiverIdx
	PairProvideSenderIdx
	PairProvideShareIdx
)

// include returned assets
const (
	PairV2ProvideAssetsIdx = iota + PairCommonMatchedLen
	PairV2ProvideReceiverIdx
	PairV2RefundAssetsIdx
	PairV2ProvideSenderIdx
	PairV2ProvideShareIdx
)

const (
	PairWithdrawRefundAssetsIdx = iota + PairCommonMatchedLen
	PairWithdrawSenderIdx
	PairWithdrawWithdrawShareIdx
)

const (
	PairInitialProvideAddrIdx = iota
	PairInitialProvideActionIdx
	PairInitialProvideAmountIdx
	PairInitialProvideToIdx
)

const (
	WasmCommonTransferCw20AddrIdx = iota
	WasmCommonTransferActionIdx
)

const (
	WasmTransferAmountIdx = iota + WasmCommonTransferMatchedLen
	WasmTransferFromIdx
	WasmTransferToIdx
)

const (
	WasmTransferFromAmountIdx = iota + WasmCommonTransferMatchedLen
	WasmTransferFromByIdx
	WasmTransferFromFromIdx
	WasmTransferFromToIdx
)

const (
	TransferRecipientIdx = iota
	TransferSenderIdx
	TransferAmountIdx
)
