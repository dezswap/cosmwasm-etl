package starfleit

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CreateCreateLogFinder(t *testing.T) {
	var logFinder eventlog.LogFinder
	var eventLogs eventlog.LogResults
	var err error
	setUp := func(factoryAddress, rawLogsStr string) {
		logFinder = nil
		eventLogs = eventlog.LogResults{}
		logFinder, err = CreateCreatePairRuleFinder(factoryAddress)
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal([]byte(rawLogsStr), &eventLogs); err != nil {
			panic(err)
		}
	}

	tcs := []struct {
		factoryAddress    string
		rawLogStr         string
		expectedResultLen int
		errMsg            string
	}{
		{FactoryAddress[TestnetPrefix], CreatePairRawLogStr, 1, "must match once"},
		{FactoryAddress[TestnetPrefix], createTwiceLogStr, 2, "must match twice"},
		{FactoryAddress[TestnetPrefix], differentTypeLogsStr, 0, "must not match with different type"},
		{FactoryAddress[TestnetPrefix], "[]", 0, "must not match with empty logs"},
	}

	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("idx(%d): %s", idx, tc.errMsg)
		assert := assert.New(t)

		setUp(tc.factoryAddress, tc.rawLogStr)
		matchedResults := logFinder.FindFromLogs(eventLogs)
		assert.Len(matchedResults, tc.expectedResultLen, errMsg)
		if tc.expectedResultLen > 0 {
			assert.Len(matchedResults[0], CreatePairMatchedLen, "must return all matched value")
		}
	}
}

func Test_LogFinders(t *testing.T) {
	var logFinder eventlog.LogFinder
	var eventLogs eventlog.LogResults

	setUp := func(rawLogsStr string, pairs map[string]bool, finderFunc func(map[string]bool) (eventlog.LogFinder, error)) {
		var err error
		logFinder = nil
		eventLogs = eventlog.LogResults{}
		logFinder, err = finderFunc(pairs)
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal([]byte(rawLogsStr), &eventLogs); err != nil {
			panic(err)
		}
	}

	tcs := []struct {
		rawLogStr         string
		pairs             map[string]bool
		finderFunc        func(map[string]bool) (eventlog.LogFinder, error)
		expectedResultLen int
		matchedLen        int
		errMsg            string
	}{
		//Swap
		{PairSwapRawLogStr, nil, CreatePairAllRulesFinder, 1, PairSwapMatchedLen, "must match once"},
		{PairSwapRawLogStr, map[string]bool{"fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh": true}, CreatePairSwapRuleFinder, 1, PairSwapMatchedLen, "must match once"},
		{PairSwapRawLogStr, map[string]bool{"DIFFERENT_PAIR_ADDR": true}, CreatePairSwapRuleFinder, 0, 0, "must not match"},
		// Provide
		{PairProvideRawLogStr, nil, CreatePairAllRulesFinder, 1, PairProvideMatchedLen, "must match once"},
		{PairProvideRawLogStr, map[string]bool{"fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh": true}, CreatePairProviderRuleFinder, 1, PairProvideMatchedLen, "must match once"},
		{PairProvideRawLogStr, map[string]bool{"DIFFERENT_PAIR_ADDR": true}, CreatePairProviderRuleFinder, 0, 0, "must not match"},
		// Withdraw
		{PairWithdrawRawLogStr, nil, CreatePairAllRulesFinder, 1, PairWithdrawMatchedLen, "must match once"},
		{PairWithdrawRawLogStr, map[string]bool{"fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh": true}, CreatePairWithdrawRuleFinder, 1, PairWithdrawMatchedLen, "must match once"},
		{PairWithdrawRawLogStr, map[string]bool{"DIFFERENT_PAIR_ADDR": true}, CreatePairWithdrawRuleFinder, 0, 0, "must not match"},
	}

	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("idx(%d): %s", idx, tc.errMsg)
		assert := assert.New(t)

		setUp(tc.rawLogStr, tc.pairs, tc.finderFunc)
		matchedResults := logFinder.FindFromLogs(eventLogs)
		assert.Len(matchedResults, tc.expectedResultLen, errMsg)
		if tc.expectedResultLen > 0 {
			assert.Len(matchedResults[0], tc.matchedLen, "must return all matched value")
		}
	}

}

const (
	differentTypeLogsStr = `[{ "type":"wrongType", "attributes":[{ "key":"_contract_address", "value":"fetch1j4kgjl6h4rt96uddtzdxdu39h0mhn4vrtydufdrk4uxxnrpsnw2qug2yx2"},
			{ "key":"action", "value":"create_pair"},
			{ "key":"pair", "value":"fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm-afetch"},
			{ "key":"_contract_address", "value":"fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
			{ "key":"liquidity_token_addr", "value":"fetch1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
			{ "key":"_contract_address", "value":"fetch1j4kgjl6h4rt96uddtzdxdu39h0mhn4vrtydufdrk4uxxnrpsnw2qug2yx2"}]}]`
	createTwiceLogStr = `[{ "type":"wasm", "attributes":[{ "key":"_contract_address", "value":"fetch1kmag3937lrl6dtsv29mlfsedzngl9egv5c3apnr468q50gu04zrqea398u"},
{ "key":"action", "value":"create_pair"},{ "key":"pair", "value":"fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm-afetch"},{ "key":"_contract_address", "value":"fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
{ "key":"liquidity_token_addr", "value":"fetch1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
{ "key":"_contract_address", "value":"fetch1j4kgjl6h4rt96uddtzdxdu39h0mhn4vrtydufdrk4uxxnrpsnw2qug2yx2"}]},
{ "type":"wasm", "attributes":[{ "key":"_contract_address", "value":"fetch1kmag3937lrl6dtsv29mlfsedzngl9egv5c3apnr468q50gu04zrqea398u"},{ "key":"action", "value":"create_pair"},
{ "key":"pair", "value":"fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm-afetch"},{ "key":"_contract_address", "value":"fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
{ "key":"liquidity_token_addr", "value":"fetch1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
{ "key":"_contract_address", "value":"fetch1j4kgjl6h4rt96uddtzdxdu39h0mhn4vrtydufdrk4uxxnrpsnw2qug2yx2"}]}
]`
)

const CreatePairRawLogStr = `[
    {
        "type": "execute",
        "attributes": [
            { "key": "_contract_address", "value": "fetch1j4kgjl6h4rt96uddtzdxdu39h0mhn4vrtydufdrk4uxxnrpsnw2qug2yx2"}
        ]
    },
    {
        "type": "instantiate",
        "attributes": [
            { "key": "_contract_address", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            { "key": "code_id", "value": "111"},
            { "key": "_contract_address", "value": "fetch1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
            { "key": "code_id", "value": "110"}
        ]
    },
    {
        "type": "message",
        "attributes": [
            { "key": "action", "value": "/cosmwasm.wasm.v1.MsgExecuteContract"},
            { "key": "module", "value": "wasm"},
            { "key": "sender", "value": "fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"}
        ]
    },
    {
        "type": "reply",
        "attributes": [
            { "key": "_contract_address", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            { "key": "_contract_address", "value": "fetch1j4kgjl6h4rt96uddtzdxdu39h0mhn4vrtydufdrk4uxxnrpsnw2qug2yx2"}
        ]
    },
    {
        "type": "wasm",
        "attributes": [
            { "key": "_contract_address", "value": "fetch1kmag3937lrl6dtsv29mlfsedzngl9egv5c3apnr468q50gu04zrqea398u"},
            { "key": "action", "value": "create_pair"},
            { "key": "pair", "value": "fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm-fetch1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            { "key": "_contract_address", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            { "key": "liquidity_token_addr", "value": "fetch1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
            { "key": "_contract_address", "value": "fetch1j4kgjl6h4rt96uddtzdxdu39h0mhn4vrtydufdrk4uxxnrpsnw2qug2yx2"},
            { "key": "pair_contract_addr", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            { "key": "liquidity_token_addr", "value": "fetch1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"}
        ]
    }
]`

const PairSwapRawLogStr = `[
    {
        "type": "execute",
        "attributes": [
            {"key": "_contract_address", "value": "fetch1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "_contract_address", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "_contract_address", "value": "fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"}
        ]},
    {
        "type": "message",
        "attributes": [
            {"key": "action", "value": "/cosmwasm.wasm.v1.MsgExecuteContract"},
            {"key": "module", "value": "wasm"},
            {"key": "sender", "value": "fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"}
        ]},
    {
        "type": "wasm",
        "attributes": [
            {"key": "_contract_address", "value": "fetch1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "action", "value": "send"},
            {"key": "from", "value": "fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "to", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "amount", "value": "36691384354750000000"},
            {"key": "_contract_address", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "action", "value": "swap"},
            {"key": "ask_asset", "value": "fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},
            {"key": "commission_amount", "value": "111860776010889756"},
            {"key": "offer_amount", "value": "36691384354750000000"},
            {"key": "offer_asset", "value": "fetch1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "receiver", "value": "fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "return_amount", "value": "37175064560952362156"},
            {"key": "sender", "value": "fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "spread_amount", "value": "1360327158956032802"},
            {"key": "_contract_address", "value": "fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},
            {"key": "action", "value": "transfer"},
            {"key": "amount", "value": "37175064560952362156"},
            {"key": "from", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "to", "value": "fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"}
        ]}
]`

const PairProvideRawLogStr = `[
    {
        "type": "execute",
        "attributes": [
            {"key": "_contract_address","value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "_contract_address","value": "fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},
            {"key": "_contract_address","value": "fetch1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "_contract_address","value": "fetch1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"}]},
    {
        "type": "message",
        "attributes": [
            {"key": "action","value": "/cosmwasm.wasm.v1.MsgExecuteContract"},
            {"key": "module","value": "wasm"},
            {"key": "sender","value": "fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"}]},
    {
        "type": "wasm",
        "attributes": [
            {"key": "_contract_address","value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "action","value": "provide_liquidity"},
            {"key": "sender","value": "fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "receiver","value": "fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "assets","value": "1000000000000000000000fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm, 1000000000000000000000fetch1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "share","value": "1000000000000000000000"},
            {"key": "_contract_address","value": "fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},
            {"key": "action","value": "transfer_from"},
            {"key": "amount","value": "1000000000000000000000"},
            {"key": "by","value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "from","value": "fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "to","value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "_contract_address","value": "fetch1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "action","value": "transfer_from"},
            {"key": "amount","value": "1000000000000000000000"},
            {"key": "by","value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "from","value": "fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},
            {"key": "to","value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "_contract_address","value": "fetch1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
            {"key": "action","value": "mint"},
            {"key": "amount","value": "1000000000000000000000"},
            {"key": "to","value": "fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"}]}
]`

const PairWithdrawRawLogStr = `[
    {
        "type": "execute",
        "attributes": [
            {"key": "_contract_address", "value": "fetch1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
            {"key": "_contract_address", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "_contract_address", "value": "fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},
            {"key": "_contract_address", "value": "fetch1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "_contract_address", "value": "fetch1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"}
        ]},
    {
        "type": "message",
        "attributes": [
            {"key": "action", "value": "/cosmwasm.wasm.v1.MsgExecuteContract"},
            {"key": "module", "value": "wasm"},
            {"key": "sender", "value": "fetch1s4gljj0ksjkhh5vsk3lvw2s9rpflyq6k7e575x"}
        ]},
    {
        "type": "wasm",
        "attributes": [
            {"key": "_contract_address", "value": "fetch1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
            {"key": "action", "value": "send"},
            {"key": "from", "value": "fetch1s4gljj0ksjkhh5vsk3lvw2s9rpflyq6k7e575x"},
            {"key": "to", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "amount", "value": "1098669138945462355"},
            {"key": "_contract_address", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "action", "value": "withdraw_liquidity"},
            {"key": "refund_assets", "value": "1100109276349974322fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm, 1097303402006688516fetch1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "sender", "value": "fetch1s4gljj0ksjkhh5vsk3lvw2s9rpflyq6k7e575x"},
            {"key": "withdrawn_share", "value": "1098669138945462355"},
            {"key": "_contract_address", "value": "fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},
            {"key": "action", "value": "transfer"},
            {"key": "amount", "value": "1100109276349974322"},
            {"key": "from", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "to", "value": "fetch1s4gljj0ksjkhh5vsk3lvw2s9rpflyq6k7e575x"},
            {"key": "_contract_address", "value": "fetch1v2ezcmgzmvwdtp9m0nyfy38p85dnkn0excnyy6dqylm65fhft0qsrzmktv"},
            {"key": "action", "value": "transfer"},
            {"key": "amount", "value": "1097303402006688516"},
            {"key": "from", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},
            {"key": "to", "value": "fetch1s4gljj0ksjkhh5vsk3lvw2s9rpflyq6k7e575x"},
            {"key": "_contract_address", "value": "fetch1aye7rggr2w0dgpwuwul0y6nyxau2k5jjrpmrxtkcvsd7nlx2nz0su357u5"},
            {"key": "action", "value": "burn"},
            {"key": "amount", "value": "1098669138945462355"},
            {"key": "from", "value": "fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"}
        ]
	}
]`

const TransferRawLogStr = `[
	{"type":"execute","attributes":[{"key":"_contract_address","value":"fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"}]},
	{"type":"message","attributes":[{"key":"action","value":"/cosmwasm.wasm.v1.MsgExecuteContract"},{"key":"module","value":"wasm"},{"key":"sender","value":"fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"}]},
	{"type":"wasm","attributes":[{"key":"_contract_address","value":"fetch1w6hv0suf8dmpq8kxd8a6yy9fnmntlh7hh9kl37qmax7kyzfd047qnnp0mm"},{"key":"action","value":"transfer"},{"key":"from","value":"fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},{"key":"to","value":"fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},{"key":"amount","value":"1000000"}]}
	]`

const WasmTransferRawLogStr = `[
	{"type":"coin_received","attributes":[{"key":"receiver","value":"fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},{"key":"amount","value":"1000000afetch"}]},
	{"type":"coin_spent","attributes":[{"key":"spender","value":"fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},{"key":"amount","value":"1000000afetch"}]},
	{"type":"message","attributes":[{"key":"action","value":"/cosmos.bank.v1beta1.MsgSend"},{"key":"sender","value":"fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},{"key":"module","value":"bank"}]},
	{"type":"transfer","attributes":[{"key":"recipient","value":"fetch1ng9mj65a5cunzvkdqctgsv3pewgrx2hvk9tnrww77v3tk2lp7c9qllk0xh"},{"key":"sender","value":"fetch190465x8qz4p7uxylrmwcn8rufkv30j655h6h7q"},{"key":"amount","value":"1000000afetch"}]}
	]`

// Since cosmos-sdk v0.50 a tx keeps one wasm event per contract emission, each ending
// with a msg_index attribute, instead of one merged wasm event per message.
// fetchhub-4 AB89942C05459804782896C27B9575405695BAE779FC8D4693274B3FA8C26A5A
func Test_CreateCreateLogFinder_Sdk50SplitEvents(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	eventLogs := eventlog.LogResults{}
	require.NoError(json.Unmarshal([]byte(sdk50CreatePairRawLogStr), &eventLogs))

	logFinder, err := CreateCreatePairRuleFinder(FactoryAddress[MainnetPrefix])
	require.NoError(err)

	matchedResults := logFinder.FindFromLogs(eventLogs)
	require.Len(matchedResults, 1)
	require.Len(matchedResults[0], CreatePairMatchedLen)
	assert.Equal("fetch1z2fy5y08nvx3tvs0e48vn88gldr7wn9u5chgvm55n0r4kpjlgk7sarnunq", matchedResults[0][FactoryPairAddrIdx].Value)
	assert.Equal("fetch1axuck3rz2xf3p44tg04hzj6sk0kxkl98ls7mplu306hxfnwnt04sjzmnrv", matchedResults[0][FactoryLpAddrIdx].Value)
}

func Test_CreateCreateLogFinder_EmptyFactoryAddress(t *testing.T) {
	_, err := CreateCreatePairRuleFinder("")
	assert.Error(t, err)
}

// Attributes are grouped per event type by the source stores, so the three wasm
// events of a create_pair arrive as one attribute sequence.
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
