package api

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/store"
	"golang.org/x/net/webdav"
)

func (s *Server) listWebDevServices(c *gin.Context) {
	list, err := s.store.ListWebDAVMappings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list})
}

func (s *Server) addWebDevService(c *gin.Context) {
	var req store.WebDAVMapping
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	normalized, err := normalizeWebDevServicePayload(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := s.store.AddWebDAVMapping(normalized)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "id": id})
}

func (s *Server) updateWebDevService(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req store.WebDAVMapping
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	normalized, err := normalizeWebDevServicePayload(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.store.UpdateWebDAVMapping(id, normalized); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) deleteWebDevService(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.store.DeleteWebDAVMapping(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) startWebDevService(c *gin.Context) {
	item, ok := s.getWebDAVByParamID(c)
	if !ok {
		return
	}
	item.Enabled = true
	if err := s.store.UpdateWebDAVMapping(item.ID, item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) stopWebDevService(c *gin.Context) {
	item, ok := s.getWebDAVByParamID(c)
	if !ok {
		return
	}
	item.Enabled = false
	if err := s.store.UpdateWebDAVMapping(item.ID, item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) restartWebDevService(c *gin.Context) {
	item, ok := s.getWebDAVByParamID(c)
	if !ok {
		return
	}
	item.Enabled = true
	if err := s.store.UpdateWebDAVMapping(item.ID, item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) healthWebDevService(c *gin.Context) {
	item, ok := s.getWebDAVByParamID(c)
	if !ok {
		return
	}
	if err := validateLocalDirectory(item.LocalPath); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "directory available"})
}

func (s *Server) serveWebDAVByID(c *gin.Context) {
	item, ok := s.getWebDAVByParamID(c)
	if !ok {
		return
	}
	if !item.Enabled {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "mapping disabled"})
		return
	}
	if err := validateLocalDirectory(item.LocalPath); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username, password, okAuth := c.Request.BasicAuth()
	if !okAuth || subtle.ConstantTimeCompare([]byte(username), []byte(item.Username)) != 1 ||
		subtle.ConstantTimeCompare([]byte(password), []byte(item.Password)) != 1 {
		c.Header("WWW-Authenticate", `Basic realm="SkyLink WebDAV"`)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	davPath := c.Param("davPath")
	if davPath == "" {
		davPath = "/"
	}
	davPath = path.Clean("/" + strings.TrimPrefix(davPath, "/"))
	if strings.Contains(davPath, "..") {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid dav path"})
		return
	}

	var fsys webdav.FileSystem = webdav.Dir(item.LocalPath)
	if item.ReadOnly {
		fsys = readOnlyFS{base: webdav.Dir(item.LocalPath)}
	}
	handler := &webdav.Handler{
		Prefix:     "",
		FileSystem: fsys,
		LockSystem: webdav.NewMemLS(),
	}
	c.Request.URL.Path = davPath
	handler.ServeHTTP(c.Writer, c.Request)
}

func (s *Server) getWebDAVByParamID(c *gin.Context) (*store.WebDAVMapping, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return nil, false
	}
	svc, err := s.store.GetWebDAVMapping(id)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, false
	}
	return svc, true
}

func normalizeWebDevServicePayload(req *store.WebDAVMapping) (*store.WebDAVMapping, error) {
	name := strings.TrimSpace(req.Name)
	localPath := strings.TrimSpace(req.LocalPath)
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if name == "" {
		return nil, errf("name is required")
	}
	if localPath == "" {
		return nil, errf("local_path is required")
	}
	if username == "" {
		return nil, errf("username is required")
	}
	if password == "" {
		return nil, errf("password is required")
	}
	if err := validateLocalDirectory(localPath); err != nil {
		return nil, err
	}

	return &store.WebDAVMapping{
		Name:      name,
		LocalPath: filepath.Clean(localPath),
		Username:  username,
		Password:  password,
		Enabled:   req.Enabled,
		ReadOnly:  req.ReadOnly,
	}, nil
}

func validateLocalDirectory(pathValue string) error {
	cleaned := filepath.Clean(strings.TrimSpace(pathValue))
	if cleaned == "" || cleaned == "." {
		return errf("local_path is invalid")
	}
	info, err := os.Stat(cleaned)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errf("local_path not found")
		}
		return errf("local_path is not accessible")
	}
	if !info.IsDir() {
		return errf("local_path must be a directory")
	}
	return nil
}

func errf(msg string) error {
	return &payloadError{msg: msg}
}

type payloadError struct {
	msg string
}

func (e *payloadError) Error() string { return e.msg }

type readOnlyFS struct {
	base webdav.FileSystem
}

func (r readOnlyFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return fs.ErrPermission
}

func (r readOnlyFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if (flag & (os.O_WRONLY | os.O_RDWR | os.O_APPEND | os.O_CREATE | os.O_TRUNC)) != 0 {
		return nil, fs.ErrPermission
	}
	return r.base.OpenFile(ctx, name, os.O_RDONLY, perm)
}

func (r readOnlyFS) RemoveAll(ctx context.Context, name string) error {
	return fs.ErrPermission
}

func (r readOnlyFS) Rename(ctx context.Context, oldName, newName string) error {
	return fs.ErrPermission
}

func (r readOnlyFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	return r.base.Stat(ctx, name)
}
