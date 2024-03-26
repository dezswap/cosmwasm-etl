package columbusv2

type PairAction string

const (
	SwapAction     = PairAction("swap")
	ProvideAction  = PairAction("provide_liquidity")
	WithdrawAction = PairAction("withdraw_liquidity")
)

const (
	SortedTransferMatchedLen = SortedTransferSenderIdx + 1
)

const (
	PairAddrIdx = iota
	PairActionIdx
)

const (
	SortedTransferAmountIdx = iota
	SortedTransferRecipientIdx
	SortedTransferSenderIdx
)

const (
	PairAddrKey   = "_contract_address"
	PairActionKey = "action"
)

const (
	PairSwapAskAssetKey         = "ask_asset"
	PairSwapCommissionAmountKey = "commission_amount"
	PairSwapOfferAmountKey      = "offer_amount"
	PairSwapOfferAssetKey       = "offer_asset"
	PairSwapReceiverKey         = "receiver"
	PairSwapReturnAmountKey     = "return_amount"
	PairSwapSenderKey           = "sender"
	PairSwapSpreadAmountKey     = "spread_amount"
)

const (
	SortedTransferAmountKey    = "amount"
	SortedTransferRecipientKey = "recipient"
	SortedTransferSenderKey    = "sender"
)

const (
	PairProvideAssetsKey      = "assets"
	PairProvideSenderKey      = "sender"
	PairProvideReceiverKey    = "receiver"
	PairProvideShareKey       = "share"
	PairProvideRefundAssetKey = "refund_assets"
)

const (
	PairWithdrawRefundAssetsKey  = "refund_assets"
	PairWithdrawSenderKey        = "sender"
	PairWithdrawWithdrawShareKey = "withdrawn_share"
)
