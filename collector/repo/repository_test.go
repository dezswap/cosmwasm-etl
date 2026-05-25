package repo

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newMockRepo(t *testing.T) (*repository, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	return NewWithDB(gormDB).(*repository), mock
}

func TestGetSyncedHeightNotFound(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "collector_synced_heights"`)).
		WillReturnRows(sqlmock.NewRows([]string{"chain_id", "height", "created_at", "updated_at"}))

	_, err := repo.GetSyncedHeight("phoenix-1")

	require.ErrorIs(t, err, ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSyncedHeightUnavailable(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "collector_synced_heights"`)).
		WillReturnError(&pq.Error{Code: "42P01", Message: "relation does not exist"})

	_, err := repo.GetSyncedHeight("phoenix-1")

	require.ErrorIs(t, err, ErrUnavailable)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSyncedHeightReturnsHardError(t *testing.T) {
	repo, mock := newMockRepo(t)
	expected := errors.New("database failed")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "collector_synced_heights"`)).
		WillReturnError(expected)

	_, err := repo.GetSyncedHeight("phoenix-1")

	require.ErrorIs(t, err, expected)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockTxs(t *testing.T) {
	repo, mock := newMockRepo(t)
	ts := time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"chain_id", "height", "block_time", "txs", "created_at", "updated_at"}).
		AddRow("phoenix-1", 10, ts, schemas.CollectorJSON(`[{"hash":"hash","sender":"sender","timestamp":"2026-05-19T01:02:03Z","logResults":[]}]`), ts, ts)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "collector_blocks"`)).WillReturnRows(rows)

	txs, blockTime, err := repo.GetBlockTxs("phoenix-1", 10)

	require.NoError(t, err)
	require.Equal(t, ts, blockTime)
	require.Equal(t, parser.RawTxs{{Hash: "hash", Sender: "sender", Timestamp: ts, LogResults: eventlog.LogResults{}}}, txs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockTxsNotFound(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "collector_blocks"`)).
		WillReturnRows(sqlmock.NewRows([]string{"chain_id", "height", "block_time", "txs", "created_at", "updated_at"}))

	_, _, err := repo.GetBlockTxs("phoenix-1", 10)

	require.ErrorIs(t, err, ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockTxsMalformedJSONDoesNotMapToFallbackError(t *testing.T) {
	repo, mock := newMockRepo(t)
	ts := time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"chain_id", "height", "block_time", "txs", "created_at", "updated_at"}).
		AddRow("phoenix-1", 10, ts, schemas.CollectorJSON(`{bad json`), ts, ts)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "collector_blocks"`)).WillReturnRows(rows)

	_, _, err := repo.GetBlockTxs("phoenix-1", 10)

	require.Error(t, err)
	require.False(t, errors.Is(err, ErrNotFound))
	require.False(t, errors.Is(err, ErrUnavailable))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPoolInfos(t *testing.T) {
	repo, mock := newMockRepo(t)
	ts := time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"chain_id", "height", "pool_infos", "created_at", "updated_at"}).
		AddRow("phoenix-1", 10, schemas.CollectorJSON(`[{"contractAddr":"pair","assets":[{"addr":"asset0","amount":"1"}],"lpAddr":"lp","totalShare":"2"}]`), ts, ts)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "collector_pool_snapshots"`)).WillReturnRows(rows)

	poolInfos, err := repo.GetPoolInfos("phoenix-1", 10)

	require.NoError(t, err)
	require.Equal(t, []dex.PoolInfo{{
		ContractAddr: "pair",
		Assets:       []dex.Asset{{Addr: "asset0", Amount: "1"}},
		LpAddr:       "lp",
		TotalShare:   "2",
	}}, poolInfos)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPoolInfosMalformedJSONDoesNotMapToFallbackError(t *testing.T) {
	repo, mock := newMockRepo(t)
	ts := time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"chain_id", "height", "pool_infos", "created_at", "updated_at"}).
		AddRow("phoenix-1", 10, schemas.CollectorJSON(`{bad json`), ts, ts)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "collector_pool_snapshots"`)).WillReturnRows(rows)

	_, err := repo.GetPoolInfos("phoenix-1", 10)

	require.Error(t, err)
	require.False(t, errors.Is(err, ErrNotFound))
	require.False(t, errors.Is(err, ErrUnavailable))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPoolInfosUnavailable(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "collector_pool_snapshots"`)).
		WillReturnError(&pq.Error{Code: "42P01", Message: "relation does not exist"})

	_, err := repo.GetPoolInfos("phoenix-1", 10)

	require.ErrorIs(t, err, ErrUnavailable)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveHeightUpsertsBlockPoolAndSyncedHeight(t *testing.T) {
	repo, mock := newMockRepo(t)
	ts := time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)

	mock.ExpectBegin()
	expectUpsert(mock, `collector_blocks`)
	expectUpsert(mock, `collector_pool_snapshots`)
	expectSyncedHeightInsert(mock)
	mock.ExpectCommit()

	err := repo.SaveHeight(
		"phoenix-1",
		10,
		ts,
		parser.RawTxs{{Hash: "hash", Timestamp: ts}},
		[]dex.PoolInfo{{ContractAddr: "pair"}},
		true,
	)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveHeightWithoutPoolSnapshotSkipsPoolUpsert(t *testing.T) {
	repo, mock := newMockRepo(t)
	ts := time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)

	mock.ExpectBegin()
	expectUpsert(mock, `collector_blocks`)
	expectSyncedHeightInsert(mock)
	mock.ExpectCommit()

	err := repo.SaveHeight(
		"phoenix-1",
		10,
		ts,
		parser.RawTxs{{Hash: "hash", Timestamp: ts}},
		nil,
		false,
	)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveHeightRollsBackWhenBlockUpsertFails(t *testing.T) {
	repo, mock := newMockRepo(t)
	expected := errors.New("block insert failed")

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "collector_blocks"`)).
		WillReturnError(expected)
	mock.ExpectRollback()

	err := repo.SaveHeight("phoenix-1", 10, time.Now(), parser.RawTxs{}, nil, false)

	require.ErrorIs(t, err, expected)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveHeightRollsBackWhenPoolUpsertFails(t *testing.T) {
	repo, mock := newMockRepo(t)
	expected := errors.New("pool insert failed")

	mock.ExpectBegin()
	expectUpsert(mock, `collector_blocks`)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "collector_pool_snapshots"`)).
		WillReturnError(expected)
	mock.ExpectRollback()

	err := repo.SaveHeight("phoenix-1", 10, time.Now(), parser.RawTxs{}, []dex.PoolInfo{}, true)

	require.ErrorIs(t, err, expected)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveHeightRollsBackWhenSyncedHeightUpsertFails(t *testing.T) {
	repo, mock := newMockRepo(t)
	expected := errors.New("synced insert failed")

	mock.ExpectBegin()
	expectUpsert(mock, `collector_blocks`)
	mock.ExpectExec(syncedHeightInsertPattern()).
		WillReturnError(expected)
	mock.ExpectRollback()

	err := repo.SaveHeight("phoenix-1", 10, time.Now(), parser.RawTxs{}, nil, false)

	require.ErrorIs(t, err, expected)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUndefinedTableHandlesDriverAndSQLStateErrors(t *testing.T) {
	require.True(t, isUndefinedTable(&pq.Error{Code: "42P01"}))
	require.True(t, isUndefinedTable(errors.New(`relation "collector_blocks" does not exist (SQLSTATE 42P01)`)))
	require.False(t, isUndefinedTable(errors.New("connection refused")))
}

func TestIsUndefinedTableRejectsUnrelatedDoesNotExistErrors(t *testing.T) {
	require.False(t, isUndefinedTable(errors.New(`database "collector" does not exist`)))
	require.False(t, isUndefinedTable(errors.New(`role "etl" does not exist`)))
}

func expectUpsert(mock sqlmock.Sqlmock, table string) {
	mock.ExpectExec(
		regexp.QuoteMeta(
			`INSERT INTO "`+table+`"`) +
			`.*` + regexp.QuoteMeta(`ON CONFLICT`),
	).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func expectSyncedHeightInsert(mock sqlmock.Sqlmock) {
	mock.ExpectExec(syncedHeightInsertPattern()).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func syncedHeightInsertPattern() string {
	return regexp.QuoteMeta(`INSERT INTO "collector_synced_heights"`) +
		`.*` +
		regexp.QuoteMeta(`GREATEST(collector_synced_heights.height, EXCLUDED.height)`)
}
