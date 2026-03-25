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

CREATE INDEX IF NOT EXISTS idx_mappings_host ON mappings(host);
CREATE INDEX IF NOT EXISTS idx_webdev_services_name ON webdev_services(name);
CREATE INDEX IF NOT EXISTS idx_webdav_mappings_name ON webdav_mappings(name);
CREATE INDEX IF NOT EXISTS idx_smb_mappings_name ON smb_mappings(name);
CREATE INDEX IF NOT EXISTS idx_smb_mappings_share_name ON smb_mappings(share_name);
`

// Store SQLite 存储
type Store struct {
	db *sql.DB
}

// New 打开或创建数据库并执行迁移
func New(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
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

// Close 关闭数据库
func (s *Store) Close() error {
	return s.db.Close()
}
