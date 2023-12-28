package rpc

type RpcRes[T any] struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  T      `json:"result"`
}

type BlockRes struct {
	Block struct {
		Data struct {
			Txs []string `json:"txs"`
		} `json:"data"`
	} `json:"block"`
}

type BlockResultRes struct {
	Height     string        `json:"height"`
	TxsResults []TxResultRes `json:"txs_results"`
}
type TxResultRes struct {
	Code int    `json:"code"`
	Data string `json:"data"`
	Log  string `json:"log"` // json string of tx results unmarshall type must be EventRes
}

type EventRes struct {
	Type       string         `json:"type"`
	Attributes []AttributeRes `json:"attributes"`
}
type AttributeRes struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
