package col4

import (
	"encoding/hex"
	"time"

	"github.com/tendermint/tendermint/types"
)

type RpcRes[T any] struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  T      `json:"result"`
}

type RpcBlockRes struct {
	Block struct {
		Header struct {
			Height string    `json:"height"`
			Time   time.Time `json:"time"`
		} `json:"header"`
		Data struct {
			Txs types.Txs `json:"txs"`
		} `json:"data"`
	} `json:"block"`
}

func (r *RpcBlockRes) TxsHashStrings() []string {
	hashes := make([]string, len(r.Block.Data.Txs))
	for i, tx := range r.Block.Data.Txs {
		hashes[i] = hex.EncodeToString(tx.Hash())
	}
	return hashes
}

type RpcBlockResultRes struct {
	Height     string           `json:"height"`
	TxsResults []RpcTxResultRes `json:"txs_results"`
}
type RpcTxResultRes struct {
	Code int    `json:"code"`
	Data string `json:"data"`
	Log  string `json:"log"` // json string of tx results, result of json.Unmarshall must be RpcEventRes
}

type RpcEventRes struct {
	Type       string            `json:"type"`
	Attributes []RpcAttributeRes `json:"attributes"`
}
type RpcAttributeRes struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type LcdContractStateRes[T any] struct {
	Height string `json:"height"`
	Result T      `json:"result"`
}

type LcdTxRes struct {
	Height string `json:"height"`
	TxHash string `json:"txhash"`
	RawLog string `json:"raw_log"`
	Tx     TxRes  `json:"tx"`
}

type TxRes struct {
	Type  string     `json:"type"`
	Value TxValueRes `json:"value"`
}

type TxValueRes struct {
	Msg []MsgRes `json:"msg"`
}

type MsgRes struct {
	Type  string          `json:"type"`
	Value WasmMsgValueRes `json:"value"`
}

type WasmMsgValueRes struct {
	Sender string `json:"sender"`
}

const (
	LCD_TERRA_TX_TYPE          = "core/StdTx"
	LCD_TERRA_TX_MSG_WASM_TYPE = "wasm/MsgExecuteContract"
)
