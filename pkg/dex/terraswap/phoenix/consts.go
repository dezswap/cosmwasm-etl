package phoenix

type PairAction string

const (
	SwapAction     = PairAction("swap")
	ProvideAction  = PairAction("provide_liquidity")
	WithdrawAction = PairAction("withdraw_liquidity")
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
