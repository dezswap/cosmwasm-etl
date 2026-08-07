//go:build mig
// +build mig

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

const migTableName = "collector_migration"

func main() {
	rollBack := os.Args[1:]
	c := configs.New().Rdb

	params := url.Values{}
	params.Set("x-migrations-table", migTableName)

	m, err := migrate.New("file://db/migrations/collector", c.PostgresURL(params))
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
