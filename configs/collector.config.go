package configs

import (
	"github.com/spf13/viper"
)

type CollectorConfig struct {
	ChainId                    string
	PairFactoryContractAddress string
	NodeConfig                 NodeConfig
	FcdConfig                  FcdConfig
}

type FcdConfig struct {
	RdbConfig       RdbConfig
	Url             string
	TargetAddresses []string
	UntilHeight     uint
}

func collectorConfig(v *viper.Viper) CollectorConfig {
	return CollectorConfig{
		ChainId: v.GetString("collector.chainId"),
		NodeConfig: NodeConfig{
			GrpcConfig:      grpcConfig(v, "collector.node"),
			FailoverLcdHost: v.GetString("collector.node.failover_lcd_host"),
		},
		PairFactoryContractAddress: v.GetString("collector.pair_factory_contract_address"),
		FcdConfig: FcdConfig{
			RdbConfig: func() RdbConfig {
				sslMode := v.GetString("collector.fcd.rdb.sslmode")
				if sslMode == "" {
					sslMode = "disable"
				}
				return RdbConfig{
					Host:     v.GetString("collector.fcd.rdb.host"),
					Port:     v.GetInt("collector.fcd.rdb.port"),
					Username: v.GetString("collector.fcd.rdb.username"),
					Password: v.GetString("collector.fcd.rdb.password"),
					Database: v.GetString("collector.fcd.rdb.database"),
					SslMode:  sslMode,
				}
			}(),
			Url:             v.GetString("collector.fcd.url"),
			TargetAddresses: v.GetStringSlice("collector.fcd.target_addresses"),
			UntilHeight:     v.GetUint("collector.fcd.until_height"),
		},
	}
}
