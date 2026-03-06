package configs

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type NodeConfig struct {
	GrpcConfig      GrpcConfig
	FailoverLcdHost string

	RestClientConfig RestClientConfig
	HttpClientConfig HttpClientConfig
}

type HttpClientConfig struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     Duration
	DisableKeepAlives   bool
	ForceAttemptHTTP2   bool
	Timeout             Duration
}

type GrpcConfig struct {
	Host         string
	Port         int
	BackoffDelay Duration
	NoTLS        bool
}

type RestClientConfig struct {
	LcdHost string
	RpcHost string
}

// Duration is wrapper type for custom unmarshalling
type Duration struct {
	time.Duration
}

func httpClientConfig(v *viper.Viper, prefix string) HttpClientConfig {
	if prefix != "" {
		prefix += "."
	}
	v.SetDefault(prefix+"http.maxIdleConns", 20)
	v.SetDefault(prefix+"http.maxIdleConnsPerHost", 5)
	v.SetDefault(prefix+"http.idleConnTimeout", "30s")

	idleConnTimeout, err := time.ParseDuration(v.GetString(prefix + "http.idleConnTimeout"))
	if err != nil {
		idleConnTimeout = 30 * time.Second
	}
	timeout, _ := time.ParseDuration(v.GetString(prefix + "http.timeout"))

	return HttpClientConfig{
		MaxIdleConns:        v.GetInt(prefix + "http.maxIdleConns"),
		MaxIdleConnsPerHost: v.GetInt(prefix + "http.maxIdleConnsPerHost"),
		IdleConnTimeout:     Duration{idleConnTimeout},
		DisableKeepAlives:   v.GetBool(prefix + "http.disableKeepAlives"),
		ForceAttemptHTTP2:   v.GetBool(prefix + "http.forceAttemptHTTP2"),
		Timeout:             Duration{timeout},
	}
}

func grpcConfig(v *viper.Viper, prefix string) GrpcConfig {
	if prefix != "" {
		prefix += "."
	}

	backoffDelay, err := time.ParseDuration(v.GetString(fmt.Sprint(prefix, "grpc.backoffdelay")))
	if err != nil {
		backoffDelay, _ = time.ParseDuration("3s")
	}
	return GrpcConfig{
		Host:         v.GetString(fmt.Sprint(prefix, "grpc.host")),
		Port:         v.GetInt(fmt.Sprint(prefix, "grpc.port")),
		BackoffDelay: Duration{backoffDelay},
		NoTLS:        v.GetBool(fmt.Sprint(prefix, "grpc.noTLS")),
	}
}
