package dex

import (
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// insert implements parser
func Test_insert(t *testing.T) {
	const (
		chainId = "chainId"
		height  = uint64(100)
	)
	poolInfo := PoolInfo{ContractAddr: "ContractAddr", TotalShare: "1000", Assets: []Asset{{"Asset0", "100"}, {"Asset1", "100"}}}

	tcs := []struct {
		txs       []ParsedTx
		poolInfos []PoolInfo
		errMsg    string
	}{
		{
			[]ParsedTx{createTx, swapTx, provideTx, withdrawTx, transferTx},
			[]PoolInfo{poolInfo},
			"",
		},
		{
			[]ParsedTx{swapTx, provideTx, withdrawTx, transferTx},
			[]PoolInfo{poolInfo},
			"return error",
		},
	}

	for _, tc := range tcs {
		repoMock := RepoMock{}
		pairDtos := []Pair{}
		for _, tx := range tc.txs {
			if tx.Type == CreatePair {
				pairDto := Pair{
					ContractAddr: tx.ContractAddr,
					Assets:       []string{tx.Assets[0].Addr, tx.Assets[1].Addr},
					LpAddr:       tx.LpAddr,
				}
				pairDtos = append(pairDtos, pairDto)
			}
		}
		var err error
		if tc.errMsg != "" {
			err = errors.New(tc.errMsg)
		}
		repoMock.On("Insert", height-1, height, tc.txs, tc.poolInfos, pairDtos).Return(err)

		app := dexApp{Repo: &repoMock}
		err = app.insert(height-1, height, tc.txs, tc.poolInfos)
		if tc.errMsg != "" {
			assert.Error(t, err, tc.errMsg)
			repoMock.AssertExpectations(t)
		} else {
			assert.NoError(t, err)
		}
	}
}
func Test_srcHeightCheck(t *testing.T) {
	app := &dexApp{
		sameHeightTolerance: 3,
		lastSrcHeight:       100,
		sameHeightCount:     3,
	}

	testCases := []struct {
		srcHeight   uint64
		expectError bool
	}{
		{srcHeight: 100, expectError: true},
		{srcHeight: 101, expectError: false},
		{srcHeight: 101, expectError: false},
		{srcHeight: 101, expectError: false},
		{srcHeight: 101, expectError: false},
		{srcHeight: 101, expectError: true},
	}

	for i, tc := range testCases {
		err := app.checkRemoteHeight(tc.srcHeight)
		if (err != nil) != tc.expectError {
			t.Errorf("Test case %d failed: srcHeight=%d, expectError=%v, got error=%v",
				i, tc.srcHeight, tc.expectError, err)
		} else {
			app.lastSrcHeight = tc.srcHeight
		}
	}
}
func Test_validate(t *testing.T) {
	tcs := []struct {
		actualPools []PoolInfo
		exceptions  []string
		errMsg      string
		explain     string
	}{
		{
			[]PoolInfo{
				{ContractAddr: "ContractAddr1", TotalShare: "1000", Assets: []Asset{{"Asset0", "100"}, {"Asset1", "100"}}},
				{ContractAddr: "ContractAddr2", TotalShare: "2000", Assets: []Asset{{"Asset0", "200"}, {"Asset1", "200"}}},
			},
			[]string{},
			"",
			"must match the expected pools",
		},
		{
			[]PoolInfo{
				{ContractAddr: "ContractAddr1", TotalShare: "1000", Assets: []Asset{{"Asset0", "100"}, {"Asset1", "100"}}},
			},
			[]string{"ContractAddr2"},
			"",
			"ContractAddr2 is in exception list, must be skipped although actual pool does not match the expected pool",
		},
		{
			[]PoolInfo{
				{ContractAddr: "ContractAddr1", TotalShare: "1000", Assets: []Asset{{"Asset0", "100"}, {"Asset1", "100"}}},
				{ContractAddr: "ContractAddr2", TotalShare: "2000", Assets: []Asset{{"Asset0", "200"}, {"Asset1", "100"}}},
			},
			[]string{"ContractAddr2"},
			"",
			"ContractAddr2 is in exception list, must be skipped although asset1 amount does not match the expected pool",
		},
		{
			[]PoolInfo{
				{ContractAddr: "ContractAddr1", TotalShare: "1000", Assets: []Asset{{"Asset0", "100"}, {"Asset1", "100"}}},
				{ContractAddr: "ContractAddr2", TotalShare: "2000", Assets: []Asset{{"Asset0", "200"}, {"Asset1", "200"}}},
				{ContractAddr: "ContractAddr3", TotalShare: "3000", Assets: []Asset{{"Asset0", "300"}, {"Asset1", "300"}}},
				{ContractAddr: "ContractAddr4", TotalShare: "4000", Assets: []Asset{{"Asset0", "400"}, {"Asset1", "400"}}},
			},
			[]string{"ContractAddr3", "ContractAddr4"},
			"",
			"must be skipped because ContractAddr3 and ContractAddr4 are in exception list",
		},
	}
	expectedPools := []PoolInfo{
		{ContractAddr: "ContractAddr1", TotalShare: "1000", Assets: []Asset{{"Asset0", "100"}, {"Asset1", "100"}}},
		{ContractAddr: "ContractAddr2", TotalShare: "2000", Assets: []Asset{{"Asset0", "200"}, {"Asset1", "200"}}},
	}
	from, to := uint64(0), uint64(0)

	for idx, tc := range tcs {
		repoMock := RepoMock{}
		repoMock.On("ParsedPoolsInfo", from, to).Return(tc.actualPools, nil)
		repoMock.On("ValidationExceptionList").Return(tc.exceptions, nil)
		dexApp := dexApp{Repo: &repoMock}
		err := dexApp.validate(uint64(from), uint64(to), expectedPools)
		errMsg := fmt.Sprintf("tc(%d): %s", idx, tc.explain)
		if tc.errMsg != "" {
			assert.Error(t, err, errMsg)
		} else {
			assert.NoError(t, err, errMsg)
		}
	}
}

var (
	createTx   = ParsedTx{"", time.Time{}, CreatePair, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", "1000"}, {"Asset1", "1000"}}, "Lp", "1000", "", nil}
	swapTx     = ParsedTx{"", time.Time{}, Swap, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", "1000"}, {"Asset1", "-1000"}}, "", "", "1", nil}
	provideTx  = ParsedTx{"", time.Time{}, Provide, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", "1000"}, {"Asset1", "1000"}}, "Lp", "1000", "", nil}
	withdrawTx = ParsedTx{"", time.Time{}, Withdraw, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", "-1000"}, {"Asset1", "-1000"}}, "Lp", "1000", "", nil}
	transferTx = ParsedTx{"", time.Time{}, Transfer, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", ""}, {"Asset1", "1000"}}, "", "", "", nil}
)
