package easytier

// Peer 表示 mesh 中的一个对等节点（来自 easytier-cli peer）
type Peer struct {
	IPv4     string  `json:"ipv4"`
	Hostname string  `json:"hostname"`
	Cost     int     `json:"cost"`
	Latency  float64 `json:"latency_ms"`
	Loss     string  `json:"loss"`
	Tunnel   string  `json:"tunnel"`
	NAT      string  `json:"nat"`
	Version  string  `json:"version"`
}

// Route 表示路由表项（来自 easytier-cli route）
type Route struct {
	IPv4           string  `json:"ipv4"`
	Hostname       string  `json:"hostname"`
	ProxyCIDRs     string  `json:"proxy_cidrs"`
	NextHopIPv4    string  `json:"next_hop_ipv4"`
	NextHopHost    string  `json:"next_hop_hostname"`
	NextHopLatency float64 `json:"next_hop_lat"`
	Cost           int     `json:"cost"`
}

// Node 表示本机节点信息（来自 easytier-cli node 或 peer 中的本地项）
type Node struct {
	IPv4     string `json:"ipv4"`
	Hostname string `json:"hostname"`
	Version  string `json:"version"`
}

// Status 为 GET /api/easytier/status 的响应结构
type Status struct {
	OK       bool    `json:"ok"`
	Error    string  `json:"error,omitempty"`
	Version  string  `json:"version,omitempty"`
	SelfIPv4 string  `json:"self_ipv4,omitempty"`
	SelfHost string  `json:"self_hostname,omitempty"`
	Peers    []Peer  `json:"peers"`
	Routes   []Route `json:"routes"`
}

// VersionCheck 为 GET /api/easytier/version/check 的响应结构
type VersionCheck struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url,omitempty"`
	ReleaseNotesURL string `json:"release_notes_url,omitempty"`
}
