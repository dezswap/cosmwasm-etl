package terraswap

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/parser/dex/terraswap/columbus_v1"
	"github.com/dezswap/cosmwasm-etl/parser/dex/terraswap/phoenix"
	ts "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
)

func New(repo dex.PairRepo, logger logging.Logger, c configs.ParserConfig) (dex.TargetApp, error) {
	switch ts.TerraswapFactory(c.FactoryAddress) {
	case ts.MAINNET_FACTORY:
		return phoenix.New(repo, logger, c)
	case ts.CLASSIC_V2_FACTORY, ts.PISCO_FACTORY:
		return nil, errors.New("not implemented yet")
	case ts.CLASSIC_V1_FACTORY:
		return columbus_v1.New(repo, logger, c)
	default:
		return nil, errors.Errorf("invalid factory address: %s", c.FactoryAddress)
	}
}
