package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

func loadFromYAML(path string, app *App, cf *Cloudflare, et *EasyTier) *Cloudflare {
	if path == "" {
		return cf
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cf
	}

	var file struct {
		App      *App        `yaml:"app"`
		CF       *Cloudflare `yaml:"cloudflare"`
		EasyTier *EasyTier   `yaml:"easytier"`
	}
	var raw struct {
		App      map[string]any `yaml:"app"`
		EasyTier map[string]any `yaml:"easytier"`
	}

	if err := yaml.Unmarshal(data, &file); err != nil {
		return cf
	}
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
	return cf
}
