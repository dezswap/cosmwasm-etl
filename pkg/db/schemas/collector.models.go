package schemas

import "time"

type FcdTxLog struct {
	Id        uint32    `json:"id"`
	FcdOffset uint32    `json:"fcdOffset"`
	Height    uint32    `json:"height"`
	TxIndex   uint8     `json:"txIndex"`
	Timestamp time.Time `json:"timestamp"`
	Hash      string    `json:"hash"`
	Address   string    `json:"address"`
	EventLog  string    `json:"eventLog"`
}

func (c FcdTxLog) TableName() string {
	return "fcd_tx_log"
}
