package dezswap

import "strings"

func getFactoryAddress(chainId string) string {
	prefix := strings.Split(chainId, "-")[0]
	return FactoryAddress[prefix]
}
