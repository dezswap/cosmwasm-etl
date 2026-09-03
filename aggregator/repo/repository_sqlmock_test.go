package repo

// Repository tests that need no database: sqlmock asserts the statements and the
// transaction boundaries a call produces. Tests that need real postgres belong in
// repository_test.go, behind requireDb.

import (
	"context"
	"database/sql/driver"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	rootdb "github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
	"github.com/stretchr/testify/require"
	gormschema "gorm.io/gorm/schema"
)

func TestWithinTxRollsBackCallbackError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}
	expectedErr := errors.New("transaction failed")
	mock.ExpectBegin()
	mock.ExpectRollback()

	err = repository.WithinTx(context.Background(), func(Repo) error {
		return expectedErr
	})

	require.ErrorIs(t, err, expectedErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

// HoldingPairIds and Accounts used to reach for the raw *sql.DB, which gorm unwraps
// out of the enclosing transaction and which also drops the scoped context. A second
// Begin here, or a bind-parameter mismatch, means one of them escaped again.
func TestRawQueriesJoinEnclosingTransaction(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pattern string
		args    []driver.Value
		rows    *sqlmock.Rows
		call    func(ctx context.Context, r Repo) error
	}{
		{
			name:    "HoldingPairIds",
			pattern: "FROM account_stats_30m",
			args:    []driver.Value{"test-chain", uint64(7)},
			rows:    sqlmock.NewRows([]string{"pair_id"}).AddRow(3).AddRow(9),
			call: func(ctx context.Context, r Repo) error {
				ids, err := r.HoldingPairIds(ctx, 7)
				if err == nil {
					require.Equal(t, []uint64{3, 9}, ids)
				}
				return err
			},
		},
		{
			name:    "Accounts",
			pattern: "FROM account",
			args:    []driver.Value{"test-chain", float64(1200)},
			rows:    sqlmock.NewRows([]string{"id", "address"}).AddRow(1, "addr0"),
			call: func(ctx context.Context, r Repo) error {
				accounts, err := r.Accounts(ctx, 1200)
				if err == nil {
					require.Equal(t, map[uint64]string{1: "addr0"}, accounts)
				}
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer sqlDB.Close()

			gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
			require.NoError(t, err)
			repository := &repoImpl{db: gormDB, chainId: "test-chain"}

			mock.ExpectBegin()
			mock.ExpectQuery(tc.pattern).WithArgs(tc.args...).WillReturnRows(tc.rows)
			mock.ExpectCommit()

			require.NoError(t, repository.WithinTx(context.Background(), func(r Repo) error { return tc.call(context.Background(), r) }))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// CreateAccounts used to reach for the raw *sql.DB, which gorm unwraps out of the
// enclosing transaction. A second Begin here means it escaped again.
func TestCreateAccountsJoinsEnclosingTransaction(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "account"`).
		WithArgs("addr0", "addr1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2))
	mock.ExpectCommit()

	err = repository.WithinTx(context.Background(), func(txRepo Repo) error {
		return txRepo.CreateAccounts(context.Background(), []string{"addr0", "addr1"})
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// pair_stats_recent is shared by every chain pointing at the database, so the prune
// has to be scoped to this repository's chain. Without the chain_id predicate the
// statement below deletes the other chains' rows too.
func TestDeletePairStatsRecentScopesToChain(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}

	deleteBefore := time.Unix(1700000000, 0).UTC()
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "pair_stats_recent" WHERE timestamp < \$1 and chain_id = \$2`).
		WithArgs(util.ToEpoch(deleteBefore), "test-chain").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repository.DeletePairStatsRecent(context.Background(), deleteBefore))
	require.NoError(t, mock.ExpectationsWereMet())
}

// A canceled context must stop DeleteDuplicates before it reaches the database.
func TestDeleteDuplicatesHonorsContext(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = repository.DeleteDuplicates(ctx, time.Unix(0, 0))

	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, mock.ExpectationsWereMet())
}

// DeleteDuplicates spans its own transaction, so callers get all five tables purged
// as one unit without wrapping it. A per-delete Begin/Commit here would mean a
// cancellation could leave only some tables cleaned.
func TestDeleteDuplicatesRunsEveryDeleteInOneTransaction(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}

	mock.ExpectBegin()
	for range 5 {
		mock.ExpectExec("DELETE FROM").WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	require.NoError(t, repository.DeleteDuplicates(context.Background(), time.Unix(0, 0)))
	require.NoError(t, mock.ExpectationsWereMet())
}

// A failed delete must roll the whole cleanup back rather than commit a partial one.
func TestDeleteDuplicatesRollsBackOnFailure(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}
	expectedErr := errors.New("delete failed")

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM").WillReturnError(expectedErr)
	mock.ExpectRollback()

	require.ErrorIs(t, repository.DeleteDuplicates(context.Background(), time.Unix(0, 0)), expectedErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Nesting is still safe: gorm turns the inner transaction into a savepoint instead
// of a second Begin, and leaves releasing it to the enclosing commit.
func TestDeleteDuplicatesNestsInsideAnEnclosingTransaction(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}

	mock.ExpectBegin()
	mock.ExpectExec("SAVEPOINT").WillReturnResult(sqlmock.NewResult(0, 0))
	for range 5 {
		mock.ExpectExec("DELETE FROM").WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	err = repository.WithinTx(context.Background(), func(r Repo) error {
		return r.DeleteDuplicates(context.Background(), time.Unix(0, 0))
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Every writer taking a slice has to split it. A single INSERT would build the whole
// statement in memory and fail outright once a busy window crosses the 65535 limit.
func TestSliceWritersSplitIntoBatches(t *testing.T) {
	for _, tc := range []struct {
		name      string
		statement string
		write     func(ctx context.Context, r Repo, rows int) error
	}{
		{
			name:      "UpdatePairStatsRecent",
			statement: `INSERT INTO "pair_stats_recent"`,
			write: func(ctx context.Context, r Repo, rows int) error {
				return r.UpdatePairStatsRecent(ctx, make([]schemas.PairStatsRecent, rows))
			},
		},
		{
			name:      "UpdateLpHistory",
			statement: `INSERT INTO "lp_history"`,
			write: func(ctx context.Context, r Repo, rows int) error {
				return r.UpdateLpHistory(ctx, make([]schemas.LpHistory, rows))
			},
		},
		{
			name:      "UpdatePairStats",
			statement: `INSERT INTO "pair_stats_30m"`,
			write: func(ctx context.Context, r Repo, rows int) error {
				return r.UpdatePairStats(ctx, zeroAmountPairStats(rows))
			},
		},
		{
			name:      "UpdateAccountStats",
			statement: `INSERT INTO "account_stats_30m"`,
			write: func(ctx context.Context, r Repo, rows int) error {
				return r.UpdateAccountStats(ctx, make([]schemas.AccountStats30m, rows))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer sqlDB.Close()

			gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
			require.NoError(t, err)
			repository := &repoImpl{db: gormDB, chainId: "test-chain"}

			// one row past the batch size is the smallest input that has to split
			mock.ExpectBegin()
			for range 2 {
				mock.ExpectExec(tc.statement).WillReturnResult(sqlmock.NewResult(0, InsertBatchSize))
			}
			mock.ExpectCommit()

			require.NoError(t, tc.write(context.Background(), repository, InsertBatchSize+1))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// zeroAmountPairStats builds rows the writer accepts: every numeric column carries a
// value, so a test exercising something else is not tripped by the amount check.
func zeroAmountPairStats(rows int) []schemas.PairStats30m {
	stats := make([]schemas.PairStats30m, rows)
	for i := range stats {
		stats[i] = schemas.NewPairStat30min("test-chain", "uusd", time.Unix(0, 0).UTC(), uint64(i))
	}

	return stats
}

// An unset amount reaches postgres as an empty string and fails the numeric column with
// a driver error naming neither the pair nor the column, so the writer rejects it first.
func TestUpdatePairStatsRejectsEmptyAmounts(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}

	stats := zeroAmountPairStats(1)
	stats[0].PairId = 713
	stats[0].LastSwapPrice = ""

	err = repository.UpdatePairStats(context.Background(), stats)

	// no statement is expected, so a write would have failed the sqlmock connection and
	// returned a driver error here instead of the validation message
	require.ErrorContains(t, err, "last_swap_price")
	require.ErrorContains(t, err, "713")
	require.ErrorContains(t, err, "test-chain")
}

// A row carrying another chain's id would be written under that id, keyed correctly for
// a chain this repo does not serve, and postgres would take it without complaint.
func TestUpdatePairStatsRejectsForeignChain(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}

	stats := zeroAmountPairStats(1)
	stats[0].PairId = 713
	stats[0].ChainId = "other-chain"

	err = repository.UpdatePairStats(context.Background(), stats)

	require.ErrorContains(t, err, "713")
	require.ErrorContains(t, err, "other-chain")
	require.ErrorContains(t, err, "test-chain")
}

// Two rows on one window collide inside the upsert, which postgres reports without
// naming either row, so the writer rejects them first.
func TestUpdatePairStatsRejectsDuplicateWindows(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}

	stats := zeroAmountPairStats(2)
	stats[1].PairId = stats[0].PairId
	stats[1].Timestamp = stats[0].Timestamp

	err = repository.UpdatePairStats(context.Background(), stats)

	require.ErrorContains(t, err, "duplicate")
	require.ErrorContains(t, err, "test-chain")
}

// A normal run writes one pair across windows and one window across pairs. Neither may
// be mistaken for the collision above.
func TestUpdatePairStatsAcceptsDistinctWindows(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}

	// zeroAmountPairStats numbers the pairs apart on one timestamp, so only the repeated
	// pair needs a window of its own
	stats := zeroAmountPairStats(3)
	stats[2] = schemas.NewPairStat30min("test-chain", "uusd", time.Unix(1800, 0).UTC(), stats[0].PairId)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "pair_stats_30m"`).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	require.NoError(t, repository.UpdatePairStats(context.Background(), stats))
	require.NoError(t, mock.ExpectationsWereMet())
}

// The refreshed columns are read off the model, so a column added to PairStats30m and
// left out of the upsert would keep its first written value on every rerun.
func TestUpdatePairStatsUpsertsEveryColumnOnWindowKey(t *testing.T) {
	// the matcher accepts anything, it is only here to capture the statement gorm built
	var executed string
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(
		sqlmock.QueryMatcherFunc(func(_, actualSQL string) error {
			executed = actualSQL
			return nil
		})))
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}

	mock.ExpectBegin()
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repository.UpdatePairStats(context.Background(), zeroAmountPairStats(1)))
	require.NoError(t, mock.ExpectationsWereMet())

	require.Contains(t, executed, `ON CONFLICT ("chain_id","timestamp","pair_id") DO UPDATE SET`)

	naming := gormschema.NamingStrategy{}
	model := reflect.TypeOf(schemas.PairStats30m{})
	for i := 0; i < model.NumField(); i++ {
		column := naming.ColumnName("", model.Field(i).Name)
		if column == "chain_id" || column == "timestamp" || column == "pair_id" {
			// the conflict target identifies the row, rewriting it would be a no-op
			continue
		}
		require.Contains(t, executed, `"`+column+`"=excluded.`+column,
			"a rerun has to refresh %s", column)
	}

	// not a model field, so the loop above cannot cover it
	require.Contains(t, executed, `"modified_at"=date_part('epoch'::text, now())`)
}

// The validation list is intentionally explicit so it can name the empty column and
// read its value without reflection on every row. Keep that list tied to the model so
// a newly added numeric string cannot silently bypass validation.
func TestPairStatAmountsStayInSyncWithModel(t *testing.T) {
	covered := make(map[string]struct{}, len(pairStatAmounts))
	for _, amount := range pairStatAmounts {
		require.NotContains(t, covered, amount.column, "duplicate pairStatAmounts column")
		covered[amount.column] = struct{}{}
	}

	expected := make(map[string]struct{})
	model := reflect.TypeOf(schemas.PairStats30m{})
	naming := gormschema.NamingStrategy{}
	for i := 0; i < model.NumField(); i++ {
		field := model.Field(i)
		if field.Type.Kind() != reflect.String || field.Name == "ChainId" || field.Name == "PriceToken" {
			continue
		}
		expected[naming.ColumnName("", field.Name)] = struct{}{}
	}

	require.Equal(t, expected, covered)
}

// One batch size serves every writer, so it has to clear the widest model's ceiling.
// Counts come from the models themselves: adding a field narrows that ceiling, and it
// should fail here rather than on a busy window.
func TestBatchSizesStayUnderBindParameterLimit(t *testing.T) {
	const maxBindParams = 65535

	for _, tc := range []struct {
		name  string
		model any
		// fields the writer passes to Omit, which cost no bind parameter
		omitted int
	}{
		{name: "account", model: schemas.Account{}, omitted: 2}, // Id, CreatedAt
		{name: "lp_history", model: schemas.LpHistory{}},
		{name: "pair_stats_recent", model: schemas.PairStatsRecent{}},
		{name: "pair_stats_30m", model: schemas.PairStats30m{}},
		{name: "account_stats_30m", model: schemas.AccountStats30m{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			columnCnt := reflect.TypeOf(tc.model).NumField() - tc.omitted
			require.LessOrEqual(t, InsertBatchSize*columnCnt, maxBindParams)
		})
	}
}

func TestSliceWritersSkipEmptyInput(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}
	ctx := context.Background()

	require.NoError(t, repository.UpdatePairStatsRecent(ctx, nil))
	require.NoError(t, repository.UpdateLpHistory(ctx, nil))
	require.NoError(t, repository.UpdatePairStats(ctx, nil))
	require.NoError(t, repository.UpdateAccountStats(ctx, nil))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateAccountsSkipsEmptyInput(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := rootdb.OpenGormPostgresWithConn(sqlDB)
	require.NoError(t, err)
	repository := &repoImpl{db: gormDB, chainId: "test-chain"}

	require.NoError(t, repository.CreateAccounts(context.Background(), nil))
	require.NoError(t, mock.ExpectationsWereMet())
}
