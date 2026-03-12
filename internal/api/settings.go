package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	settingFrpCnameTarget  = "frp.cname_target"
	settingCFCnameProxied  = "cf.cname_proxied"
)

func (s *Server) getSettings(c *gin.Context) {
	frp, _ := s.store.GetSetting(settingFrpCnameTarget)
	proxied, _ := s.store.GetSetting(settingCFCnameProxied)
	c.JSON(http.StatusOK, gin.H{
		"frp_cname_target": strings.TrimSpace(frp),
		"cf_cname_proxied": parseBoolDefault(proxied, true),
	})
}

func (s *Server) updateSettings(c *gin.Context) {
	var req struct {
		FRPCNAMETarget string `json:"frp_cname_target"`
		CFCNAMEProxied *bool  `json:"cf_cname_proxied"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	frp := normalizeDomain(req.FRPCNAMETarget)
	// 允许清空：写入空字符串
	if err := s.store.SetSetting(settingFrpCnameTarget, frp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.CFCNAMEProxied != nil {
		if err := s.store.SetSetting(settingCFCnameProxied, formatBool(*req.CFCNAMEProxied)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func normalizeDomain(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".")
	return s
}

func parseBoolDefault(v string, def bool) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "true" || v == "1" || v == "yes" || v == "on" {
		return true
	}
	if v == "false" || v == "0" || v == "no" || v == "off" {
		return false
	}
	return def
}

func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

