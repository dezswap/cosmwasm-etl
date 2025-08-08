package dex

import (
	"fmt"
	"math/big"
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

func TestToBigInt(t *testing.T) {
	ln, _ := new(big.Int).SetString("9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999", 10)
	tests := []struct {
		name    string
		input   string
		want    *big.Int
		wantErr bool
	}{
		{
			name:    "valid positive number",
			input:   "123456789",
			want:    big.NewInt(123456789),
			wantErr: false,
		},
		{
			name:    "valid zero",
			input:   "0",
			want:    big.NewInt(0),
			wantErr: false,
		},
		{
			name:    "negative number",
			input:   "-123",
			want:    big.NewInt(-123),
			wantErr: false,
		},
		{
			name:    "valid large number",
			input:   "9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999",
			want:    ln,
			wantErr: false,
		},
		{
			name:    "invalid number with letters",
			input:   "123abc",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToBigInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToBigInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Cmp(tt.want) != 0 {
				t.Errorf("ToBigInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
