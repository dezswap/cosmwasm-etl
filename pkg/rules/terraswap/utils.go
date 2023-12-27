package terraswap

func IsFactoryAddress(address string) bool {
	for _, v := range FactoryAddress {
		if v == address {
			return true
		}
	}
	return false
}
