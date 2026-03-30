package config

import (
	"os"
	"strings"
)

func applyEnvOverrides(app *App, et *EasyTier) {
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
}
