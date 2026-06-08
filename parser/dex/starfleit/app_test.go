package starfleit

import (
	"testing"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
