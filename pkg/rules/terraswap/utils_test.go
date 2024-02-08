package terraswap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isFactoryAddr(t *testing.T) {

	tcs := []struct {
		factoryAddr string
		expected    bool
		errMsg      string
	}{
		{FactoryAddress["empty"], false, "there is no chain named empty"},
		{FactoryAddress[Mainnet], true, "must return true for phoenix factory address"},
		{FactoryAddress[Pisco], true, "must return true for pisco factory address"},
		{FactoryAddress[ClassicV1], true, "must return true for columbus factory address"},
		{FactoryAddress[ClassicV2], true, "must return true for columbus factory address"},
	}

	assert := assert.New(t)
	for _, tc := range tcs {
		actual := IsFactoryAddress(tc.factoryAddr)
		assert.Equal(tc.expected, actual, tc.errMsg)
	}
}
