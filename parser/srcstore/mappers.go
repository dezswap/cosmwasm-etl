package srcstore

import (
	"github.com/aws/smithy-go/time"
	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
)

type mapper interface {
	blockToRawTxs(block *datastore.BlockTxsDTO) parser.RawTxs
	txToParserRawTx(rawTx datastore.TxDTO) parser.RawTx

	rawPoolInfosToPoolInfos(rawPoolInfos *datastore.PoolInfoList) []parser.PoolInfo
	rawPoolInfoToPoolInfo(pairAddr string, rawPoolInfo datastore.PoolInfoDTO) parser.PoolInfo
}

type mapperImpl struct{}

var _ mapper = &mapperImpl{}

// blockToRawTxs implements mapper
func (m *mapperImpl) blockToRawTxs(block *datastore.BlockTxsDTO) parser.RawTxs {
	rawTxs := parser.RawTxs{}
	for _, tx := range block.Txs {
		rawTxs = append(rawTxs, m.txToParserRawTx(tx))
	}
	return rawTxs
}

// txToRawTx implements mapper
func (*mapperImpl) txToParserRawTx(tx datastore.TxDTO) parser.RawTx {
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

// rawPoolInfosToPoolInfos implements mapper
func (m *mapperImpl) rawPoolInfosToPoolInfos(rawPoolInfos *datastore.PoolInfoList) []parser.PoolInfo {
	infos := []parser.PoolInfo{}
	for addr, rawPoolInfo := range rawPoolInfos.Pairs {
		infos = append(infos, m.rawPoolInfoToPoolInfo(addr, rawPoolInfo))
	}
	return infos
}

// rawPoolInfoToPoolInfo implements mapper
func (*mapperImpl) rawPoolInfoToPoolInfo(contractAddr string, rawPoolInfo datastore.PoolInfoDTO) parser.PoolInfo {
	poolInfo := parser.PoolInfo{
		ContractAddr: contractAddr,
		Assets: []parser.Asset{
			{
				Addr:   rawPoolInfo.Assets[0].Info.DenomOrAddress,
				Amount: rawPoolInfo.Assets[0].Amount.String(),
			},
			{
				Addr:   rawPoolInfo.Assets[1].Info.DenomOrAddress,
				Amount: rawPoolInfo.Assets[1].Amount.String(),
			},
		},
		TotalShare: rawPoolInfo.TotalShare.String(),
	}
	return poolInfo
}
