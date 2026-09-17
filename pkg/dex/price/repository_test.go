package price

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	rootdb "github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := rootdb.OpenGormPostgresWithConn(db)
	require.NoError(t, err)

	return gormDB, mock
}

// A pool the hidden token shares with the price token is priced directly, without any
// route, so retiring its routes does not stop it from being quoted.
func TestTxsExcludesHiddenTokens(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := &srcRepoImpl{db: gormDB, chainId: "test-chain"}

	mock.ExpectQuery(regexp.QuoteMeta(`token_exception`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chain_id", "height", "asset0", "asset1"}).
			AddRow(1, "test-chain", 10, "uusd", "B"))

	txs, err := repo.Txs(context.Background(), 10)

	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// The cursor the next round starts from is max(price.height), and a height whose swaps
// are all hidden writes no price row.
func TestNextHeightExcludesHiddenTokens(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := &srcRepoImpl{db: gormDB, chainId: "test-chain"}

	mock.ExpectQuery(regexp.QuoteMeta(`token_exception`)).
		WillReturnRows(sqlmock.NewRows([]string{"height"}).AddRow(42))

	height, err := repo.NextHeight(context.Background(), 10)

	require.NoError(t, err)
	assert.Equal(t, int64(42), height)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// One query because this runs once per height.
func TestRouteRevisionReadsBothSignalsInOneQuery(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := &srcRepoImpl{db: gormDB, chainId: "test-chain"}

	mock.ExpectQuery(regexp.QuoteMeta(`md5(string_agg(contract`)).
		WithArgs("test-chain", "test-chain").
		WillReturnRows(sqlmock.NewRows([]string{"updated_at", "hidden_digest"}).
			AddRow(100.0, "6f1ed002ab5595859014ebf0951522d9"))

	revision, err := repo.RouteRevision(context.Background())

	require.NoError(t, err)
	assert.Equal(t, RouteRevision{UpdatedAt: 100, HiddenDigest: "6f1ed002ab5595859014ebf0951522d9"}, revision)
	assert.NoError(t, mock.ExpectationsWereMet())
}
