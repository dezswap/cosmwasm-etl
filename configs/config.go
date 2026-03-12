package configs

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	defaultConfigFileName = "config"
	envPrefix             = "app"
)

var defaultConfig = Config{
	Collector: CollectorConfig{
		NodeConfig: NodeConfig{
			HttpClientConfig: defaultHttpClientConfig,
		},
	},
	Parser: ParserConfig{
		DexConfig: ParserDexConfig{
			NodeConfig: NodeConfig{
				HttpClientConfig: defaultHttpClientConfig,
			},
		},
	},
	Rdb: defaultRdbConfig,
}

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

// Config aggregation
type Config struct {
	Aggregator AggregatorConfig `mapstructure:"aggregator"`
	Collector  CollectorConfig  `mapstructure:"collector"`
	Parser     ParserConfig     `mapstructure:"parser"`
	Log        LogConfig        `mapstructure:"log"`
	Sentry     SentryConfig     `mapstructure:"sentry"`
	Rdb        RdbConfig        `mapstructure:"rdb"`
	S3         S3Config         `mapstructure:"s3"`
}

// Init is explicit initializer for Config
func New() Config {
	v, err := initViper(defaultConfigFileName)
	if err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			panic(err)
		}
	}

	var cfg = defaultConfig
	if err = v.Unmarshal(&cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.TextUnmarshallerHookFunc(),
		),
	)); err != nil {
		panic(fmt.Errorf("unmarshal config: %w", err))
	}

	// common configuration for collector/parser/aggregator
	// check env variables have been set when no file exists
	if cfg.Log.ChainId == "" || cfg.Log.Environment == "" {
		panic(fmt.Errorf("APP_LOG_CHAINID or APP_LOG_ENV is not set"))
	}

	return cfg
}

func NewWithFileName(fileName string) Config {
	v, err := initViper(fileName)
	if err != nil {
		panic(err)
	}

	var cfg = defaultConfig
	if err = v.Unmarshal(&cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.TextUnmarshallerHookFunc(),
		),
	)); err != nil {
		panic(fmt.Errorf("unmarshal config: %w", err))
	}

	return cfg
}

func initViper(configName string) (*viper.Viper, error) {
	v := viper.NewWithOptions(viper.ExperimentalBindStruct())

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

func (c Config) Redacted() Config {
	cp := c

	// Collector
	cp.Collector.FcdConfig.RdbConfig.Password = "***"
	cp.S3.Secret = "***"

	// Parser
	cp.Rdb.Password = "***"

	// Aggregator
	cp.Aggregator.SrcDb.Password = "***"
	cp.Aggregator.DestDb.Password = "***"

	return cp
}

func (cfg Config) Pretty() string {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Sprintf("config(marshal_error=%v)", err)
	}
	return string(b)
}
