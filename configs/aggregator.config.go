package configs

import (
	"time"
)

const DefaultTaskWaitTimeout = 30 * time.Minute

type AggregatorConfig struct {
	ChainId         string        `mapstructure:"chainid"`
	PriceToken      string        `mapstructure:"pricetoken"`
	StartTs         time.Time     `mapstructure:"startts"`
	CleanDups       bool          `mapstructure:"cleandups"`
	TaskWaitTimeout time.Duration `mapstructure:"taskwaittimeout"`
	SrcDb           RdbConfig     `mapstructure:"srcdb"`
	DestDb          RdbConfig     `mapstructure:"destdb"`
	Router          RouterConfig  `mapstructure:"router"`
}

// defaultAggregatorConfig returns aggregator defaults that are not covered by
// zero values.
func defaultAggregatorConfig() AggregatorConfig {
	return AggregatorConfig{
		TaskWaitTimeout: DefaultTaskWaitTimeout,
	}
}
