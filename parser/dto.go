package parser

import (
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
)

type RawTx struct {
	Hash       string              `json:"hash"`
	Sender     string              `json:"sender"`
	Timestamp  time.Time           `json:"timestamp,omitempty"`
	LogResults eventlog.LogResults `json:"logResults"`
}

type RawTxs []RawTx
