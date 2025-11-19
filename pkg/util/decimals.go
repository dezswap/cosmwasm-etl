package util

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"
)

const maxDecimalDigits = 18

func StringAmountToDecimal(amount string, decimals int64) (math.LegacyDec, error) {
	amountD, err := math.LegacyNewDecFromStr(amount)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return amountD.Quo(math.LegacyNewDec(10).Power(uint64(decimals))), nil
}

// ExponentToDecimal converts a numeric string into a LegacyDec.
// Note: Only up to 18 decimal places are used. Digits beyond the 18th
// decimal place are truncated and ignored.
func ExponentToDecimal(value string) (math.LegacyDec, error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return math.LegacyDec{}, fmt.Errorf("empty value")
	}

	// Extract optional sign
	sign := ""
	if s[0] == '+' || s[0] == '-' {
		sign = s[:1]
		s = s[1:]
		if s == "" {
			return math.LegacyDec{}, fmt.Errorf("invalid value: %s", value)
		}
	}

	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	if intPart == "" {
		intPart = "0"
	}

	// No fractional part
	if len(parts) == 1 {
		return math.LegacyNewDecFromStr(sign + intPart)
	}

	fracPart := parts[1]

	// Truncate to at most 18 decimal places (rounding toward zero)
	if len(fracPart) > maxDecimalDigits {
		fracPart = fracPart[:maxDecimalDigits]
	}

	return math.LegacyNewDecFromStr(sign + intPart + "." + fracPart)
}
