package terraswap

import (
	p_dex "github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/dex"
)

type mapper interface {
	dexPairToPair(pair *dex.Pair) p_dex.Pair
}
type mapperImpl struct{}

var _ mapper = &mapperImpl{}

// dexPairToPair implements mapper.
func (*mapperImpl) dexPairToPair(pair *dex.Pair) p_dex.Pair {
	p := p_dex.Pair{
		ContractAddr: pair.ContractAddr,
		LpAddr:       pair.LiquidityToken,
		Assets:       []string{},
	}
	for _, asset := range pair.AssetInfos {
		if asset.NativeToken != nil {
			p.Assets = append(p.Assets, asset.NativeToken.Denom)
		}
		if asset.Token != nil {
			p.Assets = append(p.Assets, asset.Token.ContractAddr)
		}
	}
	return p
}
