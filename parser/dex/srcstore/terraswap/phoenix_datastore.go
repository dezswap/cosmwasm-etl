package terraswap

import (
	"encoding/json"
	"time"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/lcd"

	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
	"github.com/pkg/errors"
)

// https://github.com/phoenix-directive/core/releases/tag/v2.17.0
// https://github.com/cosmos/cosmos-sdk/blob/release/v0.50.x/UPGRADING.md
const cosmosSdk50StartHeight = 16395000

type cosmos47Resultog struct {
	MsgIndex int               `json:"msg_index"`
	Log      string            `json:"log"`
	Events   []rpc.RpcEventRes `json:"events"`
}

type phoenixSourceDataStore struct {
	rpc rpc.Rpc
	terraswap.QueryClient
	chainDataAdapter
}

var _ dex.SourceDataStore = &phoenixSourceDataStore{}

func NewPhoenixStore(factoryAddress string, rpc rpc.Rpc, lcd lcd.Lcd[cosmos45.LcdTxRes], client terraswap.QueryClient) dex.SourceDataStore {
	return &phoenixSourceDataStore{
		rpc,
		client,
		&col5ChainDataAdapter{
			factoryAddress: factoryAddress,
			mapper:         &mapperImpl{},
			rpc:            rpc,
			lcd:            lcd,
			QueryClient:    client,
		},
	}
}

func (r *phoenixSourceDataStore) GetPoolInfos(height uint64) ([]dex.PoolInfo, error) {
	allPairs, err := r.AllPairs(height)
	if err != nil {
		return nil, errors.Wrap(err, "phoenixSourceDataStore.GetPoolInfos")
	}

	poolInfos := make([]dex.PoolInfo, len(allPairs))

	for idx, pair := range allPairs {
		poolRes, err := r.QueryPool(pair.ContractAddr, height)
		if err != nil {
			return nil, errors.Wrap(err, "phoenixSourceDataStore.GetPoolInfos")
		}

		poolInfos[idx] = dex.PoolInfo{
			ContractAddr: pair.ContractAddr,
			Assets: []dex.Asset{
				{Addr: pair.Assets[0], Amount: poolRes.Assets[0].Amount},
				{Addr: pair.Assets[1], Amount: poolRes.Assets[1].Amount},
			},
			LpAddr:     pair.LpAddr,
			TotalShare: poolRes.TotalShare,
		}
	}
	return poolInfos, nil
}

func (r *phoenixSourceDataStore) GetSourceSyncedHeight() (uint64, error) {
	height, err := r.rpc.RemoteBlockHeight()
	if err != nil {
		return 0, errors.Wrap(err, "phoenixSourceDataStore.GetSourceSyncedHeight")
	}

	return height, nil
}

func (r *phoenixSourceDataStore) GetSourceTxs(height uint64) (parser.RawTxs, error) {
	rpcRes, err := r.rpc.Block(height)
	if err != nil {
		return nil, errors.Wrap(err, "phoenixSourceDataStore.GetSourceTxs")
	}
	blockRes := rpcRes.Result
	blockTime := blockRes.Block.Header.Time
	txHashes := blockRes.TxsHashStrings()

	rpcResultRes, err := r.rpc.BlockResults(height)
	if err != nil {
		return nil, errors.Wrap(err, "phoenixSourceDataStore.GetSourceTxs")
	}

	txResults := rpcResultRes.Result.TxsResults
	if len(txHashes) != len(txResults) {
		return nil, errors.New("phoenixSourceDataStore.GetSourceTxs: txs length mismatch")
	}

	rawTxs := []parser.RawTx{}
	for i, txHash := range txHashes {
		if txResults[i].Code != 0 {
			continue
		}

		var tx parser.RawTx
		var err error
		if height > cosmosSdk50StartHeight {
			tx, err = r.convertLogToRawTx(txHash, txResults[i].Events, blockTime)
		} else {
			var logs []cosmos47Resultog
			if err = json.Unmarshal([]byte(txResults[i].Log), &logs); err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal log JSON for tx %s", txHash)
			}
			var events []rpc.RpcEventRes
			for _, l := range logs {
				events = append(events, l.Events...)
			}
			tx, err = r.convertLogToRawTx(txHash, events, blockTime)
		}
		if err != nil {
			return nil, err
		}

		rawTxs = append(rawTxs, tx)
	}
	return rawTxs, nil
}

func (r *phoenixSourceDataStore) convertLogToRawTx(txHash string, events []rpc.RpcEventRes, blockTs time.Time) (parser.RawTx, error) {
	logResultMap := r.groupLogAttrByType(events)
	tx := parser.RawTx{
		Hash:       txHash,
		Timestamp:  blockTs,
		LogResults: make([]eventlog.LogResult, 0, len(logResultMap)),
	}

	for logType, logs := range logResultMap {
		tx.LogResults = append(tx.LogResults, eventlog.LogResult{
			Type:       logType,
			Attributes: logs,
		})
		if logType == eventlog.Message {
			for _, attr := range logs {
				if attr.Key == "sender" {
					tx.Sender = attr.Value
					break
				}
			}
		}
	}

	var err error
	if tx.Sender == "" {
		if tx.Sender, err = r.TxSenderOf(txHash); err != nil {
			return parser.RawTx{}, errors.Wrapf(err, "failed to retrieve sender for tx %s from TxSenderOf", txHash)
		}
	}

	return tx, nil
}

func (r *phoenixSourceDataStore) groupLogAttrByType(events []rpc.RpcEventRes) map[eventlog.LogType]eventlog.Attributes {
	logResultMap := make(map[eventlog.LogType]eventlog.Attributes)

	for _, event := range events {
		attributes := eventlog.Attributes{}
		for _, attr := range event.Attributes {
			attributes = append(attributes, eventlog.Attribute{
				Key:   attr.Key,
				Value: attr.Value,
			})
		}
		logType := eventlog.LogType(event.Type)
		if attrs, ok := logResultMap[logType]; ok {
			attributes = append(attrs, attributes...)
		}
		logResultMap[logType] = attributes
	}
	return logResultMap
}
