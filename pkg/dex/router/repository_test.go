package router

import (
	"context"
	"errors"
	"fmt"
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

func TestUpdateRoutes_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := &srcRepoImpl{db: gormDB, chainId: "test-chain"}

	indexToAsset := map[int]string{
		0: "asset0",
		1: "asset1",
		2: "asset2",
	}
	routesMap := map[int]map[int][][]int{
		0: {
			1: {{0, 1}},
			2: {{0, 1, 2}},
		},
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "route"`)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	err := repo.UpdateRoutes(context.Background(), indexToAsset, routesMap)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateRoutes_EmptyRoutes(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := &srcRepoImpl{db: gormDB, chainId: "test-chain"}

	indexToAsset := map[int]string{}
	routesMap := map[int]map[int][][]int{}

	err := repo.UpdateRoutes(context.Background(), indexToAsset, routesMap)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateRoutes_DBError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := &srcRepoImpl{db: gormDB, chainId: "test-chain"}

	indexToAsset := map[int]string{0: "asset0", 1: "asset1"}
	routesMap := map[int]map[int][][]int{
		0: {1: {{0, 1}}},
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "route"`)).
		WillReturnError(errors.New("db error"))
	mock.ExpectRollback()

	err := repo.UpdateRoutes(context.Background(), indexToAsset, routesMap)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repo.UpdateRoutes")
}

func TestUpdateRoutes_DataTransformation(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := &srcRepoImpl{db: gormDB, chainId: "test-chain"}

	indexToAsset := map[int]string{
		0: "terra1abc",
		1: "terra1def",
		2: "terra1ghi",
	}
	routesMap := map[int]map[int][][]int{
		0: {
			2: {{0, 1, 2}},
		},
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "route"`)).
		WithArgs("test-chain", "terra1abc", "terra1ghi", 2, `{"terra1abc","terra1def","terra1ghi"}`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.UpdateRoutes(context.Background(), indexToAsset, routesMap)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateRoutes_BatchesLargeDataset(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := &srcRepoImpl{db: gormDB, chainId: "test-chain"}

	// Generate enough routes to exceed batchSize (10,000)
	numAssets := 150 // 150*149/2 = 11,175 pairs
	indexToAsset := make(map[int]string, numAssets)
	for i := 0; i < numAssets; i++ {
		indexToAsset[i] = fmt.Sprintf("asset%d", i)
	}

	routesMap := make(map[int]map[int][][]int)
	for i := 0; i < numAssets; i++ {
		routesMap[i] = make(map[int][][]int)
		for j := i + 1; j < numAssets; j++ {
			routesMap[i][j] = [][]int{{i, j}}
		}
	}

	mock.ExpectBegin()
	// Expect 2 INSERT statements due to batching (10,000 + 1,175)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "route"`)).
		WillReturnResult(sqlmock.NewResult(0, 10000))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "route"`)).
		WillReturnResult(sqlmock.NewResult(0, 1175))
	mock.ExpectCommit()

	err := repo.UpdateRoutes(context.Background(), indexToAsset, routesMap)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// The count and the coverage flag arrive as two columns of one row, so the mapping is
// worth pinning: a silent scan failure would report zero pairs and no missing routes,
// which reads exactly like a healthy route table.
func TestPairStatus(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		unrouted bool
	}{
		{name: "routes cover every pair", total: 3, unrouted: false},
		{name: "a pair has no route", total: 3, unrouted: true},
		{name: "no pairs at all", total: 0, unrouted: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock := setupMockDB(t)
			repo := &srcRepoImpl{db: gormDB, chainId: "test-chain"}

			mock.ExpectQuery(regexp.QuoteMeta(`from pair p`)).
				WithArgs("test-chain").
				WillReturnRows(sqlmock.NewRows([]string{"total", "unrouted"}).
					AddRow(tt.total, tt.unrouted))

			count, unrouted, err := repo.PairStatus(context.Background())

			require.NoError(t, err)
			assert.Equal(t, tt.total, count)
			assert.Equal(t, tt.unrouted, unrouted)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPairStatus_DBError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := &srcRepoImpl{db: gormDB, chainId: "test-chain"}

	mock.ExpectQuery(regexp.QuoteMeta(`from pair p`)).WillReturnError(errors.New("db error"))

	_, _, err := repo.PairStatus(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repo.PairStatus")
}
