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

// CreateTransferRuleFinder finds normalized native transfer events.
// Transfer attributes are normalized to amount, recipient, sender before parsing.
// Sender is optional because the bank module's MsgMultiSend emits one "transfer"
// event per output (recipient+amount only) without a per-output sender, since a
// multisend can have multiple inputs. Only amount and recipient are required, and
// the remaining attributes (e.g. sender, when present) are appended until the
// next amount.
func CreateTransferRuleFinder(pairs map[string]bool) (eventlog.LogFinder, error) {
	var recipientFilter func(v string) bool
	if pairs != nil {
		recipientFilter = func(v string) bool {
			_, ok := pairs[v]
			return ok
		}
	}

	rule := eventlog.Rule{
		Type:  eventlog.TransferType,
		Until: TransferAmountKey,
		Items: eventlog.RuleItems{
			{Key: TransferAmountKey},
			{Key: TransferRecipientKey, Filter: recipientFilter},
		},
	}
	return eventlog.NewLogFinder(rule)
}

// NormalizeTransferAttrs sorts one native "transfer" log's attributes into
// amount, recipient, sender order so CreateTransferRuleFinder can match them
// regardless of the order the chain emitted them in (see its doc comment).
// Every DEX parser's ParseTxs must call this, for each transfer-type log,
// before running its Transfer parser — there is no other enforcement point,
// since callers (including cmd/parser/diagnose) may invoke a TargetApp's
// ParseTxs directly.
func NormalizeTransferAttrs(attrs eventlog.Attributes) (eventlog.Attributes, error) {
	sorted, err := eventlog.SortAttributes(attrs, []string{
		TransferAmountKey,
		TransferRecipientKey,
		TransferSenderKey,
	})
	if err != nil {
		return nil, err
	}
	return *sorted, nil
}

var initialProvideRule = eventlog.Rule{Type: eventlog.WasmType, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: "_contract_address", Filter: nil},
	eventlog.RuleItem{Key: "action", Filter: func(v string) bool {
		return v == "mint"
	}},
	eventlog.RuleItem{Key: "amount", Filter: nil},
	eventlog.RuleItem{Key: "to", Filter: nil},
}}

func CreateBurnRuleFinder() (eventlog.LogFinder, error) {
	return eventlog.NewLogFinder(burnRule)
}

var burnRule = eventlog.Rule{Type: eventlog.WasmType, Items: eventlog.RuleItems{
	eventlog.RuleItem{Key: BurnAddrKey, Filter: nil},
	eventlog.RuleItem{Key: BurnActionKey, Filter: func(v string) bool {
		return v == "burn"
	}},
	eventlog.RuleItem{Key: BurnAmountKey, Filter: nil},
	eventlog.RuleItem{Key: BurnFromKey, Filter: nil},
}}
