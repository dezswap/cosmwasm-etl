package conx

import "strings"

func IsCw20(addr string) bool {
	return strings.HasPrefix(addr, Cw20Prefix)
}

// IsNativeToken returns true if the given address is native token address.
// excepts CW20 token address every token is native token.
func IsNativeToken(addr string) bool {
	return !IsCw20(addr)
}
