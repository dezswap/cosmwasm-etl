package repo

import (
	"database/sql"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const configName = "config.test"
const chainName = "columbus-5"

var loc = flag.String("type", "", "enable db test")

var (
	testConfig configs.Config

	end = util.ToEpoch(time.Date(2022, 10, 13, 4, 58, 12, 0, time.UTC))

	accounts = []schemas.Account{
		{
			Id:      1,
			Address: "terra0wall1et1",
		},
		{
			Id:      2,
			Address: "terra0wall1et2",
		},
	}

	accountStats = []schemas.HAccountStats30m{
		{
			YearUtc:       2022,
			MonthUtc:      10,
			DayUtc:        13,
			HourUtc:       4,
			MinuteUtc:     30,
			Ts:            1665636627,
			ChainId:       chainName,
			AccountId:     accounts[0].Id,
			PairId:        0,
			TxCnt:         2,
			Asset0Amount:  "38517017",
			Asset1Amount:  "1398850",
			TotalLpAmount: "3627963",
		},
		{
			YearUtc:       2022,
			MonthUtc:      10,
			DayUtc:        13,
			HourUtc:       4,
			MinuteUtc:     30,
			Ts:            1665636627,
			ChainId:       chainName,
			AccountId:     accounts[1].Id,
			PairId:        1,
			TxCnt:         1,
			Asset0Amount:  "13517017",
			Asset1Amount:  "909068",
			TotalLpAmount: "447645",
		},
	}
)

// go test -v -type db
func TestMain(m *testing.M) {
	testConfig = configs.NewWithFileName(configName)

	flag.Parse()
	if *loc == "db" {
		Logger = logging.Discard

		code := m.Run()
		os.Exit(code)
	}
}

func TestDeleteDuplicates(t *testing.T) {
	assert := assert.New(t)

	expectedPairStatsCnt, expectedAccountStatsCnt := 1, 0

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	assert.NoError(err)
	defer db.Close()

	// prepare
	gormDb.Exec(`TRUNCATE TABLE pair_stats_30m`)
	gormDb.Exec(`
INSERT INTO pair_stats_30m (year_utc, month_utc, day_utc, hour_utc, minute_utc, timestamp, chain_id, pair_id, tx_cnt, provider_cnt, asset0_volume, asset1_volume, asset0_liquidity, asset1_liquidity, commission0, commission1)
VALUES (2022, 10, 13, 2, 0, 1665626400, 'columbus-5', 3, 24, 0, '2788291', '1005263', '10000000', '10000000', '9250379', '18874')`)
	gormDb.Exec(`
INSERT INTO pair_stats_30m (year_utc, month_utc, day_utc, hour_utc, minute_utc, timestamp, chain_id, pair_id, tx_cnt, provider_cnt, asset0_volume, asset1_volume, asset0_liquidity, asset1_liquidity, commission0, commission1)
VALUES (2022, 10, 13, 2, 30, 1665628200, 'columbus-5', 3, 7, 0, '5169822', '-380047', '10000000', '10000000', '89759', '5504')`)
	gormDb.Exec(`
INSERT INTO pair_stats_30m (year_utc, month_utc, day_utc, hour_utc, minute_utc, timestamp, chain_id, pair_id, tx_cnt, provider_cnt, asset0_volume, asset1_volume, asset0_liquidity, asset1_liquidity, commission0, commission1)
VALUES (2022, 10, 13, 3, 0, 1665630000, 'columbus-5', 3, 4, 0, '3195129', '-265058', '10000000', '10000000', '14457', '1546')`)

	/*
			gormDb.Exec(`TRUNCATE TABLE h_account_stats_30m`)
			gormDb.Exec(`
		INSERT INTO public.h_account_stats_30m (year_utc, month_utc, day_utc, hour_utc, minute_utc, ts, chain_id, account_id, pair_id, tx_cnt, asset0_amount, asset1_amount, total_lp_amount)
		VALUES (2022, 10, 13, 5, 0, 1665637200, 'columbus-5', 1, 3, 1, 13517017, 909068, 447645)
		`)
			gormDb.Exec(`
		INSERT INTO public.h_account_stats_30m (year_utc, month_utc, day_utc, hour_utc, minute_utc, ts, chain_id, account_id, pair_id, tx_cnt, asset0_amount, asset1_amount, total_lp_amount)
		VALUES (2022, 10, 13, 5, 0, 1665637200, 'columbus-5', 1, 4, 1, 25000000, 489782, 3180318)
		`)
	*/

	// execute
	repo := New(chainName, testConfig.Aggregator.DestDb)
	defer repo.Close()
	err = repo.DeleteDuplicates(util.ToTime(1665628200))

	// verify
	var pairStatsCnt, accountStatsCnt int
	gormDb.Raw("SELECT COUNT(*) FROM pair_stats_30m").Scan(&pairStatsCnt)
	// gormDb.Raw("SELECT COUNT(*) FROM h_account_stats_30m").Scan(&accountStatsCnt)

	assert.NoError(err)
	assert.Equal(expectedPairStatsCnt, pairStatsCnt)
	assert.Equal(expectedAccountStatsCnt, accountStatsCnt)
}

func TestUpdatePairStats(t *testing.T) {
	assert := assert.New(t)

	expected := schemas.NewPairStat30min(chainName, "axpla", util.ToTime(1665626400), 3)
	expected.TxCnt = 24
	expected.ProviderCnt = uint64(0)
	expected.Volume0 = "2788291"
	expected.Volume1 = "1005263"
	expected.Commission0 = "9250379"
	expected.Commission1 = "18874"

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	assert.NoError(err)
	defer db.Close()

	// prepare
	gormDb.Exec(`TRUNCATE TABLE pair_stats_30m`)

	// execute
	repo := New(chainName, testConfig.Aggregator.DestDb)
	defer repo.Close()
	err = repo.UpdatePairStats([]schemas.PairStats30m{expected})

	// verify
	actual := []schemas.PairStats30m{}
	gormDb.Find(&actual)

	assert.NoError(err)
	assert.Len(actual, 1)
	assert.Equal(expected.YearUtc, actual[0].YearUtc)
	assert.Equal(expected.MonthUtc, actual[0].MonthUtc)
	assert.Equal(expected.DayUtc, actual[0].DayUtc)
	assert.Equal(expected.HourUtc, actual[0].HourUtc)
	assert.Equal(expected.MinuteUtc, actual[0].MinuteUtc)
	assert.Equal(expected.PairId, actual[0].PairId)
	assert.Equal(expected.TxCnt, actual[0].TxCnt)
	assert.Equal(expected.ProviderCnt, actual[0].ProviderCnt)
	assert.Equal(expected.Volume0, actual[0].Volume0)
	assert.Equal(expected.Volume1, actual[0].Volume1)
	assert.Equal(expected.Commission0, actual[0].Commission0)
	assert.Equal(expected.Commission1, actual[0].Commission1)
	assert.Equal(expected.PriceToken, actual[0].PriceToken)
	assert.Equal(expected.Timestamp, actual[0].Timestamp)
}

func TestUpdateAccountStats(t *testing.T) {
	assert := assert.New(t)

	expected := schemas.NewUserStat30min(chainName, util.ToTime(1665637200), 3, 1)
	expected.TxCnt = 1
	expected.Asset0Amount = "13517017"
	expected.Asset1Amount = "909068"
	expected.TotalLpAmount = "447645"

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	assert.NoError(err)
	defer db.Close()

	// prepare
	gormDb.Exec(`TRUNCATE TABLE h_account_stats_30m`)

	// execute
	repo := New(chainName, testConfig.Aggregator.DestDb)
	defer repo.Close()
	err = repo.UpdateAccountStats(expected)

	// verify
	actual := []schemas.HAccountStats30m{}
	gormDb.Find(&actual)

	assert.NoError(err)
	assert.Len(actual, 1)
	assert.Equal(expected.YearUtc, actual[0].YearUtc)
	assert.Equal(expected.MonthUtc, actual[0].MonthUtc)
	assert.Equal(expected.DayUtc, actual[0].DayUtc)
	assert.Equal(expected.HourUtc, actual[0].HourUtc)
	assert.Equal(expected.MinuteUtc, actual[0].MinuteUtc)
	assert.Equal(expected.Ts, actual[0].Ts)
	assert.Equal(expected.PairId, actual[0].PairId)
	assert.Equal(expected.TxCnt, actual[0].TxCnt)
	assert.Equal(expected.Asset0Amount, actual[0].Asset0Amount)
	assert.Equal(expected.Asset1Amount, actual[0].Asset1Amount)
	assert.Equal(expected.TotalLpAmount, actual[0].TotalLpAmount)
}

func TestCreateAccounts(t *testing.T) {
	assert := assert.New(t)

	expected := []string{"terra1234", "terra5678", "terra9012"}

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	assert.NoError(err)
	defer db.Close()

	// prepare
	gormDb.Exec(`TRUNCATE TABLE account`)

	// execute
	repo := New(chainName, testConfig.Aggregator.DestDb)
	defer repo.Close()
	err = repo.CreateAccounts(expected)
	assert.NoError(err)

	// verify
	rows, _ := db.Query("SELECT address FROM account ORDER BY id ASC")
	defer rows.Close()

	actual := []string{}
	for rows.Next() {
		var account string

		err = rows.Scan(&account)
		assert.NoError(err)

		actual = append(actual, account)
	}

	assert.EqualValues(expected, actual)
}

func TestHoldingPairIds(t *testing.T) {
	assert := assert.New(t)

	expected := []uint64{1} // the index of `pairs`

	// prepare
	db, gormDb, err := initDb(testConfig.Aggregator.SrcDb)
	assert.NoError(err)
	defer db.Close()

	createTestAccountStats(gormDb)

	repo := New(chainName, testConfig.Aggregator.DestDb)
	defer repo.Close()
	actual, err := repo.HoldingPairIds(accounts[1].Id)

	// verify
	assert.NoError(err)
	assert.EqualValues(expected, actual)
}

func TestAccounts(t *testing.T) {
	assert := assert.New(t)

	expected := make(map[uint64]string)
	expected[accounts[0].Id] = accounts[0].Address
	expected[accounts[1].Id] = accounts[1].Address

	// prepare
	db, gormDb, err := initDb(testConfig.Aggregator.SrcDb)
	assert.NoError(err)
	defer db.Close()

	createTestAccounts(gormDb)
	createTestAccountStats(gormDb)

	// execute
	repo := New(chainName, testConfig.Aggregator.SrcDb)
	defer repo.Close()
	actual, err := repo.Accounts(end)

	// verify
	assert.NoError(err)
	assert.EqualValues(expected, actual)
}

func TestClose(t *testing.T) {
	assert := assert.New(t)

	repo := New(chainName, testConfig.Aggregator.SrcDb)
	err := repo.Close()

	assert.NoError(err)
}

func initDb(config configs.RdbConfig) (*sql.DB, *gorm.DB, error) {
	pq := db.PostgresDb{}
	err := pq.Init(config)
	if err != nil {
		return nil, nil, err
	}

	// verify
	gormDb, _ := gorm.Open(postgres.New(postgres.Config{
		Conn: pq.Db,
	}), &gorm.Config{})

	return pq.Db, gormDb, nil
}

func createTestAccounts(db *gorm.DB) {
	db.Exec(`TRUNCATE TABLE account`)
	db.Omit("CreatedAt").Create(&accounts)
}

func createTestAccountStats(db *gorm.DB) {
	db.Exec(`TRUNCATE TABLE h_account_stats_30m`)

	for _, stats := range accountStats {
		db.Omit("Id", "CreatedAt").Create(&stats)
	}
}
