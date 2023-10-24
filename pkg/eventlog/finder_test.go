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
