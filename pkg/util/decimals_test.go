package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseDecWithDecimal(t *testing.T) {
	assert := assert.New(t)

	dec, err := StringAmountToDecimal("1000000", 6)

	assert.NoError(err)
	assert.Equal(dec.String(), "1.000000000000000000")
}

func TestParseDecWithDecimal_Negative(t *testing.T) {
	assert := assert.New(t)

	dec, err := StringAmountToDecimal("-1000000", 6)

	assert.NoError(err)
	assert.Equal(dec.String(), "-1.000000000000000000")
}
