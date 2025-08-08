package srcstore

import (
	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	p_srcstore "github.com/dezswap/cosmwasm-etl/parser/srcstore"
)

type mapper interface {
	p_srcstore.Mapper
	rawPoolInfosToPoolInfos(rawPoolInfos *datastore.PoolInfoList) []dex.PoolInfo
	rawPoolInfoToPoolInfo(pairAddr string, rawPoolInfo datastore.PoolInfoWithLpAddr) dex.PoolInfo
}

type mapperImpl struct{ p_srcstore.Mapper }

func newMapper(m p_srcstore.Mapper) mapper {
	return &mapperImpl{m}
}

// blockToRawTxs implements mapper
// rawPoolInfosToPoolInfos implements mapper
func (m *mapperImpl) rawPoolInfosToPoolInfos(rawPoolInfos *datastore.PoolInfoList) []dex.PoolInfo {
	infos := []dex.PoolInfo{}
	for addr, rawPoolInfo := range rawPoolInfos.Pairs {
		infos = append(infos, m.rawPoolInfoToPoolInfo(addr, rawPoolInfo))
	}
	return infos
}

// rawPoolInfoToPoolInfo implements mapper
func (*mapperImpl) rawPoolInfoToPoolInfo(contractAddr string, rawPoolInfo datastore.PoolInfoWithLpAddr) dex.PoolInfo {
	poolInfo := dex.PoolInfo{
		ContractAddr: contractAddr,
		Assets: []dex.Asset{
			{
				Addr:   rawPoolInfo.Assets[0].Info.DenomOrAddress,
				Amount: rawPoolInfo.Assets[0].Amount.String(),
			},
			{
				Addr:   rawPoolInfo.Assets[1].Info.DenomOrAddress,
				Amount: rawPoolInfo.Assets[1].Amount.String(),
			},
		},
		LpAddr:     rawPoolInfo.LpAddr,
		TotalShare: rawPoolInfo.TotalShare.String(),
	}
	return poolInfo
}
