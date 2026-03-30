package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// migrateDDNSRecordType 为 ddns_config 表添加 record_type 列（已有则跳过）
func migrateDDNSRecordType(db *sql.DB) error {
	_, err := db.Exec("SELECT record_type FROM ddns_config LIMIT 1")
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "record_type") || strings.Contains(err.Error(), "no such column") {
		_, err = db.Exec("ALTER TABLE ddns_config ADD COLUMN record_type TEXT NOT NULL DEFAULT 'A'")
		return err
	}
	return err
}

// migrateSMBGrantAccount 为 smb_mappings 增加 grant_account（SMB /GRANT 主体，空则沿用 Everyone 等默认行为）
func migrateSMBGrantAccount(db *sql.DB) error {
	_, err := db.Exec(`SELECT grant_account FROM smb_mappings LIMIT 1`)
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "no such column") || strings.Contains(msg, "grant_account") {
		_, err = db.Exec(`ALTER TABLE smb_mappings ADD COLUMN grant_account TEXT NOT NULL DEFAULT ''`)
		return err
	}
	return err
}

// migrateDriveAccountsV2 将旧版 drive_accounts(token_sha256) 迁移为 username/password_hash 账号体系。
// 迁移后旧 token 方式不可用；管理员需要为账号设置/重置密码。
func migrateDriveAccountsV2(db *sql.DB) error {
	cols, err := tableColumns(db, "drive_accounts")
	if err != nil {
		return err
	}
	if len(cols) == 0 || cols["password_hash"] {
		return nil
	}
	if !cols["token_sha256"] || !cols["name"] {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
CREATE TABLE IF NOT EXISTS drive_accounts_v2 (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  root_path TEXT NOT NULL,
  quota_bytes INTEGER NOT NULL DEFAULT 0,
  used_bytes INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 1,
  last_used_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_drive_accounts_v2_username ON drive_accounts_v2(username);
`)
	if err != nil {
		return err
	}

	type row struct {
		id         int64
		name       string
		rootPath   string
		quotaBytes int64
		usedBytes  int64
		enabled    int
		lastUsed   sql.NullInt64
		createdAt  int64
		updatedAt  int64
	}
	rows, err := tx.Query(`
SELECT id, name, root_path, quota_bytes, used_bytes, enabled, last_used_at, created_at, updated_at
FROM drive_accounts
ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	seen := map[string]bool{}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.name, &r.rootPath, &r.quotaBytes, &r.usedBytes, &r.enabled, &r.lastUsed, &r.createdAt, &r.updatedAt); err != nil {
			return err
		}
		username := strings.TrimSpace(r.name)
		if username == "" {
			username = fmt.Sprintf("user-%d", r.id)
		}
		base := username
		i := 1
		for seen[strings.ToLower(username)] {
			i++
			username = fmt.Sprintf("%s-%d", base, i)
		}
		seen[strings.ToLower(username)] = true

		var lastUsed any = nil
		if r.lastUsed.Valid {
			lastUsed = r.lastUsed.Int64
		}
		if _, err := tx.Exec(
			`INSERT INTO drive_accounts_v2 (id, username, password_hash, root_path, quota_bytes, used_bytes, enabled, last_used_at, created_at, updated_at)
			 VALUES (?, ?, '', ?, ?, ?, ?, ?, ?, ?)`,
			r.id, username, r.rootPath, r.quotaBytes, r.usedBytes, r.enabled, lastUsed, r.createdAt, r.updatedAt,
		); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := tx.Exec(`DROP TABLE drive_accounts`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE drive_accounts_v2 RENAME TO drive_accounts`); err != nil {
		return err
	}
	return tx.Commit()
}

func tableColumns(db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return map[string]bool{}, nil
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

