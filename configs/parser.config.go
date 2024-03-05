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
	DexConfig *ParserDexConfig
}

func parserConfig(v *viper.Viper) ParserConfig {
	dex := parserDexConfig(v)
	return ParserConfig{
		DexConfig: dex,
	}
}

type ParserDexConfig struct {
	ChainId             string
	FactoryAddress      string
	TargetApp           dex.DexType
	SameHeightTolerance uint
	ErrTolerance        uint

	PoolSnapshotInterval uint
	ValidationInterval   uint

	NodeConfig NodeConfig
}

func parserDexConfig(v *viper.Viper) *ParserDexConfig {
	if v.Get("parser.dex") == nil {
		return nil
	}

	v.SetDefault("parser.dex.sameHeightTolerance", PARSER_SAME_HEIGHT_TOLERANCE)
	v.SetDefault("parser.dex.poolSnapshotInterval", PARSER_POOL_SNAPSHOT_INTERVAL)
	v.SetDefault("parser.dex.validationInterval", PARSER_VALIDATION_INTERVAL)
	return &ParserDexConfig{
		ChainId:             v.GetString("parser.dex.chainId"),
		FactoryAddress:      v.GetString("parser.dex.factoryAddress"),
		TargetApp:           dex.ToDexType(v.GetString("parser.dex.targetApp")),
		SameHeightTolerance: v.GetUint("parser.dex.sameHeightTolerance"),
		ErrTolerance:        v.GetUint("parser.dex.errTolerance"),

		PoolSnapshotInterval: v.GetUint("parser.dex.poolSnapshotInterval"),
		ValidationInterval:   v.GetUint("parser.dex.validationInterval"),

		NodeConfig: NodeConfig{
			GrpcConfig:      grpcConfig(v, "parser.dex.node"),
			FailoverLcdHost: v.GetString("parser.dex.node.failover_lcd_host"),

			RestClientConfig: RestClientConfig{
				LcdHost: v.GetString("parser.dex.node.rest.lcd"),
				RpcHost: v.GetString("parser.dex.node.rest.rpc"),
			},
		},
	}
}
