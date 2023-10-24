package configs

import (
	"time"

	"github.com/spf13/viper"
)

type GrpcConfig struct {
	Host            string
	Port            int
	BackoffMaxDelay Duration
	NoTLS           bool
}

// Duration is wrapper type for custom unmarshalling
type Duration struct {
	time.Duration
}

func grpcConfig(v *viper.Viper) GrpcConfig {
	backoffDelay, err := time.ParseDuration(v.GetString("backoffdelay"))
	if err != nil {
		backoffDelay, _ = time.ParseDuration("3s")
	}
	return GrpcConfig{
		Host:            v.GetString("host"),
		Port:            v.GetInt("port"),
		BackoffMaxDelay: Duration{backoffDelay},
		NoTLS:           v.GetBool("noTLS"),
	}
}
