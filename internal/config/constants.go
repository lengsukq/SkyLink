package config

const (
	DefaultProxyPort = 18080
	DefaultAdminPort = 19080
	DefaultDBPath    = "./data/skylink.db"
	// DefaultEasyTierRPC 为 SkyLink 连接 EasyTier 守护进程的默认地址。
	// 以守护进程模式为主，默认假定守护进程在本机监听 15888 端口。
	DefaultEasyTierRPC        = "127.0.0.1:15888"
	DefaultEasyTierTag        = "latest"
	DefaultEasyTierDaemonPath = "easytier-core"
	DefaultEasyTierRuntimeDir = "./data/easytier-bin"
)
