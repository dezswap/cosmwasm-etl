package parser

import (
	"database/sql"
	"flag"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/faker"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const configName = "config.test"
const chainName = "columbus-5"

var loc = flag.String("type", "", "enable db test")

var (
	start = util.ToEpoch(time.Date(2022, 10, 13, 4, 50, 27, 0, time.UTC))
	end   = util.ToEpoch(time.Date(2022, 10, 13, 4, 58, 12, 0, time.UTC))

	pairs = []schemas.Pair{
		{
			ChainId:  chainName,
			Contract: "terra0con1tract1",
			Asset0:   "terra0asset1",
			Asset1:   "terra0asset2",
			Lp:       "terra0l1p1",
		},
		{
			ChainId:  chainName,
			Contract: "terra0con1tract2",
			Asset0:   "terra0asset3",
			Asset1:   "terra0asset4",
			Lp:       "terra0l1p2",
		},
		{
			ChainId:  chainName,
			Contract: "terra0con1tract3",
			Asset0:   "terra0asset5",
			Asset1:   "terra0asset6",
			Lp:       "terra0l1p3",
		},
	}

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

	provideTxs = []schemas.ParsedTx{
		{
			ChainId:          chainName,
			Height:           9787251,
			Timestamp:        start,
			Hash:             "7348EBEC7428CD5F3CA1DC6B15604FAF3C8BEFE8213F1D2F8733F37E25CC1991",
			Sender:           accounts[0].Address,
			Type:             parser.Provide,
			Contract:         pairs[0].Contract,
			Asset0:           pairs[0].Asset0,
			Asset0Amount:     "25000000",
			Asset1:           pairs[0].Asset1,
			Asset1Amount:     "489782",
			Lp:               pairs[0].Lp,
			LpAmount:         "3180318",
			CommissionAmount: "0",
		},
		{
			ChainId:          chainName,
			Height:           9787329,
			Timestamp:        end - 1,
			Hash:             "D59E85ADF83DF8D9D4B00C4D89056FB53B54B6F47CCBA5BD5561842499B3095A",
			Sender:           accounts[0].Address,
			Type:             parser.Provide,
			Contract:         pairs[0].Contract,
			Asset0:           pairs[0].Asset0,
			Asset0Amount:     "13517017",
			Asset1:           pairs[0].Asset1,
			Asset1Amount:     "909068",
			Lp:               pairs[0].Lp,
			LpAmount:         "447645",
			CommissionAmount: "0",
		},
		{
			ChainId:          chainName,
			Height:           9787329,
			Timestamp:        end - 1,
			Hash:             "271CBA45F65D39ACF147F9B21F2991DE6669F218C2CBA256FB8948B11F34D2B0",
			Sender:           accounts[1].Address,
			Type:             parser.Provide,
			Contract:         pairs[1].Contract,
			Asset0:           pairs[1].Asset0,
			Asset0Amount:     "13517017",
			Asset1:           pairs[1].Asset1,
			Asset1Amount:     "909068",
			Lp:               pairs[1].Lp,
			LpAmount:         "447645",
			CommissionAmount: "0",
		},
	}

	swapTxs = []schemas.ParsedTx{
		{
			ChainId:          chainName,
			Height:           9787251,
			Timestamp:        start,
			Hash:             "A07CECA3F91454AFD8D78A7E242EE6A5371EBBACE1F4219FC7134C5620C92C5C",
			Sender:           accounts[0].Address,
			Type:             parser.Swap,
			Contract:         pairs[0].Contract,
			Asset0:           pairs[0].Asset0,
			Asset0Amount:     "-4804871",
			Asset1:           pairs[0].Asset1,
			Asset1Amount:     "249266",
			Lp:               pairs[0].Lp,
			LpAmount:         "0",
			CommissionAmount: "14457",
		},
		{
			ChainId:          chainName,
			Height:           9787329,
			Timestamp:        end - 1,
			Hash:             "F2A46F9128C41BBD9CBC8F99316E80EBAA71CD644CAE095E473E049799068FEE",
			Sender:           accounts[1].Address,
			Type:             parser.Swap,
			Contract:         pairs[0].Contract,
			Asset0:           pairs[0].Asset0,
			Asset0Amount:     "1000000",
			Asset1:           pairs[0].Asset1,
			Asset1Amount:     "-41616",
			Lp:               pairs[0].Lp,
			LpAmount:         "0",
			CommissionAmount: "125",
		},
	}
)

func TestMain(m *testing.M) {
	flag.Parse()

	code := m.Run()
	os.Exit(code)
}

type baseSuite struct {
	suite.Suite
	DB   *gorm.DB
	Mock sqlmock.Sqlmock

	Repo readRepoImpl
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

	s.Repo = readRepoImpl{db: s.DB, chainId: "local"}
	s.C = configs.RdbConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "cosmwasm_etl",
		Username: "app",
		Password: "appPW",
	}
}

type readSyncedHeightSuite struct {
	baseSuite
}

func (s *readSyncedHeightSuite) Test_ReadGetSyncedHeight() {
	heights := []uint64{}
	_ = faker.FakeData(&heights)

	for idx, height := range heights {
		assert := assert.New(s.T())
		rows := sqlmock.NewRows([]string{"height"}).
			AddRow(height)
		s.Mock.ExpectQuery(`^SELECT (.+) FROM "synced_height" WHERE chain_id = `).WillReturnRows(rows)

		msg := fmt.Sprintf("tc(%d)", idx)
		actual, err := s.Repo.GetSyncedHeight()
		assert.NoError(err, msg)
		assert.Equal(height, actual, msg)
	}
}

type readPairsSuite struct {
	baseSuite
}

func (s *readPairsSuite) Test_GetPairs() {
	pairs := []schemas.Pair{}
	_ = faker.FakeData(&pairs)

	assert := assert.New(s.T())
	rows := sqlmock.NewRows([]string{"chain_id", "contract", "asset0", "asset1", "lp", "meta"})
	for _, pair := range pairs {
		rows.AddRow(s.Repo.chainId, pair.Contract, pair.Asset0, pair.Asset1, pair.Lp, nil)
	}
	s.Mock.ExpectQuery(`^SELECT \* FROM "pair" WHERE "pair"\."chain_id" = \$1`).WillReturnRows(rows)

	actual, err := s.Repo.GetPairs()
	assert.NoError(err)

	for idx := range actual {
		assert.Equal(pairs[idx].Contract, actual[idx].Contract)
		assert.Equal(pairs[idx].Asset0, actual[idx].Asset0)
		assert.Equal(pairs[idx].Asset1, actual[idx].Asset1)
		assert.Equal(pairs[idx].Lp, actual[idx].Lp)
	}
}

type readPoolInfosSuite struct {
	baseSuite
}

func (s *readPoolInfosSuite) Test_GetPoolInfos() {
	poolInfos := []schemas.PoolInfo{}
	_ = faker.FakeData(&poolInfos)
	height := uint64(100)
	assert := assert.New(s.T())
	rows := sqlmock.NewRows([]string{"chain_id", "height", "contract", "asset0_amount", "asset1_amount", "lp_amount", "meta"})
	for _, pi := range poolInfos {
		rows.AddRow(s.Repo.chainId, height, pi.Contract, pi.Asset0Amount, pi.Asset1Amount, pi.LpAmount, nil)
	}
	s.Mock.ExpectQuery(`^SELECT \* FROM "pool_info" WHERE "pool_info"\."chain_id" = \$1 AND "pool_info"\."height" = \$2`).WillReturnRows(rows)

	actual, err := s.Repo.GetPoolInfosByHeight(height)
	assert.NoError(err)

	for idx, row := range actual {
		assert.Equal(height, row.Height)
		assert.Equal(poolInfos[idx].Contract, row.Contract)
		assert.Equal(poolInfos[idx].Asset0Amount, row.Asset0Amount)
		assert.Equal(poolInfos[idx].Asset1Amount, row.Asset1Amount)
		assert.Equal(poolInfos[idx].LpAmount, row.LpAmount)
	}

	s.Mock.ExpectQuery(`^SELECT \* FROM "pool_info" WHERE "pool_info"\."chain_id" = \$1 AND "pool_info"\."height" = \$2`).WillReturnRows(sqlmock.NewRows([]string{}))
	actual, err = s.Repo.GetPoolInfosByHeight(height)
	assert.NoError(err)
	assert.Len(actual, 0)
}

type readParsedTxsSuite struct {
	baseSuite
}

func (s *readParsedTxsSuite) Test_GetParsedTxs() {
	parsedTxs := []schemas.ParsedTx{}
	_ = faker.FakeData(&parsedTxs)
	height := uint64(100)
	assert := assert.New(s.T())
	rows := sqlmock.NewRows([]string{"chain_id", "height", "timestamp", "hash", "type", "sender", "contract", "asset0_amount", "asset1_amount", "lp_amount", "commission_amount", "meta"})
	for _, tx := range parsedTxs {
		tx.Timestamp = math.Floor(tx.Timestamp)
		rows.AddRow(s.Repo.chainId, tx.Height, tx.Timestamp, tx.Hash, tx.Type, tx.Sender, tx.Contract, tx.Asset0Amount, tx.Asset1Amount, tx.LpAmount, tx.CommissionAmount, nil)
	}
	s.Mock.ExpectQuery(`^SELECT \* FROM "parsed_tx" WHERE "parsed_tx"\."chain_id" = \$1 AND "parsed_tx"\."height" = \$2`).WillReturnRows(rows)

	actual, err := s.Repo.GetParsedTxs(height)
	assert.NoError(err)
	assert.Len(actual, len(parsedTxs))

	s.Mock.ExpectQuery(`^SELECT \* FROM "parsed_tx" WHERE "parsed_tx"\."chain_id" = \$1 AND "parsed_tx"\."height" = \$2`).WillReturnRows(sqlmock.NewRows([]string{}))
	actual, err = s.Repo.GetParsedTxs(height)
	assert.NoError(err)
	assert.Len(actual, 0)
}

func (s *readParsedTxsSuite) Test_GetParsedTxsOfPair() {
	parsedTxs := []schemas.ParsedTx{}
	_ = faker.FakeData(&parsedTxs)
	height := uint64(100)
	pair := "pair"
	assert := assert.New(s.T())
	rows := sqlmock.NewRows([]string{"chain_id", "height", "timestamp", "hash", "type",
		"sender", "contract", "asset0_amount", "asset1_amount", "lp_amount",
		"commission_amount", "commission0_amount", "commission1_amount", "meta"})
	for _, tx := range parsedTxs {
		tx.Timestamp = math.Floor(tx.Timestamp)
		tx.Contract = pair
		rows.AddRow(s.Repo.chainId, tx.Height, tx.Timestamp, tx.Hash, tx.Type,
			tx.Sender, tx.Contract, tx.Asset0Amount, tx.Asset1Amount, tx.LpAmount,
			tx.CommissionAmount, tx.Commission0Amount, tx.Commission1Amount, tx.Meta)
	}
	s.Mock.ExpectQuery(`^SELECT \* FROM "parsed_tx" WHERE "parsed_tx"\."chain_id" = \$1 AND "parsed_tx"\."height" = \$2 AND "parsed_tx"\."contract" = \$3`).WillReturnRows(rows)

	actual, err := s.Repo.GetParsedTxsOfPair(height, pair)
	assert.NoError(err)
	assert.Len(actual, len(parsedTxs))

	s.Mock.ExpectQuery(`^SELECT \* FROM "parsed_tx" WHERE "parsed_tx"\."chain_id" = \$1 AND "parsed_tx"\."height" = \$2 AND "parsed_tx"\."contract" = \$3`).WillReturnRows(sqlmock.NewRows([]string{}))
	actual, err = s.Repo.GetParsedTxsOfPair(height, pair)
	assert.NoError(err)
	assert.Len(actual, 0)
}

type aggregatorReadRepoSuite struct {
	suite.Suite
	DB *gorm.DB

	Repo ReadRepository
	C    configs.RdbConfig
}

func (s *aggregatorReadRepoSuite) SetupSuite() {
	s.C = configs.NewWithFileName(configName).Aggregator.SrcDb
	s.Repo = NewReadRepo(chainName, s.C)

	pq := db.PostgresDb{}
	err := pq.Init(s.C)
	require.NoError(s.T(), err)

	s.DB, err = gorm.Open(postgres.New(postgres.Config{
		Conn: pq.Db,
	}), &gorm.Config{})
	require.NoError(s.T(), err)
}

func (s *aggregatorReadRepoSuite) Test_OldestTxTimestamp() {
	assert := assert.New(s.T())

	expected := float64(1665624218)

	// prepare
	s.DB.Exec(`TRUNCATE TABLE parsed_tx`)
	s.DB.Exec(`
INSERT INTO parsed_tx(chain_id, height, "timestamp", hash, "type", sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount)
VALUES($1, 9785167, EXTRACT(EPOCH FROM TIMESTAMP '2022-10-13 01:23:38.000000'), 'C1151FEF9593FE768FD71C85C441E1AF2E5654E73C3516DDBDDD10E6D446357D', 'swap', 'terra1zr7x25fjkkcn9pa5hrleg9rz2r5hh9jwjdgkje', 'terra1rnc5cp7r9nxrskhup9uqs9v0e43hmm6u9xydec', 'uusd', -690000000, 'terra1aa00lpfexyycedfg5k2p60l9djcmw0ue5l8fhc', 134054, 'terra13p6glny4ltdtrlptq2g9gs3e8m35t55uwgs9ea', 0, 403)`,
		chainName)
	s.DB.Exec(`
INSERT INTO parsed_tx(chain_id, height, "timestamp", hash, "type", sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount)
VALUES($1, 9788000, EXTRACT(EPOCH FROM TIMESTAMP '2022-10-13 06:04:42.000000'), 'EBA270C24E1350CE1B3FB2B31827AB44674B35117F4CCE9644009A1823705BB2', 'swap', 'terra1jwy50dplpx847zssxhacrtzqnakmzymskktd98', 'terra13awymgywq8nth34qgjaa6rm6junfqv3nxaupnw', 'uusd', 7959952, 'terra1rhhvx8nzfrx5fufkuft06q5marfkucdqwq5sjw', -155147, 'terra1aw52tgnmu3dfeguy36vm3we3ju0v6dhnvkj2cg', 0, 466)`,
		chainName)
	s.DB.Exec(`
INSERT INTO parsed_tx(chain_id, height, "timestamp", hash, "type", sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount)
VALUES($1, 9787251, EXTRACT(EPOCH FROM TIMESTAMP '2022-10-13 04:50:27.000000'), '7348EBEC7428CD5F3CA1DC6B15604FAF3C8BEFE8213F1D2F8733F37E25CC1991', 'provide', 'terra1p7zn9xwr4pdjvy59y075quq8t3rvgvx7d785qz', 'terra13awymgywq8nth34qgjaa6rm6junfqv3nxaupnw', 'uusd', 25000000, 'terra1rhhvx8nzfrx5fufkuft06q5marfkucdqwq5sjw', 489782, 'terra1aw52tgnmu3dfeguy36vm3we3ju0v6dhnvkj2cg', 3180318, 0)`,
		chainName)

	// execute
	actual, err := s.Repo.OldestTxTimestamp()

	// verify
	assert.NoError(err)
	assert.Equal(expected, actual)
}

func (s *aggregatorReadRepoSuite) Test_LatestTxTimestamp() {
	assert := assert.New(s.T())

	expected := float64(1665641082)

	// prepare
	s.DB.Exec(`TRUNCATE TABLE parsed_tx`)
	s.DB.Exec(`
INSERT INTO parsed_tx(chain_id, height, "timestamp", hash, "type", sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount)
VALUES($1, 9785167, EXTRACT(EPOCH FROM TIMESTAMP '2022-10-13 01:23:38.000000'), 'C1151FEF9593FE768FD71C85C441E1AF2E5654E73C3516DDBDDD10E6D446357D', 'swap', 'terra1zr7x25fjkkcn9pa5hrleg9rz2r5hh9jwjdgkje', 'terra1rnc5cp7r9nxrskhup9uqs9v0e43hmm6u9xydec', 'uusd', -690000000, 'terra1aa00lpfexyycedfg5k2p60l9djcmw0ue5l8fhc', 134054, 'terra13p6glny4ltdtrlptq2g9gs3e8m35t55uwgs9ea', 0, 403)`,
		chainName)
	s.DB.Exec(`
INSERT INTO parsed_tx(chain_id, height, "timestamp", hash, "type", sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount)
VALUES($1, 9788000, EXTRACT(EPOCH FROM TIMESTAMP '2022-10-13 06:04:42.000000'), 'EBA270C24E1350CE1B3FB2B31827AB44674B35117F4CCE9644009A1823705BB2', 'swap', 'terra1jwy50dplpx847zssxhacrtzqnakmzymskktd98', 'terra13awymgywq8nth34qgjaa6rm6junfqv3nxaupnw', 'uusd', 7959952, 'terra1rhhvx8nzfrx5fufkuft06q5marfkucdqwq5sjw', -155147, 'terra1aw52tgnmu3dfeguy36vm3we3ju0v6dhnvkj2cg', 0, 466)`,
		chainName)
	s.DB.Exec(`
INSERT INTO parsed_tx(chain_id, height, "timestamp", hash, "type", sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount)
VALUES($1, 9787251, EXTRACT(EPOCH FROM TIMESTAMP '2022-10-13 04:50:27.000000'), '7348EBEC7428CD5F3CA1DC6B15604FAF3C8BEFE8213F1D2F8733F37E25CC1991', 'provide', 'terra1p7zn9xwr4pdjvy59y075quq8t3rvgvx7d785qz', 'terra13awymgywq8nth34qgjaa6rm6junfqv3nxaupnw', 'uusd', 25000000, 'terra1rhhvx8nzfrx5fufkuft06q5marfkucdqwq5sjw', 489782, 'terra1aw52tgnmu3dfeguy36vm3we3ju0v6dhnvkj2cg', 3180318, 0)`,
		chainName)

	// execute
	actual, err := s.Repo.LatestTxTimestamp()

	assert.NoError(err)
	assert.Equal(expected, actual)
}

func (s *aggregatorReadRepoSuite) Test_PairIds() {
	assert := assert.New(s.T())

	expected := make([]uint64, len(pairs))
	i := uint64(0)
	for i < uint64(len(pairs)) {
		expected[i] = i
		i++
	}

	// prepare
	createTestPairs(s.DB)

	// execute
	actual, err := s.Repo.PairIds()

	// verify
	assert.NoError(err)
	assert.EqualValues(expected, actual)
}

func (s *aggregatorReadRepoSuite) Test_NewPairIds() {
	assert := assert.New(s.T())

	expected := []uint64{0} // the index of `pairs`

	// prepare
	createTestAccounts(s.DB)
	createTestPairs(s.DB)
	createTestTxs(s.DB, parser.Provide)

	actual, err := s.Repo.NewPairIds(accounts[0].Address, start, end)

	// verify
	assert.NoError(err)
	assert.EqualValues(expected, actual)
}

func (s *aggregatorReadRepoSuite) Test_NewAccounts() {
	assert := assert.New(s.T())

	expected := []string{accounts[0].Address, accounts[1].Address}

	// prepare
	s.DB.Exec(`TRUNCATE TABLE account`)
	createTestTxs(s.DB, parser.Provide)

	// execute
	actual, err := s.Repo.NewAccounts(start, end)

	// verify
	assert.NoError(err)
	assert.EqualValues(expected, actual)
}

func (s *aggregatorReadRepoSuite) Test_ProviderCount() {
	assert := assert.New(s.T())

	expected := uint64(1)
	pairId := uint64(0)

	// prepare
	createTestPairs(s.DB)
	createTestTxs(s.DB, parser.Provide)

	// execute
	actual, err := s.Repo.ProviderCount(pairId, start, end)

	// verify
	assert.NoError(err)
	assert.Equal(expected, actual)
}

func (s *aggregatorReadRepoSuite) Test_TxCountOfAccount() {
	assert := assert.New(s.T())

	expected := uint64(2)
	pairId := uint64(0)

	// prepare
	createTestAccounts(s.DB)
	createTestPairs(s.DB)
	createTestTxs(s.DB, parser.Provide)

	// execute
	actual, err := s.Repo.TxCountOfAccount(accounts[0].Address, pairId, start, end)

	// verify
	assert.NoError(err)
	assert.Equal(expected, actual)
}

func (s *aggregatorReadRepoSuite) Test_AssetAmountInPair() {
	assert := assert.New(s.T())

	expectedAsset0, expectedAsset1, expectedLp := "38517017", "1398850", "3627963"
	pairId := uint64(0)

	// prepare
	createTestAccounts(s.DB)
	createTestPairs(s.DB)
	createTestTxs(s.DB, parser.Provide)

	// execute
	actualAsset0, actualAsset1, actualLp, err := s.Repo.AssetAmountInPair(pairId, start, end)

	// verify
	assert.NoError(err)
	assert.Equal(expectedAsset0, actualAsset0)
	assert.Equal(expectedAsset1, actualAsset1)
	assert.Equal(expectedLp, actualLp)
}

func (s *aggregatorReadRepoSuite) Test_AssetAmountInPairOfAccount() {
	assert := assert.New(s.T())

	expectedAsset0, expectedAsset1, expectedLp := "38517017", "1398850", "3627963"
	pairId := uint64(0)

	// prepare
	createTestAccounts(s.DB)
	createTestPairs(s.DB)
	createTestTxs(s.DB, parser.Provide)

	// execute
	actualAsset0, actualAsset1, actualLp, err := s.Repo.AssetAmountInPairOfAccount(accounts[0].Address, pairId, start, end)

	// verify
	assert.NoError(err)
	assert.Equal(expectedAsset0, actualAsset0)
	assert.Equal(expectedAsset1, actualAsset1)
	assert.Equal(expectedLp, actualLp)
}

func (s *aggregatorReadRepoSuite) Test_CommissionAmountInPair() {
	assert := assert.New(s.T())

	expectedAsset0, expectedAsset1 := "14457", "125"
	pairId := uint64(0)

	// prepare
	createTestPairs(s.DB)
	createTestTxs(s.DB, parser.Swap)

	actualAsset0, actualAsset1, err := s.Repo.CommissionAmountInPair(pairId, start, end)

	assert.NoError(err)
	assert.Equal(expectedAsset0, actualAsset0)
	assert.Equal(expectedAsset1, actualAsset1)
}

func createTestPairs(db *gorm.DB) {
	db.Exec(`TRUNCATE TABLE pair`)
	for id, pair := range pairs {
		db.Exec(`INSERT INTO pair(id, chain_id, contract, asset0, asset1, lp) VALUES($1, $2, $3, $4, $5, $6)`,
			id, pair.ChainId, pair.Contract, pair.Asset0, pair.Asset1, pair.Lp)
	}
}

func createTestAccounts(db *gorm.DB) {
	db.Exec(`TRUNCATE TABLE account`)
	db.Omit("CreatedAt").Create(&accounts)
}

func createTestTxs(db *gorm.DB, txType parser.TxType) {
	db.Exec(`TRUNCATE TABLE parsed_tx`)

	switch txType {
	case parser.Provide:
		db.Omit("Id", "CreatedAt").Create(&provideTxs)
	case parser.Swap:
		db.Omit("Id", "CreatedAt").Create(&swapTxs)
	}
}

func Test_readRepo(t *testing.T) {
	parser.FakerCustomGenerator()
	faker.CustomGenerator()
	suite.Run(t, new(readSyncedHeightSuite))
	suite.Run(t, new(readPairsSuite))
	suite.Run(t, new(readPoolInfosSuite))
	suite.Run(t, new(readParsedTxsSuite))

	// go test -v -type db
	if *loc == "db" {
		suite.Run(t, new(aggregatorReadRepoSuite))
	}
}
