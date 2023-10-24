package datastore

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/faker"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type readStoreTestSuite struct {
	suite.Suite
	s3        *s3ClientMock
	readStore readStore
}

func (s *readStoreTestSuite) SetupSuite() {
	mock := s3ClientMock{}
	s.readStore = readStore{chainId: "test", s3Client: &mock}
	s.s3 = &mock
}

func (s *readStoreTestSuite) Test_GetLatestHeight() {
	tcs := []struct {
		expectedHeight int64
		err            error
	}{
		{0, nil},
		{10, nil},
		{-1, errors.New("must fail")},
	}

	for idx, tc := range tcs {
		s.s3.Mock.On("GetLatestProcessedBlockNumber", mock.Anything, mock.Anything).Return(tc.expectedHeight, tc.err).Once()
		assert := assert.New(s.T())
		actual, actualErr := s.readStore.GetLatestHeight()
		if tc.err != nil {
			assert.Equal(actual, uint64(0), "must return 0")
			assert.Error(actualErr)
		} else {
			assert.Equal(uint64(tc.expectedHeight), actual, fmt.Sprintf("idx(%d) has different height", idx))
		}
	}
}

func (s *readStoreTestSuite) Test_GetBlockByHeight_Success() {
	block := FakeBlockDto()
	mockData, err := json.Marshal(block)
	if err != nil {
		panic(err)
	}
	s.s3.Mock.On("GetFileFromS3", mock.Anything).Return(mockData, nil).Once()

	actualBlock, err := s.readStore.GetBlockByHeight(uint64(10))
	if err != nil {
		panic(err)
	}

	data, err := json.Marshal(actualBlock)
	if err != nil {
		panic(err)
	}

	assert.Equal(s.T(), data, mockData, "must return the same block")
}

func (s *readStoreTestSuite) Test_GetBlockByHeight_Fail() {
	for idx := 0; idx < 10; idx++ {
		err := errors.New("must fail")
		s.s3.Mock.On("GetFileFromS3", mock.Anything).Return([]byte{}, err).Once()

		_, actual := s.readStore.GetBlockByHeight(uint64(idx))

		assert.Error(s.T(), actual, fmt.Sprintf("idx(%d) must have error", idx))
	}
}

func (s *readStoreTestSuite) Test_GetPoolStatusOfAllPairsByHeight() {
	for idx := 0; idx < 10; idx++ {
		poolInfos := FakePoolInfoList()
		mockData, err := json.Marshal(poolInfos)
		if err != nil {
			panic(err)
		}
		s.s3.Mock.On("GetFileFromS3", mock.Anything).Return(mockData, nil).Once()

		actualBlock, err := s.readStore.GetPoolStatusOfAllPairsByHeight(uint64(idx))
		if err != nil {
			panic(err)
		}

		data, err := json.Marshal(actualBlock)
		if err != nil {
			panic(err)
		}
		assert.Equal(s.T(), data, mockData, fmt.Sprintf("tc(%d): must return the same block", idx))
	}
	// fail case
	s.s3.Mock.On("GetFileFromS3", mock.Anything).Return([]byte{}, errors.New("fail")).Once()
	_, err := s.readStore.GetPoolStatusOfAllPairsByHeight(uint64(10))
	assert.Error(s.T(), err)

}

func Test_readStore(t *testing.T) {
	faker.CustomGenerator()
	suite.Run(t, new(readStoreTestSuite))
}
