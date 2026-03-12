package configs

import (
	"time"
)

type AggregatorConfig struct {
	ChainId    string       `mapstructure:"chainid"`
	PriceToken string       `mapstructure:"pricetoken"`
	StartTs    time.Time    `mapstructure:"startts"`
	CleanDups  bool         `mapstructure:"cleandups"`
	SrcDb      RdbConfig    `mapstructure:"srcdb"`
	DestDb     RdbConfig    `mapstructure:"destdb"`
	Router     RouterConfig `mapstructure:"router"`
}
