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
	if err := applySQLiteConnectionSettings(db); err != nil {
		db.Close()
		return nil, err
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

// applySQLiteConnectionSettings 降低 SQLITE_BUSY：多连接并发写时 SQLite 只允许多个读者或单个写入者。
// 索引重建会产生大量连续写入，默认连接池会开多条连接，容易与其它 API 写入交错并立即报 busy。
func applySQLiteConnectionSettings(db *sql.DB) error {
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	pragmas := []string{
		`PRAGMA busy_timeout = 20000`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA synchronous = NORMAL`,
		`PRAGMA foreign_keys = ON`,
	}
	for _, q := range pragmas {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("sqlite pragma %q: %w", q, err)
		}
	}
	return nil
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

