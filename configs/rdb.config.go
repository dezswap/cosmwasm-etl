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

func (c RdbConfig) PostgresURL() string {
	return c.postgresURL(url.Values{})
}

func (c RdbConfig) MigrationURL(migrationTable string) string {
	values := url.Values{}
	values.Set("x-migrations-table", migrationTable)

	return c.postgresURL(values)
}

func (c RdbConfig) postgresURL(extraQuery url.Values) string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.Username, c.Password),
		Host:   c.Endpoint(),
		Path:   c.Database,
	}

	query := u.Query()
	query.Set("sslmode", c.SslMode)
	for key, values := range extraQuery {
		for _, value := range values {
			query.Add(key, value)
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
