package dex

type DexType string

const (
	Terraswap DexType = "terraswap"
	Dezswap   DexType = "dezswap"
	Starfleit DexType = "starfleit"
	Unknown   DexType = ""
)

func ToDexType(candidate string) DexType {
	switch DexType(candidate) {
	case Terraswap:
		return Terraswap
	case Dezswap:
		return Dezswap
	case Starfleit:
		return Starfleit
	default:
		return Unknown
	}
}
