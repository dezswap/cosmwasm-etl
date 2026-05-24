package configs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCollectorNodeConfigWithFallbackKeepsCollectorEndpointsWhenConfigured(t *testing.T) {
	collectorConfig := CollectorConfig{NodeConfig: testCollectorNodeConfig("collector-rpc", "collector-lcd", time.Second)}
	fallback := testCollectorNodeConfig("parser-rpc", "parser-lcd", 2*time.Second)

	actual := collectorConfig.NodeConfigWithFallback(fallback)

	require.Equal(t, collectorConfig.NodeConfig, actual)
}

func TestCollectorNodeConfigWithFallbackUsesEntireFallbackWhenCollectorEndpointsAreEmpty(t *testing.T) {
	collectorConfig := CollectorConfig{NodeConfig: testCollectorNodeConfig("", "", time.Second)}
	fallback := testCollectorNodeConfig("parser-rpc", "parser-lcd", 2*time.Second)

	actual := collectorConfig.NodeConfigWithFallback(fallback)

	require.Equal(t, fallback, actual)
}

func TestCollectorNodeConfigWithFallbackFillsMissingCollectorRPCEndpoint(t *testing.T) {
	collectorConfig := CollectorConfig{NodeConfig: testCollectorNodeConfig("", "collector-lcd", time.Second)}
	fallback := testCollectorNodeConfig("parser-rpc", "parser-lcd", 2*time.Second)

	actual := collectorConfig.NodeConfigWithFallback(fallback)

	require.Equal(t, "parser-rpc", actual.RestClientConfig.RpcHost)
	require.Equal(t, "collector-lcd", actual.RestClientConfig.LcdHost)
	require.Equal(t, collectorConfig.NodeConfig.HttpClientConfig, actual.HttpClientConfig)
}

func TestCollectorNodeConfigWithFallbackFillsMissingCollectorLCDEndpoint(t *testing.T) {
	collectorConfig := CollectorConfig{NodeConfig: testCollectorNodeConfig("collector-rpc", "", time.Second)}
	fallback := testCollectorNodeConfig("parser-rpc", "parser-lcd", 2*time.Second)

	actual := collectorConfig.NodeConfigWithFallback(fallback)

	require.Equal(t, "collector-rpc", actual.RestClientConfig.RpcHost)
	require.Equal(t, "parser-lcd", actual.RestClientConfig.LcdHost)
	require.Equal(t, collectorConfig.NodeConfig.HttpClientConfig, actual.HttpClientConfig)
}

func testCollectorNodeConfig(rpcHost string, lcdHost string, timeout time.Duration) NodeConfig {
	return NodeConfig{
		RestClientConfig: RestClientConfig{RpcHost: rpcHost, LcdHost: lcdHost},
		HttpClientConfig: HttpClientConfig{
			Timeout:         &Duration{Duration: timeout},
			IdleConnTimeout: &Duration{Duration: timeout},
		},
	}
}
