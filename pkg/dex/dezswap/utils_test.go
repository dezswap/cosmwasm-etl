package dezswap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getFactoryAddress(t *testing.T) {

	tcs := []struct {
		chainId         string
		expectedAddress string
		errMsg          string
	}{
		{"empty", "", "there is no chain named empty"},
		{"dimension_37-1", FactoryAddress[MainnetPrefix], "must return dimension factory address"},
		{"cube_47-5", FactoryAddress[TestnetPrefix], "must return cube factory address"},
	}

	assert := assert.New(t)
	for _, tc := range tcs {
		addr := getFactoryAddress(tc.chainId)
		assert.Equal(tc.expectedAddress, addr, tc.errMsg)
	}
}
