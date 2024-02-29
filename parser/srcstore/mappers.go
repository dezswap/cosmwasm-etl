package srcstore

import (
	"github.com/aws/smithy-go/time"
	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
)

type Mapper interface {
	BlockToRawTxs(block *datastore.BlockTxsDTO) parser.RawTxs
	TxToParserRawTx(rawTx datastore.TxDTO) parser.RawTx
}

type mapperImpl struct{}

var _ Mapper = &mapperImpl{}

func NewMapper() Mapper {
	return &mapperImpl{}
}

// BlockToRawTxs implements mapper
func (m *mapperImpl) BlockToRawTxs(block *datastore.BlockTxsDTO) parser.RawTxs {
	rawTxs := parser.RawTxs{}
	for _, tx := range block.Txs {
		rawTxs = append(rawTxs, m.TxToParserRawTx(tx))
	}
	return rawTxs
}

// txToRawTx implements mapper
func (*mapperImpl) TxToParserRawTx(tx datastore.TxDTO) parser.RawTx {
	rawTx := parser.RawTx{}
	rawTx.Hash = tx.TxHash
	rawTx.LogResults = eventlog.LogResults{}
	logResultMap := make(map[eventlog.LogType]eventlog.Attributes)

	t, _ := time.ParseDateTime(tx.Timestamp)
	rawTx.Timestamp = t
	for _, event := range tx.Events {
		attributes := eventlog.Attributes{}
		for _, attr := range event.Attributes {
			attributes = append(attributes, eventlog.Attribute{
				Key:   string(attr.Key),
				Value: string(attr.Value),
			})
		}
		logType := eventlog.LogType(event.Type)
		if attrs, ok := logResultMap[logType]; ok {
			attributes = append(attrs, attributes...)
		}
		logResultMap[logType] = attributes
	}
	for logType, logs := range logResultMap {
		rawTx.LogResults = append(rawTx.LogResults, eventlog.LogResult{
			Type:       logType,
			Attributes: logs,
		})
		if logType == eventlog.Message {
			for _, attr := range logs {
				if attr.Key == "sender" {
					rawTx.Sender = attr.Value
					break
				}
			}
		}
	}
	return rawTx
}
