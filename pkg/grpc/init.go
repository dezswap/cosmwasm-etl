package grpc

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

var logger logging.Logger

func init() {
	// FIXME: Log config should come from the file
	logger = logging.New("common", configs.LogConfig{})
}
