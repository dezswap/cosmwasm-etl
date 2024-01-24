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
