package api

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/drive"
)

func (s *Server) drivePreviewURL(c *gin.Context) {
	acc := currentDriveAccount(c)
	if acc == nil {
		driveError(c, http.StatusUnauthorized, driveErrUnauthorized, "unauthorized")
		return
	}
	path := strings.TrimSpace(c.Query("path"))
	if path == "" {
		driveError(c, http.StatusBadRequest, driveErrInvalidRequest, "path is required")
		return
	}

	rel := drive.CleanUserPath(path)
	full, err := drive.ResolveWithinRoot(acc.RootPath, rel)
	if err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, "invalid path")
		return
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, "invalid path")
		return
	}

	defaultExpires := previewExpiresBySize(info.Size())
	expiresSec := defaultExpires
	if strings.TrimSpace(c.Query("expires")) != "" {
		expiresSec = parseIntQuery(c.Query("expires"), defaultExpires)
	}
	if expiresSec < 30 {
		expiresSec = 30
	}
	// Max 6 hours for safety
	if expiresSec > 6*60*60 {
		expiresSec = 6 * 60 * 60
	}

	token, err := drive.NewFileToken([]byte(s.driveJWTSecret), acc.ID, rel, time.Now().Add(time.Duration(expiresSec)*time.Second))
	if err != nil {
		driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
		return
	}
	u := url.URL{Path: "/api/drive/preview"}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()

	driveOK(c, gin.H{
		"url":        u.String(),
		"expires_in": expiresSec,
	})
}

func (s *Server) drivePreviewServe(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		c.Status(http.StatusUnauthorized)
		return
	}
	claims, err := drive.ParseFileToken([]byte(s.driveJWTSecret), token)
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return
	}
	acc, err := s.store.GetDriveAccount(claims.AccountID)
	if err != nil || acc == nil || !acc.Enabled {
		c.Status(http.StatusUnauthorized)
		return
	}
	full, err := drive.ResolveWithinRoot(acc.RootPath, claims.Path)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	// Let net/http handle range requests for streaming.
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Disposition", "inline")
	// Disable caching for sensitive content.
	c.Header("Cache-Control", "no-store")
	c.File(full)
}

func previewExpiresBySize(sizeBytes int64) int {
	const mb = 1024 * 1024
	const gb = 1024 * mb
	switch {
	case sizeBytes >= int64(gb):
		return 3 * 60 * 60
	case sizeBytes >= 200*int64(mb):
		return 2 * 60 * 60
	default:
		return 1 * 60 * 60
	}
}
