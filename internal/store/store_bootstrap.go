package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

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
	if err := runStoreMigrations(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func runStoreMigrations(db *sql.DB) error {
	if err := migrateDDNSRecordType(db); err != nil {
		return err
	}
	if err := migrateSMBGrantAccount(db); err != nil {
		return err
	}
	if err := migrateDriveAccountsV2(db); err != nil {
		return err
	}
	return nil
}

