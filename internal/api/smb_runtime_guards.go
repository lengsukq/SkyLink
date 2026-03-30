package api

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/smb"
	"github.com/skylink/skylink/internal/store"
)

func isWindowsRuntime() bool {
	return runtime.GOOS == "windows"
}

func (s *Server) guardSMBWindows(c *gin.Context) bool {
	if isWindowsRuntime() {
		return true
	}
	c.JSON(http.StatusNotImplemented, gin.H{"error": "smb is only supported on windows"})
	return false
}

func (s *Server) restoreSMBShareState(item *store.SMBMapping) error {
	if item == nil {
		return nil
	}
	if !item.Enabled {
		return smb.DeleteShare(item.ShareName)
	}
	return smb.CreateOrUpdateShare(item.ShareName, item.LocalPath, item.ReadOnly, item.GrantAccount)
}

