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
					Attributes{{Key: "contract", Value: "a"}, {Key: "contract", Value: "b"}, {Key: "contract", Value: "c"}, {Key: "contract", Value: "d"}},
				},
				LogResult{
					WasmType,
					Attributes{{Key: "contract", Value: "a"}, {Key: "contract", Value: "b"}, {Key: "contract", Value: "c"}, {Key: "contract", Value: "d"}},
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
					Attributes{{Key: "a", Value: "Value is not important"}, {Key: "a", Value: "TEST"}, {Key: "a", Value: ""}},
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
					Attributes{{Key: "a", Value: "a"}, {Key: "b", Value: "b"}, {Key: "c", Value: "c"}},
				},
			},
			RuleItems{
				{Key: "a", Filter: nil},
				{Key: "b", Filter: "b"},
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
					Attributes{{Key: "a", Value: "a"}, {Key: "b", Value: "b"}, {Key: "c", Value: "c"}, {Key: "a", Value: "a"}, {Key: "b", Value: "b"}, {Key: "c", Value: "c"}, {Key: "a", Value: "a"}, {Key: "b", Value: "b"}, {Key: "c", Value: "c"}},
				},
			},
			RuleItems{
				{Key: "a", Filter: nil},
				{Key: "b", Filter: "b"},
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
				{Key: "contract", Value: "a"},
				{Key: "contract", Value: "b"},
				{Key: "contract", Value: "c"},
				{Key: "contract", Value: "d"},
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
				{Key: "a", Value: "Value is not important"},
				{Key: "a", Value: "TEST"},
				{Key: "a", Value: ""},
			},
			RuleItems{RuleItem{Key: "a", Filter: nil}},
			"",
			3,
			"must match if the attribute has same key",
		},
		{
			Attributes{
				{Key: "a", Value: "a"},
				{Key: "b", Value: "b"},
				{Key: "c", Value: "c"},
			},
			RuleItems{
				{Key: "a", Filter: nil},
				{Key: "b", Filter: "b"},
				{"c", func(c string) bool { return c == "c" }},
			},
			"",
			1,
			"must match all key",
		},
		{
			Attributes{
				{Key: "a", Value: "a"},
				{Key: "b", Value: "b"},
				{Key: "c", Value: "c"},
				{Key: "a", Value: "a"},
				{Key: "b", Value: "b"},
				{Key: "c", Value: "c"},
				{Key: "a", Value: "a"},
				{Key: "b", Value: "b"},
				{Key: "c", Value: "c"},
			},
			RuleItems{
				{Key: "a", Filter: nil},
				{Key: "b", Filter: "b"},
				{"c", func(c string) bool { return false }},
			},
			"",
			0,
			"c must not be matched",
		},
		{
			Attributes{
				{Key: "a", Value: "a"},
				{Key: "b", Value: "b"},
				{Key: "c", Value: "c"},
				{Key: "a", Value: "a"},
				{Key: "b", Value: "b"},
				{Key: "c", Value: "c"},
				{Key: "a", Value: "a"},
				{Key: "b", Value: "b"},
				{Key: "c", Value: "c"},
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
				{Key: "a", Value: "a"},
				{Key: "b", Value: "b"},
				{Key: "c", Value: "c"},
				{Key: "a", Value: "a"},
				{Key: "b", Value: "b"},
				{Key: "c", Value: "c"},
				{Key: "a", Value: "a"},
				{Key: "b", Value: "b"},
				{Key: "c", Value: "c"},
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
				{Key: "_contract_address", Value: "factory_address"},
				{Key: "action", Value: "create_pair"},
				{Key: "pair", Value: "asset1_address-asset2_address"},
				{Key: "msg_index", Value: "0"},
				{Key: "_contract_address", Value: "pair_address"},
				{Key: "liquidity_token_addr", Value: "lp_token_address"},
				{Key: "msg_index", Value: "0"},
				{Key: "_contract_address", Value: "factory_address"},
				{Key: "pair_contract_addr", Value: "pair_address"},
				{Key: "liquidity_token_addr", Value: "lp_token_address"},
				{Key: "msg_index", Value: "0"},
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
				{Key: "_contract_address", Value: "factory_address"},
				{Key: "action", Value: "create_pair"},
				{Key: "pair", Value: "asset1_address-asset2_address"},
				{Key: "msg_index", Value: "0"},
				{Key: "_contract_address", Value: "pair_address"},
				{Key: "liquidity_token_addr", Value: "lp_token_address"},
				{Key: "msg_index", Value: "0"},
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
				{Key: "_contract_address", Value: "pair_address"},
				{Key: "action", Value: "swap"},
				{Key: "sender", Value: "sender_address"},
				{Key: "receiver", Value: "receiver_address"},
				{Key: "offer_asset", Value: "asset1_address"},
				{Key: "ask_asset", Value: "asset2_address"},
				{Key: "offer_amount", Value: "1000000"},
				{Key: "return_amount", Value: "1000"},
				{Key: "spread_amount", Value: "1"},
				{Key: "commission_amount", Value: "3"},
				{Key: "msg_index", Value: "0"},
				{Key: "_contract_address", Value: "asset2_address"},
				{Key: "action", Value: "transfer"},
				{Key: "amount", Value: "1000"},
				{Key: "from", Value: "pair_address"},
				{Key: "to", Value: "sender_address"},
				{Key: "msg_index", Value: "0"},
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
				{Key: "_contract_address", Value: "asset2_address"},
				{Key: "action", Value: "increase_allowance"},
				{Key: "owner", Value: "provider_address"},
				{Key: "spender", Value: "pair_address"},
				{Key: "amount", Value: "1000000"},
				{Key: "msg_index", Value: "0"},
				{Key: "_contract_address", Value: "pair_address"},
				{Key: "action", Value: "provide_liquidity"},
				{Key: "sender", Value: "provider_address"},
				{Key: "receiver", Value: "provider_address"},
				{Key: "assets", Value: "1000000asset1, 1000000asset2"},
				{Key: "share", Value: "1000000"},
				{Key: "refund_assets", Value: "0asset1, 0asset2"},
				{Key: "msg_index", Value: "1"},
				{Key: "_contract_address", Value: "asset2_address"},
				{Key: "action", Value: "transfer_from"},
				{Key: "amount", Value: "1000000"},
				{Key: "by", Value: "pair_address"},
				{Key: "from", Value: "provider_address"},
				{Key: "to", Value: "pair_address"},
				{Key: "msg_index", Value: "1"},
				{Key: "_contract_address", Value: "lp_token_address"},
				{Key: "action", Value: "mint"},
				{Key: "amount", Value: "1000000"},
				{Key: "to", Value: "provider_address"},
				{Key: "msg_index", Value: "1"},
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

func TestFindFromAttr_SeparatesResultsByMsgIndex(t *testing.T) {
	attrs := Attributes{
		{Key: "_contract_address", Value: "pair_address", MsgIndex: 0},
		{Key: "action", Value: "swap", MsgIndex: 0},
		{Key: "sender", Value: "sender0", MsgIndex: 0},
		{Key: "_contract_address", Value: "pair_address", MsgIndex: 1},
		{Key: "action", Value: "swap", MsgIndex: 1},
		{Key: "sender", Value: "sender1", MsgIndex: 1},
	}
	rule := Rule{
		Type: WasmType,
		Items: RuleItems{
			{Key: "_contract_address", Filter: "pair_address"},
			{Key: "action", Filter: "swap"},
		},
		Until: "_contract_address",
	}
	finder, err := NewLogFinder(rule)
	assert.NoError(t, err)

	results := finder.FindFromAttrs(attrs)
	assert.Len(t, results, 2)
	assert.Equal(t, 0, MsgIndex(results[0]))
	assert.Equal(t, 1, MsgIndex(results[1]))
	assert.Equal(t, "sender0", results[0][2].Value)
	assert.Equal(t, "sender1", results[1][2].Value)
}

func TestFindFromAttr_DoesNotMatchRuleAcrossMsgIndex(t *testing.T) {
	attrs := Attributes{
		{Key: "_contract_address", Value: "pair_address", MsgIndex: 0},
		{Key: "action", Value: "swap", MsgIndex: 1},
	}
	rule := Rule{
		Type: WasmType,
		Items: RuleItems{
			{Key: "_contract_address", Filter: "pair_address"},
			{Key: "action", Filter: "swap"},
		},
	}
	finder, err := NewLogFinder(rule)
	assert.NoError(t, err)

	results := finder.FindFromAttrs(attrs)
	assert.Empty(t, results)
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
				{Key: "_contract_address", Value: "token_address"},
				{Key: "action", Value: "transfer"},
				{Key: "token_id", Value: "1357"},
				{Key: "amount", Value: "1"},
				{Key: "from", Value: "from_address"},
				{Key: "to", Value: "to_address"},
				{Key: "_contract_address", Value: "token_address"},
				{Key: "action", Value: "transfer"},
				{Key: "token_id", Value: "1357"},
				{Key: "amount", Value: "1"},
				{Key: "from", Value: "from_address"},
				{Key: "to", Value: "to_address"},
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
