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
		&mapperImpl{},
		&s.mock,
	}
}

func (s *rawStoreTestSuite) Test_GetSourceSyncedHeight() {
	tcs := []struct {
		expectedHeight uint64
		err            error
	}{
		{0, nil},
		{10, nil},
		{0, errors.New("must fail")},
	}

	for idx, tc := range tcs {
		s.mock.On("GetLatestHeight").Return(tc.expectedHeight, tc.err).Once()
		assert := assert.New(s.T())
		actual, actualErr := s.store.GetSourceSyncedHeight()
		if tc.err != nil {
			assert.Equal(actual, uint64(0), "must return 0")
			assert.Error(actualErr)
		} else {
			assert.Equal(uint64(tc.expectedHeight), actual, fmt.Sprintf("idx(%d) has different height", idx))
		}
	}
}

func (s *rawStoreTestSuite) Test_GetSourceTxs() {
	for idx := 0; idx < 10; idx++ {
		block := datastore.FakeBlockDto()
		s.mock.On("GetBlockByHeight", mock.Anything).Return(&block, nil).Once()

		rawTxs, err := s.store.GetSourceTxs(uint64(idx))
		if err != nil {
			panic(err)
		}
		assert.Len(s.T(), rawTxs, len(block.Txs), fmt.Sprintf("tc(%d): must return length of txs", idx))
	}

	// fail case
	s.mock.On("GetBlockByHeight", mock.Anything).Return(nil, errors.New("must fail")).Once()

	_, err := s.store.GetSourceTxs(uint64(100))

	assert.Error(s.T(), err)
}

func Test_readStore(t *testing.T) {
	suite.Run(t, new(rawStoreTestSuite))
}
