package terraswap

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
		{"phoenix-1", FactoryAddress[MainnetPrefix], "must return phoenix factory address"},
		{"columbus", FactoryAddress[ClassicPrefix], "must return columbus factory address"},
		{"pisco", FactoryAddress[TestnetPrefix], "must return pisco factory address"},
	}

	assert := assert.New(t)
	for _, tc := range tcs {
		addr := getFactoryAddress(tc.chainId)
		assert.Equal(tc.expectedAddress, addr, tc.errMsg)
	}
}
