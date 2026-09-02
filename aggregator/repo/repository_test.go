package repo

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	accountStats = []schemas.AccountStats30m{
		{
			YearUtc:              2022,
			MonthUtc:             10,
			DayUtc:               13,
			HourUtc:              4,
			MinuteUtc:            30,
			Timestamp:            1665636627,
			ChainId:              chainName,
			AccountId:            accounts[0].Id,
			Address:              accounts[0].Address,
			PairId:               0,
			TxCnt:                2,
			SwapTxCnt:            1,
			ProvideTxCnt:         1,
			SwapVolumeInPrice:    "10",
			ProvideValueInPrice:  "20",
			WithdrawValueInPrice: "0",
			NetFlowInPrice:       "30",
			PriceToken:           "uusd",
			NetAsset0Amount:      "100",
			NetAsset1Amount:      "200",
			NetLpAmount:          "300",
		},
		{
			YearUtc:              2022,
			MonthUtc:             10,
			DayUtc:               13,
			HourUtc:              4,
			MinuteUtc:            30,
			Timestamp:            1665636627,
			ChainId:              chainName,
			AccountId:            accounts[1].Id,
			Address:              accounts[1].Address,
			PairId:               1,
			TxCnt:                1,
			ProvideTxCnt:         1,
			SwapVolumeInPrice:    "0",
			ProvideValueInPrice:  "50",
			WithdrawValueInPrice: "0",
			NetFlowInPrice:       "50",
			PriceToken:           "uusd",
			NetAsset0Amount:      "1000",
			NetAsset1Amount:      "2000",
			NetLpAmount:          "400",
		},
	}
)

// go test -v -type db
func TestMain(m *testing.M) {
	testConfig = configs.NewWithFileName(configName)

	flag.Parse()
	os.Exit(m.Run())
}

// requireDb skips tests that need a live postgres. TestMain used to gate the whole
// package on the flag instead, which silently skipped the sqlmock-based tests too.
func requireDb(t *testing.T) {
	t.Helper()
	if *loc != "db" {
		t.Skip("needs a database; run with -type db")
	}
}

// mustNewRepo opens a repository and closes its pool when the test ends. Callers
// must not close it themselves; sql.DB.Close is idempotent, so the one test that
// exercises Close directly is still safe.
func mustNewRepo(t *testing.T, config configs.RdbConfig) Repo {
	t.Helper()
	repository, err := New(chainName, config)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, repository.Close()) })

	return repository
}

// The 30 minute tables are purged by the bucket the timestamp falls in, not by the
// timestamp itself, so a rerun starting mid bucket cannot leave half of one behind.
func TestDeleteDuplicates(t *testing.T) {
	requireDb(t)
	ctx := context.Background()

	assert := assert.New(t)

	// 02:00, 02:30 and 03:00 UTC. Deleting from 02:30 clears the bucket ending at 03:00,
	// which is the last row alone.
	timestamps := []float64{1665626400, 1665628200, 1665630000}
	deleteFrom := util.ToTime(1665628200)
	// asserting which rows survive, not how many: a purge of the wrong end of the range
	// leaves the same count behind
	expectedRemaining := []float64{1665626400, 1665628200}

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	assert.NoError(err)
	defer db.Close()

	// prepare: rows go in through the models, so a schema change fails the insert here
	// instead of silently leaving the tables empty
	gormDb.Exec(`TRUNCATE TABLE pair_stats_30m, account_stats_30m`)
	for _, ts := range timestamps {
		pairStat := schemas.NewPairStat30min(chainName, "axpla", util.ToTime(ts), 3)
		require.NoError(t, gormDb.Create(&pairStat).Error)

		accountStat := schemas.NewAccountStat30min(chainName, util.ToTime(ts), 3, 1, "terra0wal1let2")
		require.NoError(t, gormDb.Create(&accountStat).Error)
	}

	// execute
	repo := mustNewRepo(t, testConfig.Aggregator.DestDb)
	err = repo.DeleteDuplicates(ctx, deleteFrom)
	assert.NoError(err)

	// verify
	var pairStatsTimestamps, accountStatsTimestamps []float64
	gormDb.Raw("SELECT timestamp FROM pair_stats_30m ORDER BY timestamp").Scan(&pairStatsTimestamps)
	gormDb.Raw("SELECT timestamp FROM account_stats_30m ORDER BY timestamp").Scan(&accountStatsTimestamps)

	assert.Equal(expectedRemaining, pairStatsTimestamps)
	assert.Equal(expectedRemaining, accountStatsTimestamps)
}

func TestLatestPairStat(t *testing.T) {
	requireDb(t)
	ctx := context.Background()

	assert := assert.New(t)

	pairId, otherPairId := uint64(3), uint64(4)

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	assert.NoError(err)
	defer db.Close()

	// prepare
	gormDb.Exec(`TRUNCATE TABLE pair_stats_30m`)
	// written out of order, so a query relying on insertion order picks the wrong row
	for _, row := range []struct {
		ts         float64
		liquidity0 string
	}{
		{1665626400, "100"},
		{1665630000, "300"},
		{1665628200, "200"},
		// a rerun leaves a second row on the newest timestamp: nothing but the id tells
		// the two apart, and the later one is the one to carry over
		{1665630000, "301"},
	} {
		stat := schemas.NewPairStat30min(chainName, "axpla", util.ToTime(row.ts), pairId)
		stat.Liquidity0 = row.liquidity0
		require.NoError(t, gormDb.Create(&stat).Error)
	}
	// a newer row of another chain sharing the pair id must not win
	otherChainStat := schemas.NewPairStat30min("other-chain", "axpla", util.ToTime(1665633600), pairId)
	require.NoError(t, gormDb.Create(&otherChainStat).Error)

	// execute
	repo := mustNewRepo(t, testConfig.Aggregator.DestDb)
	latest, found, err := repo.LatestPairStat(ctx, pairId)
	_, missing, missingErr := repo.LatestPairStat(ctx, otherPairId)

	// verify
	assert.NoError(err)
	assert.True(found)
	// the newest row of this chain, whatever order they were written in
	assert.Equal(float64(1665630000), latest.Timestamp)
	assert.Equal("301", latest.Liquidity0)

	assert.NoError(missingErr)
	assert.False(missing, "a pair without stats must be reported as missing, not as a zero row")
}

func TestUpdatePairStats(t *testing.T) {
	requireDb(t)
	ctx := context.Background()

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
	repo := mustNewRepo(t, testConfig.Aggregator.DestDb)
	err = repo.UpdatePairStats(ctx, []schemas.PairStats30m{expected})

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
	requireDb(t)
	ctx := context.Background()

	assert := assert.New(t)

	expected := schemas.NewAccountStat30min(chainName, util.ToTime(1665637200), 3, 1, "xplaaabb")
	expected.TxCnt = 1
	expected.SwapTxCnt = 1
	expected.SwapVolumeInPrice = "100"
	expected.PriceToken = "uusd"
	expected.NetAsset0Amount = "-1000000"
	expected.NetAsset1Amount = "2000000"

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	assert.NoError(err)
	defer db.Close()

	// prepare
	gormDb.Exec(`TRUNCATE TABLE account_stats_30m`)

	// execute
	repo := mustNewRepo(t, testConfig.Aggregator.DestDb)
	err = repo.UpdateAccountStats(ctx, []schemas.AccountStats30m{expected})

	// verify
	actual := []schemas.AccountStats30m{}
	gormDb.Find(&actual)

	assert.NoError(err)
	assert.Len(actual, 1)
	assert.Equal(expected.YearUtc, actual[0].YearUtc)
	assert.Equal(expected.MonthUtc, actual[0].MonthUtc)
	assert.Equal(expected.DayUtc, actual[0].DayUtc)
	assert.Equal(expected.HourUtc, actual[0].HourUtc)
	assert.Equal(expected.MinuteUtc, actual[0].MinuteUtc)
	assert.Equal(expected.Timestamp, actual[0].Timestamp)
	assert.Equal(expected.AccountId, actual[0].AccountId)
	assert.Equal(expected.Address, actual[0].Address)
	assert.Equal(expected.PairId, actual[0].PairId)
	assert.Equal(expected.TxCnt, actual[0].TxCnt)
	assert.Equal(expected.SwapTxCnt, actual[0].SwapTxCnt)
	assert.Equal(expected.SwapVolumeInPrice, actual[0].SwapVolumeInPrice)
	assert.Equal(expected.PriceToken, actual[0].PriceToken)
	assert.Equal(expected.NetAsset0Amount, actual[0].NetAsset0Amount)
	assert.Equal(expected.NetAsset1Amount, actual[0].NetAsset1Amount)
}

func TestUpdateAccountStatsUpsertsOnAccountIdPairTimestamp(t *testing.T) {
	requireDb(t)
	ctx := context.Background()

	assert := assert.New(t)
	require := require.New(t)

	ts := util.ToTime(1665637200)
	initial := schemas.NewAccountStat30min(chainName, ts, 3, 1, "terra0old")
	initial.TxCnt = 1
	initial.SwapTxCnt = 1
	initial.SwapVolumeInPrice = "100"
	initial.PriceToken = "uusd"
	initial.NetAsset0Amount = "10"
	initial.NetAsset1Amount = "20"
	initial.NetLpAmount = "30"

	updated := schemas.NewAccountStat30min(chainName, ts, 3, 1, "terra0new")
	updated.TxCnt = 7
	updated.SwapTxCnt = 2
	updated.ProvideTxCnt = 3
	updated.WithdrawTxCnt = 2
	updated.SwapVolumeInPrice = "200"
	updated.ProvideValueInPrice = "300"
	updated.WithdrawValueInPrice = "400"
	updated.NetFlowInPrice = "500"
	updated.PriceToken = "uusd"
	updated.NetAsset0Amount = "-60"
	updated.NetAsset1Amount = "70"
	updated.NetLpAmount = "-80"

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	require.NoError(err)
	defer db.Close()
	require.NoError(gormDb.Exec(`TRUNCATE TABLE account_stats_30m`).Error)

	repo := mustNewRepo(t, testConfig.Aggregator.DestDb)
	require.NoError(repo.UpdateAccountStats(ctx, []schemas.AccountStats30m{initial}))
	require.NoError(repo.UpdateAccountStats(ctx, []schemas.AccountStats30m{updated}))

	var count int64
	require.NoError(gormDb.Model(&schemas.AccountStats30m{}).Count(&count).Error)
	assert.EqualValues(1, count)

	var actual schemas.AccountStats30m
	require.NoError(gormDb.First(&actual).Error)
	assert.Equal(updated.Address, actual.Address)
	assert.Equal(updated.TxCnt, actual.TxCnt)
	assert.Equal(updated.SwapTxCnt, actual.SwapTxCnt)
	assert.Equal(updated.ProvideTxCnt, actual.ProvideTxCnt)
	assert.Equal(updated.WithdrawTxCnt, actual.WithdrawTxCnt)
	assert.Equal(updated.SwapVolumeInPrice, actual.SwapVolumeInPrice)
	assert.Equal(updated.ProvideValueInPrice, actual.ProvideValueInPrice)
	assert.Equal(updated.WithdrawValueInPrice, actual.WithdrawValueInPrice)
	assert.Equal(updated.NetFlowInPrice, actual.NetFlowInPrice)
	assert.Equal(updated.NetAsset0Amount, actual.NetAsset0Amount)
	assert.Equal(updated.NetAsset1Amount, actual.NetAsset1Amount)
	assert.Equal(updated.NetLpAmount, actual.NetLpAmount)
}

func TestUpdateAccountStatsNoopOnEmptyInput(t *testing.T) {
	requireDb(t)
	ctx := context.Background()

	assert := assert.New(t)
	require := require.New(t)

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	require.NoError(err)
	defer db.Close()
	require.NoError(gormDb.Exec(`TRUNCATE TABLE account_stats_30m`).Error)

	existing := schemas.NewAccountStat30min(chainName, util.ToTime(1665637200), 3, 1, "terra0existing")
	existing.TxCnt = 1
	require.NoError(gormDb.Omit("Id", "CreatedAt").Create(&existing).Error)

	repo := mustNewRepo(t, testConfig.Aggregator.DestDb)
	err = repo.UpdateAccountStats(ctx, []schemas.AccountStats30m{})

	var count int64
	require.NoError(gormDb.Model(&schemas.AccountStats30m{}).Count(&count).Error)
	assert.NoError(err)
	assert.EqualValues(1, count)
}

func TestCreateAccounts(t *testing.T) {
	requireDb(t)
	ctx := context.Background()

	assert := assert.New(t)

	expected := []string{"terra1234", "terra5678", "terra9012"}

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	assert.NoError(err)
	defer db.Close()

	// prepare
	gormDb.Exec(`TRUNCATE TABLE account`)

	// execute
	repo := mustNewRepo(t, testConfig.Aggregator.DestDb)
	err = repo.CreateAccounts(ctx, expected)
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

func TestAccountIdsReturnsOnlyExistingAccounts(t *testing.T) {
	requireDb(t)
	ctx := context.Background()

	assert := assert.New(t)
	require := require.New(t)

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	require.NoError(err)
	defer db.Close()

	createTestAccounts(gormDb)

	repo := mustNewRepo(t, testConfig.Aggregator.DestDb)
	actual, err := repo.AccountIds(ctx, []string{accounts[0].Address, "terra0missing"})

	assert.NoError(err)
	assert.Equal(map[string]uint64{accounts[0].Address: accounts[0].Id}, actual)

	empty, err := repo.AccountIds(ctx, []string{})
	assert.NoError(err)
	assert.Empty(empty)
}

func TestHoldingPairIds(t *testing.T) {
	requireDb(t)
	ctx := context.Background()

	assert := assert.New(t)

	expected := []uint64{1} // the index of `pairs`

	// prepare
	db, gormDb, err := initDb(testConfig.Aggregator.SrcDb)
	assert.NoError(err)
	defer db.Close()

	createTestAccountStats(gormDb)

	repo := mustNewRepo(t, testConfig.Aggregator.DestDb)
	actual, err := repo.HoldingPairIds(ctx, accounts[1].Id)

	// verify
	assert.NoError(err)
	assert.EqualValues(expected, actual)
}

func TestHoldingPairIdsExcludesZeroOrNegativeNetLp(t *testing.T) {
	requireDb(t)
	ctx := context.Background()

	assert := assert.New(t)
	require := require.New(t)

	db, gormDb, err := initDb(testConfig.Aggregator.SrcDb)
	require.NoError(err)
	defer db.Close()
	require.NoError(gormDb.Exec(`TRUNCATE TABLE account_stats_30m`).Error)

	stats := []schemas.AccountStats30m{
		schemas.NewAccountStat30min(chainName, util.ToTime(1665637200), 1, accounts[0].Id, accounts[0].Address),
		schemas.NewAccountStat30min(chainName, util.ToTime(1665637200), 2, accounts[0].Id, accounts[0].Address),
		schemas.NewAccountStat30min(chainName, util.ToTime(1665637200), 3, accounts[0].Id, accounts[0].Address),
	}
	stats[0].NetLpAmount = "10"
	stats[1].NetLpAmount = "0"
	stats[2].NetLpAmount = "-5"
	require.NoError(gormDb.Omit("Id", "CreatedAt").Create(&stats).Error)

	repo := mustNewRepo(t, testConfig.Aggregator.SrcDb)
	actual, err := repo.HoldingPairIds(ctx, accounts[0].Id)

	assert.NoError(err)
	assert.EqualValues([]uint64{1}, actual)
}

func TestAccounts(t *testing.T) {
	requireDb(t)
	ctx := context.Background()

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
	repo := mustNewRepo(t, testConfig.Aggregator.SrcDb)
	actual, err := repo.Accounts(ctx, end)

	// verify
	assert.NoError(err)
	assert.EqualValues(expected, actual)
}

func TestAccountsIncludesPositiveLpAccountsAndRecentlyCreatedAccounts(t *testing.T) {
	requireDb(t)
	ctx := context.Background()

	assert := assert.New(t)
	require := require.New(t)

	endTs := float64(2000)
	oldCreatedAt := float64(1000)
	recentCreatedAt := float64(2000)
	testAccounts := []schemas.Account{
		{Id: 11, Address: "terra0lp", CreatedAt: oldCreatedAt},
		{Id: 12, Address: "terra0recent", CreatedAt: recentCreatedAt},
		{Id: 13, Address: "terra0old", CreatedAt: oldCreatedAt},
	}

	db, gormDb, err := initDb(testConfig.Aggregator.SrcDb)
	require.NoError(err)
	defer db.Close()
	require.NoError(gormDb.Exec(`TRUNCATE TABLE account, account_stats_30m`).Error)
	require.NoError(gormDb.Create(&testAccounts).Error)

	stats := []schemas.AccountStats30m{
		schemas.NewAccountStat30min(chainName, util.ToTime(1665637200), 1, testAccounts[0].Id, testAccounts[0].Address),
		schemas.NewAccountStat30min(chainName, util.ToTime(1665637200), 2, testAccounts[2].Id, testAccounts[2].Address),
	}
	stats[0].NetLpAmount = "10"
	stats[1].NetLpAmount = "0"
	require.NoError(gormDb.Omit("Id", "CreatedAt").Create(&stats).Error)

	repo := mustNewRepo(t, testConfig.Aggregator.SrcDb)
	actual, err := repo.Accounts(ctx, endTs)

	assert.NoError(err)
	assert.Equal(map[uint64]string{
		testAccounts[0].Id: testAccounts[0].Address,
		testAccounts[1].Id: testAccounts[1].Address,
	}, actual)
}

func TestAccountStats30mMigrationDefaults(t *testing.T) {
	requireDb(t)

	assert := assert.New(t)
	require := require.New(t)

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	require.NoError(err)
	defer db.Close()
	require.NoError(gormDb.Exec(`TRUNCATE TABLE account_stats_30m`).Error)
	require.NoError(gormDb.Exec(`
INSERT INTO account_stats_30m (
    year_utc, month_utc, day_utc, hour_utc, minute_utc,
    address, account_id, pair_id, chain_id, tx_cnt, timestamp
) VALUES (
    2022, 10, 13, 4, 30,
    'terra0defaults', 1, 1, $1, 1, 1665636627
)`, chainName).Error)

	var actual schemas.AccountStats30m
	require.NoError(gormDb.First(&actual).Error)
	assert.Equal(uint64(0), actual.SwapTxCnt)
	assert.Equal(uint64(0), actual.ProvideTxCnt)
	assert.Equal(uint64(0), actual.WithdrawTxCnt)
	assert.Equal("0", actual.SwapVolumeInPrice)
	assert.Equal("0", actual.ProvideValueInPrice)
	assert.Equal("0", actual.WithdrawValueInPrice)
	assert.Equal("0", actual.NetFlowInPrice)
	assert.Equal("", actual.PriceToken)
	assert.Equal("0", actual.NetAsset0Amount)
	assert.Equal("0", actual.NetAsset1Amount)
	assert.Equal("0", actual.NetLpAmount)
}

func TestAccountSchemaDoesNotIncludeProfileColumns(t *testing.T) {
	requireDb(t)

	assert := assert.New(t)
	require := require.New(t)

	db, gormDb, err := initDb(testConfig.Aggregator.DestDb)
	require.NoError(err)
	defer db.Close()

	var count int64
	require.NoError(gormDb.Raw(`
SELECT count(*)
FROM information_schema.columns
WHERE table_name = 'account'
  AND column_name IN ('first_seen_at', 'last_seen_at', 'touched_pair_cnt')
`).Scan(&count).Error)
	assert.EqualValues(0, count)
}

func TestClose(t *testing.T) {
	requireDb(t)

	assert := assert.New(t)

	repository := mustNewRepo(t, testConfig.Aggregator.SrcDb)
	err := repository.Close()

	assert.NoError(err)
}

func initDb(config configs.RdbConfig) (*sql.DB, *gorm.DB, error) {
	pq := db.PostgresDb{}
	err := pq.Init(config)
	if err != nil {
		return nil, nil, err
	}

	// verify
	gormDb, err := db.OpenGormPostgresWithConn(pq.Db)
	if err != nil {
		return nil, nil, err
	}

	return pq.Db, gormDb, nil
}

func createTestAccounts(db *gorm.DB) {
	db.Exec(`TRUNCATE TABLE account`)
	db.Omit("CreatedAt").Create(&accounts)
}

func createTestAccountStats(db *gorm.DB) {
	db.Exec(`TRUNCATE TABLE account_stats_30m`)

	for _, stats := range accountStats {
		db.Omit("Id", "CreatedAt").Create(&stats)
	}
}
