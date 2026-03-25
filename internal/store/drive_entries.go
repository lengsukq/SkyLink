package store

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type DriveEntriesSort string

const (
	DriveEntriesSortName  DriveEntriesSort = "name"
	DriveEntriesSortSize  DriveEntriesSort = "size"
	DriveEntriesSortMTime DriveEntriesSort = "mtime"
)

type DriveEntriesOrder string

const (
	DriveEntriesOrderAsc  DriveEntriesOrder = "asc"
	DriveEntriesOrderDesc DriveEntriesOrder = "desc"
)

type ListDriveEntriesParams struct {
	AccountID   int64
	ParentPath  string
	PathPrefix  string
	Recursive   bool
	Type        string
	Query       string
	Sort        DriveEntriesSort
	Order       DriveEntriesOrder
	Limit       int
	Cursor      string
	IncludeDirs bool
}

type ListDriveEntriesResult struct {
	List       []DriveEntry `json:"list"`
	NextCursor string       `json:"next_cursor"`
}

type driveEntriesCursor struct {
	V  any   `json:"v"`
	ID int64 `json:"id"`
}

func (s *Store) UpsertDriveEntry(e *DriveEntry) error {
	now := time.Now().Unix()
	if e.CreatedAt == 0 {
		e.CreatedAt = now
	}
	e.UpdatedAt = now
	isDir := 0
	if e.IsDir {
		isDir = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO drive_entries (account_id, path, parent_path, name, ext, type, is_dir, size_bytes, modified_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(account_id, path) DO UPDATE SET
		   parent_path=excluded.parent_path,
		   name=excluded.name,
		   ext=excluded.ext,
		   type=excluded.type,
		   is_dir=excluded.is_dir,
		   size_bytes=excluded.size_bytes,
		   modified_at=excluded.modified_at,
		   updated_at=excluded.updated_at`,
		e.AccountID, e.Path, e.ParentPath, e.Name, e.Ext, e.Type, isDir, e.SizeBytes, e.ModifiedAt, e.CreatedAt, e.UpdatedAt,
	)
	return err
}

func (s *Store) DeleteDriveEntryByPath(accountID int64, path string) error {
	_, err := s.db.Exec(`DELETE FROM drive_entries WHERE account_id = ? AND path = ?`, accountID, path)
	return err
}

func (s *Store) DeleteDriveEntriesByPrefix(accountID int64, prefix string) error {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		_, err := s.db.Exec(`DELETE FROM drive_entries WHERE account_id = ?`, accountID)
		return err
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	_, err := s.db.Exec(
		`DELETE FROM drive_entries WHERE account_id = ? AND (path = ? OR path LIKE ?)`,
		accountID, strings.TrimSuffix(prefix, "/"), prefix+"%",
	)
	return err
}

func (s *Store) ListDriveEntries(p ListDriveEntriesParams) (ListDriveEntriesResult, error) {
	if p.AccountID <= 0 {
		return ListDriveEntriesResult{}, fmt.Errorf("account_id is required")
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	if p.Sort == "" {
		p.Sort = DriveEntriesSortName
	}
	if p.Order == "" {
		p.Order = DriveEntriesOrderAsc
	}

	sortExpr, sortType := driveEntriesSortExpr(p.Sort)
	order := "ASC"
	if strings.EqualFold(string(p.Order), "desc") {
		order = "DESC"
	}

	where := []string{"account_id = ?"}
	args := []any{p.AccountID}

	if p.Recursive {
		prefix := strings.TrimSpace(p.PathPrefix)
		if prefix != "" {
			prefix = strings.TrimSuffix(prefix, "/")
			where = append(where, "(path = ? OR path LIKE ?)")
			args = append(args, prefix, prefix+"/%")
		}
	} else {
		where = append(where, "parent_path = ?")
		args = append(args, strings.TrimSuffix(strings.TrimSpace(p.ParentPath), "/"))
	}

	if t := strings.TrimSpace(p.Type); t != "" {
		where = append(where, "type = ?")
		args = append(args, t)
	}
	if q := strings.TrimSpace(p.Query); q != "" {
		where = append(where, "LOWER(name) LIKE ?")
		args = append(args, "%"+strings.ToLower(q)+"%")
	}
	if !p.IncludeDirs {
		where = append(where, "is_dir = 0")
	}

	cur, err := decodeDriveEntriesCursor(p.Cursor)
	if err != nil {
		return ListDriveEntriesResult{}, err
	}
	if cur != nil {
		op := ">"
		if order == "DESC" {
			op = "<"
		}
		where = append(where, fmt.Sprintf("(%s %s ? OR (%s = ? AND id %s ?))", sortExpr, op, sortExpr, op))
		args = append(args, cur.V, cur.V, cur.ID)
	}

	sqlQuery := fmt.Sprintf(
		`SELECT id, account_id, path, parent_path, name, ext, type, is_dir, size_bytes, modified_at, created_at, updated_at
		 FROM drive_entries
		 WHERE %s
		 ORDER BY is_dir DESC, %s %s, id %s
		 LIMIT ?`,
		strings.Join(where, " AND "),
		sortExpr,
		order,
		order,
	)
	args = append(args, limit+1)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return ListDriveEntriesResult{}, err
	}
	defer rows.Close()

	out := make([]DriveEntry, 0, limit)
	for rows.Next() {
		var e DriveEntry
		var isDir int
		if err := rows.Scan(
			&e.ID, &e.AccountID, &e.Path, &e.ParentPath, &e.Name, &e.Ext, &e.Type, &isDir,
			&e.SizeBytes, &e.ModifiedAt, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return ListDriveEntriesResult{}, err
		}
		e.IsDir = isDir == 1
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return ListDriveEntriesResult{}, err
	}

	nextCursor := ""
	if len(out) > limit {
		last := out[limit-1]
		out = out[:limit]
		v, _ := driveEntriesCursorValue(last, p.Sort, sortType)
		nextCursor = encodeDriveEntriesCursor(&driveEntriesCursor{V: v, ID: last.ID})
	}

	return ListDriveEntriesResult{List: out, NextCursor: nextCursor}, nil
}

func driveEntriesSortExpr(sort DriveEntriesSort) (expr string, sortType string) {
	switch sort {
	case DriveEntriesSortSize:
		return "size_bytes", "int"
	case DriveEntriesSortMTime:
		return "modified_at", "int"
	case DriveEntriesSortName:
		fallthrough
	default:
		return "LOWER(name)", "string"
	}
}

func driveEntriesCursorValue(e DriveEntry, sort DriveEntriesSort, sortType string) (any, error) {
	switch sort {
	case DriveEntriesSortSize:
		return e.SizeBytes, nil
	case DriveEntriesSortMTime:
		return e.ModifiedAt, nil
	case DriveEntriesSortName:
		fallthrough
	default:
		return strings.ToLower(e.Name), nil
	}
}

func encodeDriveEntriesCursor(c *driveEntriesCursor) string {
	if c == nil {
		return ""
	}
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeDriveEntriesCursor(v string) (*driveEntriesCursor, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(v)
	if err != nil {
		return nil, err
	}
	var c driveEntriesCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.ID <= 0 {
		return nil, sql.ErrNoRows
	}
	return &c, nil
}
