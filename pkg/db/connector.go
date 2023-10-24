package db

import (
	"github.com/dezswap/cosmwasm-etl/configs"
)

type Connector interface {
	Init(dbConfig configs.RdbConfig) error
	Close()
}
