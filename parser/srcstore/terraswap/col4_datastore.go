package terraswap

import (
	"encoding/json"
	"fmt"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/col4"
	"github.com/pkg/errors"
)

type rawDataStoreImpl struct {
	mapper
	rpc col4.Rpc
	lcd col4.Lcd
}

var _ parser.SourceDataStore = &rawDataStoreImpl{}

func NewCol4Store(rpc col4.Rpc, lcd col4.Lcd) parser.SourceDataStore {
	return &rawDataStoreImpl{
		mapper: &mapperImpl{},
		rpc:    rpc,
		lcd:    lcd,
	}
}

// GetSourceSyncedHeight implements parser.RawDataStore
func (r *rawDataStoreImpl) GetSourceSyncedHeight() (uint64, error) {
	height, err := r.rpc.RemoteBlockHeight()
	if err != nil {
		return 0, errors.Wrap(err, "rawDataStoreImpl.GetSourceSyncedHeight")
	}

	return uint64(height), nil
}

// GetPoolInfos implements parser.RawDataStore
func (r *rawDataStoreImpl) GetPoolInfos(height uint64) ([]parser.PoolInfo, error) {
	allPairs, err := r.AllPairs(height)
	if err != nil {
		return nil, errors.Wrap(err, "rawDataStoreImpl.GetPoolInfos")
	}

	queryBytes, _ := json.Marshal(&dex.PoolInfoReq{})
	poolInfos := make([]parser.PoolInfo, len(allPairs))

	for idx, pair := range allPairs {
		poolRes, err := col4.QueryContractState[dex.PoolInfoRes](r.lcd, pair.ContractAddr, string(queryBytes), height)
		if err != nil {
			return nil, errors.Wrap(err, "rawDataStoreImpl.GetPoolInfos")
		}
		poolInfos[idx] = parser.PoolInfo{
			ContractAddr: pair.ContractAddr,
			Assets: []parser.Asset{
				{Addr: pair.Assets[0], Amount: poolRes.Result.Assets[0].Amount},
				{Addr: pair.Assets[1], Amount: poolRes.Result.Assets[1].Amount},
			},
			TotalShare: poolRes.Result.TotalShare,
		}
	}
	return poolInfos, nil
}

// GetSourceTxs implements parser.RawDataStore
func (r *rawDataStoreImpl) GetSourceTxs(height uint64) (parser.RawTxs, error) {
	rpcRes, err := r.rpc.Block(height)
	if err != nil {
		return nil, errors.Wrap(err, "rawDataStoreImpl.GetSourceTxs")
	}
	blockRes := rpcRes.Result
	blockTime := blockRes.Block.Header.Time
	txHashes := blockRes.TxsHashStrings()

	rpcResultRes, err := r.rpc.BlockResults(height)
	if err != nil {
		return nil, errors.Wrap(err, "rawDataStoreImpl.GetSourceTxs")
	}

	txResults := rpcResultRes.Result.TxsResults
	if len(txHashes) != len(txResults) {
		return nil, errors.New("rawDataStoreImpl.GetSourceTxs: txs length mismatch")
	}

	type logResults []struct {
		MsgIndex int                 `json:"msg_index"`
		Log      string              `json:"log"`
		Events   eventlog.LogResults `json:"events"`
	}

	rawTxs := []parser.RawTx{}
	for idx, txHash := range txHashes {
		if txHash == "683d1c9f1c286bbe7a74ac5d556789ca90da5b23b44910d5bbbdc9d891c2014b" {
			fmt.Print("here")
		}
		tx := parser.RawTx{
			Hash:      txHash,
			Timestamp: blockTime,
		}

		logs := logResults{}
		if err := json.Unmarshal([]byte(txResults[idx].Log), &logs); err != nil {
			return nil, errors.Wrap(err, "rawDataStoreImpl.GetSourceTxs")
		}

		for _, log := range logs {
			tx.LogResults = append(tx.LogResults, log.Events...)
		}

		for _, lr := range tx.LogResults {
			if lr.Type == eventlog.Message {
				for _, attr := range lr.Attributes {
					if attr.Key == "sender" {
						tx.Sender = attr.Value
						break
					}
				}
			}
		}
		rawTxs = append(rawTxs, tx)
	}
	return rawTxs, nil
}

func (r *rawDataStoreImpl) AllPairs(height uint64) ([]parser.Pair, error) {
	req := dex.FactoryPairsReq{}
	pairs := []parser.Pair{}

	for {
		queryBytes, _ := json.Marshal(req)
		factoryRes, err := col4.QueryContractState[dex.FactoryPairsRes](r.lcd, terraswap.COLUMBUS_V1_FACTORY, string(queryBytes), height)
		if err != nil {
			return nil, errors.Wrap(err, "rawDataStoreImpl.AllPairs")
		}
		if len(factoryRes.Result.Pairs) == 0 {
			break
		}
		for _, pair := range factoryRes.Result.Pairs {
			p := r.dexPairToPair(&pair)
			pairs = append(pairs, p)
		}
		req.Pairs.StartAfter = &factoryRes.Result.Pairs[len(factoryRes.Result.Pairs)-1].AssetInfos
	}
	return pairs, nil
}
