package aggregator

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"gorm.io/gorm"

	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type ConnPool struct{ gorm.TxCommitter }

func (p ConnPool) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) { return nil, nil }
func (p ConnPool) ExecContext(_ context.Context, _ string, _ ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (p ConnPool) QueryContext(_ context.Context, _ string, _ ...interface{}) (*sql.Rows, error) {
	return nil, nil
}
func (p ConnPool) QueryRowContext(_ context.Context, _ string, _ ...interface{}) *sql.Row { return nil }
func (p ConnPool) Commit() error                                                          { return nil }
func (p ConnPool) Rollback() error                                                        { return nil }

type completedTask struct {
	height atomic.Uint64
}

func (t *completedTask) Execute(_ time.Time, _ time.Time) error {
	return nil
}

func (t *completedTask) LastProcessedHeight() uint64 {
	return t.height.Load()
}

func (t *completedTask) setLastProcessedHeight(height uint64) {
	t.height.Store(height)
}

func TestLpHistoryTaskExecute(t *testing.T) {
	assert := assert.New(t)

	history := []schemas.LpHistory{
		{
			Height:     1,
			PairId:     1,
			ChainId:    "cube_47-5",
			Liquidity0: "1000000",
			Liquidity1: "1000000",
			Timestamp:  1692939766,
		},
	}

	txs := []schemas.ParsedTxWithPrice{
		{
			PairId:            1,
			ChainId:           "cube_47-5",
			Asset0Amount:      "1000000",
			Asset1Amount:      "-500000",
			Commission0Amount: "0",
			Commission1Amount: "50000",
			Price0:            "1",
			Price1:            "1",
			Decimals0:         6,
			Decimals1:         6,
			Height:            2,
			Timestamp:         1692939767,
		},
	}

	expected := []schemas.LpHistory{
		{
			Height:     2,
			PairId:     1,
			ChainId:    "cube_47-5",
			Liquidity0: "2000000.000000000000000000",
			Liquidity1: "500000.000000000000000000",
			Timestamp:  1692939767,
		},
	}

	rp := repoMock{}
	rp.On("LastLpHistory", mock.Anything).Return(history, nil)
	rp.On("GetParsedTxsWithLimit", mock.Anything, mock.Anything).Return(txs, nil)
	rp.On("UpdateLpHistory", mock.Anything).Return(nil)

	task := lpHistoryTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		srcDb: &rp,
	}

	err := task.Execute(time.Time{}, time.Time{})
	assert.NoError(err)
	assert.Equal(expected[0], rp.updatedLpHistory[0])
}

func TestPairStatsRecentUpdateTaskExecute(t *testing.T) {
	assert := assert.New(t)

	height := uint64(100000)
	txs := []schemas.ParsedTxWithPrice{
		{
			PairId:            1,
			ChainId:           "",
			Asset0Amount:      "3000000",
			Asset1Amount:      "3000000",
			Asset0Liquidity:   "3000000",
			Asset1Liquidity:   "3000000",
			Commission0Amount: "1000000",
			Commission1Amount: "1000000",
			Price0:            "1",
			Price1:            "2",
			Decimals0:         6,
			Decimals1:         6,
			Height:            height,
			Timestamp:         float64(0),
		},
		{
			PairId:            1,
			ChainId:           "",
			Asset0Amount:      "3000000",
			Asset1Amount:      "4000000",
			Asset0Liquidity:   "6000000",
			Asset1Liquidity:   "7000000",
			Commission0Amount: "1000000",
			Commission1Amount: "2000000",
			Price0:            "1",
			Price1:            "2",
			Decimals0:         6,
			Decimals1:         6,
			Height:            height + 1,
			Timestamp:         float64(0),
		},
	}

	priceMap := map[uint64][]schemas.Price{
		1: {
			schemas.Price{
				Height:  height,
				TokenId: 1,
				Price:   "1",
			},
			schemas.Price{
				Height:  height + 1,
				TokenId: 1,
				Price:   "2",
			},
		},
		2: {
			schemas.Price{
				Height:  height,
				TokenId: 2,
				Price:   "1",
			},
			schemas.Price{
				Height:  height,
				TokenId: 2,
				Price:   "2",
			},
		},
	}

	expected := []schemas.PairStatsRecent{
		{
			PairId:             1,
			ChainId:            "",
			Volume0:            "3000000.000000000000000000",
			Volume1:            "4000000.000000000000000000",
			Volume0InPrice:     "6.000000000000000000",
			Volume1InPrice:     "8.000000000000000000",
			Liquidity0:         "6000000.000000000000000000",
			Liquidity1:         "7000000.000000000000000000",
			Liquidity0InPrice:  "12.000000000000000000",
			Liquidity1InPrice:  "14.000000000000000000",
			Commission0:        "1000000.000000000000000000",
			Commission1:        "2000000.000000000000000000",
			Commission0InPrice: "2.000000000000000000",
			Commission1InPrice: "4.000000000000000000",
			Height:             height + 1,
			Timestamp:          float64(0),
		},
	}

	rp := repoMock{}
	rp.On("HeightOnTimestamp").Return(txs[0].Height, nil)
	rp.On("LastHeightOfPrice").Return(txs[len(txs)-1].Height, nil)
	rp.On("GetRecentParsedTxs", mock.Anything, mock.Anything, mock.Anything).Return(txs, nil)
	rp.On("RecentPrices", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(priceMap, nil)
	rp.On("BeginTx").Return(&gorm.DB{Statement: &gorm.Statement{ConnPool: &ConnPool{}}}, nil)

	task := pairStatsRecentUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		priceToken: "",
		srcDb:      &rp,
	}

	err := task.Execute(time.Time{}, time.Time{})
	assert.NoError(err)
	assert.Equal(expected[0], rp.updatedPairStatsRecent[len(rp.updatedPairStatsRecent)-1])
}

func TestPairStatsUpdateTaskExecute(t *testing.T) {
	assert := assert.New(t)

	end := time.Unix(1666765800, 0).UTC() // 2022-10-26 06:30:00 UTC
	pairId := uint64(1)
	txCnt := 1
	providerCnt := uint64(4)

	stats := []schemas.PairStats30m{
		{
			YearUtc:            end.Year(),
			MonthUtc:           int(end.Month()),
			DayUtc:             end.Day(),
			HourUtc:            end.Hour(),
			MinuteUtc:          end.Minute(),
			PairId:             pairId,
			Volume0:            "6000000.000000000000000000",
			Volume1:            "7000000.000000000000000000",
			Volume0InPrice:     "9.000000000000000000",
			Volume1InPrice:     "11.000000000000000000",
			LastSwapPrice:      "0.750000000000000000",
			Commission0:        "2000000.000000000000000000",
			Commission1:        "3000000.000000000000000000",
			Commission0InPrice: "3.000000000000000000",
			Commission1InPrice: "5.000000000000000000",
			TxCnt:              txCnt,
			ProviderCnt:        providerCnt,
			Timestamp:          float64(end.Unix()),
		},
	}

	lpMap := map[uint64]schemas.PairStats30m{
		pairId: {
			PairId:            pairId,
			Liquidity0:        "7000000.000000000000000000",
			Liquidity1:        "8000000.000000000000000000",
			Liquidity0InPrice: "14.000000000000000000",
			Liquidity1InPrice: "16.000000000000000000",
		},
	}

	expected := schemas.PairStats30m{
		YearUtc:            end.Year(),
		MonthUtc:           int(end.Month()),
		DayUtc:             end.Day(),
		HourUtc:            end.Hour(),
		MinuteUtc:          end.Minute(),
		PairId:             pairId,
		Volume0:            "6000000.000000000000000000",
		Volume1:            "7000000.000000000000000000",
		Volume0InPrice:     "9.000000000000000000",
		Volume1InPrice:     "11.000000000000000000",
		LastSwapPrice:      "0.750000000000000000",
		Liquidity0:         "7000000.000000000000000000",
		Liquidity1:         "8000000.000000000000000000",
		Liquidity0InPrice:  "14.000000000000000000",
		Liquidity1InPrice:  "16.000000000000000000",
		Commission0:        "2000000.000000000000000000",
		Commission1:        "3000000.000000000000000000",
		Commission0InPrice: "3.000000000000000000",
		Commission1InPrice: "5.000000000000000000",
		TxCnt:              txCnt,
		ProviderCnt:        providerCnt,
		Timestamp:          float64(end.Unix()),
	}

	rp := repoMock{}
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)
	rp.On("LastHeightOfPrice").Return(uint64(0), nil)
	rp.On("PairStats", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(stats, nil)
	rp.On("LiquiditiesOfPairStats", mock.Anything, mock.Anything, mock.Anything).Return(lpMap, nil)

	task := pairStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		srcDb:       &rp,
		prevStatMap: make(map[uint64]schemas.PairStats30m),
	}
	err := task.Execute(time.Time{}, end)

	assert.NoError(err)
	assert.Equal(expected, rp.updatedPairStats[0])
}

func TestExecuteAccountStatsUpdateTask(t *testing.T) {
	assert := assert.New(t)

	end := time.Unix(1666765800, 0).UTC() // 2022-10-26 06:30:00 UTC
	accountAddress := "terra0wal1let2"
	accountId := uint64(7)
	pairId := uint64(1)
	txCnt := uint64(5)
	priceToken := "uusd"

	expected := []schemas.AccountStats30m{
		{
			YearUtc:             end.Year(),
			MonthUtc:            int(end.Month()),
			DayUtc:              end.Day(),
			HourUtc:             end.Hour(),
			MinuteUtc:           end.Minute(),
			Timestamp:           util.ToEpoch(end),
			AccountId:           accountId,
			Address:             accountAddress,
			PairId:              pairId,
			TxCnt:               txCnt,
			SwapTxCnt:           3,
			ProvideTxCnt:        1,
			SwapVolumeInPrice:   "10",
			ProvideValueInPrice: "20",
			PriceToken:          priceToken,
			NetLpAmount:         "30",
		},
	}

	stats := []schemas.AccountStats30m{{
		Address:             accountAddress,
		PairId:              pairId,
		TxCnt:               txCnt,
		SwapTxCnt:           3,
		ProvideTxCnt:        1,
		SwapVolumeInPrice:   "10",
		ProvideValueInPrice: "20",
		PriceToken:          priceToken,
		NetLpAmount:         "30",
	}}

	rp := repoMock{}
	rp.On("AccountStats", mock.Anything, mock.Anything, priceToken).Return(stats, nil)
	rp.On("AccountIds").Return(map[string]uint64{accountAddress: accountId}, nil)
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		priceToken: priceToken,
		srcDb:      &rp,
	}
	err := task.Execute(time.Time{}, end)

	assert.NoError(err)
	assert.Equal(expected, rp.updatedAccountStats)
}

func TestExecuteAccountStatsUpdateTaskDeduplicatesAccountCreation(t *testing.T) {
	assert := assert.New(t)

	priceToken := "uusd"
	accountAddress := "terra0duplicated"
	stats := []schemas.AccountStats30m{
		{Address: accountAddress, PairId: 1, TxCnt: 1},
		{Address: accountAddress, PairId: 2, TxCnt: 1},
	}

	rp := repoMock{}
	rp.On("AccountStats", mock.Anything, mock.Anything, priceToken).Return(stats, nil)
	rp.On("AccountIds").Return(map[string]uint64{accountAddress: 11}, nil)
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "cube_47-5",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		priceToken: priceToken,
		srcDb:      &rp,
	}
	err := task.Execute(time.Time{}, time.Unix(1666765800, 0).UTC())

	assert.NoError(err)
	assert.Equal([]string{accountAddress}, rp.updatedAccounts)
	assert.Len(rp.updatedAccountStats, 2)
	assert.Equal(uint64(11), rp.updatedAccountStats[0].AccountId)
	assert.Equal(uint64(11), rp.updatedAccountStats[1].AccountId)
}

func TestExecuteAccountStatsUpdateTaskReturnsErrorWhenAccountIdMissing(t *testing.T) {
	assert := assert.New(t)

	priceToken := "uusd"
	accountAddress := "terra0missing"
	stats := []schemas.AccountStats30m{{Address: accountAddress, PairId: 1, TxCnt: 1}}

	rp := repoMock{}
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)
	rp.On("AccountStats", mock.Anything, mock.Anything, priceToken).Return(stats, nil)
	rp.On("AccountIds").Return(map[string]uint64{}, nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "cube_47-5",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		priceToken: priceToken,
		srcDb:      &rp,
	}
	err := task.Execute(time.Time{}, time.Unix(1666765800, 0).UTC())

	assert.ErrorContains(err, "account id not found")
	assert.Empty(rp.updatedAccountStats)
}

func TestExecuteAccountStatsUpdateTaskPropagatesCreateAccountsError(t *testing.T) {
	assert := assert.New(t)

	priceToken := "uusd"
	createErr := errors.New("create accounts failed")
	stats := []schemas.AccountStats30m{{Address: "terra0fail", PairId: 1, TxCnt: 1}}

	rp := repoMock{createAccountsErr: createErr}
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)
	rp.On("AccountStats", mock.Anything, mock.Anything, priceToken).Return(stats, nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "cube_47-5",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		priceToken: priceToken,
		srcDb:      &rp,
	}
	err := task.Execute(time.Time{}, time.Unix(1666765800, 0).UTC())

	assert.ErrorIs(err, createErr)
	assert.Empty(rp.updatedAccountStats)
	rp.AssertNotCalled(t, "AccountIds", mock.Anything)
}

func TestExecuteAccountStatsUpdateTaskPassesPriceToken(t *testing.T) {
	assert := assert.New(t)

	priceToken := "uusd"
	end := time.Unix(1666765800, 0).UTC()

	rp := repoMock{}
	rp.On("AccountStats", util.ToEpoch(time.Time{}), util.ToEpoch(end), priceToken).Return([]schemas.AccountStats30m{}, nil)
	rp.On("HeightOnTimestamp").Return(uint64(0), nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "cube_47-5",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		priceToken: priceToken,
		srcDb:      &rp,
	}
	err := task.Execute(time.Time{}, end)

	assert.NoError(err)
	rp.AssertCalled(t, "AccountStats", util.ToEpoch(time.Time{}), util.ToEpoch(end), priceToken)
}

func TestExecuteAccountStatsUpdateTaskNoStats(t *testing.T) {
	assert := assert.New(t)

	rp := repoMock{}
	rp.On("AccountStats", mock.Anything, mock.Anything, "").Return([]schemas.AccountStats30m{}, nil)
	rp.On("HeightOnTimestamp").Return(uint64(10), nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "cube_47-5",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		srcDb: &rp,
	}
	err := task.Execute(time.Time{}, time.Unix(1666765800, 0).UTC())

	assert.NoError(err)
	assert.Empty(rp.updatedAccounts)
	assert.Empty(rp.updatedAccountStats)
	assert.Equal(uint64(10), task.LastProcessedHeight())
}

func TestExecuteAccountStatsUpdateTaskWaitsForParentPriceTask(t *testing.T) {
	assert := assert.New(t)

	end := time.Unix(1666765800, 0).UTC()
	endHeight := uint64(10)
	accountStatsCalled := make(chan struct{}, 1)

	rp := repoMock{}
	rp.On("HeightOnTimestamp").Return(endHeight, nil)
	rp.On("AccountStats", mock.Anything, mock.Anything, "").Run(func(_ mock.Arguments) {
		accountStatsCalled <- struct{}{}
	}).Return([]schemas.AccountStats30m{}, nil)

	parent := &completedTask{}
	parent.setLastProcessedHeight(endHeight - 1)
	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId:     "cube_47-5",
			destDb:      &rp,
			parentTasks: []task{parent},
			logger:      logging.Discard,
		},
		srcDb: &rp,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- task.Execute(time.Time{}, end)
	}()

	select {
	case <-accountStatsCalled:
		assert.Fail("AccountStats was called before the parent price task reached the target height")
	case <-time.After(WaitPeriod / 2):
	}

	parent.setLastProcessedHeight(endHeight)

	var err error
	select {
	case err = <-errCh:
	case <-time.After(WaitPeriod * 2):
		assert.Fail("account stats update task did not complete after the parent price task reached the target height")
		return
	}

	assert.NoError(err)
	assert.Equal(endHeight, task.LastProcessedHeight())
	rp.AssertCalled(t, "AccountStats", util.ToEpoch(time.Time{}), util.ToEpoch(end), "")
}
