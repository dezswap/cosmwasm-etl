package eventlog

import (
	"reflect"

	"github.com/pkg/errors"
)

type Rule struct {
	Type  LogType
	Items []RuleItem
	Until string
}

func NewRule(logType LogType, items RuleItems, until string) (Rule, error) {
	for _, item := range items {
		if err := checkRuleItem(item.Key, item.Filter); err != nil {
			return Rule{}, errors.Wrap(err, "NewRule")
		}
	}
	return Rule{logType, items, until}, nil
}

// Target receives string, func(v string) bool or nil
type RuleItem struct {
	Key    string
	Filter interface{}
}
type RuleItems []RuleItem

func checkRuleItem(key string, target interface{}) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}
	_, isString := target.(string)
	_, isMatchable := target.(func(v string) bool)
	if target != nil && !isString && !isMatchable {
		return errors.New("target must be the one of string, func(v string)bool or nil")
	}
	return nil
}

func (r *RuleItem) Match(a Attribute) bool {
	if r.Key != a.Key {
		return false
	}

	if target, ok := r.Filter.(string); ok {
		return target == a.Value
	}

	f, ok := r.Filter.(func(v string) bool)
	if ok && f != nil {
		return f(a.Value)
	}

	// IsNil occur panic
	defer func() bool {
		r := recover()
		return r != nil
	}()

	return f == nil || r.Filter == nil || reflect.ValueOf(r.Filter).IsNil()
}
