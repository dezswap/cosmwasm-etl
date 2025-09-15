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
