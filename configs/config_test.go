package configs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

	tmp := t.TempDir()
	defer withTestBasepath(t, tmp)()

	cfg := New()
	require.Equal(t, expectedEnv, cfg.Log.Environment)
	require.Equal(t, expectedChainID, cfg.Log.ChainId)
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
