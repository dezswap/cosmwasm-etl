package router

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
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

	err := repo.UpdateRoutes(indexToAsset, routesMap)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateRoutes_EmptyRoutes(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := &srcRepoImpl{db: gormDB, chainId: "test-chain"}

	indexToAsset := map[int]string{}
	routesMap := map[int]map[int][][]int{}

	mock.ExpectBegin()
	mock.ExpectCommit()

	err := repo.UpdateRoutes(indexToAsset, routesMap)

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

	err := repo.UpdateRoutes(indexToAsset, routesMap)

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

	err := repo.UpdateRoutes(indexToAsset, routesMap)

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

	err := repo.UpdateRoutes(indexToAsset, routesMap)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
