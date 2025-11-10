package db

import (
	"database/sql"
	"fmt"

	"github.com/dezswap/cosmwasm-etl/configs"
	_ "github.com/lib/pq"
)

type PostgresDb struct {
	Db *sql.DB
}

func (x *PostgresDb) Init(dbConfig configs.RdbConfig) error {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host, dbConfig.Port, dbConfig.Username, dbConfig.Password, dbConfig.Database, dbConfig.SslMode)

	if x.Db != nil {
		x.Close()
	}

	var err error
	if x.Db, err = sql.Open("postgres", psqlInfo); err != nil {
		return err
	}

	// test connection
	if err = x.Db.Ping(); err != nil {
		return err
	}

	return nil
}

func (x PostgresDb) Close() {
	x.Db.Close()
}
