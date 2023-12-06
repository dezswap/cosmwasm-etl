package configs

import (
	"github.com/spf13/viper"
)

type CollectorConfig struct {
	ChainId                    string
	PairFactoryContractAddress string
	NodeConfig                 NodeConfig
}

func collectorConfig(v *viper.Viper) CollectorConfig {
	return CollectorConfig{
		ChainId: v.GetString("collector.chainId"),
		NodeConfig: NodeConfig{
			GrpcConfig:      grpcConfig(v, "collector.node"),
			FailoverLcdHost: v.GetString("collector.node.failover_lcd_host"),
		},
		PairFactoryContractAddress: v.GetString("collector.pair_factory_contract_address"),
	}
}
