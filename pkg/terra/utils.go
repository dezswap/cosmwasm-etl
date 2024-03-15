package terra

import "strings"

func IsCw20(addr string) bool {
	return strings.HasPrefix(addr, CW20_PREFIX)
}

// IsNativeToken returns true if the given address is native token address.
// excepts CW20 token address every token is native token.
func IsNativeToken(addr string) bool {
	return !IsCw20(addr)
}
