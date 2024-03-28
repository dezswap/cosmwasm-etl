package columbusv1

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
		factoryAddrKey = string(dts.CLASSIC_V1_FACTORY)
		height         = uint(100)
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
		app := terraswapApp{&repo, &dex.PairParsers{CreatePairParser: &createPairParser}, dex.DexMixin{}}

		dexApp := dex.NewDexApp(&app, &rawStore, &repo, logging.New("test", configs.LogConfig{}), configs.ParserDexConfig{FactoryAddress: factoryAddrKey})
		pairMap := map[string]dex.Pair{pair.ContractAddr: pair}

		pairs := []dex.Pair{}
		for len(pairs) < tc.pairCount {
			pairs = append(pairs, dex.FakeParserPairs()...)
		}
		pairs = pairs[0:tc.pairCount]
		for _, p := range pairs {
			pairMap[p.ContractAddr] = p
		}
		repo.On("GetPairs").Return(pairMap, nil)

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
	swapTx         = dex.ParsedTx{hash, time.Time{}, dex.Swap, sender, "PAIR_ADDR", [2]dex.Asset{{"Asset0", "1000"}, {"Asset1", "-1000"}}, "", "", "1", map[string]interface{}{"tax_amount": dex.Asset{pair.Assets[1], "0"}}}
	provideTx      = dex.ParsedTx{hash, time.Time{}, dex.Provide, sender, "PAIR_ADDR", [2]dex.Asset{{"Asset0", "1000"}, {"Asset1", "1000"}}, "Lp", "1000", "", nil}
	withdrawTx     = dex.ParsedTx{hash, time.Time{}, dex.Withdraw, sender, "PAIR_ADDR", [2]dex.Asset{{"Asset0", "0"}, {"Asset1", "0"}}, "Lp", "1000", "", map[string]interface{}{"withdraw_assets": []dex.Asset{{"Asset0", "-1000"}, {"Asset1", "-1000"}}}}
	transferTx     = dex.ParsedTx{hash, time.Time{}, dex.Transfer, sender, "PAIR_ADDR", [2]dex.Asset{{"Asset0", ""}, {"Asset1", "1000"}}, "", "", "", make(map[string]interface{})}
	wasmTransferTx = dex.ParsedTx{hash, time.Time{}, dex.Transfer, sender, "PAIR_ADDR", [2]dex.Asset{{"Asset0", "1000"}, {"Asset1", ""}}, "", "", "", make(map[string]interface{})}
)

const (
	swapLogStr = `[{"type":"from_contract","attributes":[{"key":"contract_address","value":"PAIR_ADDR"},{"key":"action","value":"swap"},{"key":"offer_asset","value":"Asset0"},{"key":"ask_asset","value":"Asset1"},{"key":"offer_amount","value":"1000"},
{"key":"return_amount","value":"1000"},{"key":"tax_amount","value":"0"},{"key":"spread_amount","value":"10"},{"key":"commission_amount","value":"1"},{"key":"contract_address","value":"Asset1"},{"key":"action","value":"transfer"},
{"key":"from","value":"A"},
{"key":"to","value":"terra1tv7x48jderh5n9jva3vnsduhdprxpapagcly6s"},{"key":"amount","value":"100583"}]}]`
	provideLogStr  = `[{"type":"from_contract","attributes":[{"key":"contract_address","value":"PAIR_ADDR"},{"key":"action","value":"provide_liquidity"},{"key":"assets","value":"1000Asset0, 1000Asset1"},{"key":"share","value":"1000"},{"key":"contract_address","value":"asset1"},{"key":"action","value":"transfer_from"},{"key":"from","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"},{"key":"to","value":"A"},{"key":"by","value":"A"},{"key":"amount","value":"2013569"},{"key":"contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"action","value":"mint"},{"key":"to","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"},{"key":"amount","value":"998735"}]}]`
	withdrawLogStr = `[{"type":"from_contract","attributes":[{"key":"contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"action","value":"send"},{"key":"from","value":"terra1cupj7d70jrtjxqhpr6s3qq68t8ky4smcjvccm4"},
	{"key":"to","value":"A"},{"key":"amount","value":"12418119"},{"key":"contract_address","value":"PAIR_ADDR"},{"key":"action","value":"withdraw_liquidity"},{"key":"withdrawn_share","value":"1000"},{"key":"refund_assets","value":"1000Asset0, 1000Asset1"},{"key":"contract_address","value":"asset1"},{"key":"action","value":"transfer"},{"key":"from","value":"A"},{"key":"to","value":"terra1cupj7d70jrtjxqhpr6s3qq68t8ky4smcjvccm4"},{"key":"amount","value":"24999998"},{"key":"contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"action","value":"burn"},{"key":"from","value":"A"},{"key":"amount","value":"12418119"}]}]`
	wasmTransferLogStr = `[{"type":"from_contract","attributes":[{"key":"contract_address","value":"Asset1"},{"key":"action","value":"transfer"},{"key":"from","value":"sender"},
	{"key":"to","value":"PAIR_ADDR"},{"key":"amount","value":"1000"}]}]`
	transferLogStr = `[{"type":"transfer","attributes":[{"key":"recipient","value":"PAIR_ADDR"},{"key":"sender","value":"sender"},{"key":"amount","value":"1000Asset0"}]}]`
)
