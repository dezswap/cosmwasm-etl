package terraswap

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/parser/dex/terraswap/columbusv1"
	"github.com/dezswap/cosmwasm-etl/parser/dex/terraswap/columbusv2"
	"github.com/dezswap/cosmwasm-etl/parser/dex/terraswap/phoenix"
	ts "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
)

func New(repo dex.PairRepo, logger logging.Logger, c configs.ParserDexConfig) (dex.TargetApp, error) {
	switch ts.TerraswapFactory(c.FactoryAddress) {
	case ts.MAINNET_FACTORY, ts.PISCO_FACTORY:
		return phoenix.New(repo, logger, c)
	case ts.CLASSIC_V2_FACTORY:
		return columbusv2.New(repo, logger, c)
	case ts.CLASSIC_V1_FACTORY:
		return columbusv1.New(repo, logger, c)
	default:
		return nil, errors.Errorf("invalid factory address: %s", c.FactoryAddress)
	}
}
