package api

import (
	"context"
	"net/http"
	"time"

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
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if err := s.driveIndexer.StartRebuild(ctx, s.store, acc); err != nil {
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
