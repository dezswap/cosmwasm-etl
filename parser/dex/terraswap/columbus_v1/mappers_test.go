package columbus_v1

import (
	"fmt"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/parser/dex"
	el "github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/stretchr/testify/assert"
)

func Test_mapperMixin(t *testing.T) {
	mapperMixin := mapperMixin{}

	tcs := []struct {
		matchedResults el.MatchedResult
		expectedLen    int
		errMsg         string
	}{
		{
			el.MatchedResult{
				{Key: "recipient", Value: "Pair"}, {Key: "sender", Value: "A"}, {Key: "amount", Value: "1000Asset1"},
			},
			3,
			"",
		},
		{
			el.MatchedResult{
				{Key: "recipient", Value: "Pair"}, {Key: "sender", Value: "A"},
				{Key: "amount", Value: "1000Asset1"}, {Key: "WRONG_MATCHED_LENGTH", Value: "LENGTH"},
			},
			3,
			"must return error when matched result length is not equal to expected",
		},
		{
			el.MatchedResult{
				{Key: "recipient", Value: "Pair"}, {Key: "sender", Value: "A"}, {Key: "amount", Value: ""},
			},
			3,
			"empty value must return error",
		},
	}

	for idx, tc := range tcs {
		assert := assert.New(t)
		errMsg := fmt.Sprintf("tc(%d)", idx)

		err := mapperMixin.checkResult(tc.matchedResults, tc.expectedLen)
		if tc.errMsg != "" {
			assert.Error(err, errMsg, tc.errMsg)
		} else {
			assert.NoError(err, errMsg)
		}
	}
}

func Test_TransferMapper(t *testing.T) {
	const userAddr = "user"
	pair := dex.Pair{ContractAddr: "Pair", LpAddr: "LiquidityToken", Assets: []string{"Asset1", "Asset2"}}
	pairSet := map[string]dex.Pair{pair.ContractAddr: pair}
	tcs := []struct {
		mapper         dex.Mapper
		matchedResults el.MatchedResult
		expectedTx     *dex.ParsedTx
		errMsg         string
	}{
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000Asset1"},
			},
			&dex.ParsedTx{"", time.Time{}, dex.Transfer, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], ""}}, "", "", "", make(map[string]interface{})},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000Asset1,2000Asset2"},
			},
			&dex.ParsedTx{"", time.Time{}, dex.Transfer, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], "2000"}}, "", "", "", make(map[string]interface{})},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "123456789012345678901234567890123456Asset2"},
			},
			&dex.ParsedTx{"", time.Time{}, dex.Transfer, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], ""}, {pair.Assets[1], "123456789012345678901234567890123456"}}, "", "", "", make(map[string]interface{})},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: "WRONG_PAIR"}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000Asset1"},
			},
			nil,
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000WrongAsset1"},
			},
			&dex.ParsedTx{"", time.Time{}, dex.Transfer, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], ""}, {pair.Assets[1], ""}}, "", "", "", map[string]interface{}{"WrongAsset1": "1000"}},
			"",
		},
		/// wasm transfer
		{
			&wasmCommonTransferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.Assets[0]}, {Key: "action", Value: "transfer"}, {Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr}, {Key: "amount", Value: "1000"},
			},
			&dex.ParsedTx{"", time.Time{}, dex.Transfer, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], ""}}, "", "", "", make(map[string]interface{})},
			"",
		},
		{
			&wasmCommonTransferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.Assets[1]}, {Key: "action", Value: "transfer"},
				{Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr},
				{Key: "amount", Value: "123456789012345678901234567890123456"},
			},
			&dex.ParsedTx{"", time.Time{}, dex.Transfer, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], ""}, {pair.Assets[1], "123456789012345678901234567890123456"}}, "", "", "", make(map[string]interface{})},
			"",
		},
		{
			&wasmCommonTransferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.Assets[0]}, {Key: "action", Value: "transfer"}, {Key: "from", Value: userAddr},
				{Key: "to", Value: "WRONG_PAIR"}, {Key: "amount", Value: "1000"},
			},
			nil,
			"",
		},
		{
			&wasmCommonTransferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: "WRONG_CW_20"}, {Key: "action", Value: "transfer"},
				{Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr}, {Key: "amount", Value: "1000"},
			},
			&dex.ParsedTx{"", time.Time{}, dex.Transfer, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], ""}, {pair.Assets[1], ""}}, "", "", "", map[string]interface{}{"WRONG_CW_20": "1000"}},
			"",
		},
	}

	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("tc(%d)", idx)
		assert := assert.New(t)

		tx, err := tc.mapper.MatchedToParsedTx(tc.matchedResults)
		if tc.errMsg != "" {
			assert.Error(err, errMsg, tc.errMsg)
		}
		assert.Equal(tc.expectedTx, tx, errMsg)
	}
}

func Test_CreatePairMapper(t *testing.T) {
	const factoryAddr = "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"

	tcs := []struct {
		mapper         dex.Mapper
		matchedResults el.MatchedResult
		expectedTx     *dex.ParsedTx
		errMsg         string
	}{
		{
			&createPairMapper{},
			el.MatchedResult{
				{Key: "contract_address", Value: factoryAddr}, {Key: "action", Value: "create_pair"},
				{Key: "pair", Value: "terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al-uluna"},
				{Key: "contract_address", Value: "A"}, {Key: "liquidity_token_addr", Value: "terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
			},
			&dex.ParsedTx{"", time.Time{}, dex.CreatePair, "", "A", [2]dex.Asset{{"terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al", ""}, {"uluna", ""}}, "terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7", "", "", nil},
			"",
		},
		{
			&createPairMapper{},
			el.MatchedResult{
				{Key: "contract_address", Value: factoryAddr}, {Key: "action", Value: "create_pair"},
				{Key: "pair", Value: "INVALID_PAIR"},
				{Key: "contract_address", Value: "B"}, {Key: "liquidity_token_addr", Value: "terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
			},
			nil,
			"expected assets length(2)",
		},
		{
			&createPairMapper{},
			el.MatchedResult{
				{Key: "contract_address", Value: "IT REQUIRED MORE MATCHED"}, {Key: "action", Value: "create_pair"},
			},
			nil,
			"expected results length(5)",
		},
	}

	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("tc(%d)", idx)
		assert := assert.New(t)

		tx, err := tc.mapper.MatchedToParsedTx(tc.matchedResults)
		if tc.errMsg != "" {
			assert.Error(err, errMsg, tc.errMsg)
		}
		assert.Equal(tc.expectedTx, tx, errMsg)
	}
}

func Test_PairMapper(t *testing.T) {

	pair := dex.Pair{ContractAddr: "Pair", LpAddr: "LiquidityToken", Assets: []string{"Asset1", "Asset2"}}
	pairSet := map[string]dex.Pair{pair.ContractAddr: pair}
	tcs := []struct {
		mapper         dex.Mapper
		matchedResults el.MatchedResult
		expectedTx     *dex.ParsedTx
		errMsg         string
	}{

		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "swap"},
				{Key: "offer_asset", Value: pair.Assets[0]}, {Key: "ask_asset", Value: pair.Assets[1]},
				{Key: "offer_amount", Value: "100000"}, {Key: "return_amount", Value: "100583"},
				{Key: "tax_amount", Value: "0"},
				{Key: "spread_amount", Value: "2"}, {Key: "commission_amount", Value: "302"},
			},
			&dex.ParsedTx{"", time.Time{}, dex.Swap, "", pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "100000"}, {pair.Assets[1], "-100583"}}, "", "", "302", map[string]interface{}{"tax_amount": dex.Asset{pair.Assets[1], "0"}}},
			"",
		},
		{

			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "swap"},
				{Key: "offer_asset", Value: pair.Assets[1]}, {Key: "ask_asset", Value: pair.Assets[0]},
				{Key: "offer_amount", Value: "100000"}, {Key: "return_amount", Value: "100583"},
				{Key: "tax_amount", Value: "583"},
				{Key: "spread_amount", Value: "2"}, {Key: "commission_amount", Value: "300"},
			},
			&dex.ParsedTx{"", time.Time{}, dex.Swap, "", pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "-100000"}, {pair.Assets[1], "100000"}}, "", "", "300", map[string]interface{}{"tax_amount": dex.Asset{pair.Assets[0], "583"}}},
			"",
		},
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: "IT REQUIRED MORE MATCHED"}, {Key: "action", Value: "create_pair"},
			},
			nil,
			"expected results length(10)",
		},

		/// Provide
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "assets", Value: fmt.Sprintf("%s%s, %s%s", "1000", pair.Assets[0], "10000", pair.Assets[1])},
				{Key: "share", Value: "998735"},
			},
			&dex.ParsedTx{"", time.Time{}, dex.Provide, "", pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], "10000"}}, pair.LpAddr, "998735", "", nil},
			"",
		},
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "assets", Value: fmt.Sprintf("%s,%s %s%s", "1000", pair.Assets[0], "10000", pair.Assets[1])},
				{Key: "share", Value: "998735"},
			},
			nil,
			"Wrong format of assets must return error",
		},

		/// Withdraw
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.ContractAddr},
				{Key: "action", Value: "withdraw_liquidity"},
				{Key: "withdrawn_share", Value: "12418119"},
				{Key: "refund_assets", Value: fmt.Sprintf("%s%s, %s%s", "24999998", pair.Assets[0], "24939789", pair.Assets[1])},
			},
			&dex.ParsedTx{"", time.Time{}, dex.Withdraw, "", pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "0"}, {pair.Assets[1], "0"}}, pair.LpAddr, "12418119", "", map[string]interface{}{"withdraw_assets": []dex.Asset{{pair.Assets[0], "-24999998"}, {pair.Assets[1], "-24939789"}}}},
			"",
		},
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "assets", Value: fmt.Sprintf("%s,%s %s%s", "1000", pair.Assets[0], "10000", pair.Assets[1])},
				{Key: "share", Value: "998735"},
			},
			nil,
			"Wrong format of assets must return error",
		},
	}

	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("tc(%d)", idx)
		assert := assert.New(t)

		tx, err := tc.mapper.MatchedToParsedTx(tc.matchedResults)
		if tc.errMsg != "" {
			assert.Error(err, errMsg, tc.errMsg)
		} else {
			assert.NoError(err, err)
			assert.Equal(tc.expectedTx, tx, errMsg)
		}
	}
}
