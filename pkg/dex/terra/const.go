package terra

type Factory string

const (
	MainnetFactory   Factory = "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
	ClassicV1Factory Factory = "terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"
	ClassicV2Factory Factory = "terra1jkndu9w5attpz09ut02sgey5dd3e8sq5watzm0"
	PiscoFactory     Factory = "terra1jha5avc92uerwp9qzx3flvwnyxs3zax2rrm6jkcedy2qvzwd2k7qk7yxcl"
	InvalidFactory   Factory = "invalid"
)

const (
	ColumbusV1FactoryInstantiateHeight = 548_978
	Columbus4EndHeight                 = 4_724_000
)

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

const PairCommonMatchedLen = PairActionIdx + 1

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
