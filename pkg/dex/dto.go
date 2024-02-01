package dex

type ContractQueryRequest interface {
	FactoryPairsReq | PoolInfoReq
}

type FactoryPairsReq struct {
	Pairs struct {
		Limit      int           `json:"limit,omitempty"`
		StartAfter *[2]AssetInfo `json:"start_after,omitempty"`
	} `json:"pairs"`
}

type FactoryPairsRes struct {
	Pairs []Pair `json:"pairs"`
}

type Pair struct {
	AssetInfos     [2]AssetInfo `json:"asset_infos"`
	ContractAddr   string       `json:"contract_addr"`
	LiquidityToken string       `json:"liquidity_token"`
	AssetDecimals  *[2]uint8    `json:"asset_decimals,omitempty"`
}

type Asset struct {
	Info   AssetInfo `json:"info"`
	Amount string    `json:"amount"`
}

type AssetInfo struct {
	Token       *AssetCWToken     `json:"token,omitempty"`
	NativeToken *AssetNativeToken `json:"native_token,omitempty"`
}

type AssetCWToken struct {
	ContractAddr string `json:"contract_addr,omitempty"`
}
type AssetNativeToken struct {
	Denom string `json:"denom,omitempty"`
}

type PoolInfoReq struct {
	Pool struct{} `json:"pool"`
}

type PoolInfoRes struct {
	Assets     []Asset `json:"assets"`
	TotalShare string  `json:"total_share"`
}
