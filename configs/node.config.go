package configs

import (
	"time"
)

type NodeConfig struct {
	GrpcConfig       GrpcConfig       `mapstructure:"grpc"`
	FailoverLcdHost  string           `mapstructure:"failover_lcd_host"`
	RestClientConfig RestClientConfig `mapstructure:"rest"`
	HttpClientConfig HttpClientConfig `mapstructure:"http"`
}

type HttpClientConfig struct {
	MaxIdleConns        int       `mapstructure:"maxidleconns"`
	MaxIdleConnsPerHost int       `mapstructure:"maxidleconnsperhost"`
	IdleConnTimeout     *Duration `mapstructure:"idleconntimeout"`
	DisableKeepAlives   bool      `mapstructure:"disablekeepalives"`
	ForceAttemptHTTP2   bool      `mapstructure:"forceattempthttp2"`
	Timeout             *Duration `mapstructure:"timeout"`
}

type GrpcConfig struct {
	Host         string   `mapstructure:"host"`
	Port         int      `mapstructure:"port"`
	BackoffDelay Duration `mapstructure:"backoffdelay"`
	NoTLS        bool     `mapstructure:"notls"`
}

type RestClientConfig struct {
	LcdHost string `mapstructure:"lcd"`
	RpcHost string `mapstructure:"rpc"`
}

// Duration is a wrapper type for automatic string → time.Duration unmarshalling via mapstructure.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	s := string(text)
	if s == "" {
		d.Duration = 0
		return nil
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

var defaultHttpClientConfig = HttpClientConfig{
	MaxIdleConns:        20,
	MaxIdleConnsPerHost: 5,
	IdleConnTimeout:     &Duration{time.Second * 30},
	Timeout:             &Duration{},
}
