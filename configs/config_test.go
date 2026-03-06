package configs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
	require.Equal(t, Duration{60 * time.Second}, http.IdleConnTimeout)
	require.Equal(t, Duration{30 * time.Second}, http.Timeout)
}

func Test_HttpClientConfig_Defaults(t *testing.T) {
	t.Setenv("APP_LOG_ENV", "local")
	t.Setenv("APP_LOG_CHAINID", "testnet-1")

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	http := cfg.Parser.DexConfig.NodeConfig.HttpClientConfig
	require.Equal(t, 20, http.MaxIdleConns)
	require.Equal(t, 5, http.MaxIdleConnsPerHost)
	require.Equal(t, Duration{30 * time.Second}, http.IdleConnTimeout)
	require.Equal(t, Duration{0}, http.Timeout)
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
