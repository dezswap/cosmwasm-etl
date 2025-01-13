package grpc

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/sirupsen/logrus"
)

var logger logging.Logger

func init() {
	logger = logging.New(
		"grpc",
		configs.LogConfig{
			Level:      logrus.InfoLevel,
			FormatJSON: true,
		},
	)
}

func SetLogConfig(c configs.LogConfig) {
	logger = logging.New("grpc", c)
}
