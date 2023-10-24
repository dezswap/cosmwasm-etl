package router

type Pair struct {
	Contract   string   `json:"contract"`
	AssetInfos []string `json:"asset_infos"`
}
