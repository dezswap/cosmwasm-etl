package terraswap

const (
	MainnetPrefix = "phoenix"
	ClassicPrefix = "columbus"
	TestnetPrefix = "pisco"
)

var FactoryAddress = map[string]string{
	"phoenix":  "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul",
	"pisco":    "terra1jha5avc92uerwp9qzx3flvwnyxs3zax2rrm6jkcedy2qvzwd2k7qk7yxcl",
	"columbus": "terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj",
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
