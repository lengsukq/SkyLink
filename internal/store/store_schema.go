package store

import "database/sql"

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

// Close 关闭数据库
func (s *Store) Close() error {
	return s.db.Close()
}

