package terraswap

import (
	"encoding/json"
	"github.com/tendermint/tendermint/types"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	lcdtypes "github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	rpctypes "github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
	"github.com/stretchr/testify/require"
)

// rpcMock implements rpctypes.Rpc for testing GetSourceTxs
type rpcMock struct {
	blockRes       *rpctypes.RpcRes[rpctypes.RpcBlockRes]
	blockResultRes *rpctypes.RpcRes[rpctypes.RpcBlockResultRes]
}

func (m *rpcMock) RemoteBlockHeight() (uint64, error) { return 0, nil }
func (m *rpcMock) Block(height ...uint64) (*rpctypes.RpcRes[rpctypes.RpcBlockRes], error) {
	return m.blockRes, nil
}
func (m *rpcMock) BlockResults(height ...uint64) (*rpctypes.RpcRes[rpctypes.RpcBlockResultRes], error) {
	return m.blockResultRes, nil
}

// lcdMock implements cosmos45.Lcd but should not be called in this test
type lcdMock struct{ called bool }

func (l *lcdMock) ContractState(address string, query string, height ...uint64) ([]byte, error) {
	l.called = true
	return nil, nil
}
func (l *lcdMock) Tx(hash string) (*lcdtypes.LcdTxRes, error) {
	l.called = true
	return nil, nil
}

func TestGetSourceTxs_MergeEventsByType(t *testing.T) {
	now := time.Now().UTC()
	block := rpctypes.RpcBlockRes{}
	block.Block.Header.Time = now
	block.Block.Data.Txs = types.Txs{[]byte{}}

	type evAttr struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	type ev struct {
		Type       string   `json:"type"`
		Attributes []evAttr `json:"attributes"`
	}
	type logEntry struct {
		MsgIndex int    `json:"msg_index"`
		Log      string `json:"log"`
		Events   []ev   `json:"events"`
	}

	logs := []logEntry{
		{MsgIndex: 0, Events: []ev{{Type: string(eventlog.WasmType), Attributes: []evAttr{{Key: "a", Value: "1"}}}}},
		{MsgIndex: 0, Events: []ev{{Type: string(eventlog.WasmType), Attributes: []evAttr{{Key: "b", Value: "2"}}}}},
		{MsgIndex: 0, Events: []ev{{Type: string(eventlog.Message), Attributes: []evAttr{{Key: "sender", Value: "terra1sender"}}}}},
	}
	logsBytes, err := json.Marshal(logs)
	require.NoError(t, err)

	blockResults := rpctypes.RpcBlockResultRes{
		Height:     "1",
		TxsResults: []rpctypes.RpcTxResultRes{{Code: 0, Log: string(logsBytes)}},
	}

	rpc := &rpcMock{
		blockRes:       &rpctypes.RpcRes[rpctypes.RpcBlockRes]{Result: block},
		blockResultRes: &rpctypes.RpcRes[rpctypes.RpcBlockResultRes]{Result: blockResults},
	}
	lcd := &lcdMock{}

	store := &baseRawDataStoreImpl{
		factoryAddress: "",
		mapper:         &mapperImpl{},
		rpc:            rpc,
		lcd:            lcd,
		QueryClient:    nil,
	}

	txs, err := store.GetSourceTxs(1)
	require.NoError(t, err)
	require.Len(t, txs, 1)

	tx := txs[0]
	require.Equal(t, now, tx.Timestamp)
	require.Equal(t, "terra1sender", tx.Sender)

	typeToAttrs := map[eventlog.LogType]eventlog.Attributes{}
	for _, lr := range tx.LogResults {
		typeToAttrs[lr.Type] = lr.Attributes
	}

	require.Contains(t, typeToAttrs, eventlog.WasmType)
	require.Contains(t, typeToAttrs, eventlog.Message)
	require.Len(t, typeToAttrs[eventlog.WasmType], 2)
	require.Len(t, typeToAttrs[eventlog.Message], 1)

	require.False(t, lcd.called)
}
