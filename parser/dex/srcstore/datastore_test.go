package srcstore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type rawStoreTestSuite struct {
	suite.Suite
	mock  datastore.ReadStoreMock
	store rawDataStoreImpl
}

func (s *rawStoreTestSuite) SetupSuite() {
	s.mock = datastore.ReadStoreMock{}
	s.store = rawDataStoreImpl{
		nil,
		&mapperImpl{},
		&s.mock,
	}
}

func (s *rawStoreTestSuite) Test_GetPoolInfos() {
	for idx := 0; idx < 10; idx++ {
		fakedPoolInfo := datastore.FakePoolInfoList()
		s.mock.On("GetPoolStatusOfAllPairsByHeight", mock.Anything).Return(&fakedPoolInfo, nil).Once()

		poolInfos, err := s.store.GetPoolInfos(uint64(idx))
		if err != nil {
			panic(err)
		}
		assert.Len(s.T(), poolInfos, len(fakedPoolInfo.Pairs), fmt.Sprintf("tc(%d): must return length of txs", idx))
	}

	// fail case
	s.mock.On("GetPoolStatusOfAllPairsByHeight", mock.Anything).Return(nil, errors.New("must fail")).Once()

	_, err := s.store.GetPoolInfos(uint64(100))

	assert.Error(s.T(), err)
}

func Test_readStore(t *testing.T) {
	suite.Run(t, new(rawStoreTestSuite))
}
