package configs

import (
	"github.com/dezswap/cosmwasm-etl/pkg/rules"
	"github.com/spf13/viper"
)

const (
	PARSER_SAME_HEIGHT_TOLERANCE = 3
)

type ParserConfig struct {
	ChainId             string
	TargetApp           rules.RuleType
	SameHeightTolerance uint
	ErrTolerance        uint

	NodeConfig NodeConfig
}

func parserConfig(v *viper.Viper) ParserConfig {

	v.SetDefault("parser.sameHeightTolerance", PARSER_SAME_HEIGHT_TOLERANCE)
	return ParserConfig{
		ChainId:             v.GetString("parser.chainId"),
		TargetApp:           rules.ToRuleType(v.GetString("parser.targetApp")),
		SameHeightTolerance: v.GetUint("parser.sameHeightTolerance"),
		ErrTolerance:        v.GetUint("parser.errTolerance"),

		NodeConfig: NodeConfig{
			GrpcConfig:      grpcConfig(v, "parser.node"),
			FailoverLcdHost: v.GetString("parser.node.failover_lcd_host"),
		},
	}
}
