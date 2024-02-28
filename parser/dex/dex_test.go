package dex

import (
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
		repoMock.On("Insert", height, tc.txs, tc.poolInfos, pairDtos).Return(err)

		app := dexApp{Repo: &repoMock}
		err = app.insert(height, tc.txs, tc.poolInfos)
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

var (
	createTx   = ParsedTx{"", time.Time{}, CreatePair, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", "1000"}, {"Asset1", "1000"}}, "Lp", "1000", "", nil}
	swapTx     = ParsedTx{"", time.Time{}, Swap, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", "1000"}, {"Asset1", "-1000"}}, "", "", "1", nil}
	provideTx  = ParsedTx{"", time.Time{}, Provide, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", "1000"}, {"Asset1", "1000"}}, "Lp", "1000", "", nil}
	withdrawTx = ParsedTx{"", time.Time{}, Withdraw, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", "-1000"}, {"Asset1", "-1000"}}, "Lp", "1000", "", nil}
	transferTx = ParsedTx{"", time.Time{}, Transfer, "sender", "PAIR_ADDR", [2]Asset{{"Asset0", ""}, {"Asset1", "1000"}}, "", "", "", nil}
)
