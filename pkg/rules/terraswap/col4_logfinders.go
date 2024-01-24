package terraswap

import (
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/pkg/errors"
)

func CreateCreatePairRuleFinder(factoryAddr string) (eventlog.LogFinder, error) {
	if factoryAddr == "" {
		errMsg := "no factory address"
		return nil, errors.New(errMsg)
	}

	rule := col4CreatePairRule
	rule.Items[FactoryAddrIdx].Filter = factoryAddr

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
	rule := col4PairCommonRule
	rule.Items[PairAddrIdx].Filter = filter
	return eventlog.NewLogFinder(rule)
}

// Track cw20 transfer
func CreateWasmCommonTransferRuleFinder(pairs map[string]bool) (eventlog.LogFinder, error) {
	return eventlog.NewLogFinder(col4WasmTransferCommonRule)
}

// Track transfer from user to Pair
func CreateTransferRuleFinder(pairs map[string]bool) (eventlog.LogFinder, error) {
	var filter func(v string) bool
	if pairs != nil {
		filter = func(v string) bool {
			_, ok := pairs[v]
			return ok
		}
	}
	rule := transferRule
	rule.Items[TransferRecipientIdx].Filter = filter

	return eventlog.NewLogFinder(rule)
}

var col4CreatePairRule = eventlog.Rule{Type: eventlog.FromContract, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "contract_address", Filter: nil},
	eventlog.RuleItem{Key: "action", Filter: "create_pair"},
	eventlog.RuleItem{Key: "pair", Filter: nil},
	eventlog.RuleItem{Key: "contract_address", Filter: nil},
	eventlog.RuleItem{Key: "liquidity_token_addr", Filter: nil},
}}

var col4PairCommonRule = eventlog.Rule{Type: eventlog.FromContract, Until: "contract_address", Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "contract_address", Filter: nil},
	eventlog.RuleItem{Key: "action", Filter: func(v string) bool {
		return v == string(SwapAction) || v == string(ProvideAction) || v == string(WithdrawAction)
	}},
}}

var col4WasmTransferCommonRule = eventlog.Rule{Type: eventlog.FromContract, Until: "contract_address", Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "contract_address", Filter: nil},
	eventlog.RuleItem{Key: "action", Filter: func(v string) bool {
		return v == WasmTransferAction || v == WasmTransferFromAction
	}},
}}

var transferRule = eventlog.Rule{Type: eventlog.TransferType, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "recipient", Filter: nil},
	eventlog.RuleItem{Key: "sender", Filter: nil},
	eventlog.RuleItem{Key: "amount", Filter: nil},
}}
