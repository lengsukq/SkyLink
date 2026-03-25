package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/drive"
	"github.com/skylink/skylink/internal/store"
)

func (s *Server) driveListEntries(c *gin.Context) {
	acc := currentDriveAccount(c)
	if acc == nil {
		driveError(c, http.StatusUnauthorized, driveErrUnauthorized, "unauthorized")
		return
	}

	parentPath := strings.TrimSpace(c.Query("parent_path"))
	pathPrefix := strings.TrimSpace(c.Query("path_prefix"))
	recursive := parseBoolQuery(c.Query("recursive"))
	typeFilter := strings.TrimSpace(c.Query("type"))
	q := strings.TrimSpace(c.Query("q"))
	sortBy := strings.TrimSpace(c.Query("sort"))
	order := strings.TrimSpace(c.Query("order"))
	limit := parseIntQuery(c.Query("limit"), driveEntriesDefaultLimit)
	cursor := strings.TrimSpace(c.Query("cursor"))
	includeDirs := parseBoolQuery(c.Query("include_dirs"))

	res, err := s.store.ListDriveEntries(store.ListDriveEntriesParams{
		AccountID:   acc.ID,
		ParentPath:  parentPath,
		PathPrefix:  pathPrefix,
		Recursive:   recursive,
		Type:        typeFilter,
		Query:       q,
		Sort:        store.DriveEntriesSort(sortBy),
		Order:       store.DriveEntriesOrder(order),
		Limit:       limit,
		Cursor:      cursor,
		IncludeDirs: includeDirs,
	})
	if err != nil {
		driveError(c, http.StatusBadRequest, driveErrInvalidRequest, err.Error())
		return
	}

	// Lazy index: if no entries yet (e.g., user root already has files), scan current directory once.
	if cursor == "" && len(res.List) == 0 && q == "" && typeFilter == "" && !recursive {
		_ = s.lazyIndexDir(acc, parentPath)
		res, _ = s.store.ListDriveEntries(store.ListDriveEntriesParams{
			AccountID:   acc.ID,
			ParentPath:  parentPath,
			PathPrefix:  pathPrefix,
			Recursive:   recursive,
			Type:        typeFilter,
			Query:       q,
			Sort:        store.DriveEntriesSort(sortBy),
			Order:       store.DriveEntriesOrder(order),
			Limit:       limit,
			Cursor:      cursor,
			IncludeDirs: includeDirs,
		})
	}
	driveOK(c, gin.H{
		"list":        res.List,
		"next_cursor": res.NextCursor,
	})
}

func (s *Server) lazyIndexDir(acc *store.DriveAccount, parentPath string) error {
	if acc == nil {
		return nil
	}
	full, err := drive.ResolveWithinRoot(acc.RootPath, parentPath)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return err
	}
	parentRel := drive.CleanUserPath(parentPath)
	for _, e := range entries {
		name := e.Name()
		rel := name
		if parentRel != "" {
			rel = filepath.ToSlash(filepath.Join(parentRel, name))
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		ext, t := drive.ClassifyFileTypeByName(name, info.IsDir())
		_ = s.store.UpsertDriveEntry(&store.DriveEntry{
			AccountID:  acc.ID,
			Path:       rel,
			ParentPath: parentRel,
			Name:       name,
			Ext:        ext,
			Type:       string(t),
			IsDir:      info.IsDir(),
			SizeBytes:  info.Size(),
			ModifiedAt: info.ModTime().Unix(),
		})
	}
	return nil
}
