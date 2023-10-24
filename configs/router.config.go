package configs

type RouterConfig struct {
	Name       string `json:"name"`
	RouterAddr string `json:"router_addr"`
	MaxPathLen uint   `json:"max_path_len"`
	WriteDb    bool   `json:"write_db"`
}
