package configs

import (
	"net/url"
	"strconv"

	"github.com/spf13/viper"
)

// db contains configs for other services
type RdbConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Database     string `mapstructure:"database"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	SslMode      string `mapstructure:"sslmode"`
	GormLogLevel string `mapstructure:"gormloglevel"`
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

// PostgresURL builds a postgres connection URL. extraParams are optional
// query params (e.g. golang-migrate driver options like x-migrations-table,
// x-multi-statement) to add to or override the defaults. Later params take
// precedence, and only the last value is used when a key has multiple values.
func (c RdbConfig) PostgresURL(extraParams ...url.Values) string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.Username, c.Password),
		Host:   c.Endpoint(),
		Path:   c.Database,
	}

	query := url.Values{}
	query.Set("sslmode", c.SslMode)
	for _, params := range extraParams {
		for key, values := range params {
			if len(values) > 0 {
				query.Set(key, values[len(values)-1])
			}
		}
	}

	u.RawQuery = query.Encode()

	return u.String()
}

func SetDefaultRdbConfig(v *viper.Viper) {
	v.SetDefault("rdb.host", defaultRdbConfig.Host)
	v.SetDefault("rdb.port", defaultRdbConfig.Port)
	v.SetDefault("rdb.database", defaultRdbConfig.Database)
	v.SetDefault("rdb.username", defaultRdbConfig.Username)
	v.SetDefault("rdb.password", defaultRdbConfig.Password)
	v.SetDefault("rdb.sslmode", defaultRdbConfig.SslMode)
	v.SetDefault("rdb.gormloglevel", defaultRdbConfig.GormLogLevel)
}
