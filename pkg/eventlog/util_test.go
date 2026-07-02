package eventlog

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSortAttributes(t *testing.T) {
	tests := []struct {
		name     string
		attrs    Attributes
		filter   []string
		expected *Attributes
	}{
		{
			name: "default tc",
			attrs: Attributes{
				{Key: "recipient", Value: "terra1abcdr"},
				{Key: "sender", Value: "terra1abcds"},
				{Key: "amount", Value: "100uluna"},
			},
			filter: []string{
				"amount",
				"recipient",
				"sender",
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
			filter: []string{
				"amount",
				"recipient",
				"sender",
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
			filter: []string{
				"amount",
				"recipient",
				"sender",
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
		{
			// https://finder.terra-classic.hexxagon.io/mainnet/tx/4E80262C9F94E11900D903D03646D75397B92B505B0850DB1E034EFB506FF964
			name: "tc with multiple key sets 2",
			attrs: Attributes{
				{Key: "amount", Value: "100uluna"},
				{Key: "recipient", Value: "terra1abcdr"},
				{Key: "sender", Value: "terra1abcds"},
				{Key: "amount", Value: "100uluna"},
				{Key: "recipient", Value: "terra1abcdr2"},
				{Key: "sender", Value: "terra1abcdr"},
				{Key: "recipient", Value: "terra1abcds"},
				{Key: "sender", Value: "terra1abcdr"},
				{Key: "amount", Value: "1uluna"},
			},
			filter: []string{
				"amount",
				"recipient",
				"sender",
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
		{
			name: "tc with optional sender mixed in key sets",
			attrs: Attributes{
				{Key: "recipient", Value: "terra1abcdr"},
				{Key: "amount", Value: "100uluna"},
				{Key: "amount", Value: "101uluna"},
				{Key: "recipient", Value: "terra1abcdr2"},
				{Key: "sender", Value: "terra1abcds2"},
				{Key: "amount", Value: "102uluna"},
				{Key: "recipient", Value: "terra1abcdr3"},
				{Key: "sender", Value: "terra1abcds3"},
				{Key: "recipient", Value: "terra1abcdr4"},
				{Key: "amount", Value: "103uluna"},
			},
			filter: []string{
				"amount",
				"recipient",
				"sender",
			},
			expected: &Attributes{
				{Key: "amount", Value: "100uluna"},
				{Key: "recipient", Value: "terra1abcdr"},
				{Key: "amount", Value: "101uluna"},
				{Key: "recipient", Value: "terra1abcdr2"},
				{Key: "sender", Value: "terra1abcds2"},
				{Key: "amount", Value: "102uluna"},
				{Key: "recipient", Value: "terra1abcdr3"},
				{Key: "sender", Value: "terra1abcds3"},
				{Key: "amount", Value: "103uluna"},
				{Key: "recipient", Value: "terra1abcdr4"},
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

func TestResultToItemMapForKeys(t *testing.T) {
	t.Run("ignores duplicate unused keys", func(t *testing.T) {
		result, err := ResultToItemMapForKeys(MatchedResult{
			{Key: "_contract_address", Value: "token"},
			{Key: "action", Value: "transfer"},
			{Key: "metadata", Value: "first"},
			{Key: "metadata", Value: "second"},
			{Key: "amount", Value: "10"},
		}, "amount")

		require.NoError(t, err)
		require.Equal(t, "10", result["amount"].Value)
	})

	t.Run("rejects duplicate consumed keys", func(t *testing.T) {
		_, err := ResultToItemMapForKeys(MatchedResult{
			{Key: "_contract_address", Value: "token"},
			{Key: "action", Value: "transfer"},
			{Key: "amount", Value: "10", MsgIndex: 3},
			{Key: "amount", Value: "20", MsgIndex: 3},
		}, "amount")

		var ambiguity *AmbiguousEventError
		require.True(t, errors.As(err, &ambiguity))
		require.Equal(t, "token", ambiguity.Contract)
		require.Equal(t, "transfer", ambiguity.Action)
		require.Equal(t, "amount", ambiguity.Key)
		require.Equal(t, []string{"10", "20"}, ambiguity.Values)
		require.Equal(t, 3, ambiguity.MsgIndex)
	})
}
