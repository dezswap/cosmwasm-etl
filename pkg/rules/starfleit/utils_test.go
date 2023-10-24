package starfleit

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
		{"fetchhub-4", FactoryAddress[MainnetPrefix], "must return fetchhub factory address"},
		{"dorado-5", FactoryAddress[TestnetPrefix], "must return dorado factory address"},
	}

	assert := assert.New(t)
	for _, tc := range tcs {
		addr := getFactoryAddress(tc.chainId)
		assert.Equal(tc.expectedAddress, addr, tc.errMsg)
	}
}
