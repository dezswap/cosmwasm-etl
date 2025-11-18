package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestExponentToDecimal_Truncate18(t *testing.T) {
	dec, err := ExponentToDecimal("1.123456789012345678901234") // 24 decimals
	require.NoError(t, err)
	require.Equal(t, "1.123456789012345678", dec.String())
}

func TestExponentToDecimal_TrimTrailingZeros(t *testing.T) {
	dec, err := ExponentToDecimal("1.2300000000000000001234") // 22 decimals
	require.NoError(t, err)
	require.Equal(t, "1.230000000000000000", dec.String())
}

func TestExponentToDecimal_WholeNumber(t *testing.T) {
	dec, err := ExponentToDecimal("42")
	require.NoError(t, err)
	require.Equal(t, "42.000000000000000000", dec.String())
}

func TestExponentToDecimal_OnlyFraction(t *testing.T) {
	dec, err := ExponentToDecimal(".1234567890123456789") // 19 decimals
	require.NoError(t, err)
	require.Equal(t, "0.123456789012345678", dec.String())
}

func TestExponentToDecimal_Negative(t *testing.T) {
	dec, err := ExponentToDecimal("-0.999999999999999999999") // 21 decimals
	require.NoError(t, err)
	require.Equal(t, "-0.999999999999999999", dec.String())
}

func TestExponentToDecimal_Empty(t *testing.T) {
	_, err := ExponentToDecimal("")
	require.Error(t, err)
}
