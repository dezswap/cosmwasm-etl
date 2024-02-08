package terraswap

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/terraswap/columbus_v1"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

func New(repo parser.PairRepo, logger logging.Logger, c configs.ParserConfig) (parser.TargetApp, error) {
	return columbus_v1.New(repo, logger, c)
}
