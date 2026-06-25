package parser

import (
	"database/sql"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	pkgdb "github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/faker"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
			Type:             dex.Provide,
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
			Type:             dex.Provide,
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
			Type:             dex.Provide,
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
			Type:             dex.Swap,
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
			Type:             dex.Swap,
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

	s.DB, err = pkgdb.OpenGormPostgresWithConn(db)
	require.NoError(s.T(), err)

	s.Repo = readRepoImpl{db: s.DB, chainId: "local"}
	s.C = configs.RdbConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "cosmwasm_etl",
		Username: "app",
		Password: "appPW",
		SslMode:  "disable",
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

	pq := pkgdb.PostgresDb{}
	err := pq.Init(s.C)
	require.NoError(s.T(), err)

	s.DB, err = pkgdb.OpenGormPostgresWithConn(pq.Db)
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
	createTestTxs(s.DB, dex.Provide)

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
	createTestTxs(s.DB, dex.Provide)

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
	createTestTxs(s.DB, dex.Provide)

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
	createTestTxs(s.DB, dex.Provide)

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
	createTestTxs(s.DB, dex.Provide)

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
	createTestTxs(s.DB, dex.Provide)

	// execute
	actualAsset0, actualAsset1, actualLp, err := s.Repo.AssetAmountInPairOfAccount(accounts[0].Address, pairId, start, end)

	// verify
	assert.NoError(err)
	assert.Equal(expectedAsset0, actualAsset0)
	assert.Equal(expectedAsset1, actualAsset1)
	assert.Equal(expectedLp, actualLp)
}

func (s *aggregatorReadRepoSuite) Test_AccountStats_UsesTxHeightPrice() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	priceToken := "uusd"
	account := "terra0wallet"
	pairId := uint64(100)
	contract := "terra0contract"
	asset := "terra0asset"

	require.NoError(s.DB.Exec(`TRUNCATE TABLE parsed_tx, price, tokens, pair CASCADE`).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO pair(id, chain_id, contract, asset0, asset1, lp) VALUES($1, $2, $3, $4, $5, $6)`,
		pairId, chainName, contract, priceToken, asset, "terra0lp",
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO tokens(id, chain_id, address, decimals) VALUES($1, $2, $3, $4), ($5, $6, $7, $8)`,
		1000, chainName, priceToken, 0,
		1001, chainName, asset, 0,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO price(height, chain_id, token_id, price, price_token_id, route_id) VALUES($1, $2, $3, $4, $5, $6)`,
		200, chainName, 1001, "3", 1000, 0,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO parsed_tx(chain_id, height, timestamp, hash, type, sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount, commission0_amount, commission1_amount)
         VALUES
         ($1, 100, $2, 'swap-hash', 'swap', $3, $4, $5, '-10', $6, '7', 'terra0lp', '0', '0', '0', '0'),
         ($1, 250, $2, 'provide-hash', 'provide', $3, $4, $5, '2', $6, '3', 'terra0lp', '5', '0', '0', '0'),
         ($1, 300, $2, 'withdraw-hash', 'withdraw', $3, $4, $5, '-1', $6, '-4', 'terra0lp', '-2', '0', '0', '0')`,
		chainName, start, account, contract, priceToken, asset,
	).Error)

	actual, err := s.Repo.AccountStats(start, end, priceToken)

	require.NoError(err)
	require.Len(actual, 1)
	assert.Equal(account, actual[0].Address)
	assert.Equal(pairId, actual[0].PairId)
	assert.Equal(uint64(3), actual[0].TxCnt)
	assert.Equal(uint64(1), actual[0].SwapTxCnt)
	assert.Equal(uint64(1), actual[0].ProvideTxCnt)
	assert.Equal(uint64(1), actual[0].WithdrawTxCnt)
	assert.Equal("10", actual[0].SwapVolumeInPrice)
	assert.Equal("11", actual[0].ProvideValueInPrice)
	assert.Equal("13", actual[0].WithdrawValueInPrice)
	assert.Equal("-12", actual[0].NetFlowInPrice)
	assert.Equal(priceToken, actual[0].PriceToken)
	assert.Equal("-9", actual[0].NetAsset0Amount)
	assert.Equal("6", actual[0].NetAsset1Amount)
	assert.Equal("3", actual[0].NetLpAmount)
}

func (s *aggregatorReadRepoSuite) Test_AccountStats_UsesInputSideSwapVolumeWhenAsset1IsInput() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	priceToken := "uusd"
	asset := "terra0asset"
	account := "terra0wallet"
	pairId := uint64(101)
	contract := "terra0asset1input"

	require.NoError(s.DB.Exec(`TRUNCATE TABLE parsed_tx, price, tokens, pair CASCADE`).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO pair(id, chain_id, contract, asset0, asset1, lp) VALUES($1, $2, $3, $4, $5, $6)`,
		pairId, chainName, contract, asset, priceToken, "terra0lp",
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO tokens(id, chain_id, address, decimals) VALUES($1, $2, $3, 0), ($4, $2, $5, 0)`,
		1100, chainName, asset, 1101, priceToken,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO price(height, chain_id, token_id, price, price_token_id, route_id) VALUES(90, $1, $2, '5', $3, 0)`,
		chainName, 1100, 1101,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO parsed_tx(chain_id, height, timestamp, hash, type, sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount, commission0_amount, commission1_amount)
         VALUES ($1, 100, $2, 'asset1-input-swap', 'swap', $3, $4, $5, '2', $6, '-8', 'terra0lp', '0', '0', '0', '0')`,
		chainName, start, account, contract, asset, priceToken,
	).Error)

	actual, err := s.Repo.AccountStats(start, end, priceToken)

	require.NoError(err)
	require.Len(actual, 1)
	assert.Equal("8", actual[0].SwapVolumeInPrice)
}

func (s *aggregatorReadRepoSuite) Test_AccountStats_UsesZeroWhenNonPriceTokenHasNoHistoricalPrice() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	priceToken := "uusd"
	asset := "terra0asset"
	account := "terra0wallet"
	pairId := uint64(102)
	contract := "terra0futureprice"

	require.NoError(s.DB.Exec(`TRUNCATE TABLE parsed_tx, price, tokens, pair CASCADE`).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO pair(id, chain_id, contract, asset0, asset1, lp) VALUES($1, $2, $3, $4, $5, $6)`,
		pairId, chainName, contract, asset, priceToken, "terra0lp",
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO tokens(id, chain_id, address, decimals) VALUES($1, $2, $3, 0), ($4, $2, $5, 0)`,
		1200, chainName, asset, 1201, priceToken,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO price(height, chain_id, token_id, price, price_token_id, route_id) VALUES(200, $1, $2, '9', $3, 0)`,
		chainName, 1200, 1201,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO parsed_tx(chain_id, height, timestamp, hash, type, sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount, commission0_amount, commission1_amount)
         VALUES ($1, 100, $2, 'missing-historical-price', 'swap', $3, $4, $5, '-10', $6, '1', 'terra0lp', '0', '0', '0', '0')`,
		chainName, start, account, contract, asset, priceToken,
	).Error)

	actual, err := s.Repo.AccountStats(start, end, priceToken)

	require.NoError(err)
	require.Len(actual, 1)
	assert.Equal("0", actual[0].SwapVolumeInPrice)
}

func (s *aggregatorReadRepoSuite) Test_AccountStats_GroupsByAccountAndPair() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	priceToken := "uusd"
	assetA := "terra0assetA"
	assetB := "terra0assetB"
	accountA := "terra0accountA"
	accountB := "terra0accountB"

	require.NoError(s.DB.Exec(`TRUNCATE TABLE parsed_tx, price, tokens, pair CASCADE`).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO pair(id, chain_id, contract, asset0, asset1, lp) VALUES
         (201, $1, 'terra0pairA', $2, $3, 'terra0lpA'),
         (202, $1, 'terra0pairB', $2, $4, 'terra0lpB')`,
		chainName, priceToken, assetA, assetB,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO tokens(id, chain_id, address, decimals) VALUES
         (2100, $1, $2, 0),
         (2101, $1, $3, 0),
         (2102, $1, $4, 0)`,
		chainName, priceToken, assetA, assetB,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO parsed_tx(chain_id, height, timestamp, hash, type, sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount, commission0_amount, commission1_amount)
         VALUES
         ($1, 100, $2, 'a-pair-a', 'provide', $3, 'terra0pairA', $4, '1', $5, '2', 'terra0lpA', '3', '0', '0', '0'),
         ($1, 101, $2, 'a-pair-b', 'provide', $3, 'terra0pairB', $4, '4', $6, '5', 'terra0lpB', '6', '0', '0', '0'),
         ($1, 102, $2, 'b-pair-a', 'provide', $7, 'terra0pairA', $4, '7', $5, '8', 'terra0lpA', '9', '0', '0', '0')`,
		chainName, start, accountA, priceToken, assetA, assetB, accountB,
	).Error)

	actual, err := s.Repo.AccountStats(start, end, priceToken)

	require.NoError(err)
	require.Len(actual, 3)
	grouped := map[string]schemas.AccountStats30m{}
	for _, stat := range actual {
		grouped[fmt.Sprintf("%s:%d", stat.Address, stat.PairId)] = stat
	}
	assert.Contains(grouped, fmt.Sprintf("%s:%d", accountA, uint64(201)))
	assert.Contains(grouped, fmt.Sprintf("%s:%d", accountA, uint64(202)))
	assert.Contains(grouped, fmt.Sprintf("%s:%d", accountB, uint64(201)))
}

func (s *aggregatorReadRepoSuite) Test_AccountStats_CountsDistinctHashesButSumsRows() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	priceToken := "uusd"
	asset := "terra0asset"
	account := "terra0wallet"
	pairId := uint64(103)
	contract := "terra0duplicatehash"

	require.NoError(s.DB.Exec(`TRUNCATE TABLE parsed_tx, price, tokens, pair CASCADE`).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO pair(id, chain_id, contract, asset0, asset1, lp) VALUES($1, $2, $3, $4, $5, $6)`,
		pairId, chainName, contract, priceToken, asset, "terra0lp",
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO tokens(id, chain_id, address, decimals) VALUES($1, $2, $3, 0), ($4, $2, $5, 0)`,
		1300, chainName, priceToken, 1301, asset,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO parsed_tx(chain_id, height, timestamp, hash, type, sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount, commission0_amount, commission1_amount)
         VALUES
         ($1, 100, $2, 'same-hash', 'swap', $3, $4, $5, '-10', $6, '1', 'terra0lp', '0', '0', '0', '0'),
         ($1, 100, $2, 'same-hash', 'swap', $3, $4, $5, '-15', $6, '2', 'terra0lp', '0', '0', '0', '0')`,
		chainName, start, account, contract, priceToken, asset,
	).Error)

	actual, err := s.Repo.AccountStats(start, end, priceToken)

	require.NoError(err)
	require.Len(actual, 1)
	assert.Equal(uint64(1), actual[0].TxCnt)
	assert.Equal(uint64(1), actual[0].SwapTxCnt)
	assert.Equal("25", actual[0].SwapVolumeInPrice)
	assert.Equal("-25", actual[0].NetAsset0Amount)
	assert.Equal("3", actual[0].NetAsset1Amount)
}

func (s *aggregatorReadRepoSuite) Test_RecentPrices_FiltersByPriceTokenId() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	priceToken := "uusd"
	otherPriceToken := "ukrw"
	asset := "terra0asset"

	require.NoError(s.DB.Exec(`TRUNCATE TABLE price, tokens CASCADE`).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO tokens(id, chain_id, address, decimals) VALUES
         (3000, $1, $2, 0),
         (3001, $1, $3, 0),
         (3002, $1, $4, 0)`,
		chainName, priceToken, otherPriceToken, asset,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO price(height, chain_id, token_id, price, price_token_id, route_id) VALUES
         (99, $1, 3002, '2', 3001, 0),
         (90, $1, 3002, '7', 3000, 0),
         (110, $1, 3002, '9', 3000, 0)`,
		chainName,
	).Error)

	actual, err := s.Repo.RecentPrices(100, 120, []string{"3002"}, priceToken)

	require.NoError(err)
	require.Contains(actual, uint64(3002))
	require.Len(actual[3002], 2)
	assert.Equal(uint64(90), actual[3002][0].Height)
	assert.Equal("7", actual[3002][0].Price)
	assert.Equal(uint64(110), actual[3002][1].Height)
	assert.Equal("9", actual[3002][1].Price)
}

func (s *aggregatorReadRepoSuite) Test_GetParsedTxsWithPriceOfPair_FiltersByPriceTokenId() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	priceToken := "uusd"
	otherPriceToken := "ukrw"
	asset := "terra0asset"
	contract := "terra0pricefilterpair"
	pairId := uint64(300)

	require.NoError(s.DB.Exec(`TRUNCATE TABLE parsed_tx, price, tokens, pair CASCADE`).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO pair(id, chain_id, contract, asset0, asset1, lp) VALUES($1, $2, $3, $4, $5, $6)`,
		pairId, chainName, contract, asset, priceToken, "terra0lp",
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO tokens(id, chain_id, address, decimals) VALUES
         (3100, $1, $2, 0),
         (3101, $1, $3, 0),
         (3102, $1, $4, 0)`,
		chainName, priceToken, otherPriceToken, asset,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO price(height, chain_id, token_id, price, price_token_id, route_id) VALUES
         (99, $1, 3102, '2', 3101, 0),
         (90, $1, 3102, '7', 3100, 0)`,
		chainName,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO parsed_tx(chain_id, height, timestamp, hash, type, sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount, commission0_amount, commission1_amount)
         VALUES ($1, 100, $2, 'price-filter-pair', 'swap', 'terra0wallet', $3, $4, '-10', $5, '1', 'terra0lp', '0', '0', '0', '0')`,
		chainName, start, contract, asset, priceToken,
	).Error)

	actual, err := s.Repo.GetParsedTxsWithPriceOfPair(pairId, priceToken, start, end)

	require.NoError(err)
	require.Len(actual, 1)
	assert.Equal("7", actual[0].Price0)
	assert.Equal("1", actual[0].Price1)
}

func (s *aggregatorReadRepoSuite) Test_PairStats_FiltersByPriceTokenId() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	priceToken := "uusd"
	otherPriceToken := "ukrw"
	asset := "terra0asset"
	contract := "terra0pairstatsfilter"
	pairId := uint64(301)

	require.NoError(s.DB.Exec(`TRUNCATE TABLE parsed_tx, price, tokens, pair CASCADE`).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO pair(id, chain_id, contract, asset0, asset1, lp) VALUES($1, $2, $3, $4, $5, $6)`,
		pairId, chainName, contract, asset, priceToken, "terra0lp",
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO tokens(id, chain_id, address, decimals) VALUES
         (3200, $1, $2, 0),
         (3201, $1, $3, 0),
         (3202, $1, $4, 0)`,
		chainName, priceToken, otherPriceToken, asset,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO price(height, chain_id, token_id, price, price_token_id, route_id) VALUES
         (99, $1, 3202, '2', 3201, 0),
         (90, $1, 3202, '7', 3200, 0)`,
		chainName,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO parsed_tx(chain_id, height, timestamp, hash, type, sender, contract, asset0, asset0_amount, asset1, asset1_amount, lp, lp_amount, commission_amount, commission0_amount, commission1_amount)
         VALUES ($1, 100, $2, 'pair-stats-price-filter', 'swap', 'terra0wallet', $3, $4, '-10', $5, '1', 'terra0lp', '0', '0', '0', '0')`,
		chainName, start, contract, asset, priceToken,
	).Error)

	actual, err := s.Repo.PairStats(start, end, priceToken, map[uint64]schemas.PairStats30m{})

	require.NoError(err)
	require.Len(actual, 1)
	volume0InPrice, err := strconv.ParseFloat(actual[0].Volume0InPrice, 64)
	require.NoError(err)
	assert.Equal(70.0, volume0InPrice)
}

func (s *aggregatorReadRepoSuite) Test_CommissionAmountInPair() {
	assert := assert.New(s.T())

	expectedAsset0, expectedAsset1 := "14457", "125"
	pairId := uint64(0)

	// prepare
	createTestPairs(s.DB)
	createTestTxs(s.DB, dex.Swap)

	actualAsset0, actualAsset1, err := s.Repo.CommissionAmountInPair(pairId, start, end)

	assert.NoError(err)
	assert.Equal(expectedAsset0, actualAsset0)
	assert.Equal(expectedAsset1, actualAsset1)
}

func (s *aggregatorReadRepoSuite) Test_LiquiditiesOfPairStats_UsesOneForPriceToken() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	priceToken := "uusd"
	pairId := uint64(100)

	require.NoError(s.DB.Exec(`TRUNCATE TABLE parsed_tx, lp_history, price, tokens, pair CASCADE`).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO pair(id, chain_id, contract, asset0, asset1, lp) VALUES($1, $2, $3, $4, $5, $6)`,
		pairId, chainName, "terra0pricepair", priceToken, "terra0asset", "terra0lp",
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO tokens(id, chain_id, address, decimals) VALUES($1, $2, $3, $4), ($5, $6, $7, $8), ($9, $10, $11, $12)`,
		1000, chainName, priceToken, 6,
		1001, chainName, "terra0asset", 6,
		1002, chainName, "ukrw", 6,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO price(height, chain_id, token_id, price, price_token_id, route_id) VALUES
         ($1, $2, $3, $4, $5, $6),
         ($7, $8, $9, $10, $11, $12)`,
		99, chainName, 1001, "9", 1002, 0,
		90, chainName, 1001, "2", 1000, 0,
	).Error)
	require.NoError(s.DB.Exec(
		`INSERT INTO lp_history(height, pair_id, chain_id, liquidity0, liquidity1, timestamp) VALUES($1, $2, $3, $4, $5, $6)`,
		100, pairId, chainName, "1000000", "2000000", start,
	).Error)

	actual, err := s.Repo.LiquiditiesOfPairStats(start, end, priceToken)

	require.NoError(err)
	require.Contains(actual, pairId)
	assert.Equal("1000000", actual[pairId].Liquidity0)
	assert.Equal("2000000", actual[pairId].Liquidity1)

	liquidity0InPrice, err := strconv.ParseFloat(actual[pairId].Liquidity0InPrice, 64)
	require.NoError(err)
	liquidity1InPrice, err := strconv.ParseFloat(actual[pairId].Liquidity1InPrice, 64)
	require.NoError(err)
	assert.Equal(1.0, liquidity0InPrice)
	assert.Equal(4.0, liquidity1InPrice)
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

func createTestTxs(db *gorm.DB, txType dex.TxType) {
	db.Exec(`TRUNCATE TABLE parsed_tx`)

	switch txType {
	case dex.Provide:
		db.Omit("Id", "CreatedAt").Create(&provideTxs)
	case dex.Swap:
		db.Omit("Id", "CreatedAt").Create(&swapTxs)
	}
}

func Test_readRepo(t *testing.T) {
	dex.FakerCustomGenerator()
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
