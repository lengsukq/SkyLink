package api

import (
	"database/sql"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/skylink/skylink/internal/store"
)

var smbShareNameRegexp = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

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

