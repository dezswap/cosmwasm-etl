package terraswap

import (
	"encoding/json"

	"github.com/dezswap/cosmwasm-etl/parser"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
	"github.com/pkg/errors"
)

type baseRawDataStoreImpl struct {
	factoryAddress string
	mapper
	rpc rpc.Rpc
	lcd cosmos45.Lcd
	terraswap.QueryClient
}

var _ p_dex.SourceDataStore = &baseRawDataStoreImpl{}

func NewBaseStore(factoryAddress string, rpc rpc.Rpc, lcd cosmos45.Lcd, client terraswap.QueryClient) p_dex.SourceDataStore {
	return &baseRawDataStoreImpl{factoryAddress, &mapperImpl{}, rpc, lcd, client}
}

// GetSourceSyncedHeight implements p_dex.RawDataStore
func (r *baseRawDataStoreImpl) GetSourceSyncedHeight() (uint64, error) {
	height, err := r.rpc.RemoteBlockHeight()
	if err != nil {
		return 0, errors.Wrap(err, "baseRawDataStoreImpl.GetSourceSyncedHeight")
	}

	return uint64(height), nil
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

	type logResults []struct {
		MsgIndex int                 `json:"msg_index"`
		Log      string              `json:"log"`
		Events   eventlog.LogResults `json:"events"`
	}

	rawTxs := []parser.RawTx{}
	for idx, txHash := range txHashes {
		if txResults[idx].Code != 0 {
			continue
		}

		tx := parser.RawTx{
			Hash:      txHash,
			Timestamp: blockTime,
		}

		logs := logResults{}
		if err := json.Unmarshal([]byte(txResults[idx].Log), &logs); err != nil {
			return nil, errors.Wrap(err, "baseRawDataStoreImpl.GetSourceTxs")
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

		if tx.Sender == "" {
			if tx.Sender, err = r.txSenderOf(txHash); err != nil {
				return nil, errors.Wrap(err, "baseRawDataStoreImpl.GetSourceTxs")
			}
		}
		rawTxs = append(rawTxs, tx)
	}
	return rawTxs, nil
}

func (r *baseRawDataStoreImpl) AllPairs(height uint64) ([]p_dex.Pair, error) {
	pairs := []p_dex.Pair{}
	var startAfter []dex.AssetInfo = nil
	for {
		factoryRes, err := r.QueryPairs(r.factoryAddress, startAfter, height)
		if err != nil {
			return nil, errors.Wrap(err, "baseRawDataStoreImpl.AllPairs")
		}

		if len(factoryRes.Pairs) == 0 {
			break
		}

		for _, pair := range factoryRes.Pairs {
			p := r.dexPairToPair(&pair)
			pairs = append(pairs, p)
		}
		startAfter = factoryRes.Pairs[len(factoryRes.Pairs)-1].AssetInfos[:]
	}

	return pairs, nil
}

func (r *baseRawDataStoreImpl) txSenderOf(hash string) (string, error) {
	res, err := r.lcd.Tx(hash)
	if err != nil {
		return "", errors.Wrap(err, "txSenderOf")
	}

	for _, msg := range res.Tx.Body.Messages {
		if msg.Type == "/cosmwasm.wasm.v1.MsgExecuteContract" {
			return msg.Sender, nil
		}
	}

	return "", nil
}
