package eventlog

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"
)

func SortAttributes(attrs Attributes, filter map[string]bool) (*Attributes, error) {
	if len(filter) == 0 {
		return nil, errors.New("filter must be provided")
	}

	filtered := make(Attributes, 0, len(attrs))
	for _, attr := range attrs {
		if _, ok := filter[attr.Key]; ok {
			filtered = append(filtered, attr)
		}
	}

	idx := 0
	end := len(filter)
	for end <= len(filtered) {
		sort.Slice(filtered[idx:end], func(i, j int) bool {
			return filtered[i].Key < attrs[j].Key
		})
		idx = end
		end = idx + len(filter)
	}

	return &filtered, nil
}

func ResultToItemMap(res MatchedResult) (map[string]MatchedItem, error) {
	m := make(map[string]MatchedItem)
	for _, r := range res {
		if _, ok := m[r.Key]; ok {
			return nil, errors.New(fmt.Sprintf("duplicated key(%s)", r.Key))
		}
		m[r.Key] = r
	}
	return m, nil
}
