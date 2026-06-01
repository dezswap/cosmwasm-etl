package db

import (
	"database/sql"

	"github.com/dezswap/cosmwasm-etl/configs"
	_ "github.com/lib/pq"
)

type PostgresDb struct {
	Db *sql.DB
}

func (x *PostgresDb) Init(dbConfig configs.RdbConfig) error {
	if x.Db != nil {
		x.Close()
	}

	var err error
	if x.Db, err = sql.Open("postgres", dbConfig.PostgresURL()); err != nil {
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
