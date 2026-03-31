package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) driveIndexRebuild(c *gin.Context) {
	id, ok := parsePositiveInt64Param(c, "id")
	if !ok {
		return
	}
	acc, err := s.store.GetDriveAccount(id)
	if err != nil {
		driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
		return
	}
	if acc == nil {
		driveError(c, http.StatusNotFound, driveErrNotFound, "not found")
		return
	}
	if err := s.driveIndexer.StartRebuild(s.store, acc); err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidRequest, err.Error())
		return
	}
	driveOK(c, gin.H{})
}

func (s *Server) driveIndexStatus(c *gin.Context) {
	id, ok := parsePositiveInt64Param(c, "id")
	if !ok {
		return
	}
	status := s.driveIndexer.GetStatus(id)
	driveOK(c, gin.H{"status": status})
}

// driveUserIndexRebuild 供网盘 JWT 调用：重建当前账号的索引，使列表与磁盘一致（与管理员 POST /accounts/:id/index/rebuild 行为相同）。
// 实际超时由 indexer.Manager 在后台 goroutine 内持有，与 HTTP handler 返回无关。
func (s *Server) driveUserIndexRebuild(c *gin.Context) {
	acc := currentDriveAccount(c)
	if acc == nil {
		driveError(c, http.StatusUnauthorized, driveErrUnauthorized, "unauthorized")
		return
	}
	if err := s.driveIndexer.StartRebuild(s.store, acc); err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidRequest, err.Error())
		return
	}
	driveOK(c, gin.H{})
}

func (s *Server) driveUserIndexStatus(c *gin.Context) {
	acc := currentDriveAccount(c)
	if acc == nil {
		driveError(c, http.StatusUnauthorized, driveErrUnauthorized, "unauthorized")
		return
	}
	status := s.driveIndexer.GetStatus(acc.ID)
	driveOK(c, gin.H{"status": status})
}
