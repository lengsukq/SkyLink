package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		pw := bearerToken(c.GetHeader("Authorization"))
		if pw == "" || !s.verifyPassword(pw) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

