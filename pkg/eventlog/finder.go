package eventlog

import "github.com/pkg/errors"

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
	ruleLen := len(f.rule.Items)
	attrLen := len(attrs)

	if ruleLen > attrLen {
		return nil
	}

	for aIdx := 0; (aIdx + ruleLen) <= attrLen; aIdx++ {
		idx := 0
		for ; idx < ruleLen; idx++ {
			if !f.rule.Items[idx].Match(attrs[aIdx+idx]) {
				break
			}
		}

		if idx != ruleLen {
			continue
		}

		matchedResult := make(MatchedResult, 0)

		if f.rule.Until != "" {
			for ; (idx+aIdx) < attrLen && attrs[idx+aIdx].Key != f.rule.Until; idx++ {
			}
		}

		for i := 0; i < idx; i++ {
			matchedResult = append(matchedResult, MatchedItem{attrs[aIdx+i].Key, attrs[aIdx+i].Value})
		}

		results = append(results, matchedResult)
		aIdx += (idx - 1)
	}
	return results
}
