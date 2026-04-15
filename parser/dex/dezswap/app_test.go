package dezswap

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	txSender = "sender"
	txHash   = "hash"

	// real addresses from logfinders_test.go in pkg/dex/dezswap
	pairAddr = "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"
	lpAddr   = "xpla1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"
	asset1   = "xpla1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"
	asset2   = "xpla1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"
	chainId  = "cube_47-5"
)

var testPair = dex.Pair{ContractAddr: pairAddr, LpAddr: lpAddr, Assets: []string{asset1, asset2}}

func Test_ParseTxs(t *testing.T) {
	type testcase struct {
		logStrs  []string
		expected []dex.ParsedTx
		desc     string
	}

	// height < TestnetV2Height(2975818) -> v1 mapper (no refund_assets in provide)
	const height = uint64(100)

	setUp := func() dex.DexParserApp {
		createPairParser := dex.ParserMock{}
		repo := dex.RepoMock{}
		rawStore := dex.RawStoreMock{}

		createPairParser.On("parse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]*dex.ParsedTx{}, nil)

		app := dezswapApp{
			PairRepo:    &repo,
			Parsers:     &dex.PairParsers{CreatePairParser: &createPairParser},
			DexMixin:    dex.DexMixin{},
			chainId:     chainId,
			pairs:       map[string]dex.Pair{pairAddr: testPair},
			lpPairAddrs: map[string]string{lpAddr: pairAddr},
		}
		dexApp := dex.NewDexApp(&app, &rawStore, &repo, logging.New("test", configs.LogConfig{}), configs.ParserDexConfig{})
		return dexApp.(dex.DexParserApp)
	}

	tcs := []testcase{
		{
			logStrs:  nil,
			expected: []dex.ParsedTx{},
			desc:     "empty logs produce no txs",
		},
		{
			logStrs:  []string{swapLogStr},
			expected: []dex.ParsedTx{swapTx},
			desc:     "swap log produces one swap tx",
		},
		{
			logStrs:  []string{provideLogStr},
			expected: []dex.ParsedTx{provideTx},
			desc:     "provide log produces one provide tx; wasm transfers are deduplicated",
		},
		{
			logStrs:  []string{withdrawLogStr},
			expected: []dex.ParsedTx{withdrawTx},
			desc:     "withdraw log produces withdraw tx; lp burn from pair contract is filtered out",
		},
	}

	for idx, tc := range tcs {
		assert := assert.New(t)
		msg := fmt.Sprintf("tc(%d): %s", idx, tc.desc)

		app := setUp()

		logs := eventlog.LogResults{}
		for _, s := range tc.logStrs {
			var results eventlog.LogResults
			if err := json.Unmarshal([]byte(s), &results); err != nil {
				t.Fatal(err)
			}
			logs = append(logs, results...)
		}

		tx := parser.RawTx{Sender: txSender, Hash: txHash, LogResults: logs}

		err := app.UpdateParsers(make(map[string]bool), uint64(height))
		assert.NoError(err)

		txs, err := app.ParseTxs(tx, height)
		assert.NoError(err, msg)
		assert.Equal(tc.expected, txs, msg)
	}
}

func Test_IsValidationExceptionCandidate(t *testing.T) {
	app := &dezswapApp{}
	assert.False(t, app.IsValidationExceptionCandidate("any_address"))
	assert.False(t, app.IsValidationExceptionCandidate(""))
}

var (
	swapTx = dex.ParsedTx{
		Hash: txHash, Timestamp: time.Time{},
		Type: dex.Swap, Sender: txSender, ContractAddr: pairAddr,
		Assets: [2]dex.Asset{
			{Addr: asset1, Amount: "-37175064560952362156"},
			{Addr: asset2, Amount: "36691384354750000000"},
		},
		CommissionAmount: "111860776010889756",
	}
	provideTx = dex.ParsedTx{
		Hash: txHash, Timestamp: time.Time{},
		Type: dex.Provide, Sender: txSender, ContractAddr: pairAddr,
		Assets: [2]dex.Asset{
			{Addr: asset1, Amount: "1000000000000000000000"},
			{Addr: asset2, Amount: "1000000000000000000000"},
		},
		LpAddr: lpAddr, LpAmount: "1000000000000000000000",
	}
	withdrawTx = dex.ParsedTx{
		Hash: txHash, Timestamp: time.Time{},
		Type: dex.Withdraw, Sender: txSender, ContractAddr: pairAddr,
		Assets: [2]dex.Asset{
			{Addr: asset1, Amount: "-1100109276349974322"},
			{Addr: asset2, Amount: "-1097303402006688516"},
		},
		LpAddr: lpAddr, LpAmount: "1098669138945462355",
	}
)

// log strings are taken from pkg/dex/dezswap/logfinders_test.go
const swapLogStr = `[
    {
        "type": "execute",
        "attributes": [
            {"key": "_contract_address", "value": "xpla1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "_contract_address", "value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "_contract_address", "value": "xpla1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"}
        ]},
    {
        "type": "message",
        "attributes": [
            {"key": "action", "value": "/cosmwasm.wasm.v1.MsgExecuteContract"},
            {"key": "module", "value": "wasm"},
            {"key": "sender", "value": "xpla190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"}
        ]},
    {
        "type": "wasm",
        "attributes": [
            {"key": "_contract_address", "value": "xpla1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "action", "value": "send"},
            {"key": "from", "value": "xpla190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "to", "value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "amount", "value": "36691384354750000000"},
            {"key": "_contract_address", "value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "action", "value": "swap"},
            {"key": "ask_asset", "value": "xpla1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},
            {"key": "commission_amount", "value": "111860776010889756"},
            {"key": "offer_amount", "value": "36691384354750000000"},
            {"key": "offer_asset", "value": "xpla1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "receiver", "value": "xpla190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "return_amount", "value": "37175064560952362156"},
            {"key": "sender", "value": "xpla190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "spread_amount", "value": "1360327158956032802"},
            {"key": "_contract_address", "value": "xpla1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},
            {"key": "action", "value": "transfer"},
            {"key": "amount", "value": "37175064560952362156"},
            {"key": "from", "value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "to", "value": "xpla190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"}
        ]}
]`

const provideLogStr = `[
    {
        "type": "execute",
        "attributes": [
            {"key": "_contract_address","value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "_contract_address","value": "xpla1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},
            {"key": "_contract_address","value": "xpla1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "_contract_address","value": "xpla1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"}]},
    {
        "type": "message",
        "attributes": [
            {"key": "action","value": "/cosmwasm.wasm.v1.MsgExecuteContract"},
            {"key": "module","value": "wasm"},
            {"key": "sender","value": "xpla190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"}]},
    {
        "type": "wasm",
        "attributes": [
            {"key": "_contract_address","value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "action","value": "provide_liquidity"},
            {"key": "sender","value": "xpla190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "receiver","value": "xpla190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "assets","value": "1000000000000000000000xpla1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm, 1000000000000000000000xpla1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "share","value": "1000000000000000000000"},
            {"key": "_contract_address","value": "xpla1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},
            {"key": "action","value": "transfer_from"},
            {"key": "amount","value": "1000000000000000000000"},
            {"key": "by","value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "from","value": "xpla190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "to","value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "_contract_address","value": "xpla1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "action","value": "transfer_from"},
            {"key": "amount","value": "1000000000000000000000"},
            {"key": "by","value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "from","value": "xpla190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "to","value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "_contract_address","value": "xpla1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
            {"key": "action","value": "mint"},
            {"key": "amount","value": "1000000000000000000000"},
            {"key": "to","value": "xpla190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"}]}
]`

const withdrawLogStr = `[
    {
        "type": "execute",
        "attributes": [
            {"key": "_contract_address", "value": "xpla1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
            {"key": "_contract_address", "value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "_contract_address", "value": "xpla1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},
            {"key": "_contract_address", "value": "xpla1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "_contract_address", "value": "xpla1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"}
        ]},
    {
        "type": "message",
        "attributes": [
            {"key": "action", "value": "/cosmwasm.wasm.v1.MsgExecuteContract"},
            {"key": "module", "value": "wasm"},
            {"key": "sender", "value": "xpla1s4gljj0ksjkhh5vsk3lvw2s9rpflyq6k7e575x"}
        ]},
    {
        "type": "wasm",
        "attributes": [
            {"key": "_contract_address", "value": "xpla1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
            {"key": "action", "value": "send"},
            {"key": "from", "value": "xpla1s4gljj0ksjkhh5vsk3lvw2s9rpflyq6k7e575x"},
            {"key": "to", "value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "amount", "value": "1098669138945462355"},
            {"key": "_contract_address", "value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "action", "value": "withdraw_liquidity"},
            {"key": "refund_assets", "value": "1100109276349974322xpla1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm, 1097303402006688516xpla1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "sender", "value": "xpla1s4gljj0ksjkhh5vsk3lvw2s9rpflyq6k7e575x"},
            {"key": "withdrawn_share", "value": "1098669138945462355"},
            {"key": "_contract_address", "value": "xpla1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},
            {"key": "action", "value": "transfer"},
            {"key": "amount", "value": "1100109276349974322"},
            {"key": "from", "value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "to", "value": "xpla1s4gljj0ksjkhh5vsk3lvw2s9rpflyq6k7e575x"},
            {"key": "_contract_address", "value": "xpla1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "action", "value": "transfer"},
            {"key": "amount", "value": "1097303402006688516"},
            {"key": "from", "value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "to", "value": "xpla1s4gljj0ksjkhh5vsk3lvw2s9rpflyq6k7e575x"},
            {"key": "_contract_address", "value": "xpla1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
            {"key": "action", "value": "burn"},
            {"key": "amount", "value": "1098669138945462355"},
            {"key": "from", "value": "xpla1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"}
        ]
    }
]`
