package parser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetAssetsFromAssetsString(t *testing.T) {

	tcs := []struct {
		amountsAssets string
		expected      []Asset
		errMsg        string
	}{
		{"1000a, 1000b", []Asset{{"a", "1000"}, {"b", "1000"}}, ""},
		{"54631040ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4, 7483704uluna", []Asset{{"ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4", "54631040"}, {"uluna", "7483704"}}, ""},
		{"1000ibc/first, 1000ibc/second", []Asset{{"ibc/first", "1000"}, {"ibc/second", "1000"}}, ""},
		{"1000, 1000b", nil, "wrong asset0 format"},
		{"10001000b", nil, "wrong format"},
		{"1000aaaa,1000b", []Asset{{"aaaa", "1000"}, {"b", "1000"}}, ""},
	}
	assert := assert.New(t)

	for idx, tc := range tcs {
		msg := fmt.Sprintf("tc(%d): %s", idx, tc.errMsg)
		assets, err := GetAssetsFromAssetsString(tc.amountsAssets)
		if tc.errMsg != "" {
			assert.Error(err, msg)
		} else {
			assert.Equal(assets, tc.expected, msg)
			assert.NoError(err, msg)
		}
	}
}

func Test_GetAssetFromAssetAmountString(t *testing.T) {
	tcs := []struct {
		amountAsset string
		expected    Asset
		errMsg      string
	}{
		{"1000a", Asset{"a", "1000"}, ""},
		{"1a", Asset{"a", "1"}, ""},
		{"1000ibc/hash", Asset{"ibc/hash", "1000"}, ""},
		{"10001000b", Asset{"b", "10001000"}, ""},
		{"10001000", Asset{}, "must provide asset address"},
		{"0001", Asset{}, "the amount string must start with a valid digit"},
	}
	assert := assert.New(t)

	for idx, tc := range tcs {
		msg := fmt.Sprintf("tc(%d): %s", idx, tc.errMsg)
		asset, err := GetAssetFromAmountAssetString(tc.amountAsset)
		if tc.errMsg != "" {
			assert.Error(err, msg)
		} else {
			assert.Equal(asset, tc.expected, msg)
			assert.NoError(err, msg)
		}
	}

}
