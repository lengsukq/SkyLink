package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/drive"
	"github.com/skylink/skylink/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type driveLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) driveLogin(c *gin.Context) {
	var req driveLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	username := strings.TrimSpace(req.Username)
	password := req.Password
	if username == "" || strings.TrimSpace(password) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}

	acc, err := s.store.GetDriveAccountByUsername(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if acc == nil || !acc.Enabled || strings.TrimSpace(acc.PasswordHash) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	token, err := drive.NewToken([]byte(s.driveJWTSecret), acc.ID, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = s.store.TouchDriveAccountLastUsed(acc.ID, time.Now().Unix())
	_ = s.store.AddDriveAuditLog(acc.ID, store.DriveAuditLogin, "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt.Unix(),
		"account": gin.H{
			"id":          acc.ID,
			"username":    acc.Username,
			"root_path":   acc.RootPath,
			"quota_bytes": acc.QuotaBytes,
			"used_bytes":  acc.UsedBytes,
		},
	})
}
