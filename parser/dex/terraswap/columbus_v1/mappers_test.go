package columbus_v1

import (
	"fmt"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	el "github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/stretchr/testify/assert"
)

func Test_PairMapper(t *testing.T) {

	pair := dex.Pair{ContractAddr: "Pair", LpAddr: "LiquidityToken", Assets: []string{"Asset1", "Asset2"}}
	pairSet := map[string]dex.Pair{pair.ContractAddr: pair}
	tcs := []struct {
		mapper         parser.Mapper[dex.ParsedTx]
		matchedResults el.MatchedResult
		expectedTx     []*dex.ParsedTx
		errMsg         string
	}{

		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "swap"},
				{Key: "offer_asset", Value: pair.Assets[0]}, {Key: "ask_asset", Value: pair.Assets[1]},
				{Key: "offer_amount", Value: "100000"}, {Key: "return_amount", Value: "100583"},
				{Key: "tax_amount", Value: "0"},
				{Key: "spread_amount", Value: "2"}, {Key: "commission_amount", Value: "302"},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Swap, "", pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "100000"}, {pair.Assets[1], "-100583"}}, "", "", "302", map[string]interface{}{"tax_amount": dex.Asset{pair.Assets[1], "0"}}}},
			"",
		},
		{

			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "swap"},
				{Key: "offer_asset", Value: pair.Assets[1]}, {Key: "ask_asset", Value: pair.Assets[0]},
				{Key: "offer_amount", Value: "100000"}, {Key: "return_amount", Value: "100583"},
				{Key: "tax_amount", Value: "583"},
				{Key: "spread_amount", Value: "2"}, {Key: "commission_amount", Value: "300"},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Swap, "", pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "-100000"}, {pair.Assets[1], "100000"}}, "", "", "300", map[string]interface{}{"tax_amount": dex.Asset{pair.Assets[0], "583"}}}},
			"",
		},
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: "IT REQUIRED MORE MATCHED"}, {Key: "action", Value: "create_pair"},
			},
			nil,
			"expected results length(10)",
		},

		/// Provide
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "assets", Value: fmt.Sprintf("%s%s, %s%s", "1000", pair.Assets[0], "10000", pair.Assets[1])},
				{Key: "share", Value: "998735"},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Provide, "", pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], "10000"}}, pair.LpAddr, "998735", "", nil}},
			"",
		},
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "assets", Value: fmt.Sprintf("%s,%s %s%s", "1000", pair.Assets[0], "10000", pair.Assets[1])},
				{Key: "share", Value: "998735"},
			},
			nil,
			"Wrong format of assets must return error",
		},

		/// Withdraw
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.ContractAddr},
				{Key: "action", Value: "withdraw_liquidity"},
				{Key: "withdrawn_share", Value: "12418119"},
				{Key: "refund_assets", Value: fmt.Sprintf("%s%s, %s%s", "24999998", pair.Assets[0], "24939789", pair.Assets[1])},
			},
			[]*dex.ParsedTx{{"", time.Time{}, dex.Withdraw, "", pair.ContractAddr, [2]dex.Asset{{pair.Assets[0], "0"}, {pair.Assets[1], "0"}}, pair.LpAddr, "12418119", "", map[string]interface{}{"withdraw_assets": []dex.Asset{{pair.Assets[0], "-24999998"}, {pair.Assets[1], "-24939789"}}}}},
			"",
		},
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "assets", Value: fmt.Sprintf("%s,%s %s%s", "1000", pair.Assets[0], "10000", pair.Assets[1])},
				{Key: "share", Value: "998735"},
			},
			nil,
			"Wrong format of assets must return error",
		},
	}

	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("tc(%d)", idx)
		assert := assert.New(t)

		tx, err := tc.mapper.MatchedToParsedTx(tc.matchedResults)
		if tc.errMsg != "" {
			assert.Error(err, errMsg, tc.errMsg)
		} else {
			assert.NoError(err, err)
			assert.Equal(tc.expectedTx, tx, errMsg)
		}
	}
}
