package configs

type CollectorConfig struct {
	ChainId                    string     `mapstructure:"chainid"`
	PairFactoryContractAddress string     `mapstructure:"pair_factory_contract_address"`
	NodeConfig                 NodeConfig `mapstructure:"node"`
	FcdConfig                  FcdConfig  `mapstructure:"fcd"`
}

type FcdConfig struct {
	RdbConfig       RdbConfig `mapstructure:"rdb"`
	Url             string    `mapstructure:"url"`
	TargetAddresses []string  `mapstructure:"target_addresses"`
	UntilHeight     uint      `mapstructure:"until_height"`
}
