package util

import (
	"strings"

	ds "github.com/dezswap/cosmwasm-etl/pkg/dex/dezswap"
	sf "github.com/dezswap/cosmwasm-etl/pkg/dex/starfleit"
)

const (
	NetworkXpla         = "XPLA Chain"
	NetworkTerra2       = "Terra 2.0"
	NetworkTerraClassic = "Terra Classic"
	NetworkASIAlliance  = "ASI Alliance"
	NetworkUnknown      = "unknown network"
)

const (
	phoenixPrefix  = "phoenix"
	piscoPrefix    = "pisco"
	columbusPrefix = "columbus"
)

func NetworkNameByChainID(chainId string) string {
	switch {
	case strings.HasPrefix(chainId, ds.MainnetPrefix), strings.HasPrefix(chainId, ds.TestnetPrefix):
		return NetworkXpla
	case strings.HasPrefix(chainId, sf.MainnetPrefix), strings.HasPrefix(chainId, sf.TestnetPrefix):
		return NetworkASIAlliance
	case strings.HasPrefix(chainId, phoenixPrefix), strings.HasPrefix(chainId, piscoPrefix):
		return NetworkTerra2
	case strings.HasPrefix(chainId, columbusPrefix):
		return NetworkTerraClassic
	default:
		return NetworkUnknown
	}
}
