package datastore

import (
	"fmt"

	"github.com/pkg/errors"
)

type readStoreWithGrpc struct {
	chainId string
	DataStore
}

var _ ReadStore = &readStoreWithGrpc{}

func NewReadStoreWithGrpc(chainId string, store DataStore) ReadStore {
	return &readStoreWithGrpc{chainId, store}
}

// GetBlockByHeight implements ReadStore
func (r *readStoreWithGrpc) GetBlockByHeight(height uint64) (*BlockTxsDTO, error) {
	rawBlock, err := r.GetBlockTxsFromHeight(int64(height))
	if err != nil {
		return nil, errors.Wrap(err, "readStoreWithGrpc.GetBlockByHeight")
	}
	txs := []TxDTO{}
	block := BlockTxsDTO{
		BlockId: rawBlock.BlockId,
	}
	for _, tx := range rawBlock.Txs {
		txDto := TxDTO{
			Height:    fmt.Sprint(tx.TxContent.TxResponse.Height),
			TxHash:    tx.TxContent.TxResponse.TxHash,
			Codespace: tx.TxContent.TxResponse.Codespace,
			Code:      tx.TxContent.TxResponse.Code,
			Data:      tx.TxContent.TxResponse.Data,
			RawLog:    tx.TxContent.TxResponse.RawLog,
			Logs:      tx.TxContent.TxResponse.Logs,
			Info:      tx.TxContent.TxResponse.Info,
			GasUsed:   fmt.Sprint(tx.TxContent.TxResponse.GasUsed),
			GasWanted: fmt.Sprint(tx.TxContent.TxResponse.GasWanted),
			Tx:        tx.TxContent.TxResponse.Tx,
			Timestamp: tx.TxContent.TxResponse.Timestamp,
			Events:    tx.TxContent.TxResponse.Events,
		}
		txs = append(txs, txDto)
	}
	block.Txs = txs

	return &block, nil

}

// GetLatestHeight implements ReadStore
func (r *readStoreWithGrpc) GetLatestHeight() (uint64, error) {
	height, err := r.GetNodeSyncedHeight()
	if err != nil {
		return uint64(0), errors.Wrap(err, "readStore.GetLatestHeight")
	}
	if height < 0 {
		return uint64(0), errors.New("readStore.GetLatestHeight(returned height is negative)")
	}
	return uint64(height), nil
}

// GetPoolStatusOfAllPairsByHeight implements ReadStore
func (r *readStoreWithGrpc) GetPoolStatusOfAllPairsByHeight(height uint64) (*PoolInfoList, error) {

	ret, err := r.GetCurrentPoolStatusOfAllPairs(int64(height))
	if err != nil {
		return nil, errors.Wrap(err, "readStoreWithGrpc.GetPoolStatusOfAllPairsByHeight")
	}

	return ret, nil
}
