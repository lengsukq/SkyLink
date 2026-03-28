package api

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/drive"
	drivesvc "github.com/skylink/skylink/internal/drive/service"
	"github.com/skylink/skylink/internal/store"
)

type driveFileItem struct {
	Name       string         `json:"name"`
	Path       string         `json:"path"`
	IsDir      bool           `json:"is_dir"`
	Size       int64          `json:"size"`
	ModifiedAt int64          `json:"modified_at"`
	Ext        string         `json:"ext"`
	Type       drive.FileType `json:"type"`
}

// driveListFiles 处理 GET /api/drive/files：按路径直接读盘列举（offset/limit）。
// 与 driveListEntries（索引表 + 懒扫描、cursor）并存；Web 内置网盘 UI 使用后者。
func (s *Server) driveListFiles(c *gin.Context) {
	acc := currentDriveAccount(c)
	if acc == nil {
		driveError(c, http.StatusUnauthorized, driveErrUnauthorized, "unauthorized")
		return
	}

	userPath := c.Query("path")
	recursive := parseBoolQuery(c.Query("recursive"))
	typeFilter := strings.TrimSpace(c.Query("type"))
	q := strings.TrimSpace(c.Query("q"))
	offset := parseIntQuery(c.Query("offset"), 0)
	limit := parseIntQuery(c.Query("limit"), driveListDefaultLimit)
	if limit <= 0 {
		limit = driveListDefaultLimit
	}
	if limit > driveListMaxLimit {
		limit = driveListMaxLimit
	}
	if offset < 0 {
		offset = 0
	}

	if typeFilter != "" && !isSupportedDriveTypeFilter(typeFilter) {
		driveError(c, http.StatusBadRequest, driveErrInvalidRequest, "invalid type")
		return
	}

	baseFull, err := drive.ResolveWithinRoot(acc.RootPath, userPath)
	if err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, "invalid path")
		return
	}
	info, err := os.Stat(baseFull)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			driveError(c, http.StatusNotFound, driveErrNotFound, "not found")
			return
		}
		driveError(c, http.StatusInternalServerError, driveErrInternal, "path not accessible")
		return
	}
	if !info.IsDir() {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, "path must be a directory")
		return
	}

	var list []driveFileItem
	hasMore := false

	if !recursive {
		entries, err := os.ReadDir(baseFull)
		if err != nil {
			driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
			return
		}
		// stable order
		sort.Slice(entries, func(i, j int) bool { return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name()) })

		startRel := drive.CleanUserPath(userPath)
		for _, e := range entries {
			name := e.Name()
			if q != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(q)) {
				continue
			}
			isDir := e.IsDir()
			ext, t := drive.ClassifyFileTypeByName(name, isDir)
			if typeFilter != "" && !isDir && string(t) != typeFilter {
				continue
			}
			inf, err := e.Info()
			if err != nil {
				continue
			}
			p := name
			if startRel != "" {
				p = filepath.ToSlash(filepath.Join(startRel, name))
			}
			item := driveFileItem{
				Name:       name,
				Path:       p,
				IsDir:      isDir,
				Size:       inf.Size(),
				ModifiedAt: inf.ModTime().Unix(),
				Ext:        ext,
				Type:       t,
			}
			list = append(list, item)
		}

		if offset >= len(list) {
			list = []driveFileItem{}
		} else {
			end := offset + limit
			if end < len(list) {
				hasMore = true
				list = list[offset:end]
			} else {
				list = list[offset:]
			}
		}
	} else {
		rootClean := filepath.Clean(acc.RootPath)
		startRel := drive.CleanUserPath(userPath)
		startFull := baseFull

		collected := 0
		skipped := 0
		err := filepath.WalkDir(startFull, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			name := d.Name()
			if q != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(q)) {
				return nil
			}
			ext, t := drive.ClassifyFileTypeByName(name, false)
			if typeFilter != "" && string(t) != typeFilter {
				return nil
			}

			if skipped < offset {
				skipped++
				return nil
			}
			if collected >= limit {
				hasMore = true
				return fs.SkipAll
			}

			inf, e := d.Info()
			if e != nil {
				return nil
			}
			rel, e := filepath.Rel(rootClean, p)
			if e != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			// Ensure recursive search is constrained under requested start path.
			if startRel != "" && !strings.HasPrefix(strings.ToLower(rel), strings.ToLower(filepath.ToSlash(startRel))+"/") {
				return nil
			}

			list = append(list, driveFileItem{
				Name:       name,
				Path:       rel,
				IsDir:      false,
				Size:       inf.Size(),
				ModifiedAt: inf.ModTime().Unix(),
				Ext:        ext,
				Type:       t,
			})
			collected++
			return nil
		})
		if err != nil && err != fs.SkipAll {
			driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
			return
		}
	}

	nextOffset := offset + len(list)
	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"list":        list,
		"offset":      offset,
		"next_offset": nextOffset,
		"limit":       limit,
		"has_more":    hasMore,
	})
}

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

func parseBoolQuery(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func parseIntQuery(v string, def int) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func isSupportedDriveTypeFilter(v string) bool {
	switch v {
	case string(drive.FileTypeImage),
		string(drive.FileTypeVideo),
		string(drive.FileTypeAudio),
		string(drive.FileTypeDocument),
		string(drive.FileTypeArchive),
		string(drive.FileTypeOther):
		return true
	default:
		return false
	}
}
