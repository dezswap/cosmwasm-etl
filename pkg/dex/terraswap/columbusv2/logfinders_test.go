package columbusv2

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	dts "github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"

	"github.com/stretchr/testify/assert"
)

func Test_CreateCreateLogFinder(t *testing.T) {
	factoryAddr := string(dts.MAINNET_FACTORY)
	var logFinder eventlog.LogFinder
	var eventLogs eventlog.LogResults
	var err error
	setUp := func(factoryAddr, rawLogsStr string) {
		logFinder = nil
		eventLogs = eventlog.LogResults{}
		logFinder, err = CreateCreatePairRuleFinder(factoryAddr)
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal([]byte(rawLogsStr), &eventLogs); err != nil {
			panic(err)
		}
	}

	tcs := []struct {
		factoryAddr       string
		rawLogStr         string
		expectedResultLen int
		errMsg            string
	}{
		{factoryAddr, CreatePairRawLogStr, 1, "must match once"},
		{factoryAddr, createTwiceLogStr, 2, "must match twice"},
		{factoryAddr, differentTypeLogsStr, 0, "must not match with different type"},
		{factoryAddr, "[]", 0, "must not match with empty logs"},
	}

	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("idx(%d): %s", idx, tc.errMsg)
		assert := assert.New(t)

		setUp(tc.factoryAddr, tc.rawLogStr)
		matchedResults := logFinder.FindFromLogs(eventLogs)
		assert.Len(matchedResults, tc.expectedResultLen, errMsg)
		if tc.expectedResultLen > 0 {
			assert.Len(matchedResults[0], dex.CreatePairMatchedLen, "must return all matched value")
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
		rawLogStr  string
		pairs      map[string]bool
		finderFunc func(map[string]bool) (eventlog.LogFinder, error)
		matchedLen int
		errMsg     string
	}{
		//Swap
		{PairSwapRawLogStr, nil, CreatePairCommonRulesFinder, 1, "must match once"},
		// Provide
		{PairProvideRawLogStr, nil, CreatePairCommonRulesFinder, 1, "must match once"},
		// Withdraw
		{PairWithdrawRawLogStr, nil, CreatePairCommonRulesFinder, 1, "must match once"},
		// WasmTransfer
		{WasmTransferRawLogStr, nil, CreateWasmCommonTransferRuleFinder, 1, "must match once"},
		// Transfer
		{TransferRawLogStr, nil, CreateSortedTransferRuleFinder, 1, "must match once"},
	}

	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("idx(%d): %s", idx, tc.errMsg)
		assert := assert.New(t)

		setUp(tc.rawLogStr, tc.pairs, tc.finderFunc)
		matchedResults := logFinder.FindFromLogs(eventLogs)
		assert.NotEmpty(matchedResults, errMsg)
	}

}

const (
	differentTypeLogsStr = `[{ "type":"wrongType", "attributes":[{ "key":"_contract_address", "value":"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"},
			{ "key":"action", "value":"create_pair"},
			{ "key":"pair", "value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al-uluna"},
			{ "key":"_contract_address", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
			{ "key":"liquidity_token_addr", "value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
			{ "key":"_contract_address", "value":"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"}]}]`
	createTwiceLogStr = `[{ "type":"wasm", "attributes":[{ "key":"_contract_address", "value":"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"},
{ "key":"action", "value":"create_pair"},{ "key":"pair", "value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al-uluna"},{ "key":"_contract_address", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
{ "key":"liquidity_token_addr", "value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
{ "key":"_contract_address", "value":"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"}]},
{ "type":"wasm", "attributes":[{ "key":"_contract_address", "value":"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"},{ "key":"action", "value":"create_pair"},
{ "key":"pair", "value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al-uluna"},{ "key":"_contract_address", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
{ "key":"liquidity_token_addr", "value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
{ "key":"_contract_address", "value":"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"}]}
]`
)

const CreatePairRawLogStr = `[
	{"type":"execute","attributes": [{ "key":"_contract_address", "value":"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"}]},
{ "type":"instantiate", "attributes":[
					{ "key":"_contract_address", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
					{ "key":"code_id", "value":"5"},
					{ "key":"_contract_address", "value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
					{ "key":"code_id", "value":"4"}]},

					{"type":"message", "attributes":[{ "key":"action", "value":"/cosmwasm.wasm.v1.MsgExecuteContract"},
					{ "key":"module", "value":"wasm"},
					{ "key":"sender", "value":"terra1g5cad8hl9uwldus279ddc0j4fq7xjude0ynhjv"}]},
{ "type":"reply", "attributes":[{ "key":"_contract_address", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
					{ "key":"_contract_address", "value":"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"}]},
{ "type":"wasm", "attributes":[{ "key":"_contract_address", "value":"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"},
					{ "key":"action", "value":"create_pair"},
					{ "key":"pair", "value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al-uluna"},
					{ "key":"_contract_address", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
					{ "key":"liquidity_token_addr", "value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
					{ "key":"_contract_address", "value":"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"},
					{ "key":"pair_contract_addr", "value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},
					{ "key":"liquidity_token_addr", "value":"terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"}]}]`

const PairSwapRawLogStr = `[
	{"type":"coin_received","attributes":[{"key":"receiver","value":"terra13ehuhysn5mqjeaheeuew2gjs785f6k7jm8vfsqg3jhtpkwppcmzqcu7chk"},{"key":"amount","value":"1000000uluna"},{"key":"receiver","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"amount","value":"1000000uluna"}]},
{"type":"coin_spent","attributes":[{"key":"spender","value":"terra15245qvf0g473xscam0hjfag0l8yr65h6pter3x"},{"key":"amount","value":"1000000uluna"},{"key":"spender","value":"terra13ehuhysn5mqjeaheeuew2gjs785f6k7jm8vfsqg3jhtpkwppcmzqcu7chk"},{"key":"amount","value":"1000000uluna"}]},
{"type":"execute","attributes":[{"key":"_contract_address","value":"terra13ehuhysn5mqjeaheeuew2gjs785f6k7jm8vfsqg3jhtpkwppcmzqcu7chk"},{"key":"_contract_address","value":"terra13ehuhysn5mqjeaheeuew2gjs785f6k7jm8vfsqg3jhtpkwppcmzqcu7chk"},{"key":"_contract_address","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"_contract_address","value":"terra1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqynf7kp"},{"key":"_contract_address","value":"terra13ehuhysn5mqjeaheeuew2gjs785f6k7jm8vfsqg3jhtpkwppcmzqcu7chk"}]},
{"type":"message","attributes":[{"key":"action","value":"/cosmwasm.wasm.v1.MsgExecuteContract"},{"key":"module","value":"wasm"},{"key":"sender","value":"terra15245qvf0g473xscam0hjfag0l8yr65h6pter3x"}]},
{"type":"transfer","attributes":[{"key":"recipient","value":"terra13ehuhysn5mqjeaheeuew2gjs785f6k7jm8vfsqg3jhtpkwppcmzqcu7chk"},{"key":"sender","value":"terra15245qvf0g473xscam0hjfag0l8yr65h6pter3x"},{"key":"amount","value":"1000000uluna"},{"key":"recipient","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"sender","value":"terra13ehuhysn5mqjeaheeuew2gjs785f6k7jm8vfsqg3jhtpkwppcmzqcu7chk"},{"key":"amount","value":"1000000uluna"}]},
{"type":"wasm","attributes":[
  {"key":"_contract_address","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"action","value":"swap"},{"key":"sender","value":"terra13ehuhysn5mqjeaheeuew2gjs785f6k7jm8vfsqg3jhtpkwppcmzqcu7chk"},{"key":"receiver","value":"terra15245qvf0g473xscam0hjfag0l8yr65h6pter3x"},{"key":"offer_asset","value":"uluna"},
  {"key":"ask_asset","value":"terra1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqynf7kp"},{"key":"offer_amount","value":"1000000"},{"key":"return_amount","value":"99600399601"},{"key":"spread_amount","value":"99900100"},{"key":"commission_amount","value":"299700299"},
  {"key":"_contract_address","value":"terra1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqynf7kp"},{"key":"action","value":"transfer"},{"key":"from","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"to","value":"terra15245qvf0g473xscam0hjfag0l8yr65h6pter3x"},{"key":"amount","value":"99600399601"}]}
]`

const PairProvideRawLogStr = `[
	{"type":"coin_received","attributes":[{"key":"receiver","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"amount","value":"1000000000uluna"}]},
	{"type":"coin_spent","attributes":[{"key":"spender","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"},{"key":"amount","value":"1000000000uluna"}]},
	{"type":"execute","attributes":[{"key":"_contract_address","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"_contract_address","value":"terra1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqynf7kp"},{"key":"_contract_address","value":"terra1ma0g752dl0yujasnfs9yrk6uew7d0a2zrgvg62cfnlfftu2y0egqjpj90v"}]},
	{"type":"message","attributes":[{"key":"action","value":"/cosmwasm.wasm.v1.MsgExecuteContract"},{"key":"module","value":"wasm"},{"key":"sender","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"}]},
	{"type":"transfer","attributes":[{"key":"recipient","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"sender","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"},{"key":"amount","value":"1000000000uluna"}]},
	{"type":"wasm","attributes":[
		{"key":"_contract_address","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"action","value":"provide_liquidity"},{"key":"sender","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"},{"key":"receiver","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"},
		{"key":"assets","value":"100000000000000terra1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqynf7kp, 1000000000uluna"},{"key":"share","value":"316227766016"},
		{"key":"_contract_address","value":"terra1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqynf7kp"},{"key":"action","value":"transfer_from"},{"key":"from","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"},{"key":"to","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"by","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"amount","value":"100000000000000"},
		{"key":"_contract_address","value":"terra1ma0g752dl0yujasnfs9yrk6uew7d0a2zrgvg62cfnlfftu2y0egqjpj90v"},{"key":"action","value":"mint"},{"key":"to","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"},{"key":"amount","value":"316227766016"}
	]}
]`

const PairWithdrawRawLogStr = `[
	{"type":"coin_received","attributes":[{"key":"receiver","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"},{"key":"amount","value":"1003000000uluna"}]},
	{"type":"coin_spent","attributes":[{"key":"spender","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"amount","value":"1003000000uluna"}]},
	{"type":"execute","attributes":[{"key":"_contract_address","value":"terra1ma0g752dl0yujasnfs9yrk6uew7d0a2zrgvg62cfnlfftu2y0egqjpj90v"},{"key":"_contract_address","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"_contract_address","value":"terra1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqynf7kp"},{"key":"_contract_address","value":"terra1ma0g752dl0yujasnfs9yrk6uew7d0a2zrgvg62cfnlfftu2y0egqjpj90v"}]},
	{"type":"message","attributes":[{"key":"action","value":"/cosmwasm.wasm.v1.MsgExecuteContract"},{"key":"module","value":"wasm"},{"key":"sender","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"}]},
	{"type":"transfer","attributes":[{"key":"recipient","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"},{"key":"sender","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"amount","value":"1003000000uluna"}]},
	{"type":"wasm","attributes":[
		{"key":"_contract_address","value":"terra1ma0g752dl0yujasnfs9yrk6uew7d0a2zrgvg62cfnlfftu2y0egqjpj90v"},{"key":"action","value":"send"},{"key":"from","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"},{"key":"to","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"amount","value":"316227766016"},
		{"key":"_contract_address","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"action","value":"withdraw_liquidity"},{"key":"sender","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"},{"key":"withdrawn_share","value":"316227766016"},{"key":"refund_assets","value":"99701793723021terra1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqynf7kp, 1003000000uluna"},
		{"key":"_contract_address","value":"terra1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqynf7kp"},{"key":"action","value":"transfer"},{"key":"from","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"to","value":"terra103rap9vjn3v59frjd90ucmcs5wu0dy0a59fzra"},{"key":"amount","value":"99701793723021"},
		{"key":"_contract_address","value":"terra1ma0g752dl0yujasnfs9yrk6uew7d0a2zrgvg62cfnlfftu2y0egqjpj90v"},{"key":"action","value":"burn"},{"key":"from","value":"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x"},{"key":"amount","value":"316227766016"}]
	}
]`

const WasmTransferRawLogStr = `[
	{"type":"execute","attributes":[{"key":"_contract_address","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"}]},
	{"type":"message","attributes":[{"key":"action","value":"/cosmwasm.wasm.v1.MsgExecuteContract"},{"key":"module","value":"wasm"},{"key":"sender","value":"terra1g5cad8hl9uwldus279ddc0j4fq7xjude0ynhjv"}]},
	{"type":"wasm","attributes":[{"key":"_contract_address","value":"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"},{"key":"action","value":"transfer"},{"key":"from","value":"terra1g5cad8hl9uwldus279ddc0j4fq7xjude0ynhjv"},{"key":"to","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"amount","value":"1000000"}]}
]`

const TransferRawLogStr = `[
	{"type":"coin_received","attributes":[{"key":"receiver","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"amount","value":"1000000uluna"}]},
	{"type":"coin_spent","attributes":[{"key":"spender","value":"terra1g5cad8hl9uwldus279ddc0j4fq7xjude0ynhjv"},{"key":"amount","value":"1000000uluna"}]},
	{"type":"message","attributes":[{"key":"action","value":"/cosmos.bank.v1beta1.MsgSend"},{"key":"sender","value":"terra1g5cad8hl9uwldus279ddc0j4fq7xjude0ynhjv"},{"key":"module","value":"bank"}]},
	{"type":"transfer","attributes":[{"key":"amount","value":"1000000uluna"},{"key":"recipient","value":"terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y"},{"key":"sender","value":"terra1g5cad8hl9uwldus279ddc0j4fq7xjude0ynhjv"}]}
]`
