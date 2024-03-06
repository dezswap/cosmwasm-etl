package columbus_v1

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/stretchr/testify/assert"
)

// // move to dex
// func Test_CreateCreateLogFinder(t *testing.T) {
// 	var (
// 		logFinder   eventlog.LogFinder
// 		eventLogs   eventlog.LogResults
// 		err         error
// 		factoryAddr = string(dts.CLASSIC_V1_FACTORY)
// 	)
// 	setUp := func(factoryAddr, rawLogsStr string) {
// 		logFinder = nil
// 		eventLogs = eventlog.LogResults{}
// 		logFinder, err = CreateCreatePairRuleFinder(factoryAddr)
// 		if err != nil {
// 			panic(err)
// 		}
// 		if err := json.Unmarshal([]byte(rawLogsStr), &eventLogs); err != nil {
// 			panic(err)
// 		}
// 	}

// 	tcs := []struct {
// 		factoryAddr       string
// 		rawLogStr         string
// 		expectedResultLen int
// 		errMsg            string
// 	}{
// 		{factoryAddr, CreatePairRawLogStr, 1, "must match once"},
// 		{factoryAddr, createTwiceLogStr, 2, "must match twice"},
// 		{factoryAddr, differentTypeLogsStr, 0, "must not match with different type"},
// 		{factoryAddr, "[]", 0, "must not match with empty logs"},
// 	}

// 	for idx, tc := range tcs {
// 		errMsg := fmt.Sprintf("idx(%d): %s", idx, tc.errMsg)
// 		assert := assert.New(t)

// 		setUp(tc.factoryAddr, tc.rawLogStr)
// 		matchedResults := logFinder.FindFromLogs(eventLogs)
// 		assert.Len(matchedResults, tc.expectedResultLen, errMsg)
// 		if tc.expectedResultLen > 0 {
// 			assert.Len(matchedResults[0], CreatePairMatchedLen, "must return all matched value")
// 		}
// 	}
// }

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
		{PairSwapRawLogStr, nil, CreatePairCommonRulesFinder, 1, PairSwapMatchedLen, "must match once"},
		// Provide
		{PairProvideRawLogStr, nil, CreatePairCommonRulesFinder, 1, PairProvideMatchedLen, "must match once"},
		// Withdraw
		{PairWithdrawRawLogStr, nil, CreatePairCommonRulesFinder, 1, PairWithdrawMatchedLen, "must match once"},
		// WasmTransfer
		{WasmTransferRawLogStr, nil, CreateWasmCommonTransferRuleFinder, 1, WasmTransferMatchedLen, "must match once"},
		// Transfer
		{TransferRawLogStr, nil, CreateTransferRuleFinder, 1, TransferMatchedLen, "must match once"},
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
	differentTypeLogsStr = `[{ "type":"wrongType", "attributes":[{ "key":"contract_address", "value":"terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"},
			{ "key":"action", "value":"create_pair"},
			{ "key":"pair", "value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al-uluna"},
			{ "key":"contract_address", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
			{ "key":"liquidity_token_addr", "value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
			{ "key":"contract_address", "value":"terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"}]}]`
	createTwiceLogStr = `[{ "type":"from_contract", "attributes":[{ "key":"contract_address", "value":"terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"},
{ "key":"action", "value":"create_pair"},{ "key":"pair", "value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al-uluna"},{ "key":"contract_address", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
{ "key":"liquidity_token_addr", "value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
{ "key":"contract_address", "value":"terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"}]},
{ "type":"from_contract", "attributes":[{ "key":"contract_address", "value":"terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"},{ "key":"action", "value":"create_pair"},
{ "key":"pair", "value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al-uluna"},{ "key":"contract_address", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
{ "key":"liquidity_token_addr", "value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
{ "key":"contract_address", "value":"terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"}]}
]`
)

const CreatePairRawLogStr = `[{"type":"execute","attributes": [{ "key":"contract_address", "value":"terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"}]},
{ "type":"instantiate", "attributes":[
					{ "key":"contract_address", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
					{ "key":"code_id", "value":"5"},
					{ "key":"contract_address", "value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
					{ "key":"code_id", "value":"4"}]},
					{"type":"message", "attributes":[{ "key":"action", "value":"/cosmwasm.wasm.v1.MsgExecuteContract"},
					{ "key":"module", "value":"from_contract"},
					{ "key":"sender", "value":"terra1g5cad8hl9uwldus279ddc0j4fq7xjude0ynhjv"}]},
{ "type":"reply", "attributes":[{ "key":"contract_address", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
					{ "key":"contract_address", "value":"terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"}]},
{ "type":"from_contract", "attributes":[{ "key":"contract_address", "value":"terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"},
					{ "key":"action", "value":"create_pair"},
					{ "key":"pair", "value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al-uluna"},
					{ "key":"contract_address", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
					{ "key":"liquidity_token_addr", "value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
					{ "key":"contract_address", "value":"terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"},
					{ "key":"pair_contract_addr", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
					{ "key":"liquidity_token_addr", "value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"}]}]`

const PairSwapRawLogStr = `[{"type":"coin_received","attributes":[{"key":"receiver","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"amount","value":"100000uluna"}]},
{"type":"coin_spent","attributes":[{"key":"spender","value":"terra1tv7x48jderh5n9jva3vnsduhdprxpapagcly6s"},{"key":"amount","value":"100000uluna"}]},
{"type":"execute","attributes":[{"key":"contract_address","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"contract_address","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"}]},
{"type":"message","attributes":[{"key":"action","value":"/cosmwasm.wasm.v1.MsgExecuteContract"},{"key":"module","value":"from_contract"},{"key":"sender","value":"terra1tv7x48jderh5n9jva3vnsduhdprxpapagcly6s"}]},
{"type":"transfer","attributes":[{"key":"recipient","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"sender","value":"terra1tv7x48jderh5n9jva3vnsduhdprxpapagcly6s"},{"key":"amount","value":"100000uluna"}]},
{"type":"from_contract","attributes":[{"key":"contract_address","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"action","value":"swap"},
	{"key":"receiver","value":"terra1tv7x48jderh5n9jva3vnsduhdprxpapagcly6s"},{"key":"offer_asset","value":"uluna"},{"key":"ask_asset","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"},{"key":"offer_amount","value":"100000"},
	{"key":"return_amount","value":"100583"},{"key":"spread_amount","value":"2"},{"key":"commission_amount","value":"302"},{"key":"contract_address","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"},{"key":"action","value":"transfer"},
	{"key":"from","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
	{"key":"to","value":"terra1tv7x48jderh5n9jva3vnsduhdprxpapagcly6s"},{"key":"amount","value":"100583"}]}]`

const PairProvideRawLogStr = `[
	{"type":"execute","attributes":[{"key":"contract_address","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"}]},
	{"type":"message","attributes":[{"key":"action","value":"/cosmwasm.wasm.v1.MsgExecuteContract"},{"key":"module","value":"from_contract"},{"key":"sender","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"}]},
	{"type":"from_contract","attributes":[{"key":"contract_address","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"},{"key":"action","value":"increase_allowance"},{"key":"owner","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"},{"key":"spender","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"amount","value":"2013569"}]},
	{"type":"coin_received","attributes":[{"key":"receiver","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"amount","value":"2000000uluna"}]},
	{"type":"coin_spent","attributes":[{"key":"spender","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"},{"key":"amount","value":"2000000uluna"}]},
	{"type":"execute","attributes":[{"key":"contract_address","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"contract_address","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"},{"key":"contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"}]},
	{"type":"message","attributes":[{"key":"action","value":"/cosmwasm.wasm.v1.MsgExecuteContract"},{"key":"module","value":"from_contract"},{"key":"sender","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"}]},
	{"type":"transfer","attributes":[{"key":"recipient","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"sender","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"},{"key":"amount","value":"2000000uluna"}]},
	{"type":"from_contract","attributes":[{"key":"contract_address","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"action","value":"provide_liquidity"},{"key":"assets","value":"2000000uluna, 2013569terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"},{"key":"share","value":"998735"},{"key":"contract_address","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"},{"key":"action","value":"transfer_from"},{"key":"from","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"},{"key":"to","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"by","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"amount","value":"2013569"},{"key":"contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"action","value":"mint"},{"key":"to","value":"terra160lml094xruqkufvapdm6j3qph8ppkrjt2m4dd"},{"key":"amount","value":"998735"}]}
	]`

const PairWithdrawRawLogStr = `[
	{"type":"coin_received","attributes":[{"key":"receiver","value":"terra1cupj7d70jrtjxqhpr6s3qq68t8ky4smcjvccm4"},{"key":"amount","value":"24939789uluna"}]},
	{"type":"coin_spent","attributes":[{"key":"spender","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"amount","value":"24939789uluna"}]},
	{"type":"execute","attributes":[{"key":"contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"contract_address","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"contract_address","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"},{"key":"contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"}]},
	{"type":"message","attributes":[{"key":"action","value":"/cosmwasm.wasm.v1.MsgExecuteContract"},{"key":"module","value":"from_contract"},{"key":"sender","value":"terra1cupj7d70jrtjxqhpr6s3qq68t8ky4smcjvccm4"}]},
	{"type":"transfer","attributes":[{"key":"recipient","value":"terra1cupj7d70jrtjxqhpr6s3qq68t8ky4smcjvccm4"},{"key":"sender","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"amount","value":"24939789uluna"}]},
	{"type":"from_contract","attributes":[{"key":"contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"action","value":"send"},{"key":"from","value":"terra1cupj7d70jrtjxqhpr6s3qq68t8ky4smcjvccm4"},{"key":"to","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"amount","value":"12418119"},{"key":"contract_address","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"action","value":"withdraw_liquidity"},{"key":"withdrawn_share","value":"12418119"},{"key":"refund_assets","value":"24999998terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al, 24939789uluna"},{"key":"contract_address","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"},{"key":"action","value":"transfer"},{"key":"from","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"to","value":"terra1cupj7d70jrtjxqhpr6s3qq68t8ky4smcjvccm4"},{"key":"amount","value":"24999998"},{"key":"contract_address","value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},{"key":"action","value":"burn"},{"key":"from","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"amount","value":"12418119"}]}
	]`

const WasmTransferRawLogStr = `[
	{"type":"execute","attributes":[{"key":"contract_address","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"}]},
	{"type":"message","attributes":[{"key":"action","value":"/cosmwasm.wasm.v1.MsgExecuteContract"},{"key":"module","value":"from_contract"},{"key":"sender","value":"terra1g5cad8hl9uwldus279ddc0j4fq7xjude0ynhjv"}]},
	{"type":"from_contract","attributes":[{"key":"contract_address","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"},{"key":"action","value":"transfer"},{"key":"from","value":"terra1g5cad8hl9uwldus279ddc0j4fq7xjude0ynhjv"},{"key":"to","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"amount","value":"1000000"}]}
	]`

const TransferRawLogStr = `[
	{"type":"coin_received","attributes":[{"key":"receiver","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"amount","value":"1000000uluna"}]},
	{"type":"coin_spent","attributes":[{"key":"spender","value":"terra1g5cad8hl9uwldus279ddc0j4fq7xjude0ynhjv"},{"key":"amount","value":"1000000uluna"}]},
	{"type":"message","attributes":[{"key":"action","value":"/cosmos.bank.v1beta1.MsgSend"},{"key":"sender","value":"terra1g5cad8hl9uwldus279ddc0j4fq7xjude0ynhjv"},{"key":"module","value":"bank"}]},
	{"type":"transfer","attributes":[{"key":"recipient","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"sender","value":"terra1g5cad8hl9uwldus279ddc0j4fq7xjude0ynhjv"},{"key":"amount","value":"1000000uluna"}]}
	]`
