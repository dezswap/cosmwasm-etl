package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
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
	logLevel, err := GormLogLevelFromConfig(dbConfig.GormLogLevel)
	if err != nil {
		return nil, err
	}

	pq := PostgresDb{}
	if err := pq.Init(dbConfig); err != nil {
		return nil, err
	}
	return openGormPostgresWithConn(pq.Db, logLevel, opts...)
}

// OpenGormPostgresWithConn opens a GORM Postgres connection using an existing sql.DB.
func OpenGormPostgresWithConn(conn *sql.DB, opts ...GormOption) (*gorm.DB, error) {
	return openGormPostgresWithConn(conn, logger.Silent, opts...)
}

func openGormPostgresWithConn(conn *sql.DB, logLevel logger.LogLevel, opts ...GormOption) (*gorm.DB, error) {
	postgresConfig := &postgres.Config{Conn: conn}
	gormConfig := &gorm.Config{
		Logger: NewGormLogger(logLevel),
	}
	for _, opt := range opts {
		opt(gormConfig, postgresConfig)
	}
	return gorm.Open(postgres.New(*postgresConfig), gormConfig)
}

// GormLogLevelFromConfig parses an optional GORM log level from config.
func GormLogLevelFromConfig(value string) (logger.LogLevel, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "silent":
		return logger.Silent, nil
	case "error":
		return logger.Error, nil
	case "warn", "warning":
		return logger.Warn, nil
	case "info":
		return logger.Info, nil
	default:
		return logger.Silent, fmt.Errorf("invalid gorm log level %q", value)
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
