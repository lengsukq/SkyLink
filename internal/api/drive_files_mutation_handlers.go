package api

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/drive"
	drivesvc "github.com/skylink/skylink/internal/drive/service"
	"github.com/skylink/skylink/internal/store"
)

type driveMkdirRequest struct {
	Path string `json:"path"`
}

func (s *Server) driveMkdir(c *gin.Context) {
	acc := currentDriveAccount(c)
	if acc == nil {
		driveError(c, http.StatusUnauthorized, driveErrUnauthorized, "unauthorized")
		return
	}
	var req driveMkdirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidBody, "invalid body")
		return
	}
	svc := drivesvc.NewFilesService(s.store)
	if err := svc.Mkdir(acc, req.Path, c.ClientIP()); err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, "invalid path")
		return
	}
	driveOK(c, gin.H{})
}

type driveRenameRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (s *Server) driveRename(c *gin.Context) {
	acc := currentDriveAccount(c)
	if acc == nil {
		driveError(c, http.StatusUnauthorized, driveErrUnauthorized, "unauthorized")
		return
	}
	var req driveRenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidBody, "invalid body")
		return
	}
	svc := drivesvc.NewFilesService(s.store)
	if err := svc.Rename(acc, req.From, req.To, c.ClientIP()); err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, err.Error())
		return
	}
	driveOK(c, gin.H{})
}

func (s *Server) driveDelete(c *gin.Context) {
	acc := currentDriveAccount(c)
	if acc == nil {
		driveError(c, http.StatusUnauthorized, driveErrUnauthorized, "unauthorized")
		return
	}
	userPath := c.Query("path")
	if strings.TrimSpace(userPath) == "" {
		driveError(c, http.StatusBadRequest, driveErrInvalidRequest, "path is required")
		return
	}
	svc := drivesvc.NewFilesService(s.store)
	freed, err := svc.Delete(acc, userPath, c.ClientIP())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			driveError(c, http.StatusNotFound, driveErrNotFound, "not found")
			return
		}
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, err.Error())
		return
	}
	driveOK(c, gin.H{"freed_bytes": freed})
}

func (s *Server) driveUpload(c *gin.Context) {
	acc := currentDriveAccount(c)
	if acc == nil {
		driveError(c, http.StatusUnauthorized, driveErrUnauthorized, "unauthorized")
		return
	}
	targetDir := c.PostForm("path")
	fh, err := c.FormFile("file")
	if err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidRequest, "file is required")
		return
	}
	svc := drivesvc.NewFilesService(s.store)
	written, err := svc.Upload(acc, targetDir, fh, c.ClientIP())
	if err != nil {
		if errors.Is(err, drive.ErrQuotaExceeded) {
			driveError(c, http.StatusRequestEntityTooLarge, driveErrQuotaExceeded, "quota exceeded")
			return
		}
		driveError(c, http.StatusBadRequest, driveErrInvalidRequest, err.Error())
		return
	}
	driveOK(c, gin.H{"written_bytes": written})
}

func (s *Server) driveDownload(c *gin.Context) {
	acc := currentDriveAccount(c)
	if acc == nil {
		driveError(c, http.StatusUnauthorized, driveErrUnauthorized, "unauthorized")
		return
	}
	userPath := c.Query("path")
	if strings.TrimSpace(userPath) == "" {
		driveError(c, http.StatusBadRequest, driveErrInvalidRequest, "path is required")
		return
	}
	full, err := drive.ResolveWithinRoot(acc.RootPath, userPath)
	if err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, "invalid path")
		return
	}
	info, err := os.Stat(full)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			driveError(c, http.StatusNotFound, driveErrNotFound, "not found")
			return
		}
		driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
		return
	}
	if info.IsDir() {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, "path is a directory")
		return
	}
	_ = s.store.AddDriveAuditLog(acc.ID, store.DriveAuditDownload, drive.CleanUserPath(userPath), c.ClientIP())
	c.File(full)
}

