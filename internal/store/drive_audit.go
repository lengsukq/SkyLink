package store

import "time"

type DriveAuditAction string

const (
	DriveAuditLogin    DriveAuditAction = "login"
	DriveAuditUpload   DriveAuditAction = "upload"
	DriveAuditDelete   DriveAuditAction = "delete"
	DriveAuditRename   DriveAuditAction = "rename"
	DriveAuditMkdir    DriveAuditAction = "mkdir"
	DriveAuditDownload DriveAuditAction = "download"
)

func (s *Store) AddDriveAuditLog(accountID int64, action DriveAuditAction, path string, ip string) error {
	at := time.Now().Unix()
	_, err := s.db.Exec(
		`INSERT INTO drive_audit_logs (account_id, action, path, ip, created_at) VALUES (?, ?, ?, ?, ?)`,
		accountID, string(action), path, ip, at,
	)
	return err
}
