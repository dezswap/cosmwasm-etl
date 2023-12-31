package terraswap

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/faker"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/rules/terraswap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_parseTxs(t *testing.T) {
	type testcase struct {
		rawLogs           []string
		pairCount         int
		createPairTxCount int
		expected          []parser.ParsedTx
		errMsg            string
	}

	const (
		chainId = terraswap.TestnetPrefix
		height  = uint(100)
	)
	parser.FakerCustomGenerator()
	faker.CustomGenerator()

	createTxs := []*parser.ParsedTx{}
	raws := eventlog.LogResults{}
	tx := parser.RawTx{
		Sender: "sender",
	}

	setUp := func(tc testcase) parser.Dex {
		createPairParser := parser.ParserMock{}
		repo := parser.RepoMock{}
		rawStore := parser.RawStoreMock{}
		app := terraswapApp{&repo, &parser.PairParsers{CreatePairParser: &createPairParser}, parser.DexMixin{}}

		dexApp := parser.NewDexApp(&app, &rawStore, &repo, logging.New("test", configs.LogConfig{}), configs.ParserConfig{ChainId: chainId})
		pairMap := map[string]parser.Pair{pair.ContractAddr: pair}

		pairs := []parser.Pair{}
		for len(pairs) < tc.pairCount {
			pairs = append(pairs, parser.FakeParserPairs()...)
		}
		pairs = pairs[0:tc.pairCount]
		for _, p := range pairs {
			pairMap[p.ContractAddr] = p
		}
		repo.On("GetPairs").Return(pairMap, nil)

		createTxs = []*parser.ParsedTx{}
		for len(createTxs) < tc.createPairTxCount {
			for _, ptx := range parser.FakeParserParsedTxs() {
				createTxs = append(createTxs, &ptx)
			}
		}
		createTxs = createTxs[0:tc.createPairTxCount]

		for _, tx := range createTxs {
			tx.Type = parser.CreatePair
		}

		raws = eventlog.LogResults{}
		for _, log := range tc.rawLogs {
			raws = append(raws, rawLogs(log)...)
		}
		createPairParser.On("parse", raws, mock.Anything).Return(createTxs, nil)
		tx.LogResults = raws

		return dexApp
	}

	tcs := []testcase{
		{[]string{swapLogStr, provideLogStr, withdrawLogStr, wasmTransferLogStr, transferLogStr}, 1, 0, []parser.ParsedTx{swapTx, provideTx, withdrawTx}, ""},
		{[]string{withdrawLogStr, transferLogStr, wasmTransferLogStr}, 1, 0, []parser.ParsedTx{withdrawTx, transferTx, wasmTransferTx}, ""},
		{[]string{swapLogStr, wasmTransferLogStr, transferLogStr}, 1, 0, []parser.ParsedTx{swapTx, transferTx}, ""},
		{nil, 0, 0, []parser.ParsedTx{}, ""},
		{nil, 3, 1, []parser.ParsedTx{}, ""},
	}

	for idx, tc := range tcs {
		assert := assert.New(t)
		msg := fmt.Sprintf("tc(%d): %s", idx, tc.errMsg)
		app := setUp(tc)

		txs, err := app.ParseTxs(tx, uint64(height))
		if tc.errMsg != "" {
			assert.Error(err, msg, err)
		} else {
			expected := []parser.ParsedTx{}
			for _, tx := range createTxs {
				expected = append(expected, *tx)
			}
			assert.Equal(append(expected, tc.expected...), txs, msg, err)
		}
	}
}

func rawLogs(logStr string) eventlog.LogResults {
	logs := eventlog.LogResults{}
	if err := json.Unmarshal([]byte(logStr), &logs); err != nil {
		panic(err)
	}
	return logs
}

var (
	pair           = parser.Pair{ContractAddr: "PAIR_ADDR", Assets: []string{"Asset0", "Asset1"}, LpAddr: "Lp"}
	createTx       = parser.ParsedTx{"", time.Time{}, parser.CreatePair, "sender", "PAIR_ADDR", [2]parser.Asset{{"Asset0", "1000"}, {"Asset1", "1000"}}, "Lp", "1000", "", nil}
	swapTx         = parser.ParsedTx{"", time.Time{}, parser.Swap, "sender", "PAIR_ADDR", [2]parser.Asset{{"Asset0", "1000"}, {"Asset1", "-1000"}}, "", "", "1", nil}
	provideTx      = parser.ParsedTx{"", time.Time{}, parser.Provide, "sender", "PAIR_ADDR", [2]parser.Asset{{"Asset0", "1000"}, {"Asset1", "1000"}}, "Lp", "1000", "", nil}
	withdrawTx     = parser.ParsedTx{"", time.Time{}, parser.Withdraw, "sender", "PAIR_ADDR", [2]parser.Asset{{"Asset0", "-1000"}, {"Asset1", "-1000"}}, "Lp", "1000", "", nil}
	transferTx     = parser.ParsedTx{"", time.Time{}, parser.Transfer, "sender", "PAIR_ADDR", [2]parser.Asset{{"Asset0", ""}, {"Asset1", "1000"}}, "", "", "", make(map[string]interface{})}
	wasmTransferTx = parser.ParsedTx{"", time.Time{}, parser.Transfer, "sender", "PAIR_ADDR", [2]parser.Asset{{"Asset0", "1000"}, {"Asset1", ""}}, "", "", "", make(map[string]interface{})}
)

const (
	swapLogStr = `[{"type":"wasm","attributes":[{"key":"_contract_address","value":"PAIR_ADDR"},{"key":"action","value":"swap"},{"key":"sender","value":"sender"},
{"key":"receiver","value":"receiver"},{"key":"offer_asset","value":"Asset0"},{"key":"ask_asset","value":"Asset1"},{"key":"offer_amount","value":"1000"},
{"key":"return_amount","value":"1000"},{"key":"spread_amount","value":"10"},{"key":"commission_amount","value":"1"},{"key":"_contract_address","value":"Asset1"},{"key":"action","value":"transfer"},
{"key":"from","value":"A"},
{"key":"to","value":"terra1tv7x48jderh5n9jva3vnsduhdprxpapagcly6s"},{"key":"amount","value":"100583"}]}]`
	provideLogStr = `[{"type":"wasm","attributes":[{"key":"_contract_address","value":"PAIR_ADDR"},{"key":"action","value":"provide_liquidity"},{"key":"sender","value":"sender"},
	{"key":"receiver","value":"receiver"},{"key":"assets","value":"1000Asset0, 1000Asset1"},{"key":"share","value":"1000"},{"key":"_contract_address","value":"asset1"},{"key":"action","value":"transfer_from"},{"key":"from","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"},{"key":"to","value":"A"},{"key":"by","value":"A"},{"key":"amount","value":"2013569"},{"key":"_contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"action","value":"mint"},{"key":"to","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"},{"key":"amount","value":"998735"}]}]`
	withdrawLogStr = `[{"type":"wasm","attributes":[{"key":"_contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"action","value":"send"},{"key":"from","value":"terra1cupj7d70jrtjxqhpr6s3qq68t8ky4smcjvccm4"},
	{"key":"to","value":"A"},{"key":"amount","value":"12418119"},{"key":"_contract_address","value":"PAIR_ADDR"},{"key":"action","value":"withdraw_liquidity"},{"key":"sender","value":"sender"},{"key":"withdrawn_share","value":"1000"},{"key":"refund_assets","value":"1000Asset0, 1000Asset1"},{"key":"_contract_address","value":"asset1"},{"key":"action","value":"transfer"},{"key":"from","value":"A"},{"key":"to","value":"terra1cupj7d70jrtjxqhpr6s3qq68t8ky4smcjvccm4"},{"key":"amount","value":"24999998"},{"key":"_contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"action","value":"burn"},{"key":"from","value":"A"},{"key":"amount","value":"12418119"}]}]`
	wasmTransferLogStr = `[{"type":"wasm","attributes":[{"key":"_contract_address","value":"Asset1"},{"key":"action","value":"transfer"},{"key":"from","value":"sender"},
	{"key":"to","value":"PAIR_ADDR"},{"key":"amount","value":"1000"}]}]`
	transferLogStr = `[{"type":"transfer","attributes":[{"key":"recipient","value":"PAIR_ADDR"},{"key":"sender","value":"sender"},{"key":"amount","value":"1000Asset0"}]}]`
)
