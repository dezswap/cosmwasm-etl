package httpclient

import (
	"net/http"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
)

// DefaultTimeout applies when config leaves the timeout unset.
const DefaultTimeout = 30 * time.Second

func New(c configs.HttpClientConfig) *http.Client {
	timeout := DefaultTimeout
	if c.Timeout != nil && c.Timeout.Duration > 0 {
		timeout = c.Timeout.Duration
	}

	var idleConnTimeout time.Duration
	if c.IdleConnTimeout != nil {
		idleConnTimeout = c.IdleConnTimeout.Duration
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        c.MaxIdleConns,
			MaxIdleConnsPerHost: c.MaxIdleConnsPerHost,
			IdleConnTimeout:     idleConnTimeout,
			DisableKeepAlives:   c.DisableKeepAlives,
			ForceAttemptHTTP2:   c.ForceAttemptHTTP2,
		},
	}
}
