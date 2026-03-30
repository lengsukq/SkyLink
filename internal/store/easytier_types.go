package store

// EasyTierConfig 供 API 和 env 文件使用的配置结构
type EasyTierConfig struct {
	NetworkName   string `json:"network_name"`
	NetworkSecret string `json:"network_secret"`
	Peers         string `json:"peers"`
	IPv4          string `json:"ipv4"`
	Enabled       bool   `json:"enabled"`
	ImageTag      string `json:"image_tag"`
	RPCPortal     string `json:"rpc_portal"`
	EnvFilePath   string `json:"env_file_path"`
	Hostname      string `json:"hostname"`
	ExternalNode  string `json:"external_node"`
	ProxyNetworks string `json:"proxy_networks"`
	DHCP          bool   `json:"dhcp"`
}

type EasyTierProfile struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Config EasyTierConfig `json:"config"`
}

type EasyTierProfiles struct {
	Profiles        []EasyTierProfile `json:"profiles"`
	ActiveProfileID string            `json:"active_profile_id"`
}

