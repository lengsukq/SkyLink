package config

import (
	"strings"
)

// Load 从环境变量和可选 YAML 文件加载配置
func Load(path string) (*App, *Cloudflare, *EasyTier, error) {
	app := &App{
		ProxyPort:  DefaultProxyPort,
		AdminPort:  DefaultAdminPort,
		DBPath:     DefaultDBPath,
		ConfigPath: path,
	}
	cf := &Cloudflare{}
	et := &EasyTier{
		RPCAddress:    DefaultEasyTierRPC,
		DaemonEnabled: true,
		DaemonPath:    DefaultEasyTierDaemonPath,
		RuntimeDir:    DefaultEasyTierRuntimeDir,
	}

	cf = loadFromYAML(path, app, cf, et)
	applyEnvOverrides(app, et)

	return app, cf, et, nil
}

func mergeApp(dst, src *App, hasRequireAdmin bool) {
	if src.ProxyPort > 0 {
		dst.ProxyPort = src.ProxyPort
	}
	if src.AdminPort > 0 {
		dst.AdminPort = src.AdminPort
	}
	if src.DBPath != "" {
		dst.DBPath = src.DBPath
	}
	if src.ConfigPath != "" {
		dst.ConfigPath = src.ConfigPath
	}
	if hasRequireAdmin {
		dst.RequireAdmin = src.RequireAdmin
	}
}

func truthyEnv(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func mergeEasyTier(dst, src *EasyTier, hasEnabled bool, hasDaemonEnabled bool) {
	if src.RPCAddress != "" {
		dst.RPCAddress = src.RPCAddress
	}
	if src.EnvFilePath != "" {
		dst.EnvFilePath = src.EnvFilePath
	}
	if hasEnabled {
		dst.Enabled = src.Enabled
	}
	if hasDaemonEnabled {
		dst.DaemonEnabled = src.DaemonEnabled
	}
	if src.DaemonPath != "" {
		dst.DaemonPath = src.DaemonPath
	}
	if src.RuntimeDir != "" {
		dst.RuntimeDir = src.RuntimeDir
	}
}

func hasYAMLKey(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
