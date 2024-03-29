package util

import (
	"github.com/cosmos/cosmos-sdk/types"
	"strconv"
)

func StringAmountToDecimal(amount string, decimals int64) (types.Dec, error) {
	amountD, err := types.NewDecFromStr(amount)
	if err != nil {
		return types.Dec{}, err
	}

	return amountD.Quo(types.NewDec(10).Power(uint64(decimals))), nil
}

func ExponentToDecimal(value string) (types.Dec, error) {
	lsp, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return types.Dec{}, err
	}

	valueStr := strconv.FormatFloat(lsp, 'f', -1, 64)
	return types.NewDecFromStr(valueStr)
}
