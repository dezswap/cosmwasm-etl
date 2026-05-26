package configs

type CollectorConfig struct {
	ChainId                    string     `mapstructure:"chainid"`
	PairFactoryContractAddress string     `mapstructure:"pair_factory_contract_address"`
	NodeConfig                 NodeConfig `mapstructure:"node"`
	FcdConfig                  FcdConfig  `mapstructure:"fcd"`
	StartHeight                uint64     `mapstructure:"start_height"`
	UntilHeight                uint64     `mapstructure:"until_height"`
	PoolSnapshotInterval       uint       `mapstructure:"pool_snapshot_interval"`
}

type FcdConfig struct {
	Url             string   `mapstructure:"url"`
	TargetAddresses []string `mapstructure:"target_addresses"`
}

// NodeConfigWithFallback resolves node settings for collectors migrating from
// parser-owned node configuration while preserving collector endpoint overrides.
func (c CollectorConfig) NodeConfigWithFallback(fallback NodeConfig) NodeConfig {
	if c.NodeConfig.RestClientConfig.RpcHost == "" && c.NodeConfig.RestClientConfig.LcdHost == "" {
		return fallback
	}

	nodeConfig := c.NodeConfig
	if nodeConfig.RestClientConfig.RpcHost == "" {
		nodeConfig.RestClientConfig.RpcHost = fallback.RestClientConfig.RpcHost
	}
	if nodeConfig.RestClientConfig.LcdHost == "" {
		nodeConfig.RestClientConfig.LcdHost = fallback.RestClientConfig.LcdHost
	}
	return nodeConfig
}
