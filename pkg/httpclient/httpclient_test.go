package httpclient

import (
	"net/http"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/stretchr/testify/require"
)

func TestNewAppliesHTTPClientConfiguration(t *testing.T) {
	client := New(configs.HttpClientConfig{
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     &configs.Duration{Duration: 30 * time.Second},
		DisableKeepAlives:   true,
		ForceAttemptHTTP2:   true,
		Timeout:             &configs.Duration{Duration: 5 * time.Second},
	})

	require.Equal(t, 5*time.Second, client.Timeout)
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.Equal(t, 20, transport.MaxIdleConns)
	require.Equal(t, 5, transport.MaxIdleConnsPerHost)
	require.Equal(t, 30*time.Second, transport.IdleConnTimeout)
	require.True(t, transport.DisableKeepAlives)
	require.True(t, transport.ForceAttemptHTTP2)
}

func TestNewAllowsOmittedDurations(t *testing.T) {
	client := New(configs.HttpClientConfig{})

	require.Zero(t, client.Timeout)
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.Zero(t, transport.IdleConnTimeout)
}
