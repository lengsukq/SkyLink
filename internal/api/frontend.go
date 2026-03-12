package api

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// serveFrontend 提供内嵌的静态前端；若 staticFS 为 nil 则返回 404
func (s *Server) serveFrontend(c *gin.Context) {
	if s.staticFS == nil {
		c.Status(http.StatusNotFound)
		return
	}
	path := c.Request.URL.Path
	if path == "/" {
		path = "/index.html"
	}
	path = strings.TrimPrefix(path, "/")
	if strings.Contains(path, "..") || filepath.Clean(path) != path {
		c.Status(http.StatusNotFound)
		return
	}
	// staticFS 的根即为 web/dist 内容（embed 时已用 all:../web/dist）
	data, err := fs.ReadFile(s.staticFS, path)
	if err != nil {
		data, err = fs.ReadFile(s.staticFS, "index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
	}
	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(path, ".html"):
		contentType = "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		contentType = "application/javascript"
	case strings.HasSuffix(path, ".css"):
		contentType = "text/css"
	case strings.HasSuffix(path, ".ico"):
		contentType = "image/x-icon"
	}
	c.Data(http.StatusOK, contentType, data)
}
