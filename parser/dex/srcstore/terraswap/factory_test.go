package terraswap

import (
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	dts "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/stretchr/testify/require"
)

func TestNewFromConfigCreatesSupportedStores(t *testing.T) {
	tests := []struct {
		name    string
		factory dts.TerraswapFactory
		assert  func(*testing.T, interface{})
	}{
		{
			name:    "phoenix",
			factory: dts.MAINNET_FACTORY,
			assert: func(t *testing.T, store interface{}) {
				require.IsType(t, &phoenixSourceDataStore{}, store)
			},
		},
		{
			name:    "columbus v2",
			factory: dts.CLASSIC_V2_FACTORY,
			assert: func(t *testing.T, store interface{}) {
				require.IsType(t, &baseRawDataStoreImpl{}, store)
			},
		},
		{
			name:    "columbus v1",
			factory: dts.CLASSIC_V1_FACTORY,
			assert: func(t *testing.T, store interface{}) {
				require.IsType(t, &baseRawDataStoreImpl{}, store)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store, err := NewFromConfig(factoryNodeConfig(), string(tc.factory))

			require.NoError(t, err)
			tc.assert(t, store)
		})
	}
}

func TestNewFromConfigRejectsUnsupportedFactory(t *testing.T) {
	_, err := NewFromConfig(factoryNodeConfig(), string(dts.PISCO_FACTORY))

	require.EqualError(t, err, "not implemented yet")
}

func TestNewFromConfigRejectsUnknownFactory(t *testing.T) {
	_, err := NewFromConfig(factoryNodeConfig(), "terra1unknown")

	require.EqualError(t, err, "invalid factory address: terra1unknown")
}

func factoryNodeConfig() configs.NodeConfig {
	return configs.NodeConfig{
		HttpClientConfig: configs.HttpClientConfig{
			Timeout:         &configs.Duration{Duration: time.Second},
			IdleConnTimeout: &configs.Duration{Duration: time.Second},
		},
		RestClientConfig: configs.RestClientConfig{
			RpcHost: "http://rpc",
			LcdHost: "http://lcd",
		},
	}
}
