package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/ddns"
)

func normalizeDDNSIntervalMin(v int) int {
	if v <= 0 {
		return ddns.DefaultIntervalMin
	}
	return v
}

func (s *Server) listDDNS(c *gin.Context) {
	// 需要当前 CF 账号上下文，以实现按账号隔离
	if _, err := s.cfClientForCurrentAccount(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	idVal := s.currentCFAccountID.Load()
	cfAccountID, _ := idVal.(int64)
	list, err := s.store.ListDDNSConfigs(cfAccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list})
}

func (s *Server) addDDNS(c *gin.Context) {
	var req struct {
		ZoneID      string `json:"zone_id"`
		RecordName  string `json:"record_name"`
		RecordID    string `json:"record_id"`
		RecordType  string `json:"record_type"` // "A" 或 "AAAA"，默认 A
		IntervalMin int    `json:"interval_min"`
		Enabled     bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.ZoneID == "" || req.RecordName == "" || req.RecordID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "zone_id, record_name, record_id required"})
		return
	}
	if req.RecordType != "AAAA" {
		req.RecordType = "A"
	}
	// 需要当前 CF 账号上下文，以便将配置绑定到对应账号
	if _, err := s.cfClientForCurrentAccount(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	idVal := s.currentCFAccountID.Load()
	cfAccountID, _ := idVal.(int64)
	req.IntervalMin = normalizeDDNSIntervalMin(req.IntervalMin)
	_, err := s.store.AddDDNSConfig(cfAccountID, req.ZoneID, req.RecordName, req.RecordID, req.RecordType, req.IntervalMin, req.Enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) updateDDNS(c *gin.Context) {
	id, ok := parsePositiveInt64Param(c, "id")
	if !ok {
		return
	}
	var req struct {
		IntervalMin int  `json:"interval_min"`
		Enabled     bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	req.IntervalMin = normalizeDDNSIntervalMin(req.IntervalMin)
	if err := s.store.UpdateDDNSConfig(id, req.IntervalMin, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) deleteDDNS(c *gin.Context) {
	id, ok := parsePositiveInt64Param(c, "id")
	if !ok {
		return
	}
	if err := s.store.DeleteDDNSConfig(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) getPublicIP(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), ddnsPublicIPTimeout)
	defer cancel()
	ipv4, err4 := ddns.GetPublicIPv4(ctx)
	ipv6, _ := ddns.GetPublicIPv6(ctx)
	if err4 != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err4.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ip":   ipv4,
		"ipv4": ipv4,
		"ipv6": ipv6,
	})
}
