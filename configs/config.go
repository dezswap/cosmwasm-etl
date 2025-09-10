package configs

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	fileName  = "config"
	envPrefix = "app"
)

var envConfig Config
var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

// Config aggregation
type Config struct {
	Aggregator AggregatorConfig
	Collector  CollectorConfig
	Parser     ParserConfig
	Log        LogConfig
	Sentry     SentryConfig
	Rdb        RdbConfig
	S3         S3Config
}

// Init is explicit initializer for Config
func New() Config {
	v, err := initViper(fileName)
	if err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			panic(err)
		}
	}

	envConfig = Config{
		Aggregator: aggregatorConfig(v),
		Collector:  collectorConfig(v),
		Parser:     parserConfig(v),
		Log:        logConfig(v),
		Sentry:     sentryConfig(v),
		Rdb:        rdbConfig(v),
		S3:         s3Config(v),
	}

	// common configuration for collector/parser/aggregator
	// check env variables have been set when no file exists
	if envConfig.Log.ChainId == "" || envConfig.Log.Environment == "" {
		panic(err)
	}

	return envConfig
}

func NewWithFileName(fileName string) Config {
	v, err := initViper(fileName)
	if err != nil {
		panic(err)
	}

	envConfig = Config{
		Aggregator: aggregatorConfig(v),
		Collector:  collectorConfig(v),
		Parser:     parserConfig(v),
		Log:        logConfig(v),
		Sentry:     sentryConfig(v),
		Rdb:        rdbConfig(v),
		S3:         s3Config(v),
	}
	return envConfig
}

// Get returns Config object
func Get() Config {
	return envConfig
}

func initViper(configName string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigName(configName)

	if basepath == "" {
		return nil, errors.New("package root path is not initialized")
	}
	v.AddConfigPath(fmt.Sprintf("%s/../", basepath))
	v.AddConfigPath(".") // optionally look for config in the working directory

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// All env vars starts with APP_
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		// check read fails once loading env var load
		return v, err
	}

	return v, nil
}
