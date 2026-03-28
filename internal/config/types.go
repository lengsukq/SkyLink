package config

// App 应用级配置（端口、数据路径等）
type App struct {
	ProxyPort  int    `yaml:"proxy_port"`  // 反代监听端口，默认 18080
	AdminPort  int    `yaml:"admin_port"`  // 管理 API + GUI 端口，默认 19080
	DBPath     string `yaml:"db_path"`     // SQLite 文件路径
	ConfigPath string `yaml:"config_path"` // 配置文件路径（可选）
	// RequireAdmin 仅 Windows 可选：为 true 时非管理员启动会触发 UAC 提权重开。跨平台默认关闭（macOS/Linux 忽略此项）。
	RequireAdmin bool `yaml:"require_admin"`
}

// Cloudflare 从环境变量或配置文件读取，敏感信息建议用 env
type Cloudflare struct {
	APIToken string `yaml:"api_token"` // CF API Token，权限 Zone.DNS Edit
	ZoneID   string `yaml:"zone_id"`   // 默认 Zone ID（可选）
}

// EasyTier 可选配置（RPC 地址、守护进程管理、可选 env 文件路径等）
type EasyTier struct {
	RPCAddress    string `yaml:"rpc_address"`    // EasyTier RPC 地址，守护进程监听的地址，如 127.0.0.1:15888
	Enabled       bool   `yaml:"enabled"`        // 是否启用 EasyTier 功能
	EnvFilePath   string `yaml:"env_file_path"`  // 写入的 env 文件路径（可选，仅在需与外部进程/容器共享配置时使用）
	DaemonEnabled bool   `yaml:"daemon_enabled"` // 是否由 SkyLink 直接管理 easytier-daemon 进程（推荐的裸机一站式场景）
	DaemonPath    string `yaml:"daemon_path"`    // easytier-daemon 二进制路径，空则在 PATH 中查找或由 RuntimeDownloader 决定
	RuntimeDir    string `yaml:"runtime_dir"`    // 自动下载 EasyTier 二进制的缓存目录（默认 ./data/easytier-bin）
}
