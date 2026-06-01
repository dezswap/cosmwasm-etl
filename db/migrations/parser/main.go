//go:build mig
// +build mig

package main

import (
	"os"
	"strings"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"
)

func main() {
	rollBack := os.Args[1:]
	c := configs.New().Rdb

	m, err := migrate.New("file://db/migrations/parser", c.PostgresURL())
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
