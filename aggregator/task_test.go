package aggregator

import (
	"context"
	"database/sql"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"gorm.io/gorm"
	"testing"
	"time"

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
	rp.On("PairStats", mock.Anything, mock.Anything, mock.Anything).Return(stats, nil)
	rp.On("LiquiditiesOfPairStats", mock.Anything, mock.Anything, mock.Anything).Return(lpMap, nil)

	task := pairStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		srcDb: &rp,
	}
	err := task.Execute(time.Time{}, end)

	assert.NoError(err)
	assert.Equal(expected, rp.updatedPairStats[0])
}

func TestTimeframe_0_30(t *testing.T) {
	assert := assert.New(t)

	rp := repoMock{}
	task := pairStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
		},
		srcDb: &rp,
	}

	expectedStart := time.Date(2022, 10, 25, 7, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2022, 10, 25, 7, 30, 0, 0, time.UTC)

	ts := time.Unix(1666681350, 0).UTC() // 2022-10-25 07:02:30 UTC
	actualStart, actualEnd := task.timeframe(ts)

	assert.Equal(expectedStart, actualStart)
	assert.Equal(expectedEnd, actualEnd)
}

func TestTimeframe_30_0(t *testing.T) {
	assert := assert.New(t)

	rp := repoMock{}
	task := pairStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		srcDb: &rp,
	}

	expectedStart := time.Date(2022, 10, 25, 7, 30, 0, 0, time.UTC)
	expectedEnd := time.Date(2022, 10, 25, 8, 0, 0, 0, time.UTC)

	ts := time.Unix(1666684022, 0).UTC() // 2022-10-25 07:47:02 UTC
	actualStart, actualEnd := task.timeframe(ts)

	assert.Equal(expectedStart, actualStart)
	assert.Equal(expectedEnd, actualEnd)
}

func TestTimeframe_InclusiveStart(t *testing.T) {
	assert := assert.New(t)

	rp := repoMock{}
	task := pairStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		srcDb: &rp,
	}

	expectedStart := time.Date(2022, 10, 25, 7, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2022, 10, 25, 7, 30, 0, 0, time.UTC)

	ts := time.Unix(1666681200, 0).UTC() // 2022-10-25 07:00:00 UTC
	actualStart, actualEnd := task.timeframe(ts)

	assert.Equal(expectedStart, actualStart)
	assert.Equal(expectedEnd, actualEnd)
}

func TestTimeframe_ExclusiveEnd(t *testing.T) {
	assert := assert.New(t)

	rp := repoMock{}
	task := pairStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		srcDb: &rp,
	}

	expectedStart := time.Date(2022, 10, 25, 7, 30, 0, 0, time.UTC)
	expectedEnd := time.Date(2022, 10, 25, 8, 0, 0, 0, time.UTC)

	ts := time.Unix(1666683000, 0).UTC() // 2022-10-25 07:30:00 UTC
	actualStart, actualEnd := task.timeframe(ts)

	assert.Equal(expectedStart, actualStart)
	assert.Equal(expectedEnd, actualEnd)
}

func TestExecuteAccountStatsUpdateTask(t *testing.T) {
	assert := assert.New(t)

	end := time.Unix(1666765800, 0).UTC() // 2022-10-26 06:30:00 UTC
	accountId, pairId := uint64(1), uint64(1)
	txCnt := uint64(5)
	asset0, asset1, lp := "1000000", "2000000", "1000000"

	expected := schemas.HAccountStats30m{
		YearUtc:       end.Year(),
		MonthUtc:      int(end.Month()),
		DayUtc:        end.Day(),
		HourUtc:       end.Hour(),
		MinuteUtc:     end.Minute(),
		Ts:            util.ToEpoch(end),
		AccountId:     accountId,
		PairId:        pairId,
		TxCnt:         txCnt,
		Asset0Amount:  asset0,
		Asset1Amount:  asset1,
		TotalLpAmount: lp,
	}

	accounts := make(map[uint64]string)
	accounts[accountId] = "terra0wal1let2"

	rp := repoMock{}
	rp.On("NewAccounts", mock.Anything, mock.Anything).Return([]string{}, nil)
	rp.On("Accounts", mock.Anything).Return(accounts, nil)
	rp.On("HoldingPairIds", mock.Anything).Return([]uint64{}, nil)
	rp.On("NewPairIds", mock.Anything, mock.Anything, mock.Anything).Return([]uint64{pairId}, nil)
	rp.On("TxCountOfAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(txCnt, nil)
	rp.On("AssetAmountInPairOfAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(asset0, asset1, lp, nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		srcDb: &rp,
	}
	err := task.Execute(time.Time{}, end)

	assert.NoError(err)
	assert.Equal(expected, rp.updatedAccountStats)
}

func TestUpdateAccounts(t *testing.T) {
	assert := assert.New(t)

	expected := []string{"terra1234", "terra5678", "terra9012"}

	rp := repoMock{}
	rp.On("NewAccounts", mock.Anything, mock.Anything).Return(expected, nil)

	task := accountStatsUpdateTask{
		taskImpl: taskImpl{
			chainId: "",
			destDb:  &rp,
			logger:  logging.Discard,
		},
		srcDb: &rp,
	}
	err := task.updateAccounts(1666731600, 1666765800)

	assert.NoError(err)
	assert.Equal(expected, rp.updatedAccounts)
}
