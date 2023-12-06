package aggregator

import (
	"os"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/aggregator/repo"
	"gorm.io/gorm"

	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/stretchr/testify/mock"
)

type repoMock struct {
	mock.Mock

	calledGetParsedTxsWithLimit bool

	updatedLpHistory       []schemas.LpHistory
	updatedPairStatsRecent []schemas.PairStatsRecent
	updatedPairStats       []schemas.PairStats30m
	updatedAccountStats    []schemas.AccountStats30m
	updatedAccounts        []string
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func (r *repoMock) LatestTimestamp(_ string) (float64, error) {
	return 0, nil
}

func (r *repoMock) GetSyncedHeight() (uint64, error) {
	return 0, nil
}

func (r *repoMock) GetPairs() ([]schemas.Pair, error) {
	return nil, nil
}

func (r *repoMock) GetPoolInfosByHeight(_ uint64) ([]schemas.PoolInfo, error) {
	return nil, nil
}

func (r *repoMock) GetParsedTxs(_ uint64) ([]schemas.ParsedTx, error) {
	return nil, nil
}

func (r *repoMock) GetParsedTxsOfPair(_ uint64, _ string) ([]schemas.ParsedTx, error) {
	return nil, nil
}

func (r *repoMock) TxHeightToSync(_ int64, _ ...string) (int64, error) {
	return 0, nil
}

func (r *repoMock) HeightOnTimestamp(_ float64) (uint64, error) {
	args := r.Mock.MethodCalled("HeightOnTimestamp")
	return args.Get(0).(uint64), args.Error(1)
}

func (r *repoMock) LastHeightOfPrice() (uint64, error) {
	args := r.Mock.MethodCalled("LastHeightOfPrice")
	return args.Get(0).(uint64), args.Error(1)
}

func (r *repoMock) GetRecentParsedTxs(_ uint64, _ uint64) ([]schemas.ParsedTxWithPrice, error) {
	args := r.Mock.MethodCalled("GetRecentParsedTxs")
	return args.Get(0).([]schemas.ParsedTxWithPrice), args.Error(1)
}

func (r *repoMock) RecentPrices(_ uint64, _ uint64, _ []string, _ string) (map[uint64][]schemas.Price, error) {
	args := r.Mock.MethodCalled("RecentPrices")
	return args.Get(0).(map[uint64][]schemas.Price), args.Error(1)
}

func (r *repoMock) GetParsedTxsWithPriceOfPair(_ uint64, _ string, _ float64, _ float64) ([]schemas.ParsedTxWithPrice, error) {
	args := r.Mock.MethodCalled("GetParsedTxsWithPriceOfPair")
	return args.Get(0).([]schemas.ParsedTxWithPrice), args.Error(1)
}

func (r *repoMock) OldestTxTimestamp() (float64, error) {
	args := r.Mock.MethodCalled("OldestTxTimestamp")
	return args.Get(0).(float64), args.Error(1)
}

func (r *repoMock) LatestTxTimestamp() (float64, error) {
	return 0, nil
}

func (r *repoMock) PairIds() ([]uint64, error) {
	args := r.Mock.MethodCalled("PairIds")
	return args.Get(0).([]uint64), args.Error(1)
}

func (r *repoMock) NewPairIds(_ string, _ float64, _ float64) ([]uint64, error) {
	args := r.Mock.MethodCalled("NewPairIds")
	return args.Get(0).([]uint64), args.Error(1)
}

func (r *repoMock) NewAccounts(_ float64, _ float64) ([]string, error) {
	args := r.Mock.MethodCalled("NewAccounts")
	return args.Get(0).([]string), args.Error(1)
}

func (r *repoMock) ProviderCount(_ uint64, _ float64, _ float64) (uint64, error) {
	args := r.Mock.MethodCalled("ProviderCount")
	return args.Get(0).(uint64), args.Error(1)
}

func (r *repoMock) TxCountOfAccount(_ string, _ uint64, _ float64, _ float64) (uint64, error) {
	args := r.Mock.MethodCalled("TxCountOfAccount")
	return args.Get(0).(uint64), args.Error(1)
}

func (r *repoMock) AssetAmountInPair(_ uint64, _ float64, _ float64) (string, string, string, error) {
	args := r.Mock.MethodCalled("AssetAmountInPair")
	return args.Get(0).(string), args.Get(1).(string), args.Get(2).(string), args.Error(3)
}

func (r *repoMock) AssetAmountInPairOfAccount(_ string, _ uint64, _ float64, _ float64) (string, string, string, error) {
	args := r.Mock.MethodCalled("AssetAmountInPairOfAccount")
	return args.Get(0).(string), args.Get(1).(string), args.Get(2).(string), args.Error(3)
}

func (r *repoMock) CommissionAmountInPair(_ uint64, _ float64, _ float64) (string, string, error) {
	args := r.Mock.MethodCalled("CommissionAmountInPair")
	return args.Get(0).(string), args.Get(1).(string), args.Error(2)
}

// aggregator/repo/repository.go interface
func (r *repoMock) LastHeightOfPairStatsRecent() (uint64, error) {
	return 0, nil
}

func (r *repoMock) GetParsedTxsWithLimit(_ uint64, _ int) ([]schemas.ParsedTxWithPrice, error) {
	args := r.Mock.MethodCalled("GetParsedTxsWithLimit")
	if r.calledGetParsedTxsWithLimit {
		return []schemas.ParsedTxWithPrice{}, args.Error(1)
	}

	r.calledGetParsedTxsWithLimit = true
	return args.Get(0).([]schemas.ParsedTxWithPrice), args.Error(1)
}

func (r *repoMock) LastLiquidity(_ uint64, _ float64) ([repo.TupleLength]string, error) {
	args := r.Mock.MethodCalled("LastLiquidity")
	return args.Get(0).([repo.TupleLength]string), args.Error(1)
}

func (r *repoMock) LastLpHistory(_ uint64) ([]schemas.LpHistory, error) {
	args := r.Mock.MethodCalled("LastLpHistory")
	return args.Get(0).([]schemas.LpHistory), args.Error(1)
}

func (r *repoMock) BeginTx() (*gorm.DB, error) {
	args := r.Mock.MethodCalled("BeginTx")
	return args.Get(0).(*gorm.DB), args.Error(1)
}

func (r *repoMock) UpdatePairStatsRecent(_ *gorm.DB, stats []schemas.PairStatsRecent) error {
	r.updatedPairStatsRecent = stats
	return nil
}

func (r *repoMock) PairStats(_ float64, _ float64, _ string) ([]schemas.PairStats30m, error) {
	args := r.Mock.MethodCalled("PairStats")
	return args.Get(0).([]schemas.PairStats30m), args.Error(1)
}

func (r *repoMock) AccountStats(_ float64, _ float64) ([]schemas.AccountStats30m, error) {
	args := r.Mock.MethodCalled("AccountStats")
	return args.Get(0).([]schemas.AccountStats30m), args.Error(1)
}

func (r *repoMock) LiquiditiesOfPairStats(_ float64, _ float64, _ string) (map[uint64]schemas.PairStats30m, error) {
	args := r.Mock.MethodCalled("LiquiditiesOfPairStats")
	return args.Get(0).(map[uint64]schemas.PairStats30m), args.Error(1)
}

func (r *repoMock) UpdateLpHistory(history []schemas.LpHistory) error {
	r.updatedLpHistory = history
	return nil
}

func (r *repoMock) DeletePairStatsRecent(_ *gorm.DB, _ time.Time) error {
	return nil
}

func (r *repoMock) DeleteDuplicates(_ time.Time) error {
	return nil
}

func (r *repoMock) UpdatePairStats(stats []schemas.PairStats30m) error {
	r.updatedPairStats = stats
	return nil
}

func (r *repoMock) UpdateAccountStats(stats []schemas.AccountStats30m) error {
	r.updatedAccountStats = stats
	return nil
}

func (r *repoMock) CreateAccounts(addresses []string) error {
	r.updatedAccounts = addresses
	return nil
}

func (r *repoMock) HoldingPairIds(_ uint64) ([]uint64, error) {
	args := r.Mock.MethodCalled("HoldingPairIds")
	return args.Get(0).([]uint64), args.Error(1)
}

func (r *repoMock) Accounts(_ float64) (map[uint64]string, error) {
	args := r.Mock.MethodCalled("Accounts")
	return args.Get(0).(map[uint64]string), args.Error(1)
}

func (r *repoMock) Close() error { return nil }
