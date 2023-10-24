package parser

import (
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
)

type TxType string

const (
	CreatePair     TxType = "create_pair"
	Swap           TxType = "swap"
	Provide        TxType = "provide"
	InitialProvide TxType = "initial_provide"
	Withdraw       TxType = "withdraw"
	Transfer       TxType = "transfer"
)

type Asset struct {
	Addr   string `json:"addr"`
	Amount string `json:"amount" faker:"amountString"`
}

type RawTx struct {
	Hash       string              `json:"hash"`
	Sender     string              `json:"sender"`
	Timestamp  time.Time           `json:"timestamp,omitempty"`
	LogResults eventlog.LogResults `json:"logResults"`
}

type RawTxs []RawTx

type ParsedTx struct {
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"` // timestamp of a block

	Type             TxType  `json:"type" faker:"parserTxType"`
	Sender           string  `json:"sender"`
	ContractAddr     string  `json:"contractAddr"`
	Assets           []Asset `json:"assets"`
	LpAddr           string  `json:"lpAddr"`
	LpAmount         string  `json:"lpAmount" faker:"amountString"`
	CommissionAmount string  `json:"commissionAmount" faker:"amountString"`

	Meta map[string]interface{} `json:"meta" faker:"meta"`
}

type PoolInfo struct {
	ContractAddr string  `json:"contractAddr"`
	Assets       []Asset `json:"assets"`
	TotalShare   string  `json:"totalShare" faker:"amountString"`
}

type Pair struct {
	ContractAddr string   `json:"contractAddr"`
	Assets       []string `json:"assets"`
	LpAddr       string   `json:"lpAddr"`
}
