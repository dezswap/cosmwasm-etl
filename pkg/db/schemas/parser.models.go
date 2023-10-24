package schemas

import (
	"github.com/dezswap/cosmwasm-etl/parser"
)

type Meta map[string]interface{}

type PoolInfo struct {
	ChainId      string `json:"chainId"`
	Height       uint64 `json:"height"`
	Contract     string `json:"contract"`
	Asset0Amount string `json:"asset0Amount" faker:"amountString"`
	Asset1Amount string `json:"asset1Amount" faker:"amountString"`
	LpAmount     string `json:"lpAmount" faker:"amountString"`

	Meta Meta `json:"meta" faker:"meta"`
}

type ParsedTx struct {
	ChainId           string        `json:"chainId"`
	Height            uint64        `json:"height"`
	Timestamp         float64       `json:"timestamp"` // timestamp of a block in second
	Hash              string        `json:"hash"`
	Sender            string        `json:"sender"`
	Type              parser.TxType `json:"type" faker:"parserTxType"`
	Contract          string        `json:"contract"`
	Asset0            string        `json:"asset0"`
	Asset0Amount      string        `json:"asset0Amount" faker:"amountString"`
	Asset1            string        `json:"asset1"`
	Asset1Amount      string        `json:"asset1Amount" faker:"amountString"`
	Lp                string        `json:"lp"`
	LpAmount          string        `json:"lpAmount" faker:"amountString"`
	CommissionAmount  string        `json:"commissionAmount" faker:"amountString"`
	Commission0Amount string        `json:"commission0Amount" faker:"amountString"`
	Commission1Amount string        `json:"commission1Amount" faker:"amountString"`

	Meta Meta `json:"meta" faker:"meta"`
}

type Pair struct {
	ChainId  string `json:"chainId"`
	Contract string `json:"contract"`
	Asset0   string `json:"asset0"`
	Asset1   string `json:"asset1"`
	Lp       string `json:"lp"`

	Meta Meta `json:"meta" faker:"meta"`
}

type SyncedHeight struct {
	ChainId string `json:"chainId"`
	Height  uint64 `json:"height"`
}
