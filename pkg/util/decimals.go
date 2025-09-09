package util

import (
	"cosmossdk.io/math"
	"strconv"
)

func StringAmountToDecimal(amount string, decimals int64) (math.LegacyDec, error) {
	amountD, err := math.LegacyNewDecFromStr(amount)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return amountD.Quo(math.LegacyNewDec(10).Power(uint64(decimals))), nil
}

func ExponentToDecimal(value string) (math.LegacyDec, error) {
	lsp, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return math.LegacyDec{}, err
	}

	valueStr := strconv.FormatFloat(lsp, 'f', -1, 64)
	return math.LegacyNewDecFromStr(valueStr)
}
