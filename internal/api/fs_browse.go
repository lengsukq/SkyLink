package api

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type fsBrowseEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// getFSBrowse 列出本机目录（供管理端选择 SMB/WebDAV/网盘根路径等）。仅管理员可调用。
// path 为空：Windows 返回盘符列表；Unix 返回根目录「/」下的子项。
func (s *Server) getFSBrowse(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("path"))
	if raw == "" {
		s.listFSRoots(c)
		return
	}

	clean, err := sanitizeBrowsePath(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "path not found"})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not a directory"})
		return
	}

	entries, err := os.ReadDir(clean)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	out := make([]fsBrowseEntry, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		full := filepath.Join(clean, name)
		out = append(out, fsBrowseEntry{Name: name, Path: full, IsDir: true})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})

	c.JSON(http.StatusOK, gin.H{
		"current": clean,
		"parent":  browseParentPath(clean),
		"entries": out,
	})
}

func (s *Server) listFSRoots(c *gin.Context) {
	if runtime.GOOS == "windows" {
		var roots []fsBrowseEntry
		for _, letter := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			p := string(letter) + `:\`
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				roots = append(roots, fsBrowseEntry{Name: string(letter) + ":", Path: p, IsDir: true})
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"current": "",
			"parent":  "",
			"entries": roots,
		})
		return
	}

	// Unix：列出根下子目录
	const root = "/"
	entries, err := os.ReadDir(root)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	out := make([]fsBrowseEntry, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		full := filepath.Join(root, name)
		out = append(out, fsBrowseEntry{Name: name, Path: full, IsDir: true})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	c.JSON(http.StatusOK, gin.H{
		"current": root,
		"parent":  "",
		"entries": out,
	})
}

func sanitizeBrowsePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errf("path is required")
	}
	clean := filepath.Clean(raw)
	if !filepath.IsAbs(clean) {
		return "", errf("path must be absolute")
	}
	if runtime.GOOS == "windows" {
		if len(clean) == 2 && clean[1] == ':' {
			clean += `\`
		}
	}
	return clean, nil
}

func browseParentPath(p string) string {
	if p == "" {
		return ""
	}
	if runtime.GOOS != "windows" {
		if p == "/" {
			return ""
		}
		parent := filepath.Dir(p)
		if parent == p || parent == "." {
			return ""
		}
		return parent
	}
	if isWindowsDriveRoot(p) {
		return ""
	}
	parent := filepath.Dir(p)
	// Dir(`C:\`) 在 Windows 上可能为 `C:`，规范为盘符根
	if len(parent) == 2 && parent[1] == ':' {
		parent += `\`
	}
	if parent == p {
		return ""
	}
	return parent
}

func isWindowsDriveRoot(p string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	c := filepath.Clean(p)
	return len(c) == 3 && c[1] == ':' && (c[2] == '\\' || c[2] == '/')
}
