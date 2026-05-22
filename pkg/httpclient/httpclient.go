package httpclient

import (
	"net/http"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
)

func New(c configs.HttpClientConfig) *http.Client {
	var timeout time.Duration
	if c.Timeout != nil {
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
