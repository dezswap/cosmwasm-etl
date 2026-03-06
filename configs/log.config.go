package configs

import "github.com/sirupsen/logrus"

const defaultLogLevel = logrus.InfoLevel

type LogConfig struct {
	// Log level for the global `Logger`
	Level string `mapstructure:"level"`
	// Should log-messages be printed as JSON?
	FormatJSON bool `mapstructure:"formatjson"`
	// The environment the service is currently running in, e.g. local/development/staging
	Environment string `mapstructure:"env"`
	ChainId     string `mapstructure:"chainid"`
}

func (c LogConfig) ParsedLevel() logrus.Level {
	l, err := logrus.ParseLevel(c.Level)
	if err != nil {
		return defaultLogLevel
	}

	return l
}
