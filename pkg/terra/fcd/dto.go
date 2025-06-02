package fcd

import (
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
)

type FcdTxsReqQuery struct {
	Limit  *int `json:"limit"`
	Offset *int `json:"next"`
}

type FcdBlockRes struct {
	Height int             `json:"height"`
	Txs    []FcdBlockTxRes `json:"txs"`
}

type FcdBlockTxRes struct {
	Code int                `json:"code"`
	Data string             `json:"data"`
	Logs []FcdBlockTxLogRes `json:"logs"`
}

type FcdBlockTxLogRes struct {
	Log      string              `json:"log"`
	Events   eventlog.LogResults `json:"events"`
	MsgIndex int                 `json:"msg_index"`
}

type FcdTxsRes struct {
	Limit int `json:"limit"`
	Next  int `json:"next"`
	Txs   []struct {
		Id      int    `json:"id"`
		ChainId string `json:"chainId"`

		Height    string `json:"height"`
		TxHash    string `json:"txhash"`
		RawLog    string `json:"raw_log"`
		Timestamp string `json:"timestamp"`
	} `json:"txs"`
}

type FcdTxRes struct {
	Tx     cosmos45.LcdTx         `json:"tx"`
	Code   int                    `json:"code"`
	Logs   []cosmos45.LcdTxLogRes `json:"logs"`
	Height string                 `json:"height"`
	TxHash string                 `json:"txhash"`
	RawLog string                 `json:"raw_log"`
}
