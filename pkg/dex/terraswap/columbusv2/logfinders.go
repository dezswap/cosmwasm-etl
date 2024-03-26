package columbusv2

import (
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/pkg/errors"
)

func CreateCreatePairRuleFinder(factoryAddr string) (eventlog.LogFinder, error) {
	if factoryAddr == "" {
		errMsg := "no factory address"
		return nil, errors.New(errMsg)
	}

	rule := createPairRule
	rule.Items[dex.FactoryAddrIdx].Filter = factoryAddr

	return eventlog.NewLogFinder(rule)
}

func CreatePairCommonRulesFinder(pairs map[string]bool) (eventlog.LogFinder, error) {
	var filter func(v string) bool
	if pairs != nil {
		filter = func(v string) bool {
			_, ok := pairs[v]
			return ok
		}
	}
	rule := pairCommonRule
	rule.Items[PairAddrIdx].Filter = filter
	return eventlog.NewLogFinder(rule)
}

// Track cw20 transfer
func CreateWasmCommonTransferRuleFinder(pairs map[string]bool) (eventlog.LogFinder, error) {
	return eventlog.NewLogFinder(wasmTransferCommonRule)
}

// Track transfer from user to Pair
func CreateSortedTransferRuleFinder(pairs map[string]bool) (eventlog.LogFinder, error) {
	var filter func(v string) bool
	if pairs != nil {
		filter = func(v string) bool {
			_, ok := pairs[v]
			return ok
		}
	}
	rule := sortedTransferRule
	rule.Items[SortedTransferRecipientIdx].Filter = filter

	return eventlog.NewLogFinder(rule)
}

var createPairRule = eventlog.Rule{Type: eventlog.WasmType, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "_contract_address", Filter: nil},
	eventlog.RuleItem{Key: "action", Filter: "create_pair"},
	eventlog.RuleItem{Key: "pair", Filter: nil},
	eventlog.RuleItem{Key: "_contract_address", Filter: nil},
	eventlog.RuleItem{Key: "liquidity_token_addr", Filter: nil},
}}

var pairCommonRule = eventlog.Rule{Type: eventlog.WasmType, Until: "_contract_address", Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "_contract_address", Filter: nil},
	eventlog.RuleItem{Key: "action", Filter: func(v string) bool {
		return v == string(SwapAction) || v == string(ProvideAction) || v == string(WithdrawAction)
	}},
}}

var wasmTransferCommonRule = eventlog.Rule{Type: eventlog.WasmType, Until: "_contract_address", Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "_contract_address", Filter: nil},
	eventlog.RuleItem{Key: "action", Filter: func(v string) bool {
		return v == dex.WasmTransferAction || v == dex.WasmTransferFromAction
	}},
}}

var sortedTransferRule = eventlog.Rule{Type: eventlog.TransferType, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "amount", Filter: nil},
	eventlog.RuleItem{Key: "recipient", Filter: nil},
	eventlog.RuleItem{Key: "sender", Filter: nil},
}}
