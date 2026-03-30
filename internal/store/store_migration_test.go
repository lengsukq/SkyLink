package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMigrateDriveAccountsV2_FromLegacyTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
CREATE TABLE drive_accounts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  token_sha256 TEXT NOT NULL,
  root_path TEXT NOT NULL,
  quota_bytes INTEGER NOT NULL DEFAULT 0,
  used_bytes INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 1,
  last_used_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
INSERT INTO drive_accounts (id, name, token_sha256, root_path, quota_bytes, used_bytes, enabled, last_used_at, created_at, updated_at)
VALUES
  (1, 'Alice', 'x', '/a', 1, 0, 1, NULL, 100, 100),
  (2, 'alice', 'y', '/b', 2, 0, 1, NULL, 101, 101),
  (3, '', 'z', '/c', 3, 0, 1, NULL, 102, 102);
`)
	if err != nil {
		t.Fatalf("seed legacy table: %v", err)
	}

	if err := migrateDriveAccountsV2(db); err != nil {
		t.Fatalf("migrateDriveAccountsV2: %v", err)
	}

	type accountRow struct {
		ID           int64
		Username     string
		PasswordHash string
	}
	rows, err := db.Query(`SELECT id, username, password_hash FROM drive_accounts ORDER BY id`)
	if err != nil {
		t.Fatalf("query migrated rows: %v", err)
	}
	defer rows.Close()

	var got []accountRow
	for rows.Next() {
		var r accountRow
		if err := rows.Scan(&r.ID, &r.Username, &r.PasswordHash); err != nil {
			t.Fatalf("scan migrated row: %v", err)
		}
		got = append(got, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(got))
	}
	if got[0].Username != "Alice" {
		t.Fatalf("unexpected username for row1: %q", got[0].Username)
	}
	if got[1].Username != "alice-2" {
		t.Fatalf("expected case-insensitive deduped username for row2, got %q", got[1].Username)
	}
	if got[2].Username != "user-3" {
		t.Fatalf("expected fallback username for empty name, got %q", got[2].Username)
	}
	for _, r := range got {
		if r.PasswordHash != "" {
			t.Fatalf("expected empty password_hash after migration, id=%d", r.ID)
		}
	}
}

