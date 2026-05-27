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

type CollectorJSON []byte

type CollectorBlock struct {
	ChainId   string        `json:"chainId"`
	Height    uint64        `json:"height"`
	BlockTime time.Time     `json:"blockTime"`
	Txs       CollectorJSON `json:"txs" gorm:"type:jsonb"`
	CreatedAt int64         `json:"createdAt" gorm:"->"`
	UpdatedAt int64         `json:"updatedAt" gorm:"->"`
}

type CollectorPoolSnapshot struct {
	ChainId   string        `json:"chainId"`
	Height    uint64        `json:"height"`
	PoolInfos CollectorJSON `json:"poolInfos" gorm:"type:jsonb"`
	CreatedAt int64         `json:"createdAt" gorm:"->"`
	UpdatedAt int64         `json:"updatedAt" gorm:"->"`
}

type CollectorSyncedHeight struct {
	ChainId   string `json:"chainId"`
	Height    uint64 `json:"height"`
	CreatedAt int64  `json:"createdAt" gorm:"->"`
	UpdatedAt int64  `json:"updatedAt" gorm:"->"`
}
