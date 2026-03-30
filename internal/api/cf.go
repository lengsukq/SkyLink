package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/cloudflare"
	"github.com/skylink/skylink/internal/store"
)

func (s *Server) listZones(c *gin.Context) {
	cfClient, err := s.cfClientForCurrentAccount()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, errorResponse{Error: err.Error()})
		return
	}
	zones, err := cfClient.ListZones()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfZonesResponse{Zones: zones})
}

func (s *Server) listDNSRecords(c *gin.Context) {
	zoneID := c.Param("zoneId")
	cfClient, err := s.cfClientForCurrentAccount()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, errorResponse{Error: err.Error()})
		return
	}
	records, err := cfClient.ListDNSRecords(zoneID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfDNSRecordsResponse{Records: records})
}

func (s *Server) createDNSRecord(c *gin.Context) {
	zoneID := c.Param("zoneId")
	var req struct {
		Type    string `json:"type"`
		Name    string `json:"name"`
		Content string `json:"content"`
		TTL     int    `json:"ttl"`
		Proxied bool   `json:"proxied"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.Type == "" {
		req.Type = "A"
	}
	cfClient, err := s.cfClientForCurrentAccount()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	rec, err := cfClient.CreateDNSRecord(zoneID, req.Type, req.Name, req.Content, req.TTL, req.Proxied)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": rec})
}

func (s *Server) updateDNSRecord(c *gin.Context) {
	zoneID := c.Param("zoneId")
	recordID := c.Param("recordId")
	var req struct {
		Type    string `json:"type"`
		Name    string `json:"name"`
		Content string `json:"content"`
		TTL     int    `json:"ttl"`
		Proxied bool   `json:"proxied"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.Type == "" {
		req.Type = "A"
	}
	cfClient, err := s.cfClientForCurrentAccount()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	rec, err := cfClient.UpdateDNSRecord(zoneID, recordID, req.Type, req.Name, req.Content, req.TTL, req.Proxied)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": rec})
}

func (s *Server) deleteDNSRecord(c *gin.Context) {
	zoneID := c.Param("zoneId")
	recordID := c.Param("recordId")
	cfClient, err := s.cfClientForCurrentAccount()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	if err := cfClient.DeleteDNSRecord(zoneID, recordID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Cloudflare account management

func (s *Server) listCFAccounts(c *gin.Context) {
	list, err := s.store.ListCFAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, len(list))
	for i, a := range list {
		out[i] = gin.H{
			"id":         a.ID,
			"name":       a.Name,
			"zone_id":    a.ZoneID,
			"created_at": a.CreatedAt,
		}
	}
	c.JSON(http.StatusOK, gin.H{"accounts": out})
}

func (s *Server) createCFAccount(c *gin.Context) {
	var req struct {
		Name     string `json:"name"`
		APIToken string `json:"api_token"`
		ZoneID   string `json:"zone_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.APIToken) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and api_token required"})
		return
	}
	token := strings.TrimSpace(req.APIToken)
	acc := &store.CFAccount{
		Name:     strings.TrimSpace(req.Name),
		APIToken: token,
	}

	// 校验 Token，并规范化默认 Zone ID（可选）
	cfClient := cloudflare.New(token)
	zones, err := cfClient.ListZones()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid api_token or unable to list zones: " + err.Error()})
		return
	}
	zoneHint := strings.TrimSpace(req.ZoneID)
	if zoneHint != "" {
		var matched string
		for _, z := range zones {
			if z.ID == zoneHint || strings.EqualFold(z.Name, zoneHint) {
				matched = z.ID
				break
			}
		}
		if matched == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "zone not found for given hint"})
			return
		}
		acc.ZoneID = matched
	} else if len(zones) == 1 {
		// 未提供时，如账号下只有一个 Zone，则自动选为默认
		acc.ZoneID = zones[0].ID
	}

	id, err := s.store.CreateCFAccount(acc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (s *Server) updateCFAccount(c *gin.Context) {
	id, ok := parsePositiveInt64Param(c, "id")
	if !ok {
		return
	}
	var req struct {
		Name     string `json:"name"`
		APIToken string `json:"api_token"`
		ZoneID   string `json:"zone_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	acc, err := s.store.GetCFAccount(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if acc == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if strings.TrimSpace(req.Name) != "" {
		acc.Name = strings.TrimSpace(req.Name)
	}

	updatedToken := strings.TrimSpace(req.APIToken)
	if updatedToken != "" {
		acc.APIToken = updatedToken
	}

	zoneHint := strings.TrimSpace(req.ZoneID)
	if zoneHint != "" || updatedToken != "" {
		// 若提供了新的 Zone 提示或更新了 Token，则重新校验并规范化 Zone ID
		token := acc.APIToken
		cfClient := cloudflare.New(token)
		zones, err := cfClient.ListZones()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid api_token or unable to list zones: " + err.Error()})
			return
		}
		if zoneHint != "" {
			var matched string
			for _, z := range zones {
				if z.ID == zoneHint || strings.EqualFold(z.Name, zoneHint) {
					matched = z.ID
					break
				}
			}
			if matched == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "zone not found for given hint"})
				return
			}
			acc.ZoneID = matched
		} else if len(zones) == 1 {
			acc.ZoneID = zones[0].ID
		}
	}
	if err := s.store.UpdateCFAccount(acc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 清理客户端缓存，避免继续使用旧 token
	s.cfClientsMu.Lock()
	delete(s.cfClients, id)
	s.cfClientsMu.Unlock()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) deleteCFAccount(c *gin.Context) {
	id, ok := parsePositiveInt64Param(c, "id")
	if !ok {
		return
	}
	// 简单实现：直接删除；如需更严格约束可检查是否被引用
	if err := s.store.DeleteCFAccount(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.cfClientsMu.Lock()
	delete(s.cfClients, id)
	s.cfClientsMu.Unlock()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) activateCFAccount(c *gin.Context) {
	id, ok := parsePositiveInt64Param(c, "id")
	if !ok {
		return
	}
	if _, err := s.cfAccountService.GetActivatableAccount(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account not found or token empty"})
		return
	}
	if err := s.setCurrentCFAccountID(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// validateCFAccount 仅用于在前端保存前校验 API Token 与可选的 Zone 提示是否有效
func (s *Server) validateCFAccount(c *gin.Context) {
	var req struct {
		APIToken string `json:"api_token"`
		ZoneID   string `json:"zone_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.APIToken) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_token required"})
		return
	}

	token := strings.TrimSpace(req.APIToken)
	cfClient := cloudflare.New(token)
	zones, err := cfClient.ListZones()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid api_token or unable to list zones: " + err.Error()})
		return
	}

	zoneHint := strings.TrimSpace(req.ZoneID)
	var normalizedZoneID string
	if zoneHint != "" {
		for _, z := range zones {
			if z.ID == zoneHint || strings.EqualFold(z.Name, zoneHint) {
				normalizedZoneID = z.ID
				break
			}
		}
		if normalizedZoneID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "zone not found for given hint"})
			return
		}
	} else if len(zones) == 1 {
		normalizedZoneID = zones[0].ID
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":        true,
		"zone_id":   normalizedZoneID,
		"zone_hint": zoneHint,
		"zones":     zones,
	})
}
