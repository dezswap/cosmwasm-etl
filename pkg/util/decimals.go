package util

import (
	"github.com/cosmos/cosmos-sdk/types"
)

func StringAmountToDecimal(amount string, decimals int64) (types.Dec, error) {
	amountD, err := types.NewDecFromStr(amount)
	if err != nil {
		return types.Dec{}, err
	}

	return amountD.Quo(types.NewDec(10).Power(uint64(decimals))), nil
}
