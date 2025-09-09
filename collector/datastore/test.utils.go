package datastore

import (
	"cosmossdk.io/math"
	"math/rand"

	"github.com/dezswap/cosmwasm-etl/pkg/faker"
	"github.com/stretchr/testify/mock"
)

func FakeBlockDto() BlockTxsDTO {
	block := BlockTxsDTO{}
	tx := TxDTO{}
	block.Txs = append(block.Txs, tx)
	_ = faker.FakeData(&block.BlockId)
	return block
}

func FakePoolInfoList() PoolInfoList {
	poolInfos := PoolInfoList{}
	_ = faker.FakeData(&poolInfos)

	for _, pool := range poolInfos.Pairs {
		share := math.NewInt(rand.Int63())
		pool.TotalShare = &share
		for _, asset := range pool.Assets {
			amount := math.NewInt(rand.Int63())
			asset.Amount = &amount
		}
	}
	return poolInfos
}

type ReadStoreMock struct {
	mock.Mock
}

var _ ReadStore = &ReadStoreMock{}

// GetBlockByHeight implements ReadStore
func (m *ReadStoreMock) GetBlockByHeight(height uint64) (*BlockTxsDTO, error) {
	args := m.MethodCalled("GetBlockByHeight", mock.Anything)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BlockTxsDTO), args.Error(1)
}

// GetLatestHeight implements ReadStore
func (m *ReadStoreMock) GetLatestHeight() (uint64, error) {
	args := m.MethodCalled("GetLatestHeight")
	return args.Get(0).(uint64), args.Error(1)
}

// GetPoolStatusOfAllPairsByHeight implements ReadStore
func (m *ReadStoreMock) GetPoolStatusOfAllPairsByHeight(uint64) (*PoolInfoList, error) {
	args := m.MethodCalled("GetPoolStatusOfAllPairsByHeight", mock.Anything)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PoolInfoList), args.Error(1)
}
