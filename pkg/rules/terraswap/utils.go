package terraswap

func IsFactoryAddress(address string) bool {
	for _, v := range FactoryAddress {
		if v == address {
			return true
		}
	}
	return false
}

func TerraswapTypeOf(address string) TerraswapType {
	for t, v := range FactoryAddress {
		if v == address {
			return t
		}
	}
	return InvalidType
}
