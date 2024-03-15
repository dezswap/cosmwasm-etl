package configs

import (
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/spf13/viper"
)

const (
	PARSER_SAME_HEIGHT_TOLERANCE  = 3
	PARSER_POOL_SNAPSHOT_INTERVAL = 100
	PARSER_VALIDATION_INTERVAL    = 100
)

type ParserConfig struct {
	ChainId             string
	FactoryAddress      string
	TargetApp           dex.DexType
	SameHeightTolerance uint
	ErrTolerance        uint

	PoolSnapshotInterval uint
	ValidationInterval   uint

	NodeConfig NodeConfig
}

func parserConfig(v *viper.Viper) ParserConfig {

	v.SetDefault("parser.sameHeightTolerance", PARSER_SAME_HEIGHT_TOLERANCE)
	v.SetDefault("parser.poolSnapshotInterval", PARSER_POOL_SNAPSHOT_INTERVAL)
	v.SetDefault("parser.validationInterval", PARSER_VALIDATION_INTERVAL)
	return ParserConfig{
		ChainId:             v.GetString("parser.chainId"),
		FactoryAddress:      v.GetString("parser.factoryAddress"),
		TargetApp:           dex.ToDexType(v.GetString("parser.targetApp")),
		SameHeightTolerance: v.GetUint("parser.sameHeightTolerance"),
		ErrTolerance:        v.GetUint("parser.errTolerance"),

		PoolSnapshotInterval: v.GetUint("parser.poolSnapshotInterval"),
		ValidationInterval:   v.GetUint("parser.validationInterval"),

		NodeConfig: NodeConfig{
			GrpcConfig:      grpcConfig(v, "parser.node"),
			FailoverLcdHost: v.GetString("parser.node.failover_lcd_host"),

			RestClientConfig: RestClientConfig{
				LcdHost: v.GetString("parser.node.rest.lcd"),
				RpcHost: v.GetString("parser.node.rest.rpc"),
			},
		},
	}
}
