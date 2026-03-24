package store

import (
	"database/sql"
	"time"
)

// ListCFAccounts 返回所有 Cloudflare 账号配置
func (s *Store) ListCFAccounts() ([]CFAccount, error) {
	rows, err := s.db.Query(
		`SELECT id, name, api_token, zone_id, created_at FROM cf_accounts ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CFAccount
	for rows.Next() {
		var a CFAccount
		if err := rows.Scan(&a.ID, &a.Name, &a.APIToken, &a.ZoneID, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetCFAccount 按 ID 获取 Cloudflare 账号配置
func (s *Store) GetCFAccount(id int64) (*CFAccount, error) {
	var a CFAccount
	err := s.db.QueryRow(
		`SELECT id, name, api_token, zone_id, created_at FROM cf_accounts WHERE id = ?`,
		id,
	).Scan(&a.ID, &a.Name, &a.APIToken, &a.ZoneID, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// CreateCFAccount 新增 Cloudflare 账号配置，返回生成的 ID
func (s *Store) CreateCFAccount(a *CFAccount) (int64, error) {
	if a == nil {
		return 0, sql.ErrConnDone
	}
	now := time.Now().Unix()
	if a.CreatedAt == 0 {
		a.CreatedAt = now
	}
	res, err := s.db.Exec(
		`INSERT INTO cf_accounts (name, api_token, zone_id, created_at) VALUES (?, ?, ?, ?)`,
		a.Name, a.APIToken, a.ZoneID, a.CreatedAt,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	a.ID = id
	return id, nil
}

// UpdateCFAccount 更新 Cloudflare 账号配置
func (s *Store) UpdateCFAccount(a *CFAccount) error {
	if a == nil || a.ID == 0 {
		return sql.ErrConnDone
	}
	_, err := s.db.Exec(
		`UPDATE cf_accounts SET name = ?, api_token = ?, zone_id = ? WHERE id = ?`,
		a.Name, a.APIToken, a.ZoneID, a.ID,
	)
	return err
}

// DeleteCFAccount 删除 Cloudflare 账号配置
func (s *Store) DeleteCFAccount(id int64) error {
	_, err := s.db.Exec(`DELETE FROM cf_accounts WHERE id = ?`, id)
	return err
}
