package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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
	interval_min INTEGER NOT NULL DEFAULT 10,
	enabled INTEGER NOT NULL DEFAULT 1,
	last_ip TEXT,
	updated_at INTEGER NOT NULL,
	UNIQUE(zone_id, record_name)
);

CREATE INDEX IF NOT EXISTS idx_mappings_host ON mappings(host);
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
	return &Store{db: db}, nil
}

// Close 关闭数据库
func (s *Store) Close() error {
	return s.db.Close()
}
