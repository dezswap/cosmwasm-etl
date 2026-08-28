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
	"github.com/stretchr/testify/require"
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
				return r.UpdatePairStats(ctx, make([]schemas.PairStats30m, rows))
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
