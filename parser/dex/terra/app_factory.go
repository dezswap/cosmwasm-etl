package terra

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/parser/dex/terra/classicv1"
	"github.com/dezswap/cosmwasm-etl/parser/dex/terra/classicv2"
	"github.com/dezswap/cosmwasm-etl/parser/dex/terra/terra2"
	terra "github.com/dezswap/cosmwasm-etl/pkg/dex/terra"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
)

func New(repo dex.PairRepo, logger logging.Logger, c configs.ParserDexConfig) (dex.TargetApp, error) {
	switch terra.Factory(c.FactoryAddress) {
	case terra.MainnetFactory, terra.PiscoFactory:
		return terra2.New(repo, logger, c)
	case terra.ClassicV2Factory:
		return classicv2.New(repo, logger, c)
	case terra.ClassicV1Factory:
		return classicv1.New(repo, logger, c)
	default:
		return nil, errors.Errorf("invalid factory address: %s", c.FactoryAddress)
	}
}
