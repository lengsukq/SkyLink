package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/drive"
	"github.com/skylink/skylink/internal/security"
	"github.com/skylink/skylink/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type createDriveAccountRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RootPath   string `json:"root_path"`
	QuotaBytes int64  `json:"quota_bytes"`
	Enabled    bool   `json:"enabled"`
}

func (s *Server) listDriveAccounts(c *gin.Context) {
	list, err := s.store.ListDriveAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list})
}

func (s *Server) createDriveAccount(c *gin.Context) {
	var req createDriveAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.RootPath = strings.TrimSpace(req.RootPath)
	if req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
		return
	}
	if err := drive.ValidateRootPath(req.RootPath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid root_path"})
		return
	}
	if req.QuotaBytes < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quota_bytes must be >= 0"})
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password is required"})
		return
	}
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	acc, err := s.store.CreateDriveAccount(store.CreateDriveAccountParams{
		Username:     req.Username,
		PasswordHash: string(hashBytes),
		RootPath:     req.RootPath,
		QuotaBytes:   req.QuotaBytes,
		Enabled:      req.Enabled,
	})
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unique constraint failed") && strings.Contains(msg, "drive_accounts.username") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "id": acc.ID, "item": acc})
}

type updateDriveAccountRequest struct {
	Username   *string `json:"username"`
	RootPath   *string `json:"root_path"`
	QuotaBytes *int64  `json:"quota_bytes"`
	Enabled    *bool   `json:"enabled"`
}

func (s *Server) updateDriveAccount(c *gin.Context) {
	id, ok := parsePositiveInt64Param(c, "id")
	if !ok {
		return
	}
	var req updateDriveAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.Username != nil {
		v := strings.TrimSpace(*req.Username)
		req.Username = &v
		if v == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
			return
		}
	}
	if req.RootPath != nil {
		v := strings.TrimSpace(*req.RootPath)
		req.RootPath = &v
		if err := drive.ValidateRootPath(v); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid root_path"})
			return
		}
	}
	if req.QuotaBytes != nil && *req.QuotaBytes < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quota_bytes must be >= 0"})
		return
	}

	acc, err := s.store.UpdateDriveAccount(id, store.UpdateDriveAccountParams{
		Username:   req.Username,
		RootPath:   req.RootPath,
		QuotaBytes: req.QuotaBytes,
		Enabled:    req.Enabled,
	})
	if err != nil {
		if err == store.ErrDriveAccountNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err == store.ErrDriveQuotaBelowUsed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "配额不能低于当前已用量"})
			return
		}
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unique constraint failed") && strings.Contains(msg, "drive_accounts.username") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "item": acc})
}

func (s *Server) deleteDriveAccount(c *gin.Context) {
	id, ok := parsePositiveInt64Param(c, "id")
	if !ok {
		return
	}
	if err := s.store.DeleteDriveAccount(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) resetDriveAccountPassword(c *gin.Context) {
	id, ok := parsePositiveInt64Param(c, "id")
	if !ok {
		return
	}

	pw := ""
	if c.Request.ContentLength > 0 {
		var req struct {
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		pw = strings.TrimSpace(req.Password)
	}

	generated := false
	if pw == "" {
		var err error
		pw, err = security.GeneratePassword(16)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		generated = true
	} else if len(pw) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 6 characters"})
		return
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	hash := string(hashBytes)
	if _, err := s.store.UpdateDriveAccount(id, store.UpdateDriveAccountParams{PasswordHash: &hash}); err != nil {
		if err == store.ErrDriveAccountNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := gin.H{"ok": true}
	if generated {
		out["password"] = pw
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) recountDriveAccountUsed(c *gin.Context) {
	id, ok := parsePositiveInt64Param(c, "id")
	if !ok {
		return
	}
	acc, err := s.store.GetDriveAccount(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if acc == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	used, err := computeDirSize(acc.RootPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := s.store.SetDriveAccountUsedBytes(acc.ID, used); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "used_bytes": used})
}

