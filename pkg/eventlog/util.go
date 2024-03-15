package eventlog

import (
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
