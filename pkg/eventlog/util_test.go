package eventlog

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSortAttributes(t *testing.T) {
	tests := []struct {
		name     string
		attrs    Attributes
		filter   map[string]bool
		expected *Attributes
	}{
		{
			name: "default tc",
			attrs: Attributes{
				{Key: "recipient", Value: "terra1abcdr"},
				{Key: "sender", Value: "terra1abcds"},
				{Key: "amount", Value: "100uluna"},
			},
			filter: map[string]bool{
				"amount":    true,
				"recipient": true,
				"sender":    true,
			},
			expected: &Attributes{
				{Key: "amount", Value: "100uluna"},
				{Key: "recipient", Value: "terra1abcdr"},
				{Key: "sender", Value: "terra1abcds"},
			},
		},
		{
			name: "tc with keys to be excluded",
			attrs: Attributes{
				{Key: "recipient", Value: "terra1abcdr"},
				{Key: "msg_index", Value: "1"},
				{Key: "sender", Value: "terra1abcds"},
				{Key: "token_id", Value: "ABCD"},
				{Key: "amount", Value: "100uluna"},
				{Key: "assets", Value: "0"},
			},
			filter: map[string]bool{
				"amount":    true,
				"recipient": true,
				"sender":    true,
			},
			expected: &Attributes{
				{Key: "amount", Value: "100uluna"},
				{Key: "recipient", Value: "terra1abcdr"},
				{Key: "sender", Value: "terra1abcds"},
			},
		},
		{
			name: "tc with multiple key sets",
			attrs: Attributes{
				{Key: "recipient", Value: "terra1abcdr"},
				{Key: "sender", Value: "terra1abcds"},
				{Key: "amount", Value: "100uluna"},
				{Key: "recipient", Value: "terra1abcdr2"},
				{Key: "sender", Value: "terra1abcdr"},
				{Key: "amount", Value: "100uluna"},
				{Key: "recipient", Value: "terra1abcds"},
				{Key: "sender", Value: "terra1abcdr"},
				{Key: "amount", Value: "1uluna"},
			},
			filter: map[string]bool{
				"amount":    true,
				"recipient": true,
				"sender":    true,
			},
			expected: &Attributes{
				{Key: "amount", Value: "100uluna"},
				{Key: "recipient", Value: "terra1abcdr"},
				{Key: "sender", Value: "terra1abcds"},
				{Key: "amount", Value: "100uluna"},
				{Key: "recipient", Value: "terra1abcdr2"},
				{Key: "sender", Value: "terra1abcdr"},
				{Key: "amount", Value: "1uluna"},
				{Key: "recipient", Value: "terra1abcds"},
				{Key: "sender", Value: "terra1abcdr"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := SortAttributes(tc.attrs, tc.filter)

			assert.NoError(t, err)
			assert.Len(t, *actual, len(*tc.expected))
			for i := range *actual {
				assert.Equal(t, (*tc.expected)[i], (*actual)[i])
			}
		})
	}
}
