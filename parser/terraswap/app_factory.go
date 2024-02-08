package terraswap

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/terraswap/columbus_v1"
	"github.com/dezswap/cosmwasm-etl/parser/terraswap/phoenix"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	ts "github.com/dezswap/cosmwasm-etl/pkg/rules/terraswap"

	"github.com/pkg/errors"
)

func New(repo parser.PairRepo, logger logging.Logger, c configs.ParserConfig) (parser.TargetApp, error) {
	switch ts.TerraswapTypeOf(c.FactoryAddress) {
	case ts.Mainnet:
		return phoenix.New(repo, logger, c)
	case ts.ClassicV2, ts.Pisco:
		return nil, errors.New("not implemented yet")
	case ts.ClassicV1:
		return columbus_v1.New(repo, logger, c)
	default:
		return nil, errors.Errorf("invalid factory address: %s", c.FactoryAddress)
	}
}
