package configs

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func Test_LogConfig_EnvVars(t *testing.T) {
	t.Setenv("APP_LOG_ENV", "staging")
	t.Setenv("APP_LOG_CHAINID", "columbus-5")
	t.Setenv("APP_LOG_LEVEL", "debug")
	t.Setenv("APP_LOG_FORMATJSON", "true")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	require.Equal(t, "staging", cfg.Log.Environment)
	require.Equal(t, "columbus-5", cfg.Log.ChainId)
	require.Equal(t, logrus.DebugLevel.String(), cfg.Log.Level)
	require.True(t, cfg.Log.FormatJSON)
}

func Test_RdbConfig_EnvVars(t *testing.T) {
	t.Setenv("APP_LOG_ENV", "local")
	t.Setenv("APP_LOG_CHAINID", "testnet-1")
	t.Setenv("APP_RDB_HOST", "db.example.com")
	t.Setenv("APP_RDB_PORT", "5433")
	t.Setenv("APP_RDB_DATABASE", "mydb")
	t.Setenv("APP_RDB_USERNAME", "user1")
	t.Setenv("APP_RDB_PASSWORD", "secret")
	t.Setenv("APP_RDB_SSLMODE", "require")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	require.Equal(t, "db.example.com", cfg.Rdb.Host)
	require.Equal(t, 5433, cfg.Rdb.Port)
	require.Equal(t, "mydb", cfg.Rdb.Database)
	require.Equal(t, "user1", cfg.Rdb.Username)
	require.Equal(t, "secret", cfg.Rdb.Password)
	require.Equal(t, "require", cfg.Rdb.SslMode)
}

func Test_RdbConfig_Defaults(t *testing.T) {
	t.Setenv("APP_LOG_ENV", "local")
	t.Setenv("APP_LOG_CHAINID", "testnet-1")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	require.Equal(t, defaultRdbConfig.Host, cfg.Rdb.Host)
	require.Equal(t, defaultRdbConfig.Port, cfg.Rdb.Port)
	require.Equal(t, defaultRdbConfig.Database, cfg.Rdb.Database)
	require.Equal(t, defaultRdbConfig.Username, cfg.Rdb.Username)
	require.Equal(t, defaultRdbConfig.Password, cfg.Rdb.Password)
	require.Equal(t, defaultRdbConfig.SslMode, cfg.Rdb.SslMode)
}

func Test_S3Config_EnvVars(t *testing.T) {
	t.Setenv("APP_LOG_ENV", "local")
	t.Setenv("APP_LOG_CHAINID", "testnet-1")
	t.Setenv("APP_S3_BUCKET", "my-bucket")
	t.Setenv("APP_S3_REGION", "us-east-1")
	t.Setenv("APP_S3_KEY", "mykey")
	t.Setenv("APP_S3_SECRET", "mysecret")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	require.Equal(t, "my-bucket", cfg.S3.Bucket)
	require.Equal(t, "us-east-1", cfg.S3.Region)
	require.Equal(t, "mykey", cfg.S3.Key)
	require.Equal(t, "mysecret", cfg.S3.Secret)
}

func Test_SentryConfig_EnvVars(t *testing.T) {
	t.Setenv("APP_LOG_ENV", "local")
	t.Setenv("APP_LOG_CHAINID", "testnet-1")
	t.Setenv("APP_SENTRY_DSN", "https://example@sentry.io/123")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	require.Equal(t, "https://example@sentry.io/123", cfg.Sentry.DSN)
}

func Test_AggregatorConfig_EnvVars(t *testing.T) {
	t.Setenv("APP_LOG_ENV", "local")
	t.Setenv("APP_LOG_CHAINID", "testnet-1")
	t.Setenv("APP_AGGREGATOR_CHAINID", "columbus-5")
	t.Setenv("APP_AGGREGATOR_PRICETOKEN", "uusd")
	t.Setenv("APP_AGGREGATOR_CLEANDUPS", "true")
	t.Setenv("APP_AGGREGATOR_SRCDB_HOST", "src-host")
	t.Setenv("APP_AGGREGATOR_SRCDB_PORT", "5432")
	t.Setenv("APP_AGGREGATOR_SRCDB_DATABASE", "srcdb")
	t.Setenv("APP_AGGREGATOR_SRCDB_USERNAME", "srcuser")
	t.Setenv("APP_AGGREGATOR_SRCDB_PASSWORD", "srcpass")
	t.Setenv("APP_AGGREGATOR_SRCDB_SSLMODE", "require")
	t.Setenv("APP_AGGREGATOR_DESTDB_HOST", "dest-host")
	t.Setenv("APP_AGGREGATOR_DESTDB_PORT", "5433")
	t.Setenv("APP_AGGREGATOR_DESTDB_DATABASE", "destdb")
	t.Setenv("APP_AGGREGATOR_DESTDB_USERNAME", "destuser")
	t.Setenv("APP_AGGREGATOR_DESTDB_PASSWORD", "destpass")
	t.Setenv("APP_AGGREGATOR_DESTDB_SSLMODE", "disable")
	t.Setenv("APP_AGGREGATOR_ROUTER_NAME", "router1")
	t.Setenv("APP_AGGREGATOR_ROUTER_ROUTER_ADDR", "terra1router")
	t.Setenv("APP_AGGREGATOR_ROUTER_MAX_HOP_COUNT", "3")
	t.Setenv("APP_AGGREGATOR_ROUTER_WRITE_DB", "true")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	agg := cfg.Aggregator
	require.Equal(t, "columbus-5", agg.ChainId)
	require.Equal(t, "uusd", agg.PriceToken)
	require.True(t, agg.CleanDups)
	// SrcDb
	require.Equal(t, "src-host", agg.SrcDb.Host)
	require.Equal(t, 5432, agg.SrcDb.Port)
	require.Equal(t, "srcdb", agg.SrcDb.Database)
	require.Equal(t, "srcuser", agg.SrcDb.Username)
	require.Equal(t, "srcpass", agg.SrcDb.Password)
	require.Equal(t, "require", agg.SrcDb.SslMode)
	// DestDb
	require.Equal(t, "dest-host", agg.DestDb.Host)
	require.Equal(t, 5433, agg.DestDb.Port)
	require.Equal(t, "destdb", agg.DestDb.Database)
	require.Equal(t, "destuser", agg.DestDb.Username)
	require.Equal(t, "destpass", agg.DestDb.Password)
	require.Equal(t, "disable", agg.DestDb.SslMode)
	// Router
	require.Equal(t, "router1", agg.Router.Name)
	require.Equal(t, "terra1router", agg.Router.RouterAddr)
	require.Equal(t, uint(3), agg.Router.MaxHopCount)
	require.True(t, agg.Router.WriteDb)
}

func Test_CollectorConfig_EnvVars(t *testing.T) {
	t.Setenv("APP_LOG_ENV", "local")
	t.Setenv("APP_LOG_CHAINID", "testnet-1")
	t.Setenv("APP_COLLECTOR_CHAINID", "columbus-5")
	t.Setenv("APP_COLLECTOR_PAIR_FACTORY_CONTRACT_ADDRESS", "terra1factory")
	t.Setenv("APP_COLLECTOR_NODE_GRPC_HOST", "grpc.example.com")
	t.Setenv("APP_COLLECTOR_NODE_GRPC_PORT", "9090")
	t.Setenv("APP_COLLECTOR_NODE_FAILOVER_LCD_HOST", "lcd-failover.example.com")
	t.Setenv("APP_COLLECTOR_FCD_RDB_HOST", "fcd-host")
	t.Setenv("APP_COLLECTOR_FCD_RDB_PORT", "5432")
	t.Setenv("APP_COLLECTOR_FCD_RDB_USERNAME", "fcduser")
	t.Setenv("APP_COLLECTOR_FCD_RDB_PASSWORD", "fcdpass")
	t.Setenv("APP_COLLECTOR_FCD_RDB_DATABASE", "fcddb")
	t.Setenv("APP_COLLECTOR_FCD_RDB_SSLMODE", "require")
	t.Setenv("APP_COLLECTOR_FCD_URL", "https://fcd.example.com")
	t.Setenv("APP_COLLECTOR_FCD_UNTIL_HEIGHT", "1000")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	col := cfg.Collector
	require.Equal(t, "columbus-5", col.ChainId)
	require.Equal(t, "terra1factory", col.PairFactoryContractAddress)
	require.Equal(t, "grpc.example.com", col.NodeConfig.GrpcConfig.Host)
	require.Equal(t, 9090, col.NodeConfig.GrpcConfig.Port)
	require.Equal(t, "lcd-failover.example.com", col.NodeConfig.FailoverLcdHost)
	require.Equal(t, "fcd-host", col.FcdConfig.RdbConfig.Host)
	require.Equal(t, 5432, col.FcdConfig.RdbConfig.Port)
	require.Equal(t, "fcduser", col.FcdConfig.RdbConfig.Username)
	require.Equal(t, "fcdpass", col.FcdConfig.RdbConfig.Password)
	require.Equal(t, "fcddb", col.FcdConfig.RdbConfig.Database)
	require.Equal(t, "require", col.FcdConfig.RdbConfig.SslMode)
	require.Equal(t, "https://fcd.example.com", col.FcdConfig.Url)
	require.Equal(t, uint(1000), col.FcdConfig.UntilHeight)
}

func Test_ParserConfig_EnvVars(t *testing.T) {
	t.Setenv("APP_LOG_ENV", "local")
	t.Setenv("APP_LOG_CHAINID", "testnet-1")
	t.Setenv("APP_PARSER_DEX_CHAINID", "columbus-5")
	t.Setenv("APP_PARSER_DEX_FACTORYADDRESS", "terra1factory")
	t.Setenv("APP_PARSER_DEX_SAMEHEIGHTTOLERANCE", "5")
	t.Setenv("APP_PARSER_DEX_ERRTOLERANCE", "2")
	t.Setenv("APP_PARSER_DEX_NODE_GRPC_HOST", "grpc.example.com")
	t.Setenv("APP_PARSER_DEX_NODE_GRPC_PORT", "9090")
	t.Setenv("APP_PARSER_DEX_NODE_FAILOVER_LCD_HOST", "lcd-failover.example.com")
	t.Setenv("APP_PARSER_DEX_NODE_REST_LCD", "https://lcd.example.com")
	t.Setenv("APP_PARSER_DEX_NODE_REST_RPC", "https://rpc.example.com")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	p := cfg.Parser.DexConfig
	require.Equal(t, "columbus-5", p.ChainId)
	require.Equal(t, "terra1factory", p.FactoryAddress)
	require.Equal(t, uint(5), p.SameHeightTolerance)
	require.Equal(t, uint(2), p.ErrTolerance)
	require.Equal(t, "grpc.example.com", p.NodeConfig.GrpcConfig.Host)
	require.Equal(t, 9090, p.NodeConfig.GrpcConfig.Port)
	require.Equal(t, "lcd-failover.example.com", p.NodeConfig.FailoverLcdHost)
	require.Equal(t, "https://lcd.example.com", p.NodeConfig.RestClientConfig.LcdHost)
	require.Equal(t, "https://rpc.example.com", p.NodeConfig.RestClientConfig.RpcHost)
}

func Test_HttpClientConfig_EnvVars(t *testing.T) {
	t.Setenv("APP_PARSER_DEX_NODE_HTTP_MAXIDLECONNS", "50")
	t.Setenv("APP_PARSER_DEX_NODE_HTTP_MAXIDLECONNSPERHOST", "10")
	t.Setenv("APP_PARSER_DEX_NODE_HTTP_IDLECONNTIMEOUT", "60s")
	t.Setenv("APP_PARSER_DEX_NODE_HTTP_TIMEOUT", "30s")

	// required fields to prevent panic
	t.Setenv("APP_LOG_ENV", "local")
	t.Setenv("APP_LOG_CHAINID", "testnet-1")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	http := cfg.Parser.DexConfig.NodeConfig.HttpClientConfig
	require.Equal(t, 50, http.MaxIdleConns)
	require.Equal(t, 10, http.MaxIdleConnsPerHost)
	require.Equal(t, &Duration{60 * time.Second}, http.IdleConnTimeout)
	require.Equal(t, &Duration{30 * time.Second}, http.Timeout)
}

func Test_HttpClientConfig_Defaults(t *testing.T) {
	t.Setenv("APP_LOG_ENV", "local")
	t.Setenv("APP_LOG_CHAINID", "testnet-1")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	http := cfg.Collector.NodeConfig.HttpClientConfig
	require.Equal(t, defaultHttpClientConfig.MaxIdleConns, http.MaxIdleConns)
	require.Equal(t, defaultHttpClientConfig.MaxIdleConnsPerHost, http.MaxIdleConnsPerHost)
	require.Equal(t, defaultHttpClientConfig.IdleConnTimeout, http.IdleConnTimeout)
	require.Equal(t, defaultHttpClientConfig.Timeout, http.Timeout)

	http = cfg.Parser.DexConfig.NodeConfig.HttpClientConfig
	require.Equal(t, defaultHttpClientConfig.MaxIdleConns, http.MaxIdleConns)
	require.Equal(t, defaultHttpClientConfig.MaxIdleConnsPerHost, http.MaxIdleConnsPerHost)
	require.Equal(t, defaultHttpClientConfig.IdleConnTimeout, http.IdleConnTimeout)
	require.Equal(t, defaultHttpClientConfig.Timeout, http.Timeout)
}

func withTestBasepath(t *testing.T, dir string) func() {
	t.Helper()
	original := basepath
	basepath = dir
	return func() { basepath = original }
}

func Test_New_EnvOnly(t *testing.T) {
	expectedEnv := "local"
	expectedChainID := "testnet-1"

	t.Setenv("APP_LOG_ENV", expectedEnv)
	t.Setenv("APP_LOG_CHAINID", expectedChainID)
	t.Setenv("APP_AGGREGATOR_SRCDB_SSLMODE", "require")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	require.Equal(t, expectedEnv, cfg.Log.Environment)
	require.Equal(t, expectedChainID, cfg.Log.ChainId)
	require.Equal(t, "require", cfg.Aggregator.SrcDb.SslMode)
}

func Test_New_NoFile_NoEnv(t *testing.T) {
	t.Setenv("APP_LOG_ENV", "")
	t.Setenv("APP_LOG_CHAINID", "")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	require.Panics(t, func() { _ = New() })
}

func TestConfig_Redacted(t *testing.T) {
	expected := "***"

	cfg := Config{
		Collector: CollectorConfig{
			FcdConfig: FcdConfig{
				RdbConfig: RdbConfig{
					Password: "original-collector-pw",
				},
			},
		},
		S3: S3Config{
			Secret: "original-s3-secret",
		},
		Rdb: RdbConfig{
			Password: "original-parser-pw",
		},
		Aggregator: AggregatorConfig{
			SrcDb: RdbConfig{
				Password: "src-db-password",
			},
			DestDb: RdbConfig{
				Password: "dest-db-password",
			},
		},
	}

	redacted := cfg.Redacted()

	// all sensitive fields must be redacted
	require.Equal(t, expected, redacted.Collector.FcdConfig.RdbConfig.Password)
	require.Equal(t, expected, redacted.S3.Secret)
	require.Equal(t, expected, redacted.Rdb.Password)
	require.Equal(t, expected, redacted.Aggregator.SrcDb.Password)
	require.Equal(t, expected, redacted.Aggregator.DestDb.Password)

	// ensure original config is not modified
	require.Equal(t, "original-collector-pw", cfg.Collector.FcdConfig.RdbConfig.Password)
	require.Equal(t, "original-s3-secret", cfg.S3.Secret)
	require.Equal(t, "original-parser-pw", cfg.Rdb.Password)
	require.Equal(t, "src-db-password", cfg.Aggregator.SrcDb.Password)
	require.Equal(t, "dest-db-password", cfg.Aggregator.DestDb.Password)
}
