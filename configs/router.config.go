package configs

type RouterConfig struct {
	Name        string `json:"name"`
	RouterAddr  string `json:"router_addr"`
	MaxHopCount uint   `json:"max_hop_count"`
	WriteDb     bool   `json:"write_db"`
}
