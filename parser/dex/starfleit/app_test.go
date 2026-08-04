package starfleit

import (
	"encoding/json"
	"testing"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_ParseTxs_CreatePairUpdatesPairState(t *testing.T) {
	const (
		txHash     = "hash"
		txSender   = "sender"
		pairAddr   = "pair"
		lpAddr     = "lp"
		asset0Addr = "asset0"
		asset1Addr = "asset1"
	)

	createPairParser := dex.ParserMock{}
	repo := dex.RepoMock{}
	assets := [2]dex.Asset{{Addr: asset0Addr}, {Addr: asset1Addr}}
	createPairTx := &dex.ParsedTx{
		Hash:         txHash,
		Type:         dex.CreatePair,
		ContractAddr: pairAddr,
		LpAddr:       lpAddr,
		Assets:       assets,
	}
	createPairParser.On("parse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]*dex.ParsedTx{createPairTx}, nil)

	app := starfleitApp{
		PairRepo:    &repo,
		Parsers:     &dex.PairParsers{CreatePairParser: &createPairParser},
		DexMixin:    dex.DexMixin{},
		pairs:       map[string]dex.Pair{},
		lpPairAddrs: map[string]string{},
	}
	tx := parser.RawTx{Sender: txSender, Hash: txHash}

	txs, err := app.ParseTxs(tx, 100)

	assert.NoError(t, err)
	assert.Equal(t, []dex.ParsedTx{{
		Hash:         txHash,
		Type:         dex.CreatePair,
		Sender:       txSender,
		ContractAddr: pairAddr,
		LpAddr:       lpAddr,
		Assets:       assets,
	}}, txs)
	assert.Equal(t, dex.Pair{
		ContractAddr: pairAddr,
		LpAddr:       lpAddr,
		Assets:       []string{asset0Addr, asset1Addr},
	}, app.pairs[pairAddr])
	assert.Equal(t, pairAddr, app.lpPairAddrs[lpAddr])
}

func Test_ParseTxs_SortsTransferAttributesWhenRandomOrder(t *testing.T) {
	const (
		txHash   = "hash"
		txSender = "sender"
		pairAddr = "pair"
		asset0   = "asset0"
		asset1   = "asset1"
	)

	createPairParser := dex.ParserMock{}
	repo := dex.RepoMock{}
	createPairParser.On("parse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]*dex.ParsedTx{}, nil)

	app := starfleitApp{
		PairRepo:    &repo,
		Parsers:     &dex.PairParsers{CreatePairParser: &createPairParser},
		DexMixin:    dex.DexMixin{},
		chainId:     "dorado-1",
		pairs:       map[string]dex.Pair{pairAddr: {ContractAddr: pairAddr, Assets: []string{asset0, asset1}}},
		lpPairAddrs: map[string]string{},
	}
	require.NoError(t, app.UpdateParsers(map[string]bool{}, 100))

	// attributes are in a random order (sender, amount, recipient) to verify
	// NormalizeTransferAttrs sorts them before matching.
	var logs eventlog.LogResults
	require.NoError(t, json.Unmarshal([]byte(`[
		{"type":"transfer","attributes":[
			{"key":"sender","value":"`+txSender+`"},
			{"key":"amount","value":"1000`+asset0+`"},
			{"key":"recipient","value":"`+pairAddr+`"}
		]}
	]`), &logs))

	tx := parser.RawTx{Sender: txSender, Hash: txHash, LogResults: logs}
	txs, err := app.ParseTxs(tx, 100)

	require.NoError(t, err)
	require.Equal(t, []dex.ParsedTx{{
		Hash:         txHash,
		Type:         dex.Transfer,
		Sender:       txSender,
		ContractAddr: pairAddr,
		Assets:       [2]dex.Asset{{Addr: asset0, Amount: "1000"}, {Addr: asset1, Amount: ""}},
		Meta:         map[string]interface{}{"recipient": pairAddr},
	}}, txs)
}
