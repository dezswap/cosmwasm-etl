//go:build faker || mig
// +build faker mig

package main

import (
	"testing"

	"github.com/dezswap/cosmwasm-etl/configs"
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/faker"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const batch_size = 100

func Test_ParserMigration(t *testing.T) {
	c := configs.NewWithFileName("config.test")
	assertLocalTestDB(t, c.Rdb)

	faker.MigFakerInit()
	p_dex.FakerCustomGenerator()
	dbCon, err := db.OpenGormPostgres(c.Rdb)
	if err != nil {
		panic(err)
	}
	tx := dbCon.Begin()
	if tx.Error != nil {
		panic(tx.Error)
	}
	defer tx.Rollback()

	parsedTxs := []schemas.ParsedTx{}
	poolInfos := []schemas.PoolInfo{}
	pairs := []schemas.Pair{}
	syncedHeights := []schemas.SyncedHeight{}
	syncedHeightSet := make(map[string]schemas.SyncedHeight)

	for i := 0; i < batch_size; i++ {
		pair := schemas.Pair{}
		parsedTx := schemas.ParsedTx{}
		poolInfo := schemas.PoolInfo{}
		syncedHeight := schemas.SyncedHeight{}
		faker.FakeData(&pair)
		faker.FakeData(&parsedTx)
		faker.FakeData(&poolInfo)
		faker.FakeData(&syncedHeight)

		parsedTx.ChainId = pair.ChainId
		parsedTx.Contract = pair.Contract

		poolInfo.ChainId = pair.ChainId
		poolInfo.Contract = pair.Contract

		syncedHeight.ChainId = pair.ChainId

		pairs = append(pairs, pair)
		poolInfos = append(poolInfos, poolInfo)
		parsedTxs = append(parsedTxs, parsedTx)
		syncedHeightSet[pair.ChainId] = syncedHeight
	}
	for _, syncedHeight := range syncedHeightSet {
		syncedHeights = append(syncedHeights, syncedHeight)
	}

	actualParsedTxs := []schemas.ParsedTx{}
	actualPoolInfos := []schemas.PoolInfo{}
	actualPairs := []schemas.Pair{}
	actualSyncedHeights := []schemas.SyncedHeight{}

	batchInsertAndRead(tx, schemas.Pair{}, pairs, &actualPairs, len(pairs))
	batchInsertAndRead(tx, schemas.ParsedTx{}, parsedTxs, &actualParsedTxs, len(parsedTxs))
	batchInsertAndRead(tx, schemas.PoolInfo{}, poolInfos, &actualPoolInfos, len(poolInfos))
	batchInsertAndRead(tx, schemas.SyncedHeight{}, syncedHeights, &actualSyncedHeights, len(syncedHeights))

	assert := assert.New(t)
	assert.Len(actualPairs, len(pairs))
	assert.Len(actualParsedTxs, len(parsedTxs))
	assert.Len(actualPoolInfos, len(poolInfos))
	assert.Len(actualSyncedHeights, len(syncedHeights))

}

// The two flags are independent, and the parser reads only the skip_parse rows.
func Test_TokenException(t *testing.T) {
	c := configs.NewWithFileName("config.test")
	assertLocalTestDB(t, c.Rdb)

	dbCon, err := db.OpenGormPostgres(c.Rdb)
	require.NoError(t, err)

	tx := dbCon.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()

	const chainId = "token-exception-chain"
	rows := []schemas.TokenException{
		{ChainId: chainId, Contract: "hidden-only", SkipParse: false, Hidden: true},
		{ChainId: chainId, Contract: "parse-only", SkipParse: true, Hidden: false},
	}
	require.NoError(t, tx.Create(&rows).Error)

	actual := []schemas.TokenException{}
	require.NoError(t, tx.Where("chain_id = ?", chainId).Order("contract").Find(&actual).Error)
	require.Len(t, actual, 2)
	assert.True(t, actual[0].Hidden)
	assert.False(t, actual[0].SkipParse)
	assert.False(t, actual[1].Hidden)
	assert.True(t, actual[1].SkipParse)

	parsed := []string{}
	require.NoError(t, tx.Model(&schemas.TokenException{}).Where(
		"chain_id = ? and skip_parse", chainId).Pluck("contract", &parsed).Error)
	assert.Equal(t, []string{"parse-only"}, parsed)
}

type Batchable interface {
	schemas.Pair | schemas.ParsedTx | schemas.PoolInfo | schemas.SyncedHeight
}

func assertLocalTestDB(t *testing.T, c configs.RdbConfig) {
	t.Helper()

	switch c.Host {
	case "localhost", "127.0.0.1", "::1":
	default:
		t.Fatalf("migration tests must use a local test DB, got host=%s database=%s", c.Host, c.Database)
	}
}

func batchInsertAndRead[T Batchable](con *gorm.DB, v T, models []T, targets *[]T, limit int) {
	if err := con.Omit("Id").Create(models).Error; err != nil {
		panic(err)
	}

	if tx := con.Model(&v).Limit(limit).Find(targets); tx.Error != nil {
		panic(errors.Wrap(tx.Error, "find"))
	}
}
