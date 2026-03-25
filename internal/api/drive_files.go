package api

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/drive"
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
	limit := parseIntQuery(c.Query("limit"), 200)
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
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
	full, err := drive.ResolveWithinRoot(acc.RootPath, req.Path)
	if err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, "invalid path")
		return
	}
	if err := os.MkdirAll(full, 0755); err != nil {
		driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
		return
	}
	rel := drive.CleanUserPath(req.Path)
	name := filepath.Base(filepath.FromSlash(rel))
	parent := ""
	if idx := strings.LastIndex(rel, "/"); idx >= 0 {
		parent = rel[:idx]
	}
	_, t := drive.ClassifyFileTypeByName(name, true)
	_ = s.store.UpsertDriveEntry(&store.DriveEntry{
		AccountID:  acc.ID,
		Path:       rel,
		ParentPath: parent,
		Name:       name,
		Ext:        "",
		Type:       string(t),
		IsDir:      true,
		SizeBytes:  0,
		ModifiedAt: 0,
	})
	_ = s.store.AddDriveAuditLog(acc.ID, store.DriveAuditMkdir, rel, c.ClientIP())
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
	fromFull, err := drive.ResolveWithinRoot(acc.RootPath, req.From)
	if err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, "invalid from")
		return
	}
	toFull, err := drive.ResolveWithinRoot(acc.RootPath, req.To)
	if err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, "invalid to")
		return
	}
	if err := os.MkdirAll(filepath.Dir(toFull), 0755); err != nil {
		driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
		return
	}
	if err := os.Rename(fromFull, toFull); err != nil {
		driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
		return
	}
	// Best-effort index update: delete old prefix, then rebuild target subtree quickly by scanning filesystem.
	fromRel := drive.CleanUserPath(req.From)
	toRel := drive.CleanUserPath(req.To)
	if fromRel != "" {
		_ = s.store.DeleteDriveEntriesByPrefix(acc.ID, fromRel)
	}
	// Rebuild target subtree entries (dir or file).
	_ = upsertEntryFromFS(s.store, acc, toRel)
	_ = s.store.AddDriveAuditLog(acc.ID, store.DriveAuditRename, toRel, c.ClientIP())
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
	var freed int64
	if info.IsDir() {
		freed, _ = computeDirSize(full)
		if err := os.RemoveAll(full); err != nil {
			driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
			return
		}
	} else {
		freed = info.Size()
		if err := os.Remove(full); err != nil {
			driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
			return
		}
	}
	if freed > 0 {
		_ = s.store.AddDriveAccountUsedBytes(acc.ID, -freed)
	}
	rel := drive.CleanUserPath(userPath)
	if rel != "" {
		_ = s.store.DeleteDriveEntriesByPrefix(acc.ID, rel)
	}
	_ = s.store.AddDriveAuditLog(acc.ID, store.DriveAuditDelete, rel, c.ClientIP())
	driveOK(c, gin.H{"freed_bytes": freed})
}

func (s *Server) driveUpload(c *gin.Context) {
	acc := currentDriveAccount(c)
	if acc == nil {
		driveError(c, http.StatusUnauthorized, driveErrUnauthorized, "unauthorized")
		return
	}
	// Refresh account for accurate quota/used check.
	fresh, err := s.store.GetDriveAccount(acc.ID)
	if err == nil && fresh != nil {
		acc = fresh
	}

	targetDir := c.PostForm("path")
	fullDir, err := drive.ResolveWithinRoot(acc.RootPath, targetDir)
	if err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, "invalid path")
		return
	}
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidRequest, "file is required")
		return
	}

	filename := filepath.Base(fh.Filename)
	if filename == "." || filename == string(filepath.Separator) || strings.TrimSpace(filename) == "" {
		driveError(c, http.StatusBadRequest, driveErrInvalidRequest, "invalid filename")
		return
	}

	if acc.QuotaBytes > 0 && fh.Size > 0 && acc.UsedBytes+fh.Size > acc.QuotaBytes {
		driveError(c, http.StatusRequestEntityTooLarge, driveErrQuotaExceeded, "quota exceeded")
		return
	}

	dstPath := filepath.Join(fullDir, filename)
	dstPath = filepath.Clean(dstPath)
	if !strings.HasPrefix(strings.ToLower(dstPath), strings.ToLower(filepath.Clean(fullDir))+string(filepath.Separator)) && strings.ToLower(dstPath) != strings.ToLower(fullDir) {
		driveError(c, http.StatusBadRequest, driveErrInvalidPath, "invalid path")
		return
	}

	src, err := fh.Open()
	if err != nil {
		driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
		return
	}
	defer src.Close()

	tmpPath := dstPath + ".uploading"
	out, err := os.Create(tmpPath)
	if err != nil {
		driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
		return
	}
	written, copyErr := io.Copy(out, src)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		driveError(c, http.StatusInternalServerError, driveErrInternal, copyErr.Error())
		return
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		driveError(c, http.StatusInternalServerError, driveErrInternal, closeErr.Error())
		return
	}

	if acc.QuotaBytes > 0 && written > 0 && acc.UsedBytes+written > acc.QuotaBytes {
		_ = os.Remove(tmpPath)
		driveError(c, http.StatusRequestEntityTooLarge, driveErrQuotaExceeded, "quota exceeded")
		return
	}

	if err := os.Rename(tmpPath, dstPath); err != nil {
		_ = os.Remove(tmpPath)
		driveError(c, http.StatusInternalServerError, driveErrInternal, err.Error())
		return
	}
	if written > 0 {
		_ = s.store.AddDriveAccountUsedBytes(acc.ID, written)
	}
	relDir := drive.CleanUserPath(targetDir)
	rel := filename
	if relDir != "" {
		rel = filepath.ToSlash(filepath.Join(relDir, filename))
	}
	_ = upsertEntryFromFS(s.store, acc, rel)
	_ = s.store.AddDriveAuditLog(acc.ID, store.DriveAuditUpload, rel, c.ClientIP())
	driveOK(c, gin.H{"written_bytes": written})
}

func upsertEntryFromFS(st *store.Store, acc *store.DriveAccount, rel string) error {
	if st == nil || acc == nil {
		return nil
	}
	rel = drive.CleanUserPath(rel)
	full, err := drive.ResolveWithinRoot(acc.RootPath, rel)
	if err != nil {
		return err
	}
	info, err := os.Stat(full)
	if err != nil {
		return err
	}
	parent := ""
	if idx := strings.LastIndex(rel, "/"); idx >= 0 {
		parent = rel[:idx]
	}
	name := info.Name()
	ext, t := drive.ClassifyFileTypeByName(name, info.IsDir())
	entry := &store.DriveEntry{
		AccountID:  acc.ID,
		Path:       rel,
		ParentPath: parent,
		Name:       name,
		Ext:        ext,
		Type:       string(t),
		IsDir:      info.IsDir(),
		SizeBytes:  info.Size(),
		ModifiedAt: info.ModTime().Unix(),
	}
	return st.UpsertDriveEntry(entry)
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

func computeDirSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total, err
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
