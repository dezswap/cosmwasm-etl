package dezswap

import (
	"fmt"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	el "github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/xpla"
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
				{Key: "amount", Value: "1000Asset1"}, {Key: "recipient", Value: "Pair"}, {Key: "sender", Value: "A"},
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
		mapper         parser.Mapper[dex.ParsedTx]
		matchedResults el.MatchedResult
		expectedTx     []*dex.ParsedTx
		errMsg         string
	}{
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: userAddr}, {Key: "sender", Value: pair.ContractAddr}, {Key: "amount", Value: "1000Asset1"},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Transfer, pair.ContractAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "-1000"}, {pair.Assets[1], ""}}, "", "", "", map[string]interface{}{"recipient": userAddr}}},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000Asset1"},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Transfer, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], ""}}, "", "", "", map[string]interface{}{"recipient": pair.ContractAddr}}},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000Asset1,2000Asset2"},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Transfer, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], "2000"}}, "", "", "", map[string]interface{}{"recipient": pair.ContractAddr}}},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "123456789012345678901234567890123456Asset2"},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Transfer, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], ""}, {pair.Assets[1], "123456789012345678901234567890123456"}}, "", "", "", map[string]interface{}{"recipient": pair.ContractAddr}}},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: "WRONG_PEER"}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000Asset1"},
			},
			nil,
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000WrongAsset1"},
			},
			nil,
			"wrong asset must return error",
		},
		// wasm transfer
		{
			&wasmTransferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.Assets[0]}, {Key: "action", Value: "transfer"}, {Key: "amount", Value: "1000"}, {Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Transfer, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], ""}}, "", "", "", map[string]interface{}{"recipient": pair.ContractAddr}}},
			"",
		},
		{
			&wasmTransferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.Assets[1]}, {Key: "action", Value: "transfer"},
				{Key: "amount", Value: "123456789012345678901234567890123456"},
				{Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Transfer, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], ""}, {pair.Assets[1], "123456789012345678901234567890123456"}}, "", "", "", map[string]interface{}{"recipient": pair.ContractAddr}}},
			"",
		},
		{
			&wasmTransferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.Assets[0]}, {Key: "action", Value: "transfer"}, {Key: "amount", Value: "1000"},
				{Key: "from", Value: userAddr}, {Key: "to", Value: "WRONG_PEER"},
			},
			nil,
			"",
		},
		{
			&wasmTransferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: "WRONG_CW_20"}, {Key: "action", Value: "transfer"}, {Key: "amount", Value: "1000"},
				{Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr},
			},
			nil,
			"wrong asset must return error",
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
	const factoryAddr = "xpla1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"

	tcs := []struct {
		mapper         parser.Mapper[dex.ParsedTx]
		matchedResults el.MatchedResult
		expectedTx     []*dex.ParsedTx
		errMsg         string
	}{
		{
			&createPairMapper{},
			el.MatchedResult{
				{Key: "_contract_address", Value: factoryAddr}, {Key: "action", Value: "create_pair"},
				{Key: "pair", Value: "xpla1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al-axpla"},
				{Key: "_contract_address", Value: "A"}, {Key: "liquidity_token_addr", Value: "xpla1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.CreatePair, "", "A", [2]dex.Asset{{"xpla1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al", ""}, {"axpla", ""}}, "xpla1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7", "", "", nil}},
			"",
		},
		{
			&createPairMapper{},
			el.MatchedResult{
				{Key: "_contract_address", Value: factoryAddr}, {Key: "action", Value: "create_pair"},
				{Key: "pair", Value: "INVALID_PAIR"},
				{Key: "_contract_address", Value: "B"}, {Key: "liquidity_token_addr", Value: "xpla1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7"},
			},
			nil,
			"expected assets length(2)",
		},
		{
			&createPairMapper{},
			el.MatchedResult{
				{Key: "_contract_address", Value: "IT REQUIRED MORE MATCHED"}, {Key: "action", Value: "create_pair"},
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

	const userAddr = "userAddr"
	pair := dex.Pair{ContractAddr: "Pair", LpAddr: "LiquidityToken", Assets: []string{"Asset1", "Asset2"}}
	pairSet := map[string]dex.Pair{pair.ContractAddr: pair}
	tcs := []struct {
		mapper         parser.Mapper[dex.ParsedTx]
		matchedResults el.MatchedResult
		expectedTx     []*dex.ParsedTx
		errMsg         string
	}{

		{
			&pairMapperImpl{&pairMapperMixin{pairSet: pairSet}},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "swap"},
				{Key: "ask_asset", Value: pair.Assets[1]}, {Key: "commission_amount", Value: "302"},
				{Key: "offer_amount", Value: "100000"}, {Key: "offer_asset", Value: pair.Assets[0]},
				{Key: "receiver", Value: userAddr}, {Key: "return_amount", Value: "100583"},
				{Key: "sender", Value: userAddr}, {Key: "spread_amount", Value: "2"},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Swap, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "100000"}, {pair.Assets[1], "-100583"}}, "", "", "302", nil}},
			"",
		},
		{

			&pairMapperImpl{&pairMapperMixin{pairSet: pairSet}},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "swap"},
				{Key: "ask_asset", Value: pair.Assets[0]}, {Key: "commission_amount", Value: "300"},
				{Key: "offer_amount", Value: "100000"}, {Key: "offer_asset", Value: pair.Assets[1]},
				{Key: "receiver", Value: userAddr}, {Key: "return_amount", Value: "100583"},
				{Key: "sender", Value: userAddr}, {Key: "spread_amount", Value: "2"},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Swap, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "-100583"}, {pair.Assets[1], "100000"}}, "", "", "300", nil}},
			"",
		},
		{
			&pairMapperImpl{&pairMapperMixin{pairSet: pairSet}},
			el.MatchedResult{
				{Key: "_contract_address", Value: "IT REQUIRED MORE MATCHED"}, {Key: "action", Value: "create_pair"},
			},
			nil,
			"expected results length(10)",
		},

		/// Provide
		{
			&pairMapperImpl{&pairMapperMixin{pairSet: pairSet}},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "sender", Value: userAddr}, {Key: "receiver", Value: userAddr},
				{Key: "assets", Value: fmt.Sprintf("%s%s, %s%s", "1000", pair.Assets[0], "10000", pair.Assets[1])},
				{Key: "share", Value: "998735"},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Provide, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], "10000"}}, pair.LpAddr, "998735", "", nil}},
			"",
		},
		{
			&pairMapperImpl{&pairV2Mapper{&pairMapperMixin{pairSet: pairSet}}},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "sender", Value: userAddr}, {Key: "receiver", Value: userAddr},
				{Key: "assets", Value: fmt.Sprintf("%s%s, %s%s", "1000", pair.Assets[0], "10000", pair.Assets[1])},
				{Key: "share", Value: "998735"}, {Key: "refund_assets", Value: fmt.Sprintf("%s%s, %s%s", "0", pair.Assets[0], "100", pair.Assets[1])},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Provide, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], "10000"}}, pair.LpAddr, "998735", "",
				map[string]interface{}{"refund_assets": []dex.Asset{{pair.Assets[0], "0"}, {pair.Assets[1], "100"}}}}},
			"",
		},
		{
			&pairMapperImpl{&pairMapperMixin{pairSet: pairSet}},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "sender", Value: userAddr}, {Key: "receiver", Value: userAddr},
				{Key: "assets", Value: fmt.Sprintf("%s,%s %s%s", "1000", pair.Assets[0], "10000", pair.Assets[1])},
				{Key: "share", Value: "998735"},
			},
			nil,
			"Wrong format of assets must return error",
		},

		/// Withdraw
		{
			&pairMapperImpl{&pairMapperMixin{pairSet: pairSet}},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr},
				{Key: "action", Value: "withdraw_liquidity"},
				{Key: "refund_assets", Value: fmt.Sprintf("%s%s, %s%s", "24999998", pair.Assets[0], "24939789", pair.Assets[1])},
				{Key: "sender", Value: userAddr},
				{Key: "withdrawn_share", Value: "12418119"},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Withdraw, userAddr, pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "-24999998"}, {pair.Assets[1], "-24939789"}}, pair.LpAddr, "12418119", "", nil}},
			"",
		},
		{
			&pairMapperImpl{&pairMapperMixin{pairSet: pairSet}},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "withdraw_liquidity"},
				{Key: "refund_assets", Value: fmt.Sprintf("%s,%s %s%s", "1000", pair.Assets[0], "10000", pair.Assets[1])},
				{Key: "sender", Value: userAddr},
				{Key: "withdrawn_share", Value: "12418119"},
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

func Test_InitialProvideMapper(t *testing.T) {

	const userAddr = "userAddr"
	pair := dex.Pair{ContractAddr: "Pair", LpAddr: "LiquidityToken", Assets: []string{"Asset1", "Asset2"}}
	tcs := []struct {
		mapper         parser.Mapper[dex.ParsedTx]
		matchedResults el.MatchedResult
		expectedTx     []*dex.ParsedTx
		errMsg         string
	}{
		/// Initial Provide
		{
			&initialProvideMapper{mapperMixin{}},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.LpAddr}, {Key: "action", Value: "mint"},
				{Key: "amount", Value: "1000"}, {Key: "to", Value: pair.ContractAddr},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.InitialProvide, "", pair.ContractAddr, [2]dex.Asset{}, pair.LpAddr, "1000", "", nil}},
			"",
		},
		{
			&initialProvideMapper{mapperMixin{}}, el.MatchedResult{
				{Key: "_contract_address", Value: pair.LpAddr}, {Key: "action", Value: "mint"},
				{Key: "amount", Value: "1000"}, {Key: "to", Value: pair.ContractAddr}, {Key: "sender", Value: pair.ContractAddr},
			},
			nil,
			"Wrong format of matched must return error",
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

func Test_sortResult(t *testing.T) {

	const userAddr = "userAddr"
	pair := dex.Pair{ContractAddr: "Pair", LpAddr: "LiquidityToken", Assets: []string{"Asset1", "Asset2"}}
	tcs := []struct {
		target   el.MatchedResult
		expected el.MatchedResult

		errMsg string
	}{
		{

			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "send"},
				{Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr},
				{Key: "amount", Value: "332157"},
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "swap"},
				{Key: "sender", Value: userAddr}, {Key: "receiver", Value: userAddr},
				{Key: "offer_asset", Value: pair.Assets[0]}, {Key: "ask_asset", Value: pair.Assets[1]},
				{Key: "offer_amount", Value: "100000"}, {Key: "return_amount", Value: "100583"},
				{Key: "spread_amount", Value: "2"}, {Key: "commission_amount", Value: "302"},
			},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "send"},
				{Key: "amount", Value: "332157"},
				{Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr},
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "swap"},
				{Key: "ask_asset", Value: pair.Assets[1]}, {Key: "commission_amount", Value: "302"},
				{Key: "offer_amount", Value: "100000"}, {Key: "offer_asset", Value: pair.Assets[0]},
				{Key: "receiver", Value: userAddr}, {Key: "return_amount", Value: "100583"},
				{Key: "sender", Value: userAddr}, {Key: "spread_amount", Value: "2"},
			},
			"",
		},
	}

	for _, tc := range tcs {
		sortResult(tc.target)
		assert.Equal(t, tc.target, tc.expected, tc.errMsg)
	}
}

func Test_PairV2ProvideMapper_applyRefund(t *testing.T) {
	tcs := []struct {
		provide  []dex.Asset
		refund   []dex.Asset
		expected []dex.Asset
		errMsg   string
	}{
		{
			[]dex.Asset{{xpla.NATIVE_GOVERNANCE_TOKEN, "1000"}, {fmt.Sprintf("%s%s", xpla.CW20_PREFIX, "Asset2"), "10000"}},
			[]dex.Asset{{xpla.NATIVE_GOVERNANCE_TOKEN, "0"}, {fmt.Sprintf("%s%s", xpla.CW20_PREFIX, "Asset2"), "100"}},
			[]dex.Asset{{xpla.NATIVE_GOVERNANCE_TOKEN, "1000"}, {fmt.Sprintf("%s%s", xpla.CW20_PREFIX, "Asset2"), "9900"}},
			"",
		},
		{
			[]dex.Asset{{xpla.NATIVE_GOVERNANCE_TOKEN, "1234567890"}, {fmt.Sprintf("%s%s", xpla.CW20_PREFIX, "Asset2"), "10000"}},
			[]dex.Asset{{xpla.NATIVE_GOVERNANCE_TOKEN, "123456"}, {fmt.Sprintf("%s%s", xpla.CW20_PREFIX, "Asset2"), "9999"}},
			[]dex.Asset{{xpla.NATIVE_GOVERNANCE_TOKEN, "1234567890"}, {fmt.Sprintf("%s%s", xpla.CW20_PREFIX, "Asset2"), "1"}},
			"",
		},
		{
			[]dex.Asset{{xpla.NATIVE_GOVERNANCE_TOKEN, "1000"}, {fmt.Sprintf("%s%s", xpla.CW20_PREFIX, "Asset2"), "10000"}},
			[]dex.Asset{{xpla.NATIVE_GOVERNANCE_TOKEN, "0"}, {fmt.Sprintf("%s%s", xpla.CW20_PREFIX, "Asset2"), "9999"}},
			[]dex.Asset{{xpla.NATIVE_GOVERNANCE_TOKEN, "1000"}, {fmt.Sprintf("%s%s", xpla.CW20_PREFIX, "Asset2"), "10"}},
			"Asset2 must be 1",
		},
		{
			[]dex.Asset{{xpla.NATIVE_GOVERNANCE_TOKEN, "1000"}, {fmt.Sprintf("%s%s", xpla.CW20_PREFIX, "Asset2"), "10000"}},
			[]dex.Asset{{xpla.NATIVE_GOVERNANCE_TOKEN, "10"}, {fmt.Sprintf("%s%s", xpla.CW20_PREFIX, "Asset2"), "9999"}},
			[]dex.Asset{{xpla.NATIVE_GOVERNANCE_TOKEN, "990"}, {fmt.Sprintf("%s%s", xpla.CW20_PREFIX, "Asset2"), "1"}},
			"Native token must not be applied",
		},
	}

	mapper := pairV2Mapper{}

	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("tc(%d) %s", idx, tc.errMsg)
		assert := assert.New(t)

		actual, err := mapper.applyRefundAsset(tc.provide, tc.refund)
		if err != nil {
			assert.NotEmpty(tc.errMsg, errMsg)
			continue
		}

		if tc.errMsg != "" {
			assert.NotEqual(tc.expected, actual, errMsg)
		} else {
			assert.Equal(tc.expected, actual, errMsg)
		}
	}

}
