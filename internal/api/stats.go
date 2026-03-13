package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/config"
	"github.com/skylink/skylink/internal/easytier"
)

func (s *Server) stats(c *gin.Context) {
	mappings, _ := s.store.ListMappings()
	ddnsList, _ := s.store.ListDDNSConfigs(0)

	cfAccountsCount := 0
	if accounts, err := s.store.ListCFAccounts(); err == nil {
		cfAccountsCount = len(accounts)
	}

	easytierEnabled := false
	easytierHasRuntime := false
	easytierDaemonRunning := false

	if cfg, err := s.store.GetEasyTierConfig(); err == nil && cfg != nil {
		easytierEnabled = cfg.Enabled
		version := strings.TrimSpace(cfg.ImageTag)
		if version == "" {
			version = config.DefaultEasyTierTag
		}
		if s.easyTierRuntime != nil {
			easytierHasRuntime = s.easyTierRuntime.HasDaemon(version, easytier.CurrentPlatform())
		}
	}

	if s.easyTierDaemon != nil && s.easyTierCfg != nil && s.easyTierCfg.DaemonEnabled {
		state := s.easyTierDaemon.Status()
		easytierDaemonRunning = state.Running
	}

	c.JSON(http.StatusOK, gin.H{
		"mappings_count":          len(mappings),
		"ddns_count":              len(ddnsList),
		"cf_accounts_count":       cfAccountsCount,
		"easytier_enabled":        easytierEnabled,
		"easytier_daemon_running": easytierDaemonRunning,
		"easytier_has_runtime":    easytierHasRuntime,
	})
}
