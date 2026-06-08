package configs

import (
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/pkg/errors"
)

const (
	PARSER_SAME_HEIGHT_TOLERANCE  = 3
	PARSER_POOL_SNAPSHOT_INTERVAL = 1000
	PARSER_VALIDATION_INTERVAL    = 1000
)

type QuarantineRetryMode string

const (
	QuarantineRetryDisabled QuarantineRetryMode = "disabled"
	QuarantineRetryStartup  QuarantineRetryMode = "startup"
	QuarantineRetryEveryRun QuarantineRetryMode = "every_run"
)

type ParserConfig struct {
	DexConfig ParserDexConfig `mapstructure:"dex"`
}

type ParserDexConfig struct {
	ChainId              string              `mapstructure:"chainid"`
	FactoryAddress       string              `mapstructure:"factoryaddress"`
	TargetApp            dex.DexType         `mapstructure:"targetapp"`
	SameHeightTolerance  uint                `mapstructure:"sameheighttolerance"`
	ErrTolerance         uint                `mapstructure:"errtolerance"`
	PoolSnapshotInterval uint                `mapstructure:"poolsnapshotinterval"`
	ValidationInterval   uint                `mapstructure:"validationinterval"`
	QuarantineRetryMode  QuarantineRetryMode `mapstructure:"quarantineretrymode"`
	NodeConfig           NodeConfig          `mapstructure:"node"`
}

func (c ParserDexConfig) Validate() error {
	if c.ChainId == "" || c.FactoryAddress == "" || c.TargetApp == dex.Unknown {
		return errors.New("required field is missing.")
	}
	if c.QuarantineRetryMode == "" {
		return nil
	}
	switch c.QuarantineRetryMode {
	case QuarantineRetryDisabled, QuarantineRetryStartup, QuarantineRetryEveryRun:
	default:
		return errors.Errorf("invalid quarantine retry mode(%s)", c.QuarantineRetryMode)
	}

	return nil
}
