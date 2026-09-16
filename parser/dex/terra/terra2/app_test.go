package terra2

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terra"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/faker"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	sender = "sender"
	hash   = "hash"
)

func Test_parseTxs(t *testing.T) {
	type testcase struct {
		rawLogs           []string
		pairCount         int
		createPairTxCount int
		expected          []dex.ParsedTx
		errMsg            string
	}

	const (
		factoryAddr = string(terra.MainnetFactory)
		height      = uint(100)
	)
	dex.FakerCustomGenerator()
	faker.CustomGenerator()

	createTxs := []*dex.ParsedTx{}
	raws := eventlog.LogResults{}

	tx := parser.RawTx{
		Sender: sender,
		Hash:   hash,
	}

	setUp := func(tc testcase) dex.DexParserApp {
		createPairParser := dex.ParserMock{}
		repo := dex.RepoMock{}
		rawStore := dex.RawStoreMock{}

		pairMap := map[string]dex.Pair{pair.ContractAddr: pair}
		pairs := []dex.Pair{}
		for len(pairs) < tc.pairCount {
			pairs = append(pairs, dex.FakeParserPairs()...)
		}
		pairs = pairs[0:tc.pairCount]
		for _, p := range pairs {
			pairMap[p.ContractAddr] = p
		}

		app := appImpl{&repo, &dex.PairParsers{CreatePairParser: &createPairParser}, dex.DexMixin{}, pairMap, make(map[string]string), make(map[string]bool)}
		dexApp := dex.NewDexApp(&app, &rawStore, &repo, logging.New("test", configs.LogConfig{}), configs.ParserDexConfig{FactoryAddress: factoryAddr})

		createTxs = []*dex.ParsedTx{}
		for len(createTxs) < tc.createPairTxCount {
			for _, ptx := range dex.FakeParserParsedTxs() {
				createTxs = append(createTxs, &ptx)
			}
		}
		createTxs = createTxs[0:tc.createPairTxCount]

		for _, tx := range createTxs {
			tx.Type = dex.CreatePair
		}

		raws = eventlog.LogResults{}
		for _, log := range tc.rawLogs {
			raws = append(raws, rawLogs(log)...)
		}
		createPairParser.On("parse", raws, mock.Anything).Return(createTxs, nil)
		tx.LogResults = raws

		return dexApp.(dex.DexParserApp)
	}

	tcs := []testcase{
		{[]string{swapLogStr, provideLogStr, withdrawLogStr, wasmTransferLogStr, transferLogStr}, 1, 0, []dex.ParsedTx{swapTx, provideTx, withdrawTx}, ""},
		{[]string{withdrawLogStr, transferLogStr, wasmTransferLogStr}, 1, 0, []dex.ParsedTx{withdrawTx, transferTx, wasmTransferTx}, ""},
		{[]string{swapLogStr, wasmTransferLogStr, transferLogStr}, 1, 0, []dex.ParsedTx{swapTx, transferTx}, ""},
		{nil, 0, 0, []dex.ParsedTx{}, ""},
		{nil, 3, 1, []dex.ParsedTx{}, ""},
		// MsgMultiSend outputs emit "transfer" events without a "sender" attribute
		// (a multisend can have multiple inputs); must fall back to the tx-level sender.
		{[]string{transferLogStrWithoutSender}, 0, 0, []dex.ParsedTx{
			{
				Hash:         hash,
				Timestamp:    time.Time{},
				Type:         dex.Transfer,
				Sender:       sender,
				ContractAddr: "PAIR_ADDR",
				Assets:       [2]dex.Asset{{Addr: "Asset0", Amount: "1000"}, {Addr: "Asset1", Amount: ""}},
				Meta:         make(map[string]interface{}),
			},
		}, ""},
	}

	for idx, tc := range tcs {
		assert := assert.New(t)
		msg := fmt.Sprintf("tc(%d): %s", idx, tc.errMsg)
		app := setUp(tc)

		err := app.UpdateParsers(make(map[string]bool), uint64(height))
		assert.NoError(err)

		txs, err := app.ParseTxs(tx, uint64(height))
		if tc.errMsg != "" {
			assert.Error(err, msg, err)
		} else {
			expected := []dex.ParsedTx{}
			for _, tx := range createTxs {
				expected = append(expected, *tx)
			}
			assert.Equal(append(expected, tc.expected...), txs, msg, err)
		}
	}
}

func Test_ParseTxs_SameTransactionCreatePairAndInitialProvide(t *testing.T) {
	const (
		pairAddr = "PAIR_ADDR"
		lpAddr   = "Lp"
		asset0   = "Asset0"
		asset1   = "Asset1"
		height   = uint64(100)
	)

	createPairParser := dex.ParserMock{}
	createPairParser.On("parse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]*dex.ParsedTx{{
			Hash:         hash,
			Type:         dex.CreatePair,
			ContractAddr: pairAddr,
			LpAddr:       lpAddr,
			Assets:       [2]dex.Asset{{Addr: asset0}, {Addr: asset1}},
		}}, nil)

	repo := dex.RepoMock{}
	app := appImpl{
		PairRepo:      &repo,
		Parsers:       &dex.PairParsers{CreatePairParser: &createPairParser},
		DexMixin:      dex.DexMixin{},
		pairs:         map[string]dex.Pair{},
		lpPairAddrs:   map[string]string{},
		flaggedAssets: map[string]bool{},
	}
	// the height loop builds the parsers before the pair exists, so the pair
	// filters must be refreshed mid-tx for the provide to be found
	require.NoError(t, app.UpdateParsers(nil, height))

	logs := rawLogs(`[{"type":"wasm","attributes":[
		{"key":"_contract_address","value":"` + pairAddr + `"},
		{"key":"action","value":"provide_liquidity"},
		{"key":"sender","value":"` + sender + `"},
		{"key":"receiver","value":"` + sender + `"},
		{"key":"assets","value":"1000` + asset0 + `, 1000` + asset1 + `"},
		{"key":"share","value":"1000"},
		{"key":"_contract_address","value":"` + lpAddr + `"},
		{"key":"action","value":"mint"},
		{"key":"amount","value":"1000"},
		{"key":"to","value":"` + pairAddr + `"}
	]}]`)

	txs, err := app.ParseTxs(parser.RawTx{Sender: sender, Hash: hash, LogResults: logs}, height)

	require.NoError(t, err)
	require.Len(t, txs, 3)
	assert.Contains(t, txs, dex.ParsedTx{
		Hash:         hash,
		Type:         dex.CreatePair,
		Sender:       sender,
		ContractAddr: pairAddr,
		LpAddr:       lpAddr,
		Assets:       [2]dex.Asset{{Addr: asset0}, {Addr: asset1}},
	})
	assert.Contains(t, txs, dex.ParsedTx{
		Hash:         hash,
		Type:         dex.Provide,
		Sender:       sender,
		ContractAddr: pairAddr,
		Assets:       [2]dex.Asset{{Addr: asset0, Amount: "1000"}, {Addr: asset1, Amount: "1000"}},
		LpAddr:       lpAddr,
		LpAmount:     "1000",
	})
	assert.Contains(t, txs, dex.ParsedTx{
		Hash:         hash,
		Type:         dex.InitialProvide,
		Sender:       sender,
		ContractAddr: pairAddr,
		LpAddr:       lpAddr,
		LpAmount:     "1000",
	})
}

func rawLogs(logStr string) eventlog.LogResults {
	logs := eventlog.LogResults{}
	if err := json.Unmarshal([]byte(logStr), &logs); err != nil {
		panic(err)
	}
	return logs
}

var (
	pair           = dex.Pair{ContractAddr: "PAIR_ADDR", Assets: []string{"Asset0", "Asset1"}, LpAddr: "Lp"}
	createTx       = dex.ParsedTx{Hash: hash, Timestamp: time.Time{}, Type: dex.CreatePair, Sender: sender, ContractAddr: "PAIR_ADDR", Assets: [2]dex.Asset{{Addr: "Asset0", Amount: "1000"}, {Addr: "Asset1", Amount: "1000"}}, LpAddr: "Lp", LpAmount: "1000", CommissionAmount: "", MsgIndex: 0, Meta: nil}
	swapTx         = dex.ParsedTx{Hash: hash, Timestamp: time.Time{}, Type: dex.Swap, Sender: sender, ContractAddr: "PAIR_ADDR", Assets: [2]dex.Asset{{Addr: "Asset0", Amount: "1000"}, {Addr: "Asset1", Amount: "-1000"}}, LpAddr: "", LpAmount: "", CommissionAmount: "1", MsgIndex: 0, Meta: nil}
	provideTx      = dex.ParsedTx{Hash: hash, Timestamp: time.Time{}, Type: dex.Provide, Sender: sender, ContractAddr: "PAIR_ADDR", Assets: [2]dex.Asset{{Addr: "Asset0", Amount: "1000"}, {Addr: "Asset1", Amount: "1000"}}, LpAddr: "Lp", LpAmount: "1000", CommissionAmount: "", MsgIndex: 0, Meta: nil}
	withdrawTx     = dex.ParsedTx{Hash: hash, Timestamp: time.Time{}, Type: dex.Withdraw, Sender: sender, ContractAddr: "PAIR_ADDR", Assets: [2]dex.Asset{{Addr: "Asset0", Amount: "-1000"}, {Addr: "Asset1", Amount: "-1000"}}, LpAddr: "Lp", LpAmount: "1000", CommissionAmount: "", MsgIndex: 0, Meta: nil}
	transferTx     = dex.ParsedTx{Hash: hash, Timestamp: time.Time{}, Type: dex.Transfer, Sender: sender, ContractAddr: "PAIR_ADDR", Assets: [2]dex.Asset{{Addr: "Asset0", Amount: ""}, {Addr: "Asset1", Amount: "1000"}}, LpAddr: "", LpAmount: "", CommissionAmount: "", MsgIndex: 0, Meta: make(map[string]interface{})}
	wasmTransferTx = dex.ParsedTx{Hash: hash, Timestamp: time.Time{}, Type: dex.Transfer, Sender: sender, ContractAddr: "PAIR_ADDR", Assets: [2]dex.Asset{{Addr: "Asset0", Amount: "1000"}, {Addr: "Asset1", Amount: ""}}, LpAddr: "", LpAmount: "", CommissionAmount: "", MsgIndex: 0, Meta: make(map[string]interface{})}
)

const (
	swapLogStr = `[{"type":"wasm","attributes":[{"key":"_contract_address","value":"PAIR_ADDR"},{"key":"action","value":"swap"},{"key":"sender","value":"sender"},{"key":"receiver","value":"receiver"},{"key":"offer_asset","value":"Asset0"},{"key":"ask_asset","value":"Asset1"},{"key":"offer_amount","value":"1000"},
{"key":"return_amount","value":"1000"},{"key":"spread_amount","value":"10"},{"key":"commission_amount","value":"1"},{"key":"_contract_address","value":"Asset1"},{"key":"action","value":"transfer"},
{"key":"from","value":"A"},
{"key":"to","value":"terra1tv7x48jderh5n9jva3vnsduhdprxpapagcly6s"},{"key":"amount","value":"100583"}]}]`
	provideLogStr  = `[{"type":"wasm","attributes":[{"key":"_contract_address","value":"PAIR_ADDR"},{"key":"action","value":"provide_liquidity"},{"key":"sender","value":"sender"},{"key":"receiver","value":"receiver"},{"key":"assets","value":"1000Asset0, 1000Asset1"},{"key":"share","value":"1000"},{"key":"_contract_address","value":"asset1"},{"key":"action","value":"transfer_from"},{"key":"from","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"},{"key":"to","value":"A"},{"key":"by","value":"A"},{"key":"amount","value":"2013569"},{"key":"_contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"action","value":"mint"},{"key":"to","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"},{"key":"amount","value":"998735"}]}]`
	withdrawLogStr = `[{"type":"wasm","attributes":[{"key":"_contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"action","value":"send"},{"key":"from","value":"terra1cupj7d70jrtjxqhpr6s3qq68t8ky4smcjvccm4"},
	{"key":"to","value":"A"},{"key":"amount","value":"12418119"},{"key":"_contract_address","value":"PAIR_ADDR"},{"key":"action","value":"withdraw_liquidity"},{"key":"sender","value":"sender"},{"key":"withdrawn_share","value":"1000"},{"key":"refund_assets","value":"1000Asset0, 1000Asset1"},{"key":"_contract_address","value":"asset1"},{"key":"action","value":"transfer"},{"key":"from","value":"A"},{"key":"to","value":"terra1cupj7d70jrtjxqhpr6s3qq68t8ky4smcjvccm4"},{"key":"amount","value":"24999998"},{"key":"_contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"action","value":"burn"},{"key":"from","value":"A"},{"key":"amount","value":"12418119"}]}]`
	wasmTransferLogStr = `[{"type":"wasm","attributes":[{"key":"_contract_address","value":"Asset1"},{"key":"action","value":"transfer"},{"key":"from","value":"sender"},
	{"key":"to","value":"PAIR_ADDR"},{"key":"amount","value":"1000"}]}]`
	// transferLogStr attributes are deliberately not in the canonical
	// amount/recipient/sender order, so every test case using it also exercises
	// NormalizeTransferAttrs.
	transferLogStr = `[{"type":"transfer","attributes":[{"key":"recipient","value":"PAIR_ADDR"},{"key":"sender","value":"sender"},{"key":"amount","value":"1000Asset0"}]}]`
	// transferLogStrWithoutSender mimics a MsgMultiSend output: bank emits "transfer"
	// with recipient+amount only, no sender (a multisend can have multiple inputs).
	transferLogStrWithoutSender = `[{"type":"transfer","attributes":[{"key":"recipient","value":"PAIR_ADDR"},{"key":"amount","value":"1000Asset0"}]}]`
)
