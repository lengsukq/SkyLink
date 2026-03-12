package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) stats(c *gin.Context) {
	mappings, _ := s.store.ListMappings()
	ddnsList, _ := s.store.ListDDNSConfigs()
	c.JSON(http.StatusOK, gin.H{
		"mappings_count": len(mappings),
		"ddns_count":    len(ddnsList),
	})
}
