package columbusv2

import (
	"fmt"

	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap/columbusv2"
	el "github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/stretchr/testify/assert"
)

func Test_PairMapper(t *testing.T) {

	pair := dex.Pair{ContractAddr: "Pair", LpAddr: "LiquidityToken", Assets: []string{"terra1Asset1", "terra1Asset2"}}
	userAddr := "userAddr"
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
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "swap"},
				{Key: "sender", Value: userAddr}, {Key: "receiver", Value: userAddr},
				{Key: "offer_asset", Value: pair.Assets[0]}, {Key: "ask_asset", Value: pair.Assets[1]},
				{Key: "offer_amount", Value: "100000"}, {Key: "return_amount", Value: "100583"},
				{Key: "spread_amount", Value: "2"}, {Key: "commission_amount", Value: "302"},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Swap, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "100000"}, {Addr: pair.Assets[1], Amount: "-100583"}}, LpAddr: "", LpAmount: "", CommissionAmount: "302", MsgIndex: 0, Meta: nil}},
			"",
		},
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "swap"},
				{Key: "sender", Value: userAddr}, {Key: "receiver", Value: userAddr},
				{Key: "offer_asset", Value: pair.Assets[1]}, {Key: "ask_asset", Value: pair.Assets[0]},
				{Key: "offer_amount", Value: "100000"}, {Key: "return_amount", Value: "100583"},
				{Key: "spread_amount", Value: "2"}, {Key: "commission_amount", Value: "300"},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Swap, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "-100583"}, {Addr: pair.Assets[1], Amount: "100000"}}, LpAddr: "", LpAmount: "", CommissionAmount: "300", MsgIndex: 0, Meta: nil}},
			"",
		},
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "swap"},
				{Key: "sender", Value: userAddr}, {Key: "receiver", Value: userAddr},
				{Key: "offer_asset", Value: pair.Assets[1]}, {Key: "ask_asset", Value: pair.Assets[0]},
				{Key: "offer_amount", Value: "100000"}, {Key: "return_amount", Value: "100583"}, {Key: "tax_amount", Value: "583"},
				{Key: "spread_amount", Value: "2"}, {Key: "commission_amount", Value: "300"},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Swap, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "-100000"}, {Addr: pair.Assets[1], Amount: "100000"}}, LpAddr: "", LpAmount: "", CommissionAmount: "300", MsgIndex: 0, Meta: nil}},
			"",
		},
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: "IT REQUIRED MORE MATCHED"}, {Key: "action", Value: "create_pair"},
			},
			nil,
			"expected results length(10)",
		},

		/// Provide
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "sender", Value: userAddr}, {Key: "receiver", Value: userAddr},
				{Key: "assets", Value: fmt.Sprintf("%s%s, %s%s", "1000", pair.Assets[0], "10000", pair.Assets[1])},
				{Key: "share", Value: "998735"},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Provide, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "1000"}, {Addr: pair.Assets[1], Amount: "10000"}}, LpAddr: pair.LpAddr, LpAmount: "998735", CommissionAmount: "", MsgIndex: 0, Meta: nil}},
			"",
		},
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "sender", Value: userAddr}, {Key: "receiver", Value: userAddr},
				{Key: "assets", Value: fmt.Sprintf("%s%s, %s%s", "1000", pair.Assets[0], "10000", pair.Assets[1])},
				{Key: "share", Value: "998735"}, {Key: columbusv2.PairProvideRefundAssetKey, Value: fmt.Sprintf("%s%s, %s%s", "100", pair.Assets[0], "100", pair.Assets[1])},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Provide, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "900"}, {Addr: pair.Assets[1], Amount: "9900"}}, LpAddr: pair.LpAddr, LpAmount: "998735", CommissionAmount: "", MsgIndex: 0, Meta: nil}},
			"",
		},
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "sender", Value: userAddr}, {Key: "receiver", Value: userAddr},
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
				{Key: "_contract_address", Value: pair.ContractAddr},
				{Key: "action", Value: "withdraw_liquidity"}, {Key: "sender", Value: userAddr},
				{Key: "withdrawn_share", Value: "12418119"},
				{Key: "refund_assets", Value: fmt.Sprintf("%s%s, %s%s", "24999998", pair.Assets[0], "24939789", pair.Assets[1])},
			},
			[]*dex.ParsedTx{{Hash: "", Timestamp: time.Time{}, Type: dex.Withdraw, Sender: userAddr, ContractAddr: pair.ContractAddr, Assets: [2]dex.Asset{{Addr: pair.Assets[0], Amount: "0"}, {Addr: pair.Assets[1], Amount: "0"}}, LpAddr: pair.LpAddr, LpAmount: "12418119", CommissionAmount: "", MsgIndex: 0, Meta: map[string]interface{}{"withdraw_assets": []dex.Asset{{Addr: pair.Assets[0], Amount: "-24999998"}, {Addr: pair.Assets[1], Amount: "-24939789"}}}}},
			"",
		},
		{
			&pairMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.ContractAddr}, {Key: "action", Value: "provide_liquidity"},
				{Key: "sender", Value: userAddr},
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
