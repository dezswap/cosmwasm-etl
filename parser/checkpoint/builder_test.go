package checkpoint

import (
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

// MockRepo implements pdex.Repo for testing
type MockRepo struct {
	syncedHeight uint64
	poolInfos    []dex.PoolInfo
}

func (m *MockRepo) GetSyncedHeight() (uint64, error) {
	return m.syncedHeight, nil
}

func (m *MockRepo) ParsedPoolsInfo(_, _ uint64) ([]dex.PoolInfo, error) {
	return m.poolInfos, nil
}

func (m *MockRepo) Insert(_, _ uint64, _ []dex.ParsedTx, _ ...interface{}) error {
	return nil
}

func (m *MockRepo) InsertPairValidationException(_ string, _ string) error {
	return nil
}

func (m *MockRepo) GetPairs() (map[string]dex.Pair, error) {
	return nil, nil
}

func (m *MockRepo) ValidationExceptionList() ([]string, error) {
	return nil, nil
}

// MockSourceDataStore implements pdex.SourceDataStore for testing
type MockSourceDataStore struct {
	syncedHeight uint64
	poolInfos    []dex.PoolInfo
}

func (m *MockSourceDataStore) GetSourceSyncedHeight() (uint64, error) {
	return m.syncedHeight, nil
}

func (m *MockSourceDataStore) GetSourceTxs(_ uint64) (parser.RawTxs, error) {
	return nil, nil
}

func (m *MockSourceDataStore) GetPoolInfos(_ uint64) ([]dex.PoolInfo, error) {
	return m.poolInfos, nil
}

func TestValidateAndGetDbHeight(t *testing.T) {
	tests := []struct {
		name         string
		dbHeight     uint64
		sourceHeight uint64
		targetHeight uint64
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "valid heights",
			dbHeight:     100,
			sourceHeight: 200,
			targetHeight: 150,
			wantErr:      false,
		},
		{
			name:         "db ahead of source",
			dbHeight:     200,
			sourceHeight: 100,
			targetHeight: 150,
			wantErr:      true,
			errMsg:       "database height is ahead of source node height",
		},
		{
			name:         "checkpoint exists",
			dbHeight:     150,
			sourceHeight: 200,
			targetHeight: 150,
			wantErr:      true,
			errMsg:       "checkpoint already exists for target height",
		},
		{
			name:         "target beyond source",
			dbHeight:     100,
			sourceHeight: 150,
			targetHeight: 200,
			wantErr:      true,
			errMsg:       "target height is beyond source node's current height(150)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockRepo{syncedHeight: tt.dbHeight}
			ds := &MockSourceDataStore{syncedHeight: tt.sourceHeight}
			builder := NewBuilder(repo, ds)

			height, err := builder.validateAndGetDbHeight(tt.targetHeight)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.dbHeight, height)
			}
		})
	}
}

func TestCalculateAssetDiff(t *testing.T) {
	tests := []struct {
		name     string
		src      []dex.Asset
		rdb      []dex.Asset
		expected []asset
		wantErr  bool
	}{
		{
			name: "Normal asset difference calculation",
			src: []dex.Asset{
				{Addr: "asset1", Amount: "100"},
				{Addr: "asset2", Amount: "200"},
			},
			rdb: []dex.Asset{
				{Addr: "asset1", Amount: "50"},
				{Addr: "asset2", Amount: "150"},
			},
			expected: []asset{
				{Addr: "asset1", Amount: big.NewInt(50)},
				{Addr: "asset2", Amount: big.NewInt(50)},
			},
			wantErr: false,
		},
		{
			name: "Zero assets",
			src: []dex.Asset{
				{Addr: "asset1", Amount: "0"},
				{Addr: "asset2", Amount: "0"},
			},
			rdb: []dex.Asset{
				{Addr: "asset1", Amount: "0"},
				{Addr: "asset2", Amount: "0"},
			},
			expected: []asset{
				{Addr: "asset1", Amount: big.NewInt(0)},
				{Addr: "asset2", Amount: big.NewInt(0)},
			},
			wantErr: false,
		},
		{
			name: "Decreased assets",
			src: []dex.Asset{
				{Addr: "asset1", Amount: "50"},
				{Addr: "asset2", Amount: "100"},
			},
			rdb: []dex.Asset{
				{Addr: "asset1", Amount: "100"},
				{Addr: "asset2", Amount: "200"},
			},
			expected: []asset{
				{Addr: "asset1", Amount: big.NewInt(-50)},
				{Addr: "asset2", Amount: big.NewInt(-100)},
			},
			wantErr: false,
		},
		{
			name: "Invalid amount format",
			src: []dex.Asset{
				{Addr: "asset1", Amount: "invalid"},
				{Addr: "asset2", Amount: "200"},
			},
			rdb: []dex.Asset{
				{Addr: "asset1", Amount: "50"},
				{Addr: "asset2", Amount: "150"},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "Non-matching asset addresses",
			src: []dex.Asset{
				{Addr: "asset1", Amount: "100"},
				{Addr: "asset2", Amount: "200"},
			},
			rdb: []dex.Asset{
				{Addr: "asset3", Amount: "50"},
				{Addr: "asset4", Amount: "150"},
			},
			expected: []asset{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calculateAssetDiff(tt.src, tt.rdb)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(result))

			for i, expected := range tt.expected {
				assert.Equal(t, expected.Addr, result[i].Addr)
				assert.Equal(t, expected.Amount.String(), result[i].Amount.String())
			}
		})
	}
}

func TestCreateDiffTx(t *testing.T) {
	tests := []struct {
		name     string
		src      dex.PoolInfo
		rdb      dex.PoolInfo
		expected *dex.ParsedTx
		wantErr  bool
	}{
		{
			name: "Swap transaction",
			src: dex.PoolInfo{
				ContractAddr: "pool1",
				Assets: []dex.Asset{
					{Addr: "asset1", Amount: "100"},
					{Addr: "asset2", Amount: "50"},
				},
				TotalShare: "1000",
				LpAddr:     "lp1",
			},
			rdb: dex.PoolInfo{
				ContractAddr: "pool1",
				Assets: []dex.Asset{
					{Addr: "asset1", Amount: "50"},
					{Addr: "asset2", Amount: "100"},
				},
				TotalShare: "1000",
				LpAddr:     "lp1",
			},
			expected: &dex.ParsedTx{
				Hash:         "-",
				Timestamp:    time.Now(),
				Type:         dex.Swap,
				Sender:       "-",
				ContractAddr: "pool1",
				Assets: [2]dex.Asset{
					{Addr: "asset1", Amount: "50"},
					{Addr: "asset2", Amount: "-50"},
				},
				LpAddr:           "lp1",
				LpAmount:         "0",
				CommissionAmount: "0",
			},
			wantErr: false,
		},
		{
			name: "Provide transaction",
			src: dex.PoolInfo{
				ContractAddr: "pool1",
				Assets: []dex.Asset{
					{Addr: "asset1", Amount: "100"},
					{Addr: "asset2", Amount: "100"},
				},
				TotalShare: "1000",
				LpAddr:     "lp1",
			},
			rdb: dex.PoolInfo{
				ContractAddr: "pool1",
				Assets: []dex.Asset{
					{Addr: "asset1", Amount: "50"},
					{Addr: "asset2", Amount: "50"},
				},
				TotalShare: "500",
				LpAddr:     "lp1",
			},
			expected: &dex.ParsedTx{
				Hash:         "-",
				Timestamp:    time.Now(),
				Type:         dex.Provide,
				Sender:       "-",
				ContractAddr: "pool1",
				Assets: [2]dex.Asset{
					{Addr: "asset1", Amount: "50"},
					{Addr: "asset2", Amount: "50"},
				},
				LpAddr:           "lp1",
				LpAmount:         "500",
				CommissionAmount: "0",
			},
			wantErr: false,
		},
		{
			name: "Withdraw transaction",
			src: dex.PoolInfo{
				ContractAddr: "pool1",
				Assets: []dex.Asset{
					{Addr: "asset1", Amount: "50"},
					{Addr: "asset2", Amount: "50"},
				},
				TotalShare: "500",
				LpAddr:     "lp1",
			},
			rdb: dex.PoolInfo{
				ContractAddr: "pool1",
				Assets: []dex.Asset{
					{Addr: "asset1", Amount: "100"},
					{Addr: "asset2", Amount: "100"},
				},
				TotalShare: "1000",
				LpAddr:     "lp1",
			},
			expected: &dex.ParsedTx{
				Hash:         "-",
				Timestamp:    time.Now(),
				Type:         dex.Withdraw,
				Sender:       "-",
				ContractAddr: "pool1",
				Assets: [2]dex.Asset{
					{Addr: "asset1", Amount: "-50"},
					{Addr: "asset2", Amount: "-50"},
				},
				LpAddr:           "lp1",
				LpAmount:         "-500",
				CommissionAmount: "0",
			},
			wantErr: false,
		},
		{
			name: "No changes",
			src: dex.PoolInfo{
				ContractAddr: "pool1",
				Assets: []dex.Asset{
					{Addr: "asset1", Amount: "100"},
					{Addr: "asset2", Amount: "100"},
				},
				TotalShare: "1000",
				LpAddr:     "lp1",
			},
			rdb: dex.PoolInfo{
				ContractAddr: "pool1",
				Assets: []dex.Asset{
					{Addr: "asset1", Amount: "100"},
					{Addr: "asset2", Amount: "100"},
				},
				TotalShare: "1000",
				LpAddr:     "lp1",
			},
			expected: nil,
			wantErr:  false,
		},
		{
			name: "Invalid total share format",
			src: dex.PoolInfo{
				ContractAddr: "pool1",
				Assets: []dex.Asset{
					{Addr: "asset1", Amount: "100"},
					{Addr: "asset2", Amount: "100"},
				},
				TotalShare: "invalid",
				LpAddr:     "lp1",
			},
			rdb: dex.PoolInfo{
				ContractAddr: "pool1",
				Assets: []dex.Asset{
					{Addr: "asset1", Amount: "100"},
					{Addr: "asset2", Amount: "100"},
				},
				TotalShare: "1000",
				LpAddr:     "lp1",
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := createDiffTx(tt.src, tt.rdb)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.expected == nil {
				assert.Nil(t, result)
				return
			}

			assert.NotNil(t, result)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.ContractAddr, result.ContractAddr)
			assert.Equal(t, tt.expected.LpAddr, result.LpAddr)
			assert.Equal(t, tt.expected.LpAmount, result.LpAmount)

			for i, expectedAsset := range tt.expected.Assets {
				assert.Equal(t, expectedAsset.Addr, result.Assets[i].Addr)
				assert.Equal(t, expectedAsset.Amount, result.Assets[i].Amount)
			}
		})
	}
}
