package columbusv2

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	dts "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/faker"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		factoryAddr = string(dts.MAINNET_FACTORY)
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

		taxPaymentParser := dex.ParserMock{}
		taxPaymentParser.On("parse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*dex.ParsedTx{}, nil)
		app := terraswapApp{&repo, &dex.PairParsers{CreatePairParser: &createPairParser, TaxPaymentParser: &taxPaymentParser}, dex.DexMixin{}, pairMap, make(map[string]bool)}
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
	}

	for idx, tc := range tcs {
		assert := assert.New(t)
		msg := fmt.Sprintf("tc(%d): %s", idx, tc.errMsg)
		app := setUp(tc)

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

func Test_taxDeduction(t *testing.T) {
	assert := assert.New(t)
	app := &terraswapApp{}

	// swap: Asset0 inflow 1000, Asset1 outflow -1000
	pairTx := &dex.ParsedTx{
		Type:         dex.Swap,
		ContractAddr: "PAIR_ADDR",
		Assets:       [2]dex.Asset{{"Asset0", "1000"}, {"Asset1", "-1000"}},
	}

	// tax_payment log: tax of 10 Asset1
	taxTx := &dex.ParsedTx{
		Type:   dex.TaxPayment,
		Assets: [2]dex.Asset{{Addr: "Asset1", Amount: "10"}, {}},
	}

	// pair transfer: net outflow of -990 Asset1 (gross 1000 - tax 10)
	pairTransferTx := &dex.ParsedTx{
		Type:         dex.Transfer,
		ContractAddr: "PAIR_ADDR",
		Assets:       [2]dex.Asset{{Addr: "Asset0", Amount: ""}, {Addr: "Asset1", Amount: "-990"}},
	}

	result := app.deductTaxFromPairTxs([]*dex.ParsedTx{taxTx}, []*dex.ParsedTx{pairTransferTx}, []*dex.ParsedTx{pairTx})
	assert.Equal("1000", result[0].Assets[0].Amount, "Asset0 inflow should be unchanged")
	assert.Equal("-990", result[0].Assets[1].Amount, "Asset1 outflow: net -990 applied from pairTransferTx")
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
	createTx       = dex.ParsedTx{hash, time.Time{}, dex.CreatePair, sender, "PAIR_ADDR", [2]dex.Asset{{"Asset0", "1000"}, {"Asset1", "1000"}}, "Lp", "1000", "", nil}
	swapTx         = dex.ParsedTx{hash, time.Time{}, dex.Swap, sender, "PAIR_ADDR", [2]dex.Asset{{"Asset0", "1000"}, {"Asset1", "-1000"}}, "", "", "1", nil}
	provideTx      = dex.ParsedTx{hash, time.Time{}, dex.Provide, sender, "PAIR_ADDR", [2]dex.Asset{{"Asset0", "1000"}, {"Asset1", "1000"}}, "Lp", "1000", "", nil}
	withdrawTx     = dex.ParsedTx{hash, time.Time{}, dex.Withdraw, sender, "PAIR_ADDR", [2]dex.Asset{{"Asset0", "0"}, {"Asset1", "0"}}, "Lp", "1000", "", map[string]interface{}{"withdraw_assets": []dex.Asset{{pair.Assets[0], "-1000"}, {pair.Assets[1], "-1000"}}}}
	transferTx     = dex.ParsedTx{hash, time.Time{}, dex.Transfer, sender, "PAIR_ADDR", [2]dex.Asset{{"Asset0", ""}, {"Asset1", "1000"}}, "", "", "", make(map[string]interface{})}
	wasmTransferTx = dex.ParsedTx{hash, time.Time{}, dex.Transfer, sender, "PAIR_ADDR", [2]dex.Asset{{"Asset0", "1000"}, {"Asset1", ""}}, "", "", "", make(map[string]interface{})}
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
	transferLogStr = `[{"type":"transfer","attributes":[{"key":"recipient","value":"PAIR_ADDR"},{"key":"sender","value":"sender"},{"key":"amount","value":"1000Asset0"}]}]`
)
