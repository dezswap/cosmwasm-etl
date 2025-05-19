package eventlog

import (
	"github.com/pkg/errors"
)

// skipKeys
// - msg_index: Added since cosmos-sdk v50, skip to check multiple events
// - token_id: dorado-1 specific e.g. E04160F77490C13B3D1AF80FCC3FCFE210FC526F2720E2AE0C3A9984D3F16DCA
var skipKeys = []string{
	"msg_index",
	"token_id",
}

type LogFinder interface {
	// return empty slice if there is no match
	FindFromLogs(logs LogResults) MatchedResults
	FindFromAttrs(attrs Attributes) MatchedResults
}

type logfinderImpl struct {
	rule Rule
}

var _ LogFinder = &logfinderImpl{}

func NewLogFinder(rule Rule) (LogFinder, error) {
	for _, i := range rule.Items {
		if err := checkRuleItem(i.Key, i.Filter); err != nil {
			return nil, errors.Wrap(err, "NewLogFinder")
		}
	}
	return &logfinderImpl{rule}, nil
}

func (f *logfinderImpl) FindFromLogs(logs LogResults) MatchedResults {
	results := MatchedResults{}
	for _, log := range logs {
		if f.rule.Type == log.Type {
			matched := f.FindFromAttrs(log.Attributes)
			results = append(results, matched...)
		}
	}
	return results
}

func (f *logfinderImpl) FindFromAttrs(attrs Attributes) MatchedResults {
	results := MatchedResults{}
	ruleItemsSize := len(f.rule.Items)
	attrsSize := len(attrs)

	if ruleItemsSize > attrsSize {
		return nil
	}

	for i := 0; (i + ruleItemsSize) <= attrsSize; i++ {
		matchedResult, nextAttrIdx := f.findMatchingSubseq(i, attrs)
		if ruleItemsSize != len(matchedResult) {
			continue
		}

		matchedResult, lastAttrIdx := f.appendUntil(nextAttrIdx, matchedResult, attrs)
		results = append(results, matchedResult)

		i = lastAttrIdx
	}
	return results
}

// findMatchingSubseq iterate and compare only the size of `rule.Items`
func (f *logfinderImpl) findMatchingSubseq(attrIdx int, attrs Attributes) (MatchedResult, int) {
	ruleItemsSize := len(f.rule.Items)
	attrsSize := len(attrs)
	matchedResult := make(MatchedResult, 0)

	i := 0
	for ; i < ruleItemsSize; i++ {
		if shouldSkipKey(attrs[attrIdx+i].Key) && (attrIdx+ruleItemsSize) < attrsSize {
			attrIdx++
		}
		if !f.rule.Items[i].Match(attrs[attrIdx+i]) {
			break
		}

		matchedResult = append(matchedResult, MatchedItem{attrs[attrIdx+i].Key, attrs[attrIdx+i].Value})
	}

	return matchedResult, attrIdx + i
}

// appendUntil `matchedResult` will include attrs up to the `Until` Key
func (f *logfinderImpl) appendUntil(attrIdx int, matchedResult MatchedResult, attrs Attributes) (MatchedResult, int) {
	attrsSize := len(attrs)

	if f.rule.Until != "" {
		for ; attrIdx < attrsSize && attrs[attrIdx].Key != f.rule.Until; attrIdx++ {
			key := attrs[attrIdx].Key
			if !shouldSkipKey(key) {
				matchedResult = append(matchedResult, MatchedItem{key, attrs[attrIdx].Value})
			}
		}
	}

	return matchedResult, attrIdx - 1
}

func shouldSkipKey(key string) bool {
	for _, s := range skipKeys {
		if key == s {
			return true
		}
	}
	return false
}
