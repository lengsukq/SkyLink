package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/smb"
	"github.com/skylink/skylink/internal/store"
)

var smbShareNameRegexp = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func (s *Server) listSMBMappings(c *gin.Context) {
	list, err := s.store.ListSMBMappings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list})
}

func (s *Server) syncLocalSMBMappings(c *gin.Context) {
	if !s.guardSMBWindows(c) {
		return
	}
	localShares, err := smb.ListLocalShares()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	added := 0
	updated := 0
	skipped := 0
	errorsList := make([]string, 0)
	for _, share := range localShares {
		shareName := strings.TrimSpace(share.Name)
		localPath := filepath.Clean(strings.TrimSpace(share.Path))
		if shouldSkipSystemShare(shareName) {
			skipped++
			continue
		}
		if shareName == "" || localPath == "" || localPath == "." {
			skipped++
			continue
		}
		if !smbShareNameRegexp.MatchString(shareName) {
			skipped++
			continue
		}

		existing, err := s.store.GetSMBMappingByShareName(shareName)
		if err != nil && err != sql.ErrNoRows {
			errorsList = append(errorsList, fmt.Sprintf("%s: %s", shareName, err.Error()))
			continue
		}
		if existing == nil || err == sql.ErrNoRows {
			item := &store.SMBMapping{
				Name:      shareName,
				LocalPath: localPath,
				ShareName: shareName,
				Enabled:   true,
				ReadOnly:  false,
			}
			if _, addErr := s.store.AddSMBMapping(item); addErr != nil {
				errorsList = append(errorsList, fmt.Sprintf("%s: %s", shareName, mapSMBStoreError(addErr)))
				continue
			}
			added++
			continue
		}

		existing.Name = shareName
		existing.LocalPath = localPath
		existing.ShareName = shareName
		existing.Enabled = true
		existing.ReadOnly = false
		if updateErr := s.store.UpdateSMBMapping(existing.ID, existing); updateErr != nil {
			errorsList = append(errorsList, fmt.Sprintf("%s: %s", shareName, mapSMBStoreError(updateErr)))
			continue
		}
		updated++
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":      len(errorsList) == 0,
		"added":   added,
		"updated": updated,
		"skipped": skipped,
		"errors":  errorsList,
		"warning": "read_only is defaulted to false when syncing from local SMB shares",
	})
}

func (s *Server) addSMBMapping(c *gin.Context) {
	if !s.guardSMBWindows(c) {
		return
	}
	var req store.SMBMapping
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	normalized, err := normalizeSMBMappingPayload(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.ensureSMBShareNameAvailable(normalized.ShareName, 0); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if normalized.Enabled {
		if err := smb.CreateOrUpdateShare(normalized.ShareName, normalized.LocalPath, normalized.ReadOnly, normalized.GrantAccount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	id, err := s.store.AddSMBMapping(normalized)
	if err != nil {
		if normalized.Enabled {
			_ = smb.DeleteShare(normalized.ShareName)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": mapSMBStoreError(err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "id": id})
}

func (s *Server) updateSMBMapping(c *gin.Context) {
	if !s.guardSMBWindows(c) {
		return
	}
	item, ok := s.getSMBMappingByParamID(c)
	if !ok {
		return
	}

	var req store.SMBMapping
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	normalized, err := normalizeSMBMappingPayload(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.ensureSMBShareNameAvailable(normalized.ShareName, item.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if normalized.Enabled {
		if err := smb.CreateOrUpdateShare(normalized.ShareName, normalized.LocalPath, normalized.ReadOnly, normalized.GrantAccount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if item.ShareName != normalized.ShareName {
			_ = smb.DeleteShare(item.ShareName)
		}
	} else {
		_ = smb.DeleteShare(normalized.ShareName)
		if item.ShareName != normalized.ShareName {
			_ = smb.DeleteShare(item.ShareName)
		}
	}
	if err := s.store.UpdateSMBMapping(item.ID, normalized); err != nil {
		_ = s.restoreSMBShareState(item)
		c.JSON(http.StatusInternalServerError, gin.H{"error": mapSMBStoreError(err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) deleteSMBMapping(c *gin.Context) {
	if !s.guardSMBWindows(c) {
		return
	}
	item, ok := s.getSMBMappingByParamID(c)
	if !ok {
		return
	}
	if err := smb.DeleteShare(item.ShareName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := s.store.DeleteSMBMapping(item.ID); err != nil {
		_ = s.restoreSMBShareState(item)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) startSMBMapping(c *gin.Context) {
	if !s.guardSMBWindows(c) {
		return
	}
	item, ok := s.getSMBMappingByParamID(c)
	if !ok {
		return
	}
	if err := smb.CreateOrUpdateShare(item.ShareName, item.LocalPath, item.ReadOnly, item.GrantAccount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	item.Enabled = true
	if err := s.store.UpdateSMBMapping(item.ID, item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) stopSMBMapping(c *gin.Context) {
	if !s.guardSMBWindows(c) {
		return
	}
	item, ok := s.getSMBMappingByParamID(c)
	if !ok {
		return
	}
	if err := smb.DeleteShare(item.ShareName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	item.Enabled = false
	if err := s.store.UpdateSMBMapping(item.ID, item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) restartSMBMapping(c *gin.Context) {
	if !s.guardSMBWindows(c) {
		return
	}
	item, ok := s.getSMBMappingByParamID(c)
	if !ok {
		return
	}
	if err := smb.CreateOrUpdateShare(item.ShareName, item.LocalPath, item.ReadOnly, item.GrantAccount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	item.Enabled = true
	if err := s.store.UpdateSMBMapping(item.ID, item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) healthSMBMapping(c *gin.Context) {
	if !s.guardSMBWindows(c) {
		return
	}
	item, ok := s.getSMBMappingByParamID(c)
	if !ok {
		return
	}
	if err := smb.HealthCheck(item.ShareName, item.LocalPath); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "share available"})
}

func (s *Server) getSMBMappingByParamID(c *gin.Context) (*store.SMBMapping, bool) {
	id, ok := parsePositiveInt64Param(c, "id")
	if !ok {
		return nil, false
	}
	item, err := s.store.GetSMBMapping(id)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, false
	}
	return item, true
}

func normalizeSMBMappingPayload(req *store.SMBMapping) (*store.SMBMapping, error) {
	name := strings.TrimSpace(req.Name)
	localPath := filepath.Clean(strings.TrimSpace(req.LocalPath))
	shareName := strings.TrimSpace(req.ShareName)
	grant := strings.TrimSpace(req.GrantAccount)
	if name == "" {
		return nil, errf("name is required")
	}
	if localPath == "" || localPath == "." {
		return nil, errf("local_path is required")
	}
	if shareName == "" {
		return nil, errf("share_name is required")
	}
	if !smbShareNameRegexp.MatchString(shareName) {
		return nil, errf("share_name contains invalid characters")
	}
	if strings.Contains(grant, ",") {
		return nil, errf("grant_account must not contain commas")
	}
	return &store.SMBMapping{
		Name:         name,
		LocalPath:    localPath,
		ShareName:    shareName,
		Enabled:      req.Enabled,
		ReadOnly:     req.ReadOnly,
		GrantAccount: grant,
	}, nil
}

func isWindowsRuntime() bool {
	return runtime.GOOS == "windows"
}

func (s *Server) guardSMBWindows(c *gin.Context) bool {
	if isWindowsRuntime() {
		return true
	}
	c.JSON(http.StatusNotImplemented, gin.H{"error": "smb is only supported on windows"})
	return false
}

func (s *Server) restoreSMBShareState(item *store.SMBMapping) error {
	if item == nil {
		return nil
	}
	if !item.Enabled {
		return smb.DeleteShare(item.ShareName)
	}
	return smb.CreateOrUpdateShare(item.ShareName, item.LocalPath, item.ReadOnly, item.GrantAccount)
}

func (s *Server) ensureSMBShareNameAvailable(shareName string, selfID int64) error {
	item, err := s.store.GetSMBMappingByShareName(shareName)
	if err == nil && item != nil && item.ID != selfID {
		return errf("share_name already exists")
	}
	if err == nil || err == sql.ErrNoRows {
		return nil
	}
	return err
}

func mapSMBStoreError(err error) string {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unique constraint failed") && strings.Contains(msg, "smb_mappings.share_name") {
		return "share_name already exists"
	}
	if strings.Contains(msg, "unique constraint failed") && strings.Contains(msg, "smb_mappings.name") {
		return "name already exists"
	}
	return err.Error()
}

func shouldSkipSystemShare(shareName string) bool {
	upper := strings.ToUpper(strings.TrimSpace(shareName))
	if upper == "" {
		return true
	}
	if upper == "ADMIN$" || upper == "IPC$" || upper == "PRINT$" {
		return true
	}
	if len(upper) == 2 && upper[1] == '$' && upper[0] >= 'A' && upper[0] <= 'Z' {
		return true
	}
	return false
}
