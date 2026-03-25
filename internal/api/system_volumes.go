package api

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/sysinfo"
)

func (s *Server) getSystemVolumes(c *gin.Context) {
	if runtime.GOOS != "windows" {
		c.JSON(http.StatusOK, gin.H{"supported": false, "volumes": []sysinfo.LogicalVolume{}})
		return
	}
	volumes, err := sysinfo.ListLogicalVolumes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"supported": true, "volumes": volumes})
}
