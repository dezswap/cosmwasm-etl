package eventlog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

func SortAttributes(attrs Attributes, filter []string) (*Attributes, error) {
	if len(filter) == 0 {
		return nil, errors.New("filter must be provided")
	}

	order := make(map[string]int, len(filter))
	for i, key := range filter {
		order[key] = i
	}

	filtered := make(Attributes, 0, len(attrs))
	for _, attr := range attrs {
		if _, ok := order[attr.Key]; ok {
			filtered = append(filtered, attr)
		}
	}

	idx := 0
	end := len(filter)
	for end <= len(filtered) {
		sort.Slice(filtered[idx:end], func(i, j int) bool {
			return order[filtered[idx:end][i].Key] < order[filtered[idx:end][j].Key]
		})
		idx = end
		end = idx + len(filter)
	}

	return &filtered, nil
}

type AmbiguousEventError struct {
	Contract string
	Action   string
	Key      string
	Values   []string
	MsgIndex int
}

func (e *AmbiguousEventError) Error() string {
	return fmt.Sprintf(
		"ambiguous event key(%s) contract(%s) action(%s) values(%s) msg_index(%d)",
		e.Key, e.Contract, e.Action, strings.Join(e.Values, ","), e.MsgIndex,
	)
}

// ResultToItemMapForKeys returns a single-value map for the event attributes a mapper actually reads.
//
// consumedKeys must contain only keys whose values are consumed by the caller. Duplicate attributes
// outside this list are ignored because Cosmos events permit repeated keys. Duplicate values for a
// consumed key are returned as AmbiguousEventError; selecting first or last is contract-specific and
// must be implemented in the protocol mapper, not here.
func ResultToItemMapForKeys(res MatchedResult, consumedKeys ...string) (map[string]MatchedItem, error) {
	targets := make(map[string]struct{}, len(consumedKeys))
	for _, key := range consumedKeys {
		targets[key] = struct{}{}
	}

	items := make(map[string][]MatchedItem, len(consumedKeys))
	for _, item := range res {
		if _, ok := targets[item.Key]; !ok {
			continue
		}
		items[item.Key] = append(items[item.Key], item)
	}

	result := make(map[string]MatchedItem, len(consumedKeys))
	for _, key := range consumedKeys {
		matches := items[key]
		if len(matches) > 1 {
			values := make([]string, 0, len(matches))
			for _, item := range matches {
				values = append(values, item.Value)
			}
			return nil, &AmbiguousEventError{
				Contract: firstResultValue(res, "_contract_address", "contract_address"),
				Action:   firstResultValue(res, "action"),
				Key:      key,
				Values:   values,
				MsgIndex: matches[0].MsgIndex,
			}
		}
		if len(matches) == 1 {
			result[key] = matches[0]
		}
	}
	return result, nil
}

func firstResultValue(res MatchedResult, keys ...string) string {
	for _, item := range res {
		for _, key := range keys {
			if item.Key == key {
				return item.Value
			}
		}
	}
	return ""
}
