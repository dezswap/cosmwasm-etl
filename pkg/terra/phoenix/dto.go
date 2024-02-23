package phoenix

type LcdContractStateRes[T any] struct {
	Data T `json:"data"`
}

type LcdTxRes struct {
	Tx         LcdTx            `json:"tx"`
	TxResponse LcdTxResponseRes `json:"tx_response"`
}

type LcdTx struct {
	Body LcdTxBody `json:"body"`
}
type LcdTxBody struct {
	Messages []LcdTxMessage `json:"messages"`
}

type LcdTxMessage struct {
	Type   string `json:"@type"`
	Sender string `json:"sender"`
}

type LcdTxResponseRes struct {
	Height string        `json:"height"`
	TxHash string        `json:"txhash"`
	Code   int           `json:"code"`
	RawLog string        `json:"raw_log"`
	Logs   []LcdTxLogRes `json:"logs"`
}

type LcdTxLogRes struct {
	MsgIndex int             `json:"msg_index"`
	Events   []LcdTxEventRes `json:"events"`
}
type LcdTxEventRes struct {
	Type       string              `json:"type"`
	Attributes []LcdTxAttributeRes `json:"attributes"`
}

type LcdTxAttributeRes struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
