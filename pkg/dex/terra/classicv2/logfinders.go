package classicv2

import (
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
)

// CreateTaxPaymentRuleFinder finds tax_payment type logs
func CreateTaxPaymentRuleFinder() (eventlog.LogFinder, error) {
	return eventlog.NewLogFinder(taxPaymentRule)
}

var taxPaymentRule = eventlog.Rule{Type: eventlog.TaxPaymentType, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "reverse_charge", Filter: nil},
	eventlog.RuleItem{Key: "tax_amount", Filter: nil},
}}

// CreateBurnRuleFinder keeps its own rule because terra emits from before amount, unlike
// the key order dex.CreateBurnRuleFinder expects.
func CreateBurnRuleFinder() (eventlog.LogFinder, error) {
	return eventlog.NewLogFinder(burnRule)
}

var burnRule = eventlog.Rule{Type: eventlog.WasmType, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: dex.BurnAddrKey, Filter: nil},
	eventlog.RuleItem{Key: dex.BurnActionKey, Filter: func(v string) bool {
		return v == "burn"
	}},
	eventlog.RuleItem{Key: dex.BurnFromKey, Filter: nil},
	eventlog.RuleItem{Key: dex.BurnAmountKey, Filter: nil},
}}
