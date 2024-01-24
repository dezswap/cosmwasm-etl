package fcd

import (
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/pkg/errors"
)

type (
	Tx  = schemas.FcdTxLog
	Txs = []Tx
)

var ADDRESS_NOT_FOUND_ERROR = errors.New("No txs found")
