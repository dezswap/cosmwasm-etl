package configs

import (
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/pkg/errors"
)

const (
	PARSER_SAME_HEIGHT_TOLERANCE  = 3
	PARSER_POOL_SNAPSHOT_INTERVAL = 100
	PARSER_VALIDATION_INTERVAL    = 100
)

type ParserConfig struct {
	DexConfig ParserDexConfig `mapstructure:"dex"`
}

type ParserDexConfig struct {
	ChainId              string      `mapstructure:"chainid"`
	FactoryAddress       string      `mapstructure:"factoryaddress"`
	TargetApp            dex.DexType `mapstructure:"targetapp"`
	SameHeightTolerance  uint        `mapstructure:"sameheighttolerance"`
	ErrTolerance         uint        `mapstructure:"errtolerance"`
	PoolSnapshotInterval uint        `mapstructure:"poolsnapshotinterval"`
	ValidationInterval   uint        `mapstructure:"validationinterval"`
	NodeConfig           NodeConfig  `mapstructure:"node"`
}

func (c ParserDexConfig) Validate() error {
	if c.ChainId == "" || c.FactoryAddress == "" || c.TargetApp == dex.Unknown {
		return errors.New("required field is missing.")
	}

	return nil
}
