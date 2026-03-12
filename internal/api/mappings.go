package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) listMappings(c *gin.Context) {
	list, err := s.store.ListMappings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list})
}

func (s *Server) addMapping(c *gin.Context) {
	var req struct {
		Host    string `json:"host"`
		Backend string `json:"backend"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Host == "" || req.Backend == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "host and backend required"})
		return
	}
	_, err := s.store.AddMapping(req.Host, req.Backend)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.reloadProxy()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) updateMapping(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Backend string `json:"backend"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Backend == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "backend required"})
		return
	}
	if err := s.store.UpdateMapping(id, req.Backend); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.reloadProxy()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) deleteMapping(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.store.DeleteMapping(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.reloadProxy()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) oneClickMapping(c *gin.Context) {
	var req struct {
		Host        string `json:"host"`         // 如 xx.yyy.com 或 xx
		Backend     string `json:"backend"`      // 如 http://127.0.0.1:3000
		ZoneID      string `json:"zone_id"`      // CF Zone ID
		CNAMETarget string `json:"cname_target"` // CNAME 指向，如樱花 frp 出口域名（可留空用默认）
		Proxied     *bool  `json:"proxied"`      // 可选：覆盖默认 proxied
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.Host == "" || req.Backend == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "host and backend required"})
		return
	}

	req.CNAMETarget = strings.TrimSpace(req.CNAMETarget)
	if req.CNAMETarget == "" {
		if v, err := s.store.GetSetting(settingFrpCnameTarget); err == nil {
			req.CNAMETarget = strings.TrimSpace(v)
		}
	}

	// 1) 本地反代
	if _, err := s.store.AddMapping(req.Host, req.Backend); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "add mapping: " + err.Error()})
		return
	}
	s.reloadProxy()

	// 2) 若提供了 CF 信息则创建/更新 CNAME（支持默认值）
	if s.cf != nil && req.ZoneID != "" && req.CNAMETarget != "" {
		proxied := true
		if req.Proxied != nil {
			proxied = *req.Proxied
		} else if v, err := s.store.GetSetting(settingCFCnameProxied); err == nil {
			proxied = parseBoolDefault(v, true)
		}

		name := req.Host
		if len(name) > 0 && name[len(name)-1] == '.' {
			name = name[:len(name)-1]
		}
		_, err := s.cf.EnsureCNAMEWithProxied(req.ZoneID, name, req.CNAMETarget, proxied)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": true, "warning": "mapping added but cf cname failed: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	if s.cf == nil {
		c.JSON(http.StatusOK, gin.H{"ok": true, "warning": "mapping added but cloudflare not configured"})
		return
	}
	if strings.TrimSpace(req.ZoneID) == "" {
		c.JSON(http.StatusOK, gin.H{"ok": true, "warning": "mapping added but zone_id not provided"})
		return
	}
	if strings.TrimSpace(req.CNAMETarget) == "" {
		c.JSON(http.StatusOK, gin.H{"ok": true, "warning": "mapping added but cname_target not provided (and no default configured)"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
