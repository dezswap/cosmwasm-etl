//go:build faker || mig
// +build faker mig

package main

import (
	"log"
	"os"
	"testing"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/faker"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm/logger"

	"gorm.io/gorm"
)

const batch_size = 100

func Test_ParserMigration(t *testing.T) {
	c := configs.New()
	faker.MigFakerInit()
	parser.FakerCustomGenerator()
	pdb := db.PostgresDb{}
	if err := pdb.Init(c.Rdb); err != nil {
		panic(err)
	}
	myLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			LogLevel:                  logger.Error, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
		},
	)
	dbCon, err := gorm.Open(postgres.New(postgres.Config{
		Conn: pdb.Db,
	}), &gorm.Config{
		Logger: myLogger,
	})
	if err != nil {
		panic(err)
	}

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

	batchInsertAndRead(dbCon, schemas.Pair{}, pairs, &actualPairs, len(pairs))
	batchInsertAndRead(dbCon, schemas.ParsedTx{}, parsedTxs, &actualParsedTxs, len(parsedTxs))
	batchInsertAndRead(dbCon, schemas.PoolInfo{}, poolInfos, &actualPoolInfos, len(poolInfos))
	batchInsertAndRead(dbCon, schemas.SyncedHeight{}, syncedHeights, &actualSyncedHeights, len(syncedHeights))

	assert := assert.New(t)
	assert.Len(actualPairs, len(pairs))
	assert.Len(actualParsedTxs, len(parsedTxs))
	assert.Len(actualPoolInfos, len(poolInfos))
	assert.Len(actualSyncedHeights, len(syncedHeights))

}

type Batchable interface {
	schemas.Pair | schemas.ParsedTx | schemas.PoolInfo | schemas.SyncedHeight
}

func batchInsertAndRead[T Batchable](con *gorm.DB, v T, models []T, targets *[]T, limit int) {
	if err := con.Create(models).Error; err != nil {
		panic(err)
	}

	if tx := con.Model(&v).Limit(limit).Find(targets); tx.Error != nil {
		panic(errors.Wrap(tx.Error, "find"))
	}
}
