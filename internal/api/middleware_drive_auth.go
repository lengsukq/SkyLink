package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/drive"
	"github.com/skylink/skylink/internal/store"
)

const driveAccountContextKey = "drive_account"

func (s *Server) driveAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		claims, err := drive.ParseToken([]byte(s.driveJWTSecret), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		acc, err := s.store.GetDriveAccount(claims.AccountID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if acc == nil || !acc.Enabled || acc.PasswordHash == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		_ = s.store.TouchDriveAccountLastUsed(acc.ID, time.Now().Unix())
		c.Set(driveAccountContextKey, acc)
		c.Next()
	}
}

func currentDriveAccount(c *gin.Context) *store.DriveAccount {
	v, ok := c.Get(driveAccountContextKey)
	if !ok {
		return nil
	}
	acc, _ := v.(*store.DriveAccount)
	return acc
}

