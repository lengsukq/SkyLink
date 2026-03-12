package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Load 从环境变量和可选 YAML 文件加载配置
func Load(path string) (*App, *Cloudflare, error) {
	app := &App{
		ProxyPort:  DefaultProxyPort,
		AdminPort:  DefaultAdminPort,
		DBPath:     DefaultDBPath,
		ConfigPath: path,
	}
	cf := &Cloudflare{}

	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			var file struct {
				App *App         `yaml:"app"`
				CF  *Cloudflare  `yaml:"cloudflare"`
			}
			if err := yaml.Unmarshal(data, &file); err == nil {
				if file.App != nil {
					mergeApp(app, file.App)
				}
				if file.CF != nil {
					cf = file.CF
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
	if v := os.Getenv("SKYLINK_ADMIN_SECRET"); v != "" {
		app.AdminSecret = v
	}
	if v := os.Getenv("CF_API_TOKEN"); v != "" {
		cf.APIToken = v
	}
	if v := os.Getenv("CF_ZONE_ID"); v != "" {
		cf.ZoneID = v
	}

	return app, cf, nil
}

func mergeApp(dst, src *App) {
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
	if src.AdminSecret != "" {
		dst.AdminSecret = src.AdminSecret
	}
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
