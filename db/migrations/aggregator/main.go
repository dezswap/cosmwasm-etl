package main

import (
	"net/url"
	"os"
	"strings"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"
)

const migTableName = "aggregator_migration"

func main() {
	rollBack := os.Args[1:]
	c := configs.New().Rdb

	// x-multi-statement is required so migrations that mix an explicit
	// BEGIN/COMMIT block with CREATE/DROP INDEX CONCURRENTLY (which cannot
	// run inside any transaction block) execute as separate statements on
	// the same connection instead of being sent as one multi-statement query.
	params := url.Values{}
	params.Set("x-migrations-table", migTableName)
	params.Set("x-multi-statement", "true")

	m, err := migrate.New("file://db/migrations/aggregator", c.PostgresURL(params))
	if err != nil {
		panic(err)
	}

	if len(rollBack) == 1 && strings.ToLower(rollBack[0]) == "down" {
		if err := m.Steps(-1); err != nil {
			panic(errors.Wrap(err, "Down"))
		}
		return
	}

	if err := m.Up(); err != nil {
		panic(errors.Wrap(err, "Up"))
	}
}
