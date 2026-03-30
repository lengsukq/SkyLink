package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS mappings (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	host TEXT NOT NULL UNIQUE,
	backend TEXT NOT NULL,
	created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS admin (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	password_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cf_accounts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	api_token TEXT NOT NULL,
	zone_id TEXT,
	created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS ddns_config (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	cf_account_id INTEGER NOT NULL DEFAULT 0,
	zone_id TEXT NOT NULL,
	record_name TEXT NOT NULL,
	record_id TEXT NOT NULL,
	record_type TEXT NOT NULL DEFAULT 'A',
	interval_min INTEGER NOT NULL DEFAULT 10,
	enabled INTEGER NOT NULL DEFAULT 1,
	last_ip TEXT,
	updated_at INTEGER NOT NULL,
	UNIQUE(zone_id, record_name, record_type)
);

CREATE TABLE IF NOT EXISTS webdev_services (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	command TEXT NOT NULL,
	work_dir TEXT NOT NULL DEFAULT '',
	env TEXT NOT NULL DEFAULT '',
	health_type TEXT NOT NULL DEFAULT 'http',
	health_target TEXT NOT NULL DEFAULT '',
	health_timeout INTEGER NOT NULL DEFAULT 3000,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS webdav_mappings (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	local_path TEXT NOT NULL,
	username TEXT NOT NULL,
	password TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	read_only INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS smb_mappings (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	local_path TEXT NOT NULL,
	share_name TEXT NOT NULL UNIQUE,
	enabled INTEGER NOT NULL DEFAULT 1,
	read_only INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS drive_accounts (
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

CREATE TABLE IF NOT EXISTS drive_entries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL,
	path TEXT NOT NULL,
	parent_path TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL,
	ext TEXT NOT NULL DEFAULT '',
	type TEXT NOT NULL DEFAULT 'other',
	is_dir INTEGER NOT NULL DEFAULT 0,
	size_bytes INTEGER NOT NULL DEFAULT 0,
	modified_at INTEGER NOT NULL DEFAULT 0,
	deleted_at INTEGER,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	UNIQUE(account_id, path)
);

CREATE TABLE IF NOT EXISTS drive_audit_logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL,
	action TEXT NOT NULL,
	path TEXT NOT NULL DEFAULT '',
	ip TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_drive_audit_logs_account_at ON drive_audit_logs(account_id, created_at);

CREATE INDEX IF NOT EXISTS idx_drive_entries_account_parent_path ON drive_entries(account_id, parent_path);
CREATE INDEX IF NOT EXISTS idx_drive_entries_account_type ON drive_entries(account_id, type);
CREATE INDEX IF NOT EXISTS idx_drive_entries_account_name ON drive_entries(account_id, name);
CREATE INDEX IF NOT EXISTS idx_drive_entries_account_mtime ON drive_entries(account_id, modified_at);

CREATE INDEX IF NOT EXISTS idx_mappings_host ON mappings(host);
CREATE INDEX IF NOT EXISTS idx_webdev_services_name ON webdev_services(name);
CREATE INDEX IF NOT EXISTS idx_webdav_mappings_name ON webdav_mappings(name);
CREATE INDEX IF NOT EXISTS idx_smb_mappings_name ON smb_mappings(name);
CREATE INDEX IF NOT EXISTS idx_smb_mappings_share_name ON smb_mappings(share_name);
CREATE INDEX IF NOT EXISTS idx_drive_accounts_username ON drive_accounts(username);
`

// Store SQLite 存储
type Store struct {
	db *sql.DB
}

// New 打开或创建数据库并执行迁移
func New(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, defaultDirectoryMode); err != nil {
		return nil, fmt.Errorf("mkdir for db: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	if err := migrateDDNSRecordType(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrateSMBGrantAccount(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrateDriveAccountsV2(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

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
	// If table doesn't exist yet, schema already created the new version.
	// If it exists, inspect columns.
	cols, err := tableColumns(db, "drive_accounts")
	if err != nil {
		return err
	}
	if len(cols) == 0 {
		return nil
	}
	if cols["password_hash"] {
		return nil // already new schema
	}
	if !cols["token_sha256"] || !cols["name"] {
		// Unknown/partial state; do nothing to avoid destructive actions.
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
		// Ensure uniqueness in target table (case-insensitive collisions on Windows).
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
		// password_hash intentionally empty -> must be reset by admin.
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
		// table likely does not exist
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

// Close 关闭数据库
func (s *Store) Close() error {
	return s.db.Close()
}
