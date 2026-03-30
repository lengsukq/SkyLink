package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

