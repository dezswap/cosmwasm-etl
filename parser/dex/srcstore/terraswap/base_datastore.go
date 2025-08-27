package terraswap

import (
	"encoding/json"
	"time"

	"github.com/dezswap/cosmwasm-etl/parser"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
	"github.com/pkg/errors"
)

type chainDataAdapter interface {
	AllPairs(height uint64) ([]p_dex.Pair, error)
	TxSenderOf(hash string) (string, error)
}

type logResults []struct {
	MsgIndex int                 `json:"msg_index"`
	Log      string              `json:"log"`
	Events   eventlog.LogResults `json:"events"`
}

type baseRawDataStoreImpl struct {
	rpc rpc.Rpc
	terraswap.QueryClient
	chainDataAdapter
}

var _ p_dex.SourceDataStore = &baseRawDataStoreImpl{}

func NewBaseStore(rpc rpc.Rpc, client terraswap.QueryClient, cda chainDataAdapter) p_dex.SourceDataStore {
	return &baseRawDataStoreImpl{rpc, client, cda}
}

// GetSourceSyncedHeight implements p_dex.RawDataStore
func (r *baseRawDataStoreImpl) GetSourceSyncedHeight() (uint64, error) {
	height, err := r.rpc.RemoteBlockHeight()
	if err != nil {
		return 0, errors.Wrap(err, "baseRawDataStoreImpl.GetSourceSyncedHeight")
	}

	return height, nil
}

// GetPoolInfos implements p_dex.RawDataStore
func (r *baseRawDataStoreImpl) GetPoolInfos(height uint64) ([]p_dex.PoolInfo, error) {
	allPairs, err := r.AllPairs(height)
	if err != nil {
		return nil, errors.Wrap(err, "baseRawDataStoreImpl.GetPoolInfos")
	}

	poolInfos := make([]p_dex.PoolInfo, len(allPairs))

	for idx, pair := range allPairs {
		poolRes, err := r.QueryPool(pair.ContractAddr, height)
		if err != nil {
			return nil, errors.Wrap(err, "baseRawDataStoreImpl.GetPoolInfos")
		}
		poolInfos[idx] = p_dex.PoolInfo{
			ContractAddr: pair.ContractAddr,
			Assets: []p_dex.Asset{
				{Addr: pair.Assets[0], Amount: poolRes.Assets[0].Amount},
				{Addr: pair.Assets[1], Amount: poolRes.Assets[1].Amount},
			},
			TotalShare: poolRes.TotalShare,
		}
	}
	return poolInfos, nil
}

// GetSourceTxs implements p_dex.RawDataStore
func (r *baseRawDataStoreImpl) GetSourceTxs(height uint64) (parser.RawTxs, error) {
	rpcRes, err := r.rpc.Block(height)
	if err != nil {
		return nil, errors.Wrap(err, "baseRawDataStoreImpl.GetSourceTxs")
	}
	blockRes := rpcRes.Result
	blockTime := blockRes.Block.Header.Time
	txHashes := blockRes.TxsHashStrings()

	rpcResultRes, err := r.rpc.BlockResults(height)
	if err != nil {
		return nil, errors.Wrap(err, "baseRawDataStoreImpl.GetSourceTxs")
	}

	txResults := rpcResultRes.Result.TxsResults
	if len(txHashes) != len(txResults) {
		return nil, errors.New("baseRawDataStoreImpl.GetSourceTxs: txs length mismatch")
	}

	rawTxs := []parser.RawTx{}
	for i, txHash := range txHashes {
		if txResults[i].Code != 0 {
			continue
		}

		tx, err := r.convertLogToRawTx(txHash, txResults[i].Log, blockTime)
		if err != nil {
			return nil, err
		}

		rawTxs = append(rawTxs, tx)
	}
	return rawTxs, nil
}

// convertLogToRawTx unmarshal raw log data into a structured RawTx, extracting event attributes and sender.
func (r *baseRawDataStoreImpl) convertLogToRawTx(txHash, log string, blockTs time.Time) (parser.RawTx, error) {
	var logs logResults
	if err := json.Unmarshal([]byte(log), &logs); err != nil {
		return parser.RawTx{}, errors.Wrapf(err, "failed to unmarshal log JSON for tx %s", txHash)
	}

	logResultMap := groupLogAttrByType(logs)
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

// groupLogAttrByType returns a map of event types(e.g., "wasm", "transfer", "send")
// to their corresponding attributes.
func groupLogAttrByType(logs logResults) map[eventlog.LogType]eventlog.Attributes {
	logResultMap := make(map[eventlog.LogType]eventlog.Attributes)

	for _, log := range logs {
		for _, event := range log.Events {
			attributes := eventlog.Attributes{}
			for _, attr := range event.Attributes {
				attributes = append(attributes, eventlog.Attribute{
					Key:   attr.Key,
					Value: attr.Value,
				})
			}
			logType := event.Type
			if attrs, ok := logResultMap[logType]; ok {
				attributes = append(attrs, attributes...)
			}
			logResultMap[logType] = attributes
		}
	}
	return logResultMap
}
