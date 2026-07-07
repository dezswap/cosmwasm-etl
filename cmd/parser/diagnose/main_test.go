package main

import (
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/parser"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_rawTxContainsContract(t *testing.T) {
	tx := parser.RawTx{
		Hash:   "hash",
		Sender: "sender",
		LogResults: eventlog.LogResults{{
			Type: eventlog.WasmType,
			Attributes: eventlog.Attributes{
				{Key: "_contract_address", Value: "pair1"},
				{Key: "action", Value: "swap"},
			},
		}},
	}

	assert.True(t, rawTxContainsContract(tx, "pair1"))
	assert.False(t, rawTxContainsContract(tx, "pair2"))
}

func Test_diagnoseRange_ReplaysOnlyMatchingTransactions(t *testing.T) {
	target := &diagnoseTargetApp{}
	source := &diagnoseSourceDataStore{
		txs: map[uint64]parser.RawTxs{
			10: {
				rawTxWithContract("matching-hash", "pair1"),
				rawTxWithContract("other-hash", "pair2"),
			},
		},
	}
	tokenExceptions := map[string]bool{"token": true}

	report, err := diagnoseRange(target, source, tokenExceptions, "chain-1", 10, 10, "pair1")

	require.NoError(t, err)
	require.Len(t, report.Results, 1)
	assert.Equal(t, "chain-1", report.ChainID)
	assert.Equal(t, uint64(10), report.From)
	assert.Equal(t, uint64(10), report.To)
	assert.Equal(t, "pair1", report.Contract)
	assert.Equal(t, "matching-hash", report.Results[0].Hash)
	assert.Equal(t, 1, report.Results[0].ParsedTxCount)
	assert.Equal(t, []uint64{10}, target.updatedHeights)
	assert.Equal(t, tokenExceptions, target.tokenExceptions)
}

type diagnoseTargetApp struct {
	updatedHeights  []uint64
	tokenExceptions map[string]bool
}

func (a *diagnoseTargetApp) ParseTxs(tx parser.RawTx, _ uint64) ([]p_dex.ParsedTx, error) {
	return []p_dex.ParsedTx{{Hash: tx.Hash, ContractAddr: "pair1", Type: p_dex.Swap}}, nil
}

func (*diagnoseTargetApp) IsValidationExceptionCandidate(string) bool {
	return false
}

func (a *diagnoseTargetApp) UpdateParsers(tokenExceptions map[string]bool, height uint64) error {
	a.tokenExceptions = tokenExceptions
	a.updatedHeights = append(a.updatedHeights, height)
	return nil
}

type diagnoseSourceDataStore struct {
	txs map[uint64]parser.RawTxs
}

func (*diagnoseSourceDataStore) GetPoolInfos(uint64) ([]p_dex.PoolInfo, error) {
	return nil, nil
}

func (*diagnoseSourceDataStore) GetSourceSyncedHeight() (uint64, error) {
	return 0, nil
}

func (s *diagnoseSourceDataStore) GetSourceTxs(height uint64) (parser.RawTxs, error) {
	return s.txs[height], nil
}

func rawTxWithContract(hash, contract string) parser.RawTx {
	return parser.RawTx{
		Hash:      hash,
		Sender:    "sender",
		Timestamp: time.Unix(1, 0),
		LogResults: eventlog.LogResults{{
			Type: eventlog.WasmType,
			Attributes: eventlog.Attributes{
				{Key: "_contract_address", Value: contract},
				{Key: "action", Value: "swap"},
			},
		}},
	}
}
