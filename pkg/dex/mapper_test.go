package dex

import (
	"fmt"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_CheckResult(t *testing.T) {
	mapperMixin := MapperMixin{}

	tcs := []struct {
		matchedResults eventlog.MatchedResult
		expectedLen    int
		errMsg         string
	}{
		{
			eventlog.MatchedResult{
				{Key: "amount", Value: "1000Asset1"}, {Key: "recipient", Value: "Pair"}, {Key: "sender", Value: "A"},
			},
			3,
			"",
		},
		{
			eventlog.MatchedResult{
				{Key: "recipient", Value: "Pair"}, {Key: "sender", Value: "A"}, {Key: "amount", Value: "1000Asset1"},
			},
			3,
			"",
		},
		{
			eventlog.MatchedResult{
				{Key: "recipient", Value: "Pair"}, {Key: "sender", Value: "A"},
				{Key: "amount", Value: "1000Asset1"}, {Key: "WRONG_MATCHED_LENGTH", Value: "LENGTH"},
			},
			3,
			"must return error when matched result length is not equal to expected",
		},
		{
			eventlog.MatchedResult{
				{Key: "recipient", Value: "Pair"}, {Key: "sender", Value: "A"}, {Key: "amount", Value: ""},
			},
			3,
			"empty value must return error",
		},
	}

	assert := assert.New(t)
	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("tc(%d)", idx)

		err := mapperMixin.CheckResult(tc.matchedResults, tc.expectedLen)
		if tc.errMsg != "" {
			assert.Error(err, errMsg, tc.errMsg)
		} else {
			assert.NoError(err, errMsg)
		}
	}
}

func Test_sortResult(t *testing.T) {
	const userAddr = "userAddr"

	pairContractAddr := "Pair"
	assets := []string{"Asset1", "Asset2"}
	tcs := []struct {
		target   eventlog.MatchedResult
		expected eventlog.MatchedResult

		errMsg string
	}{
		{
			eventlog.MatchedResult{
				{Key: "_contract_address", Value: pairContractAddr},
				{Key: "action", Value: "send"},
				{Key: "from", Value: userAddr},
				{Key: "to", Value: pairContractAddr},
				{Key: "amount", Value: "332157"},
				{Key: "_contract_address", Value: pairContractAddr},
				{Key: "action", Value: "swap"},
				{Key: "sender", Value: userAddr},
				{Key: "receiver", Value: userAddr},
				{Key: "offer_asset", Value: assets[0]},
				{Key: "ask_asset", Value: assets[1]},
				{Key: "offer_amount", Value: "100000"},
				{Key: "return_amount", Value: "100583"},
				{Key: "spread_amount", Value: "2"},
				{Key: "commission_amount", Value: "302"},
			},
			eventlog.MatchedResult{
				{Key: "_contract_address", Value: pairContractAddr},
				{Key: "action", Value: "send"},
				{Key: "amount", Value: "332157"},
				{Key: "from", Value: userAddr},
				{Key: "to", Value: pairContractAddr},
				{Key: "_contract_address", Value: pairContractAddr},
				{Key: "action", Value: "swap"},
				{Key: "ask_asset", Value: assets[1]},
				{Key: "commission_amount", Value: "302"},
				{Key: "offer_amount", Value: "100000"},
				{Key: "offer_asset", Value: assets[0]},
				{Key: "receiver", Value: userAddr},
				{Key: "return_amount", Value: "100583"},
				{Key: "sender", Value: userAddr},
				{Key: "spread_amount", Value: "2"},
			},
			"",
		},
	}

	mm := MapperMixin{}
	for _, tc := range tcs {
		mm.SortResult(tc.target)
		assert.Equal(t, tc.target, tc.expected, tc.errMsg)
	}
}
