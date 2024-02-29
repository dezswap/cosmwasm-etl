package dex

import (
	"time"

	"github.com/dezswap/cosmwasm-etl/parser"
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

var _ parser.Overrider[ParsedTx] = &ParsedTx{}

type ParsedTx struct {
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"` // timestamp of a block

	Type             TxType   `json:"type" faker:"parserTxType"`
	Sender           string   `json:"sender"`
	ContractAddr     string   `json:"contractAddr"`
	Assets           [2]Asset `json:"assets"`
	LpAddr           string   `json:"lpAddr"`
	LpAmount         string   `json:"lpAmount" faker:"amountString"`
	CommissionAmount string   `json:"commissionAmount" faker:"amountString"`

	Meta map[string]interface{} `json:"meta" faker:"meta"`
}

func (defaultVal ParsedTx) Override(tx ParsedTx) (ParsedTx, error) {
	if tx.Hash != "" {
		defaultVal.Hash = tx.Hash
	}
	if tx.Timestamp != (time.Time{}) {
		defaultVal.Timestamp = tx.Timestamp
	}
	if tx.Type != "" {
		defaultVal.Type = tx.Type
	}
	if tx.Sender != "" {
		defaultVal.Sender = tx.Sender
	}
	if tx.ContractAddr != "" {
		defaultVal.ContractAddr = tx.ContractAddr
	}
	if tx.Assets[0].Addr != "" {
		defaultVal.Assets[0].Addr = tx.Assets[0].Addr
	}
	if tx.Assets[0].Amount != "" {
		defaultVal.Assets[0].Amount = tx.Assets[0].Amount
	}
	if tx.Assets[1].Addr != "" {
		defaultVal.Assets[1].Addr = tx.Assets[1].Addr
	}
	if tx.Assets[1].Amount != "" {
		defaultVal.Assets[1].Amount = tx.Assets[1].Amount
	}
	if tx.LpAddr != "" {
		defaultVal.LpAddr = tx.LpAddr
	}
	if tx.LpAmount != "" {
		defaultVal.LpAmount = tx.LpAmount
	}
	if tx.CommissionAmount != "" {
		defaultVal.CommissionAmount = tx.CommissionAmount
	}
	if tx.Meta != nil {
		if defaultVal.Meta == nil {
			defaultVal.Meta = map[string]interface{}{}
		}
		for k, v := range tx.Meta {
			defaultVal.Meta[k] = v
		}
	}

	return defaultVal, nil
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
