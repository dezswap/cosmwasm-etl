package dezswap

import (
	"fmt"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/pkg/errors"
)

func CreateCreatePairRuleFinder(chainId string) (eventlog.LogFinder, error) {
	factoryAddr := getFactoryAddress(chainId)
	if factoryAddr == "" {
		errMsg := fmt.Sprintf("no factory address: chainId(%s)", chainId)
		return nil, errors.New(errMsg)
	}

	rule := createPairRule
	rule.Items[FactoryAddrIdx].Filter = factoryAddr

	return eventlog.NewLogFinder(rule)
}

func CreatePairAllRulesFinder(pairs map[string]bool) (eventlog.LogFinder, error) {
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

func CreatePairSwapRuleFinder(pairs map[string]bool) (eventlog.LogFinder, error) {
	var filter func(v string) bool
	if pairs != nil {
		filter = func(v string) bool {
			_, ok := pairs[v]
			return ok
		}
	}
	rule := swapRule
	rule.Items[PairAddrIdx].Filter = filter
	return eventlog.NewLogFinder(rule)
}

func CreatePairProviderRuleFinder(pairs map[string]bool) (eventlog.LogFinder, error) {
	var filter func(v string) bool
	if pairs != nil {
		filter = func(v string) bool {
			_, ok := pairs[v]
			return ok
		}
	}
	rule := provideRule
	rule.Items[PairAddrIdx].Filter = filter
	return eventlog.NewLogFinder(rule)
}

func CreatePairWithdrawRuleFinder(pairs map[string]bool) (eventlog.LogFinder, error) {
	var filter func(v string) bool
	if pairs != nil {
		filter = func(v string) bool {
			_, ok := pairs[v]
			return ok
		}
	}
	rule := withdrawRule
	rule.Items[PairAddrIdx].Filter = filter
	return eventlog.NewLogFinder(rule)
}

// Track cw20 transfer
func CreateWasmCommonTransferRuleFinder() (eventlog.LogFinder, error) {
	return eventlog.NewLogFinder(wasmTransferRule)
}

// Track transfer from user to Pair
func CreateTransferRuleFinder() (eventlog.LogFinder, error) {
	return eventlog.NewLogFinder(transferRule)
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

var swapRule = eventlog.Rule{Type: eventlog.WasmType, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "_contract_address", Filter: nil},
	eventlog.RuleItem{Key: "action", Filter: func(v string) bool {
		return v == string(SwapAction)
	}},
	eventlog.RuleItem{Key: "ask_asset", Filter: nil},
	eventlog.RuleItem{Key: "commission_amount", Filter: nil},
	eventlog.RuleItem{Key: "offer_amount", Filter: nil},
	eventlog.RuleItem{Key: "offer_asset", Filter: nil},
	eventlog.RuleItem{Key: "receiver", Filter: nil},
	eventlog.RuleItem{Key: "return_amount", Filter: nil},
	eventlog.RuleItem{Key: "sender", Filter: nil},
	eventlog.RuleItem{Key: "spread_amount", Filter: nil},
}}

var provideRule = eventlog.Rule{Type: eventlog.WasmType, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "_contract_address", Filter: nil},
	eventlog.RuleItem{Key: "action", Filter: func(v string) bool {
		return v == string(ProvideAction)
	}},
	eventlog.RuleItem{Key: "sender", Filter: nil},
	eventlog.RuleItem{Key: "receiver", Filter: nil},
	eventlog.RuleItem{Key: "assets", Filter: nil}, //0000AAAA, 0000AAAA
	eventlog.RuleItem{Key: "share", Filter: nil},
}}

var withdrawRule = eventlog.Rule{Type: eventlog.WasmType, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "_contract_address", Filter: nil},
	eventlog.RuleItem{Key: "action", Filter: func(v string) bool {
		return v == string(WithdrawAction)
	}},
	eventlog.RuleItem{Key: "refund_assets", Filter: nil}, //0000AAAA, 0000AAAA
	eventlog.RuleItem{Key: "sender", Filter: nil},
	eventlog.RuleItem{Key: "withdrawn_share", Filter: nil},
}}

var wasmTransferRule = eventlog.Rule{Type: eventlog.WasmType, Until: "_contract_address", Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "_contract_address", Filter: nil},
	eventlog.RuleItem{Key: "action", Filter: func(v string) bool {
		return v == string(WasmTransferAction) || v == string(WasmTransferFromAction)
	}},
}}

var transferRule = eventlog.Rule{Type: eventlog.TransferType, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "recipient", Filter: nil},
	eventlog.RuleItem{Key: "sender", Filter: nil},
	eventlog.RuleItem{Key: "amount", Filter: nil},
}}
