package dex

import (
	"fmt"

	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
	el "github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/stretchr/testify/assert"
)

func Test_TransferMapper(t *testing.T) {
	const userAddr = "user"
	pair := Pair{ContractAddr: "Pair", LpAddr: "LiquidityToken", Assets: []string{"Asset1", "Asset2"}}
	pairSet := map[string]Pair{pair.ContractAddr: pair}
	tcs := []struct {
		mapper         parser.Mapper[ParsedTx]
		matchedResults el.MatchedResult
		expectedTx     []*ParsedTx
		errMsg         string
	}{
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000Asset1"},
			},
			[]*ParsedTx{{"", time.Time{}, Transfer, userAddr, pair.ContractAddr, [2]Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], ""}}, "", "", "", make(map[string]interface{})}},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000Asset1,2000Asset2"},
			},
			[]*ParsedTx{{"", time.Time{}, Transfer, userAddr, pair.ContractAddr, [2]Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], "2000"}}, "", "", "", make(map[string]interface{})}},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "123456789012345678901234567890123456Asset2"},
			},
			[]*ParsedTx{{"", time.Time{}, Transfer, userAddr, pair.ContractAddr, [2]Asset{{pair.Assets[0], ""}, {pair.Assets[1], "123456789012345678901234567890123456"}}, "", "", "", make(map[string]interface{})}},
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: "WRONG_PAIR"}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000Asset1"},
			},
			nil,
			"",
		},
		{
			&transferMapper{pairSet: pairSet},
			el.MatchedResult{
				{Key: "recipient", Value: pair.ContractAddr}, {Key: "sender", Value: userAddr}, {Key: "amount", Value: "1000WrongAsset1"},
			},
			[]*ParsedTx{{"", time.Time{}, Transfer, userAddr, pair.ContractAddr, [2]Asset{{pair.Assets[0], ""}, {pair.Assets[1], ""}}, "", "", "", map[string]interface{}{"WrongAsset1": "1000"}}},
			"",
		},
		// / wasm transfer
		{
			&wasmCommonTransferMapper{cw20AddrKey: dex.WasmTransferCw20AddrKey, pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.Assets[0]}, {Key: "action", Value: "transfer"}, {Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr}, {Key: "amount", Value: "1000"},
			},
			[]*ParsedTx{{"", time.Time{}, Transfer, userAddr, pair.ContractAddr, [2]Asset{{pair.Assets[0], "1000"}, {pair.Assets[1], ""}}, "", "", "", make(map[string]interface{})}},
			"",
		},
		{
			&wasmCommonTransferMapper{cw20AddrKey: dex.WasmTransferCw20AddrKey, pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.Assets[1]}, {Key: "action", Value: "transfer"},
				{Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr},
				{Key: "amount", Value: "123456789012345678901234567890123456"},
			},
			[]*ParsedTx{{"", time.Time{}, Transfer, userAddr, pair.ContractAddr, [2]Asset{{pair.Assets[0], ""}, {pair.Assets[1], "123456789012345678901234567890123456"}}, "", "", "", make(map[string]interface{})}},
			"",
		},
		{
			&wasmCommonTransferMapper{cw20AddrKey: dex.WasmTransferCw20AddrKey, pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.Assets[0]}, {Key: "action", Value: "transfer"}, {Key: "from", Value: userAddr},
				{Key: "to", Value: "WRONG_PAIR"}, {Key: "amount", Value: "1000"},
			},
			nil,
			"",
		},
		{
			&wasmCommonTransferMapper{cw20AddrKey: dex.WasmTransferCw20AddrKey, pairSet: pairSet},
			el.MatchedResult{
				{Key: "_contract_address", Value: "WRONG_CW_20"}, {Key: "action", Value: "transfer"},
				{Key: "from", Value: userAddr}, {Key: "to", Value: pair.ContractAddr}, {Key: "amount", Value: "1000"},
			},
			[]*ParsedTx{{"", time.Time{}, Transfer, userAddr, pair.ContractAddr, [2]Asset{{pair.Assets[0], ""}, {pair.Assets[1], ""}}, "", "", "", map[string]interface{}{"WRONG_CW_20": "1000"}}},
			"",
		},
	}

	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("tc(%d)", idx)
		assert := assert.New(t)

		tx, err := tc.mapper.MatchedToParsedTx(tc.matchedResults)
		if tc.errMsg != "" {
			assert.Error(err, errMsg, tc.errMsg)
		}
		assert.Equal(tc.expectedTx, tx, errMsg)
	}
}

func Test_InitialProvideMapper(t *testing.T) {

	const userAddr = "userAddr"
	pair := Pair{ContractAddr: "Pair", LpAddr: "LiquidityToken", Assets: []string{"Asset1", "Asset2"}}
	tcs := []struct {
		mapper         parser.Mapper[ParsedTx]
		matchedResults el.MatchedResult
		expectedTx     []*ParsedTx
		errMsg         string
	}{
		/// Initial Provide
		{
			&initialProvideMapper{dex.MapperMixin{}},
			el.MatchedResult{
				{Key: "_contract_address", Value: pair.LpAddr}, {Key: "action", Value: "mint"},
				{Key: "amount", Value: "1000"}, {Key: "to", Value: pair.ContractAddr},
			},
			[]*ParsedTx{{"", time.Time{}, InitialProvide, "", pair.ContractAddr, [2]Asset{}, pair.LpAddr, "1000", "", nil}},
			"",
		},
		{
			&initialProvideMapper{dex.MapperMixin{}}, el.MatchedResult{
				{Key: "_contract_address", Value: pair.LpAddr}, {Key: "action", Value: "mint"},
				{Key: "amount", Value: "1000"}, {Key: "to", Value: pair.ContractAddr}, {Key: "sender", Value: pair.ContractAddr},
			},
			nil,
			"Wrong format of matched must return error",
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
