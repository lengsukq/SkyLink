package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
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

	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			var file struct {
				App      *App        `yaml:"app"`
				CF       *Cloudflare `yaml:"cloudflare"`
				EasyTier *EasyTier   `yaml:"easytier"`
			}
			var raw struct {
				App      map[string]any `yaml:"app"`
				EasyTier map[string]any `yaml:"easytier"`
			}
			if err := yaml.Unmarshal(data, &file); err == nil {
				_ = yaml.Unmarshal(data, &raw)
				appHasRequireAdmin := raw.App != nil && hasYAMLKey(raw.App, "require_admin")
				easyTierHasEnabled := raw.EasyTier != nil && hasYAMLKey(raw.EasyTier, "enabled")
				easyTierHasDaemonEnabled := raw.EasyTier != nil && hasYAMLKey(raw.EasyTier, "daemon_enabled")
				if file.App != nil {
					mergeApp(app, file.App, appHasRequireAdmin)
				}
				if file.CF != nil {
					cf = file.CF
				}
				if file.EasyTier != nil {
					mergeEasyTier(et, file.EasyTier, easyTierHasEnabled, easyTierHasDaemonEnabled)
				}
			}
		}
	}

	// 环境变量覆盖
	if v := os.Getenv("SKYLINK_PROXY_PORT"); v != "" {
		if p := parseInt(v); p > 0 {
			app.ProxyPort = p
		}
	}
	if v := os.Getenv("SKYLINK_ADMIN_PORT"); v != "" {
		if p := parseInt(v); p > 0 {
			app.AdminPort = p
		}
	}
	if v := os.Getenv("SKYLINK_DB_PATH"); v != "" {
		app.DBPath = v
	}
	if v, ok := os.LookupEnv("SKYLINK_REQUIRE_ADMIN"); ok {
		app.RequireAdmin = truthyEnv(v)
	}
	if v := os.Getenv("SKYLINK_EASYTIER_RPC"); v != "" {
		et.RPCAddress = v
	}
	if v := os.Getenv("SKYLINK_EASYTIER_ENV_FILE"); v != "" {
		et.EnvFilePath = v
	}
	if v := os.Getenv("SKYLINK_EASYTIER_DAEMON_ENABLED"); v != "" {
		et.DaemonEnabled = v == "1" || strings.EqualFold(v, "true")
	}
	if v := os.Getenv("SKYLINK_EASYTIER_DAEMON_PATH"); v != "" {
		et.DaemonPath = v
	}
	if v := os.Getenv("SKYLINK_EASYTIER_RUNTIME_DIR"); v != "" {
		et.RuntimeDir = v
	}

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
