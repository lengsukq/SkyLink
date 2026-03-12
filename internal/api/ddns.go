package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

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
	list, err := s.store.ListDDNSConfigs()
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
	req.IntervalMin = normalizeDDNSIntervalMin(req.IntervalMin)
	_, err := s.store.AddDDNSConfig(req.ZoneID, req.RecordName, req.RecordID, req.IntervalMin, req.Enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) updateDDNS(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.store.DeleteDDNSConfig(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) getPublicIP(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	ip, err := ddns.GetPublicIP(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ip": ip})
}
