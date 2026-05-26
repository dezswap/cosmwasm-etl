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
