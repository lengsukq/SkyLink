package config

// App 应用级配置（端口、数据路径等）
type App struct {
	ProxyPort   int    `yaml:"proxy_port"`   // 反代监听端口，默认 18080
	AdminPort   int    `yaml:"admin_port"`   // 管理 API + GUI 端口，默认 19080
	DBPath      string `yaml:"db_path"`      // SQLite 文件路径
	ConfigPath  string `yaml:"config_path"`  // 配置文件路径（可选）
}

// Cloudflare 从环境变量或配置文件读取，敏感信息建议用 env
type Cloudflare struct {
	APIToken string `yaml:"api_token"` // CF API Token，权限 Zone.DNS Edit
	ZoneID   string `yaml:"zone_id"`   // 默认 Zone ID（可选）
}
