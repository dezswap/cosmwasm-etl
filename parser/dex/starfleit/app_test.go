package starfleit

import (
	"encoding/json"
	"testing"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
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

// The finder is built from the configured factory address, not from the chain id;
// filtering create_pair by chain id silently drops every new pair.
func Test_New_ParsesCreatePairFromConfiguredFactory(t *testing.T) {
	const (
		chainId   = "fetchhub-4"
		factory   = "fetch1slz6c85kxp4ek5ufmcakfhnscv9r2snlemxgwz6cjhklgh7v2hms8rgt5v"
		pairAddr  = "fetch1z2fy5y08nvx3tvs0e48vn88gldr7wn9u5chgvm55n0r4kpjlgk7sarnunq"
		lpAddr    = "fetch1axuck3rz2xf3p44tg04hzj6sk0kxkl98ls7mplu306hxfnwnt04sjzmnrv"
		txSender  = "fetch1t0vvv6y9lrur3x3mh9d6xkj9d0teul7h264m5s"
		lastAsset = "ibc/376222D6D9DAE23092E29740E56B758580935A6D77C24C2ABD57A6A78A1F3955"
	)

	repo := dex.RepoMock{}
	repo.On("GetPairs").Return(map[string]dex.Pair{}, nil)

	app, err := New(&repo, logging.Discard, configs.ParserDexConfig{
		ChainId:        chainId,
		FactoryAddress: factory,
	})
	require.NoError(t, err)
	require.NoError(t, app.UpdateParsers(map[string]bool{}, 28940967))

	logResults := eventlog.LogResults{}
	require.NoError(t, json.Unmarshal([]byte(sdk50CreatePairRawLogStr), &logResults))

	txs, err := app.ParseTxs(parser.RawTx{Hash: "AB89942C", Sender: txSender, LogResults: logResults}, 28940967)

	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.Equal(t, dex.CreatePair, txs[0].Type)
	assert.Equal(t, pairAddr, txs[0].ContractAddr)
	assert.Equal(t, lpAddr, txs[0].LpAddr)
	assert.Equal(t, lastAsset, txs[0].Assets[1].Addr)
}

// fetchhub-4 AB89942C05459804782896C27B9575405695BAE779FC8D4693274B3FA8C26A5A,
// the first create_pair the chain emitted after the cosmos-sdk v0.50 upgrade.
const sdk50CreatePairRawLogStr = `[
	{"type":"execute","attributes":[
		{"key":"_contract_address","value":"fetch1slz6c85kxp4ek5ufmcakfhnscv9r2snlemxgwz6cjhklgh7v2hms8rgt5v"},
		{"key":"msg_index","value":"0"}]},
	{"type":"instantiate","attributes":[
		{"key":"_contract_address","value":"fetch1z2fy5y08nvx3tvs0e48vn88gldr7wn9u5chgvm55n0r4kpjlgk7sarnunq"},
		{"key":"code_id","value":"54"},
		{"key":"msg_index","value":"0"},
		{"key":"_contract_address","value":"fetch1axuck3rz2xf3p44tg04hzj6sk0kxkl98ls7mplu306hxfnwnt04sjzmnrv"},
		{"key":"code_id","value":"21"},
		{"key":"msg_index","value":"0"}]},
	{"type":"message","attributes":[
		{"key":"action","value":"/cosmwasm.wasm.v1.MsgExecuteContract"},
		{"key":"sender","value":"fetch1t0vvv6y9lrur3x3mh9d6xkj9d0teul7h264m5s"},
		{"key":"module","value":"wasm"},
		{"key":"msg_index","value":"0"}]},
	{"type":"wasm","attributes":[
		{"key":"_contract_address","value":"fetch1slz6c85kxp4ek5ufmcakfhnscv9r2snlemxgwz6cjhklgh7v2hms8rgt5v"},
		{"key":"action","value":"create_pair"},
		{"key":"pair","value":"ibc/8AF69BC1E1D72B447738B50C28B382F62F2AF65DE303021E45C0B7C851B4B2E1-ibc/376222D6D9DAE23092E29740E56B758580935A6D77C24C2ABD57A6A78A1F3955"},
		{"key":"msg_index","value":"0"},
		{"key":"_contract_address","value":"fetch1z2fy5y08nvx3tvs0e48vn88gldr7wn9u5chgvm55n0r4kpjlgk7sarnunq"},
		{"key":"liquidity_token_addr","value":"fetch1axuck3rz2xf3p44tg04hzj6sk0kxkl98ls7mplu306hxfnwnt04sjzmnrv"},
		{"key":"msg_index","value":"0"},
		{"key":"_contract_address","value":"fetch1slz6c85kxp4ek5ufmcakfhnscv9r2snlemxgwz6cjhklgh7v2hms8rgt5v"},
		{"key":"pair_contract_addr","value":"fetch1z2fy5y08nvx3tvs0e48vn88gldr7wn9u5chgvm55n0r4kpjlgk7sarnunq"},
		{"key":"liquidity_token_addr","value":"fetch1axuck3rz2xf3p44tg04hzj6sk0kxkl98ls7mplu306hxfnwnt04sjzmnrv"},
		{"key":"msg_index","value":"0"}]}
]`
