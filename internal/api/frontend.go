package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// serveFrontend 提供内嵌的静态前端；若 staticFS 为 nil 则返回 404
func (s *Server) serveFrontend(c *gin.Context) {
	if s.staticFS == nil {
		c.Status(http.StatusNotFound)
		return
	}
	requestPath := c.Request.URL.Path
	if requestPath == "/" {
		requestPath = "/index.html"
	}
	requestPath = strings.TrimPrefix(requestPath, "/")
	// URL path should be normalized with slash semantics across platforms.
	if strings.Contains(requestPath, "..") || path.Clean(requestPath) != requestPath {
		c.Status(http.StatusNotFound)
		return
	}
	// staticFS 的根即为 web/dist 内容（由 main 传入 sub-FS）
	data, err := fs.ReadFile(s.staticFS, requestPath)
	if err != nil {
		data, err = fs.ReadFile(s.staticFS, "index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
	}
	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(requestPath, ".html"):
		contentType = "text/html; charset=utf-8"
	case strings.HasSuffix(requestPath, ".js"):
		contentType = "application/javascript"
	case strings.HasSuffix(requestPath, ".css"):
		contentType = "text/css"
	case strings.HasSuffix(requestPath, ".ico"):
		contentType = "image/x-icon"
	}
	c.Data(http.StatusOK, contentType, data)
}
