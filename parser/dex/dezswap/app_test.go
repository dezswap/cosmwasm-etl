package dezswap

import (
	"testing"

	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/stretchr/testify/assert"
)

func Test_collectLpBurnTxs(t *testing.T) {
	app := &dezswapApp{
		lpPairAddrs: map[string]string{
			"LpToken": "PairContract",
		},
	}

	tcs := []struct {
		burnTxs  []*dex.ParsedTx
		expected []dex.ParsedTx
		errMsg   string
	}{
		{
			burnTxs:  []*dex.ParsedTx{{LpAddr: "LpToken", LpAmount: "-1000"}},
			expected: []dex.ParsedTx{{LpAddr: "LpToken", ContractAddr: "PairContract", LpAmount: "-1000"}},
			errMsg:   "known LP addr must be collected with pair contract addr assigned",
		},
		{
			burnTxs:  []*dex.ParsedTx{{LpAddr: "UnknownLpToken", LpAmount: "-1000"}},
			expected: []dex.ParsedTx{},
			errMsg:   "unknown LP addr must be filtered out",
		},
	}

	for _, tc := range tcs {
		assert := assert.New(t)
		result := app.collectLpBurnTxs(tc.burnTxs)
		assert.Equal(tc.expected, result, tc.errMsg)
	}
}
