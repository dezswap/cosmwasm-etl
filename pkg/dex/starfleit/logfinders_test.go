package starfleit

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/stretchr/testify/assert"
)

func Test_CreateCreateLogFinder(t *testing.T) {
	var logFinder eventlog.LogFinder
	var eventLogs eventlog.LogResults
	var err error
	setUp := func(chainId, rawLogsStr string) {
		logFinder = nil
		eventLogs = eventlog.LogResults{}
		logFinder, err = CreateCreatePairRuleFinder(chainId)
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal([]byte(rawLogsStr), &eventLogs); err != nil {
			panic(err)
		}
	}

	tcs := []struct {
		chainId           string
		rawLogStr         string
		expectedResultLen int
		errMsg            string
	}{
		{TestnetPrefix, CreatePairRawLogStr, 1, "must match once"},
		{TestnetPrefix, createTwiceLogStr, 2, "must match twice"},
		{TestnetPrefix, differentTypeLogsStr, 0, "must not match with different type"},
		{TestnetPrefix, "[]", 0, "must not match with empty logs"},
	}

	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("idx(%d): %s", idx, tc.errMsg)
		assert := assert.New(t)

		setUp(tc.chainId, tc.rawLogStr)
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
