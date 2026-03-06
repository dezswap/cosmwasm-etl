package configs

type RouterConfig struct {
	Name        string `json:"name"          mapstructure:"name"`
	RouterAddr  string `json:"router_addr"   mapstructure:"router_addr"`
	MaxHopCount uint   `json:"max_hop_count" mapstructure:"max_hop_count"`
	WriteDb     bool   `json:"write_db"      mapstructure:"write_db"`
}
