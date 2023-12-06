package configs

import (
	"strconv"

	"github.com/spf13/viper"
)

// db contains configs for other services
type RdbConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

var defaultRdbConfig = RdbConfig{
	Host:     "localhost",
	Port:     5432,
	Database: "cosmwasm_etl",
	Username: "app",
	Password: "appPW",
}

func rdbConfig(v *viper.Viper) RdbConfig {
	c := RdbConfig{
		Host:     v.GetString("rdb.host"),
		Port:     v.GetInt("rdb.port"),
		Database: v.GetString("rdb.database"),
		Username: v.GetString("rdb.username"),
		Password: v.GetString("rdb.password"),
	}
	if c.Host == "" {
		c.Host = defaultRdbConfig.Host
	}
	if c.Port == 0 {
		c.Port = defaultRdbConfig.Port
	}
	if c.Database == "" {
		c.Database = defaultRdbConfig.Database
	}
	if c.Username == "" {
		c.Username = defaultRdbConfig.Username
	}
	if c.Password == "" {
		c.Password = defaultRdbConfig.Password
	}
	return c
}

func (c RdbConfig) Endpoint() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}
