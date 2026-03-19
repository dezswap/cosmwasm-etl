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
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Transfer, Sender: pair.ContractAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "-1000"}, {Addr: pair.Assets[1], Amount: ""}}, LpAddr: "", LpAmount: "", CommissionAmount: "", Meta: map[string]interface{}{"recipient": userAddr}}},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000Asset1"},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Transfer, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "1000"}, {Addr: pair.Assets[1], Amount: ""}}, LpAddr: "", LpAmount: "", CommissionAmount: "", Meta: map[string]interface{}{"recipient": pair.ContractAddr}}},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000Asset1,2000Asset2"},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Transfer, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "1000"}, {Addr: pair.Assets[1], Amount: "2000"}}, LpAddr: "", LpAmount: "", CommissionAmount: "", Meta: map[string]interface{}{"recipient": pair.ContractAddr}}},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "123456789012345678901234567890123456Asset2"},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Transfer, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: ""}, {Addr: pair.Assets[1], Amount: "123456789012345678901234567890123456"}}, LpAddr: "", LpAmount: "", CommissionAmount: "", Meta: map[string]interface{}{"recipient": pair.ContractAddr}}},
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
		// empty amount should return error
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: ""},
			},
			nil,
			"empty amount",
		},
		// wasm transfer
		{
			&wasmTransferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.Assets[0]}, {Key: "action", Value: "transfer"}, {Key: "amount", Value: "1000"}, {Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Transfer, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "1000"}, {Addr: pair.Assets[1], Amount: ""}}, LpAddr: "", LpAmount: "", CommissionAmount: "", Meta: map[string]interface{}{"recipient": pair.ContractAddr}}},
			"",
		},
		{
			&wasmTransferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.Assets[1]}, {Key: "action", Value: "transfer"},
				{Key: "amount", Value: "123456789012345678901234567890123456"},
				{Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Transfer, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: ""}, {Addr: pair.Assets[1], Amount: "123456789012345678901234567890123456"}}, LpAddr: "", LpAmount: "", CommissionAmount: "", Meta: map[string]interface{}{"recipient": pair.ContractAddr}}},
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
		// CW1155 transfer (has token_id, no from) should be filtered out
		{
			&wasmTransferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.Assets[0]}, {Key: "action", Value: "transfer"},
				{Key: "token_id", Value: "NL2513010"},
				{Key: "amount", Value: "1"}, {Key: "to", Value: pair.ContractAddr},
			},
			nil,
			"",
		},
		// CW1155 transfer with different attribute order should be filtered out
		{
			&wasmTransferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.Assets[0]}, {Key: "action", Value: "transfer"},
				{Key: "amount", Value: "1"}, {Key: "to", Value: pair.ContractAddr},
			},
			nil,
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
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.CreatePair, Sender: "", ContractAddr: "A", Assets: [2]dex.Asset{{Addr: "xpla1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al", Amount: ""}, {Addr: "axpla", Amount: ""}}, LpAddr: "xpla1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7", LpAmount: "", CommissionAmount: "", Meta: nil}},
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
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Swap, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "100000"}, {Addr: pair.Assets[1], Amount: "-100583"}}, LpAddr: "", LpAmount: "", CommissionAmount: "302", Meta: nil}},
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
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Swap, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "-100583"}, {Addr: pair.Assets[1], Amount: "100000"}}, LpAddr: "", LpAmount: "", CommissionAmount: "300", Meta: nil}},
			"",
		},
		{
			&pairMapperImpl{&pairMapperMixin{pairSet: pairSet}},
			el.MatchedResult{
				{Key: "_contract_address", Value: "IT REQUIRED MORE MATCHED"}, {Key: "action", Value: "create_pair"},
			},
			nil,
			"expected results length at least (10)",
		},

		// V3 swap: lp_commission_amount, protocol_fee_amount, protocol_fee_collector added
		{
			&pairMapperImpl{&pairMapperMixin{pairSet: pairSet}},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr},
				{Key: "action", Value: "swap"},
				{Key: "ask_asset", Value: pair.Assets[1]},
				{Key: "commission_amount", Value: "302"},
				{Key: "lp_commission_amount", Value: "151"},
				{Key: "offer_amount", Value: "100000"},
				{Key: "offer_asset", Value: pair.Assets[0]},
				{Key: "protocol_fee_amount", Value: "151"},
				{Key: "protocol_fee_collector", Value: "FeeCollector"},
				{Key: "receiver", Value: userAddr},
				{Key: "return_amount", Value: "100583"},
				{Key: "sender", Value: userAddr},
				{Key: "spread_amount", Value: "2"},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Swap, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "100000"}, {Addr: pair.Assets[1], Amount: "-100583"}}, LpAddr: "", LpAmount: "", CommissionAmount: "302", Meta: nil}},
			"",
		},
		{
			&pairMapperImpl{&pairMapperMixin{pairSet: pairSet}},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr},
				{Key: "action", Value: "swap"},
				{Key: "ask_asset", Value: pair.Assets[0]},
				{Key: "commission_amount", Value: "300"},
				{Key: "lp_commission_amount", Value: "150"},
				{Key: "offer_amount", Value: "100000"},
				{Key: "offer_asset", Value: pair.Assets[1]},
				{Key: "protocol_fee_amount", Value: "150"},
				{Key: "protocol_fee_collector", Value: "FeeCollector"},
				{Key: "receiver", Value: userAddr},
				{Key: "return_amount", Value: "100583"},
				{Key: "sender", Value: userAddr},
				{Key: "spread_amount", Value: "2"},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Swap, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "-100583"}, {Addr: pair.Assets[1], Amount: "100000"}}, LpAddr: "", LpAmount: "", CommissionAmount: "300", Meta: nil}},
			"",
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
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Provide, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "1000"}, {Addr: pair.Assets[1], Amount: "10000"}}, LpAddr: pair.LpAddr, LpAmount: "998735", CommissionAmount: "", Meta: nil}},
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
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Provide, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "1000"}, {Addr: pair.Assets[1], Amount: "10000"}}, LpAddr: pair.LpAddr, LpAmount: "998735", CommissionAmount: "",
				Meta: map[string]interface{}{"refund_assets": []dex.Asset{{Addr: pair.Assets[0], Amount: "0"}, {Addr: pair.Assets[1], Amount: "100"}}}}},
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
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Withdraw, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "-24999998"}, {Addr: pair.Assets[1], Amount: "-24939789"}}, LpAddr: pair.LpAddr, LpAmount: "12418119", CommissionAmount: "", Meta: nil}},
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
