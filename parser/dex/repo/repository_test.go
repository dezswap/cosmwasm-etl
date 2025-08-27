package repo

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/faker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type baseSuite struct {
	suite.Suite
	DB   *gorm.DB
	Mock sqlmock.Sqlmock

	Repo repoImpl
	C    configs.RdbConfig
}

func (s *baseSuite) SetupSuite() {
	var (
		db  *sql.DB
		err error
	)

	db, s.Mock, err = sqlmock.New()
	require.NoError(s.T(), err)

	s.DB, err = gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(s.T(), err)

	s.Repo = repoImpl{mapper: &parserMapperImpl{}, db: s.DB, chainId: "local"}
	s.C = configs.RdbConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "cosmwasm_etl",
		Username: "app",
		Password: "appPW",
	}
}

type SyncedHeightSuite struct {
	baseSuite
}

func (s *SyncedHeightSuite) Test_GetSyncedHeight() {

	tcs := []struct {
		want uint64
	}{
		{10}, {1}, {999},
	}

	for idx, tc := range tcs {
		assert := assert.New(s.T())
		rows := sqlmock.NewRows([]string{"height"}).
			AddRow(tc.want)
		s.Mock.ExpectQuery(`^SELECT (.+) FROM "synced_height" WHERE (.+)?[chain_id ](.+)=`).WillReturnRows(rows)

		msg := fmt.Sprintf("tc(%d)", idx)
		height, err := s.Repo.GetSyncedHeight()
		assert.NoError(err, msg)
		assert.Equal(tc.want, height, msg)
	}
}

func (s *SyncedHeightSuite) Test_GetSyncedHeight_Create() {

	assert := assert.New(s.T())
	rows := sqlmock.NewRows([]string{}).AddRow()
	s.Mock.ExpectQuery(`^SELECT (.+) FROM "synced_height" WHERE (.+)?[chain_id ](.+)=`).WillReturnRows(rows)
	res := sqlmock.NewResult(1, 1)
	s.Mock.ExpectExec(`^INSERT INTO "synced_height" (.+) VALUES (.+)`).WillReturnResult(res)
	msg := "FirstOrCreate"
	height, err := s.Repo.GetSyncedHeight()
	assert.NoError(err, msg)
	assert.Equal(uint64(0), height, msg)
}

func (s *SyncedHeightSuite) Test_GetSyncedHeight_Fail() {

	assert := assert.New(s.T())
	s.Mock.ExpectQuery(`^SELECT (.+) FROM "synced_height" WHERE (.+)?[chain_id ](.+)=`).WillReturnError(errors.New("error"))

	_, err := s.Repo.GetSyncedHeight()
	assert.Error(err)
}

type pairsSuite struct {
	baseSuite
}

func (s *pairsSuite) Test_GetPairs() {
	pairs := dex.FakeParserPairs()

	assert := assert.New(s.T())
	rows := sqlmock.NewRows([]string{"chain_id", "contract", "asset0", "asset1", "lp", "meta"})
	for _, pair := range pairs {
		rows.AddRow(s.Repo.chainId, pair.ContractAddr, pair.Assets[0], pair.Assets[1], pair.LpAddr, nil)
	}
	s.Mock.ExpectQuery(`^SELECT (.+) FROM "pair" WHERE chain_id = `).WillReturnRows(rows)

	pairMap, err := s.Repo.GetPairs()
	assert.NoError(err)

	for _, pair := range pairs {
		target, ok := pairMap[pair.ContractAddr]
		assert.True(ok)
		assert.Equal(pair.Assets[0], target.Assets[0])
		assert.Equal(pair.Assets[1], target.Assets[1])
		assert.Equal(pair.LpAddr, target.LpAddr)
	}

}

type insertSuite struct {
	baseSuite
	parsedTxs []dex.ParsedTx
	poolInfos []dex.PoolInfo
	pairs     []dex.Pair
	height    uint64
}

func (s *insertSuite) SetupSuite() {
	s.baseSuite.SetupSuite()
	s.pairs = dex.FakeParserPairs()
	pairLen := len(s.pairs)
	parsedTxs := dex.FakeParserParsedTxs()
	for ; len(parsedTxs) < pairLen; parsedTxs = dex.FakeParserParsedTxs() {
	}
	poolInfos := dex.FakeParserPoolInfoTxs()
	for ; len(poolInfos) < pairLen; poolInfos = dex.FakeParserPoolInfoTxs() {
	}
	for idx, pair := range s.pairs {
		parsedTxs[idx].ContractAddr = pair.ContractAddr
		parsedTxs[idx].Assets[0].Addr = pair.Assets[0]
		parsedTxs[idx].Assets[1].Addr = pair.Assets[1]
		parsedTxs[idx].LpAddr = pair.LpAddr

		poolInfos[idx].ContractAddr = pair.ContractAddr
		poolInfos[idx].Assets[0].Addr = pair.Assets[0]
		poolInfos[idx].Assets[1].Addr = pair.Assets[1]
	}
	s.parsedTxs = parsedTxs[0:pairLen]
	s.poolInfos = poolInfos[0:pairLen]
	if err := faker.FakeData(&s.height); err != nil {
		panic(err)
	}
	if s.height == 0 {
		s.height += 1
	}
}

func (s *insertSuite) SetSuccessMock() {

	pairLen, poolInfosLen := int64(len(s.pairs)), int64(len(s.poolInfos))
	parsedTxIds := sqlmock.NewRows([]string{"id"})
	for i := 0; i < len(s.parsedTxs); i++ {
		parsedTxIds = parsedTxIds.AddRow(0)
	}

	s.Mock.ExpectBegin()
	s.Mock.ExpectExec("SAVEPOINT (.*)").WillReturnResult(sqlmock.NewResult(0, 0))
	s.Mock.ExpectExec(`INSERT INTO "pair" (.*)`).WillReturnResult(sqlmock.NewResult(pairLen, pairLen))
	s.Mock.ExpectExec("SAVEPOINT (.*)").WillReturnResult(sqlmock.NewResult(0, 0))
	s.Mock.ExpectQuery(`INSERT INTO "parsed_tx" (.*)`).WillReturnRows(parsedTxIds)
	s.Mock.ExpectExec("SAVEPOINT (.*)").WillReturnResult(sqlmock.NewResult(0, 0))
	s.Mock.ExpectExec(`INSERT INTO "pool_info" (.*)`).WillReturnResult(sqlmock.NewResult(poolInfosLen, poolInfosLen))
	s.Mock.ExpectExec(`UPDATE "synced_height" SET "height"=\$1 WHERE chain\_id = \$2 AND height = \$3`).WithArgs(s.height, s.Repo.chainId, s.height-1).WillReturnResult(sqlmock.NewResult(1, 1))
	s.Mock.ExpectCommit()

}

func (s *insertSuite) SetFailMock() {
	s.Mock.ExpectBegin()
	s.Mock.ExpectExec("SAVEPOINT (.*)").WillReturnResult(sqlmock.NewResult(0, 0))
	s.Mock.ExpectExec(`INSERT INTO "pair" (.*)`).WillReturnError(errors.New("contract cannot be empty"))
	s.Mock.ExpectRollback()

}

func (s *insertSuite) Test_Insert() {
	assert := assert.New(s.T())
	s.SetSuccessMock()
	err := s.Repo.Insert(s.height, s.parsedTxs, s.poolInfos, s.pairs)
	assert.NoError(err)

	s.SetFailMock()
	s.pairs[0].ContractAddr = ""
	err = s.Repo.Insert(s.height, s.parsedTxs, s.poolInfos, s.pairs)
	assert.Error(err)
}

type validationExceptionSuite struct {
	baseSuite
}

func (s *validationExceptionSuite) SetSuccessMock() {
	s.Mock.ExpectBegin()
	s.Mock.ExpectExec(`INSERT INTO "pair_validation_exception" (.+)`).WillReturnResult(sqlmock.NewResult(1, 1))
	s.Mock.ExpectCommit()

}

func (s *validationExceptionSuite) SetFailMock() {
	s.Mock.ExpectBegin()
	s.Mock.ExpectExec(`INSERT INTO "pair_validation_exception" (.+)`).WillReturnError(errors.New("insert error"))
	s.Mock.ExpectRollback()

}

func (s *validationExceptionSuite) Test_InsertPairValidationException() {
	assert := assert.New(s.T())

	s.SetSuccessMock()
	err := s.Repo.InsertPairValidationException("testnet-1", "test1abcd")
	assert.NoError(err)

	s.SetFailMock()
	err = s.Repo.InsertPairValidationException("testnet-1", "test1abcd")
	assert.Error(err)
}

func Test_repo(t *testing.T) {
	dex.FakerCustomGenerator()
	faker.CustomGenerator()
	suite.Run(t, new(SyncedHeightSuite))
	suite.Run(t, new(pairsSuite))
	suite.Run(t, new(insertSuite))
	suite.Run(t, new(validationExceptionSuite))
}
