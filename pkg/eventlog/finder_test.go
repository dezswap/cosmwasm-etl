package eventlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindFromLogs(t *testing.T) {
	testCases := []struct {
		logs        LogResults
		rule        RuleItems
		ruleUntil   string
		expectedLen int
		errMsg      string
	}{
		{
			LogResults{
				LogResult{
					WasmType,
					Attributes{{"contract", "a"}, {"contract", "b"}, {"contract", "c"}, {"contract", "d"}},
				},
				LogResult{
					WasmType,
					Attributes{{"contract", "a"}, {"contract", "b"}, {"contract", "c"}, {"contract", "d"}},
				},
			},
			RuleItems{RuleItem{"contract", func(v string) bool {
				filterMap := map[string]bool{"a": true, "b": true, "c": true}
				_, ok := filterMap[v]
				return ok
			}}},
			"",
			6,
			"must match 6 times",
		},
		{
			LogResults{
				LogResult{
					WasmType,
					Attributes{{"a", "Value is not important"}, {"a", "TEST"}, {"a", ""}},
				},
			},
			RuleItems{RuleItem{Key: "a", Filter: nil}},
			"",
			3,
			"must match if the attribute has same key",
		},
		{
			LogResults{
				LogResult{
					WasmType,
					Attributes{{"a", "a"}, {"b", "b"}, {"c", "c"}},
				},
			},
			RuleItems{
				{Key: "a", Filter: nil},
				{"b", "b"},
				{"c", func(c string) bool { return c == "c" }},
			},
			"",
			1,
			"must match all key",
		},
		{
			LogResults{
				LogResult{
					WasmType,
					Attributes{{"a", "a"}, {"b", "b"}, {"c", "c"}, {"a", "a"}, {"b", "b"}, {"c", "c"}, {"a", "a"}, {"b", "b"}, {"c", "c"}},
				},
			},
			RuleItems{
				{Key: "a", Filter: nil},
				{"b", "b"},
				{"c", func(c string) bool { return false }},
			},
			"",
			0,
			"c must not be matched",
		},
	}

	for _, tc := range testCases {
		rule, _ := NewRule(WasmType, tc.rule, tc.ruleUntil)
		finder, _ := NewLogFinder(rule)
		results := finder.FindFromLogs(tc.logs)
		assert.NotNil(t, results)
		assert.Len(t, results, tc.expectedLen, tc.errMsg)
	}

}

func TestFindFromAttr(t *testing.T) {
	testCases := []struct {
		attrs       Attributes
		rule        RuleItems
		ruleUntil   string
		expectedLen int
		errMsg      string
	}{
		{
			Attributes{
				{"contract", "a"},
				{"contract", "b"},
				{"contract", "c"},
				{"contract", "d"},
			},
			RuleItems{RuleItem{"contract", func(v string) bool {
				filterMap := map[string]bool{"a": true, "b": true, "c": true}
				_, ok := filterMap[v]
				return ok
			}}},
			"",
			3,
			"must match 3 times",
		},
		{
			Attributes{
				{"a", "Value is not important"},
				{"a", "TEST"},
				{"a", ""},
			},
			RuleItems{RuleItem{Key: "a", Filter: nil}},
			"",
			3,
			"must match if the attribute has same key",
		},
		{
			Attributes{
				{"a", "a"},
				{"b", "b"},
				{"c", "c"},
			},
			RuleItems{
				{Key: "a", Filter: nil},
				{"b", "b"},
				{"c", func(c string) bool { return c == "c" }},
			},
			"",
			1,
			"must match all key",
		},
		{
			Attributes{
				{"a", "a"},
				{"b", "b"},
				{"c", "c"},
				{"a", "a"},
				{"b", "b"},
				{"c", "c"},
				{"a", "a"},
				{"b", "b"},
				{"c", "c"},
			},
			RuleItems{
				{Key: "a", Filter: nil},
				{"b", "b"},
				{"c", func(c string) bool { return false }},
			},
			"",
			0,
			"c must not be matched",
		},
		{
			Attributes{
				{"a", "a"},
				{"b", "b"},
				{"c", "c"},
				{"a", "a"},
				{"b", "b"},
				{"c", "c"},
				{"a", "a"},
				{"b", "b"},
				{"c", "c"},
			},
			RuleItems{
				{Key: "a", Filter: nil},
			},
			"a",
			3,
			"must return 3 matched when until provided",
		},
		{
			Attributes{
				{"a", "a"},
				{"b", "b"},
				{"c", "c"},
				{"a", "a"},
				{"b", "b"},
				{"c", "c"},
				{"a", "a"},
				{"b", "b"},
				{"c", "c"},
			},
			RuleItems{
				{Key: "a", Filter: nil},
			},
			"d",
			1,
			"must return 1 matched since logs doesn't have the until value ",
		},
	}

	for _, tc := range testCases {
		rule, _ := NewRule("test", tc.rule, tc.ruleUntil)
		finder, _ := NewLogFinder(rule)
		results := finder.FindFromAttrs(tc.attrs)
		assert.NotNil(t, results)
		assert.Len(t, results, tc.expectedLen, tc.errMsg)
	}
}

func TestFindFromAttr_WithMsgIndex(t *testing.T) {
	testCases := []struct {
		attrs              Attributes
		rule               RuleItems
		ruleUntil          string
		expectedResultLen  int
		expectedMatchedLen int
	}{
		// create_pair
		{
			Attributes{
				{"_contract_address", "factory_address"},
				{"action", "create_pair"},
				{"pair", "asset1_address-asset2_address"},
				{"msg_index", "0"},
				{"_contract_address", "pair_address"},
				{"liquidity_token_addr", "lp_token_address"},
				{"msg_index", "0"},
				{"_contract_address", "factory_address"},
				{"pair_contract_addr", "pair_address"},
				{"liquidity_token_addr", "lp_token_address"},
				{"msg_index", "0"},
			},
			RuleItems{
				RuleItem{Key: "_contract_address", Filter: nil},
				RuleItem{Key: "action", Filter: "create_pair"},
				RuleItem{Key: "pair", Filter: nil},
				RuleItem{Key: "_contract_address", Filter: nil},
				RuleItem{Key: "liquidity_token_addr", Filter: nil},
			},
			"",
			1,
			5,
		},
		{
			Attributes{
				{"_contract_address", "factory_address"},
				{"action", "create_pair"},
				{"pair", "asset1_address-asset2_address"},
				{"msg_index", "0"},
				{"_contract_address", "pair_address"},
				{"liquidity_token_addr", "lp_token_address"},
				{"msg_index", "0"},
			},
			RuleItems{
				RuleItem{Key: "_contract_address", Filter: nil},
				RuleItem{Key: "action", Filter: "create_pair"},
				RuleItem{Key: "pair", Filter: nil},
				RuleItem{Key: "_contract_address", Filter: nil},
				RuleItem{Key: "liquidity_token_addr", Filter: nil},
			},
			"",
			1,
			5,
		},
		// swap
		{
			Attributes{
				{"_contract_address", "pair_address"},
				{"action", "swap"},
				{"sender", "sender_address"},
				{"receiver", "receiver_address"},
				{"offer_asset", "asset1_address"},
				{"ask_asset", "asset2_address"},
				{"offer_amount", "1000000"},
				{"return_amount", "1000"},
				{"spread_amount", "1"},
				{"commission_amount", "3"},
				{"msg_index", "0"},
				{"_contract_address", "asset2_address"},
				{"action", "transfer"},
				{"amount", "1000"},
				{"from", "pair_address"},
				{"to", "sender_address"},
				{"msg_index", "0"},
			},
			RuleItems{
				RuleItem{Key: "_contract_address", Filter: nil},
				RuleItem{Key: "action", Filter: func(v string) bool { return v == "swap" }},
			},
			"_contract_address",
			1,
			10,
		},
		// provide
		{
			Attributes{
				{"_contract_address", "asset2_address"},
				{"action", "increase_allowance"},
				{"owner", "provider_address"},
				{"spender", "pair_address"},
				{"amount", "1000000"},
				{"msg_index", "0"},
				{"_contract_address", "pair_address"},
				{"action", "provide_liquidity"},
				{"sender", "provider_address"},
				{"receiver", "provider_address"},
				{"assets", "1000000asset1, 1000000asset2"},
				{"share", "1000000"},
				{"refund_assets", "0asset1, 0asset2"},
				{"msg_index", "1"},
				{"_contract_address", "asset2_address"},
				{"action", "transfer_from"},
				{"amount", "1000000"},
				{"by", "pair_address"},
				{"from", "provider_address"},
				{"to", "pair_address"},
				{"msg_index", "1"},
				{"_contract_address", "lp_token_address"},
				{"action", "mint"},
				{"amount", "1000000"},
				{"to", "provider_address"},
				{"msg_index", "1"},
			},
			RuleItems{
				RuleItem{Key: "_contract_address", Filter: nil},
				RuleItem{Key: "action", Filter: func(v string) bool { return v == "provide_liquidity" }},
			},
			"_contract_address",
			1,
			7,
		},
	}

	for _, tc := range testCases {
		rule, _ := NewRule("_", tc.rule, tc.ruleUntil)
		finder, _ := NewLogFinder(rule)
		results := finder.FindFromAttrs(tc.attrs)
		assert.NotNil(t, results)
		assert.Len(t, results, tc.expectedResultLen)
		for _, r := range results {
			assert.Len(t, r, tc.expectedMatchedLen)
			for _, i := range r {
				assert.NotEqual(t, "msg_index", i.Key)
			}
		}
	}
}

func TestFindFromAttr_WithTokenId(t *testing.T) {
	testCases := []struct {
		attrs              Attributes
		rule               RuleItems
		ruleUntil          string
		expectedResultLen  int
		expectedMatchedLen int
	}{
		// transfer
		{
			Attributes{
				{"_contract_address", "token_address"},
				{"action", "transfer"},
				{"token_id", "1357"},
				{"amount", "1"},
				{"from", "from_address"},
				{"to", "to_address"},
				{"_contract_address", "token_address"},
				{"action", "transfer"},
				{"token_id", "1357"},
				{"amount", "1"},
				{"from", "from_address"},
				{"to", "to_address"},
			},
			RuleItems{
				RuleItem{Key: "_contract_address", Filter: nil},
				RuleItem{Key: "action", Filter: "transfer"},
			},
			"_contract_address",
			2,
			5,
		},
	}

	for _, tc := range testCases {
		rule, _ := NewRule("_", tc.rule, tc.ruleUntil)
		finder, _ := NewLogFinder(rule)
		results := finder.FindFromAttrs(tc.attrs)
		assert.NotNil(t, results)
		assert.Len(t, results, tc.expectedResultLen)
		for _, r := range results {
			assert.Len(t, r, tc.expectedMatchedLen)
			for _, i := range r {
				assert.NotEqual(t, "token_id", i.Key)
			}
		}
	}
}
