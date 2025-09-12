package eventlog

import (
	"fmt"
	"sort"

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
