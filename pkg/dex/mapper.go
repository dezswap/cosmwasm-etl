package dex

import (
	"fmt"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/pkg/errors"
	"sort"
)

const msgIndexKey = "msg_index"

type MapperMixin struct {
	PostEventAttrLen int
}

func (m *MapperMixin) CheckResult(res eventlog.MatchedResult, expectedLen int) error {
	if len(res) != expectedLen+m.PostEventAttrLen {
		msg := fmt.Sprintf("expected results length(%d)", expectedLen)
		return errors.New(msg)
	}

	for i, r := range res {
		if r.Value == "" {
			msg := fmt.Sprintf("matched result(%d) must not be empty", i)
			return errors.New(msg)
		}
	}
	return nil
}

// SortResult sorts the result by key split by "_contract_address"
// @param res will be sorted
func (*MapperMixin) SortResult(res eventlog.MatchedResult) {
	const sortSplitter = "_contract_address"
	sort := func(from, to int) {
		target := res[from:to]
		sort.Slice(target, func(i, j int) bool {
			if target[i].Key == msgIndexKey {
				return false
			}
			return target[i].Key < target[j].Key
		})
	}
	prev := 0
	for idx, v := range res {
		if v.Key == sortSplitter {
			sort(prev, idx)
			prev = idx
		}
	}
	sort(prev, len(res))
}
