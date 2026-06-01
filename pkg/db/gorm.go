package db

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormOption customizes the GORM and Postgres driver configuration before opening a database.
type GormOption func(*gorm.Config, *postgres.Config)

// OpenGormPostgres opens a GORM Postgres connection from repository database configuration.
func OpenGormPostgres(dbConfig configs.RdbConfig, opts ...GormOption) (*gorm.DB, error) {
	pq := PostgresDb{}
	if err := pq.Init(dbConfig); err != nil {
		return nil, err
	}
	return OpenGormPostgresWithConn(pq.Db, opts...)
}

// OpenGormPostgresWithConn opens a GORM Postgres connection using an existing sql.DB.
func OpenGormPostgresWithConn(conn *sql.DB, opts ...GormOption) (*gorm.DB, error) {
	postgresConfig := &postgres.Config{Conn: conn}
	gormConfig := &gorm.Config{
		Logger: NewGormLogger(logger.Silent),
	}
	for _, opt := range opts {
		opt(gormConfig, postgresConfig)
	}
	return gorm.Open(postgres.New(*postgresConfig), gormConfig)
}

// WithGormLogLevel configures the shared GORM logger with the given log level.
func WithGormLogLevel(logLevel logger.LogLevel) GormOption {
	return func(gormConfig *gorm.Config, _ *postgres.Config) {
		gormConfig.Logger = NewGormLogger(logLevel)
	}
}

// NewGormLogger creates the standard GORM logger used by database repositories.
func NewGormLogger(logLevel logger.LogLevel) logger.Interface {
	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
}
