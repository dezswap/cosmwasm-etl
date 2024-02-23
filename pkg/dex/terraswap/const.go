package terraswap

type TerraswapFactory string

const (
	MAINNET_FACTORY    TerraswapFactory = "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
	CLASSIC_V1_FACTORY TerraswapFactory = "terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"
	CLASSIC_V2_FACTORY TerraswapFactory = "terra1jkndu9w5attpz09ut02sgey5dd3e8sq5watzm0"
	PISCO_FACTORY      TerraswapFactory = "terra1jha5avc92uerwp9qzx3flvwnyxs3zax2rrm6jkcedy2qvzwd2k7qk7yxcl"
	INVALID_FACTORY    TerraswapFactory = "invalid"
)

const (
	COLUMBUS_V1_FACTORY_INSTANTIATE_HEIGHT = 548_978
	COLUMBUS_4_END_HEIGHT                  = 4_724_000
)
