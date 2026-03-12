package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) listZones(c *gin.Context) {
	zones, err := s.cf.ListZones()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"zones": zones})
}

func (s *Server) listDNSRecords(c *gin.Context) {
	zoneID := c.Param("zoneId")
	records, err := s.cf.ListDNSRecords(zoneID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"records": records})
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
	rec, err := s.cf.CreateDNSRecord(zoneID, req.Type, req.Name, req.Content, req.TTL, req.Proxied)
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
	rec, err := s.cf.UpdateDNSRecord(zoneID, recordID, req.Type, req.Name, req.Content, req.TTL, req.Proxied)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": rec})
}

func (s *Server) deleteDNSRecord(c *gin.Context) {
	zoneID := c.Param("zoneId")
	recordID := c.Param("recordId")
	if err := s.cf.DeleteDNSRecord(zoneID, recordID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
