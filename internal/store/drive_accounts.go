package store

import (
	"database/sql"
	"errors"
	"time"
)

var ErrDriveAccountNotFound = errors.New("drive account not found")

// ErrDriveQuotaBelowUsed 限额小于当前已用量时返回。
var ErrDriveQuotaBelowUsed = errors.New("quota_bytes must be at least used_bytes")

type CreateDriveAccountParams struct {
	Username     string
	PasswordHash string
	RootPath     string
	QuotaBytes   int64
	Enabled      bool
}

type UpdateDriveAccountParams struct {
	Username     *string
	PasswordHash *string
	RootPath     *string
	QuotaBytes   *int64
	Enabled      *bool
}

func (s *Store) CreateDriveAccount(p CreateDriveAccountParams) (*DriveAccount, error) {
	now := time.Now().Unix()
	enabled := 0
	if p.Enabled {
		enabled = 1
	}
	res, err := s.db.Exec(
		`INSERT INTO drive_accounts (username, password_hash, root_path, quota_bytes, used_bytes, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 0, ?, ?, ?)`,
		p.Username, p.PasswordHash, p.RootPath, p.QuotaBytes, enabled, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetDriveAccount(id)
}

func (s *Store) ListDriveAccounts() ([]DriveAccount, error) {
	rows, err := s.db.Query(
		`SELECT id, username, password_hash, root_path, quota_bytes, used_bytes, enabled, last_used_at, created_at, updated_at
		 FROM drive_accounts
		 ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DriveAccount
	for rows.Next() {
		var a DriveAccount
		var enabled int
		var lastUsed sql.NullInt64
		if err := rows.Scan(
			&a.ID, &a.Username, &a.PasswordHash, &a.RootPath, &a.QuotaBytes, &a.UsedBytes,
			&enabled, &lastUsed, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		a.Enabled = enabled == 1
		if lastUsed.Valid {
			v := lastUsed.Int64
			a.LastUsedAt = &v
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetDriveAccount(id int64) (*DriveAccount, error) {
	var a DriveAccount
	var enabled int
	var lastUsed sql.NullInt64
	err := s.db.QueryRow(
		`SELECT id, username, password_hash, root_path, quota_bytes, used_bytes, enabled, last_used_at, created_at, updated_at
		 FROM drive_accounts
		 WHERE id = ?`,
		id,
	).Scan(
		&a.ID, &a.Username, &a.PasswordHash, &a.RootPath, &a.QuotaBytes, &a.UsedBytes,
		&enabled, &lastUsed, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Enabled = enabled == 1
	if lastUsed.Valid {
		v := lastUsed.Int64
		a.LastUsedAt = &v
	}
	return &a, nil
}

func (s *Store) UpdateDriveAccount(id int64, p UpdateDriveAccountParams) (*DriveAccount, error) {
	current, err := s.GetDriveAccount(id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrDriveAccountNotFound
	}

	username := current.Username
	passwordHash := current.PasswordHash
	rootPath := current.RootPath
	quota := current.QuotaBytes
	enabled := current.Enabled

	if p.Username != nil {
		username = *p.Username
	}
	if p.PasswordHash != nil {
		passwordHash = *p.PasswordHash
	}
	if p.RootPath != nil {
		rootPath = *p.RootPath
	}
	if p.QuotaBytes != nil {
		q := *p.QuotaBytes
		if q > 0 && q < current.UsedBytes {
			return nil, ErrDriveQuotaBelowUsed
		}
		quota = q
	}
	if p.Enabled != nil {
		enabled = *p.Enabled
	}

	now := time.Now().Unix()
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err = s.db.Exec(
		`UPDATE drive_accounts
		 SET username = ?, password_hash = ?, root_path = ?, quota_bytes = ?, enabled = ?, updated_at = ?
		 WHERE id = ?`,
		username, passwordHash, rootPath, quota, enabledInt, now, id,
	)
	if err != nil {
		return nil, err
	}
	return s.GetDriveAccount(id)
}

func (s *Store) DeleteDriveAccount(id int64) error {
	_, err := s.db.Exec(`DELETE FROM drive_accounts WHERE id = ?`, id)
	return err
}

func (s *Store) TouchDriveAccountLastUsed(id int64, at int64) error {
	_, err := s.db.Exec(`UPDATE drive_accounts SET last_used_at = ? WHERE id = ?`, at, id)
	return err
}

func (s *Store) AddDriveAccountUsedBytes(id int64, delta int64) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(
		`UPDATE drive_accounts
		 SET used_bytes = CASE
		   WHEN used_bytes + ? < 0 THEN 0
		   ELSE used_bytes + ?
		 END,
		 updated_at = ?
		 WHERE id = ?`,
		delta, delta, now, id,
	)
	return err
}

func (s *Store) SetDriveAccountUsedBytes(id int64, used int64) error {
	if used < 0 {
		used = 0
	}
	now := time.Now().Unix()
	_, err := s.db.Exec(
		`UPDATE drive_accounts SET used_bytes = ?, updated_at = ? WHERE id = ?`,
		used, now, id,
	)
	return err
}

func (s *Store) GetDriveAccountByUsername(username string) (*DriveAccount, error) {
	var a DriveAccount
	var enabled int
	var lastUsed sql.NullInt64
	err := s.db.QueryRow(
		`SELECT id, username, password_hash, root_path, quota_bytes, used_bytes, enabled, last_used_at, created_at, updated_at
		 FROM drive_accounts
		 WHERE username = ?`,
		username,
	).Scan(
		&a.ID, &a.Username, &a.PasswordHash, &a.RootPath, &a.QuotaBytes, &a.UsedBytes,
		&enabled, &lastUsed, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Enabled = enabled == 1
	if lastUsed.Valid {
		v := lastUsed.Int64
		a.LastUsedAt = &v
	}
	return &a, nil
}

