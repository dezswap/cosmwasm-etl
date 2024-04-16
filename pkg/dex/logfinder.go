package dex

import "github.com/dezswap/cosmwasm-etl/pkg/eventlog"

func CreatePairInitialProvideRuleFinder(pairs map[string]bool) (eventlog.LogFinder, error) {
	var filter func(v string) bool
	if pairs != nil {
		filter = func(v string) bool {
			_, ok := pairs[v]
			return ok
		}
	}
	rule := initialProvideRule
	// filter by to address because the initial provide liquidity is minted to the pair
	rule.Items[PairInitialProvideToIdx].Filter = filter
	return eventlog.NewLogFinder(rule)
}

var initialProvideRule = eventlog.Rule{Type: eventlog.WasmType, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "_contract_address", Filter: nil},
	eventlog.RuleItem{Key: "action", Filter: func(v string) bool {
		return v == "mint"
	}},
	eventlog.RuleItem{Key: "amount", Filter: nil},
	eventlog.RuleItem{Key: "to", Filter: nil},
}}
