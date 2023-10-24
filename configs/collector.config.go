package configs

import (
	"github.com/spf13/viper"
)

type CollectorConfig struct {
	ChainId                    string
	PairFactoryContractAddress string
	NodeConfig                 NodeConfig
}
type NodeConfig struct {
	GrpcConfig      GrpcConfig
	FailoverLcdHost string
}

func collectorConfig(v *viper.Viper) CollectorConfig {
	nodeC := NodeConfig{}
	if subV := v.Sub("collector.node"); subV != nil {
		nodeC = nodeConfig(subV)
	}

	return CollectorConfig{
		ChainId:                    v.GetString("collector.chainId"),
		NodeConfig:                 nodeC,
		PairFactoryContractAddress: v.GetString("collector.pair_factory_contract_address"),
	}
}

func nodeConfig(v *viper.Viper) NodeConfig {
	grpcC := GrpcConfig{}
	if subV := v.Sub("grpc"); subV != nil {
		grpcC = grpcConfig(subV)
	}

	return NodeConfig{
		GrpcConfig:      grpcC,
		FailoverLcdHost: v.GetString("failover_lcd_host"),
	}
}
