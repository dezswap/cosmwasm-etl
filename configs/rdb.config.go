package configs

import (
	"strconv"

	"github.com/spf13/viper"
)

// db contains configs for other services
type RdbConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	SslMode  string `mapstructure:"sslmode"`
}

var defaultRdbConfig = RdbConfig{
	Host:     "localhost",
	Port:     5432,
	Database: "cosmwasm_etl",
	Username: "app",
	Password: "appPW",
	SslMode:  "disable",
}

func (c RdbConfig) Endpoint() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func SetDefaultRdbConfig(v *viper.Viper) {
	v.SetDefault("rdb.host", defaultRdbConfig.Host)
	v.SetDefault("rdb.port", defaultRdbConfig.Port)
	v.SetDefault("rdb.database", defaultRdbConfig.Database)
	v.SetDefault("rdb.username", defaultRdbConfig.Username)
	v.SetDefault("rdb.password", defaultRdbConfig.Password)
	v.SetDefault("rdb.sslmode", defaultRdbConfig.SslMode)
}
