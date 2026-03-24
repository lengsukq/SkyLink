package store

import "database/sql"

// GetAdminPasswordHash 返回当前密码 hash；未设置返回空字符串
func (s *Store) GetAdminPasswordHash() (string, error) {
	var hash string
	err := s.db.QueryRow(`SELECT password_hash FROM admin WHERE id = 1`).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return hash, nil
}

// SetAdminPasswordHash 设置/更新管理密码 hash
func (s *Store) SetAdminPasswordHash(hash string) error {
	_, err := s.db.Exec(
		`INSERT INTO admin (id, password_hash) VALUES (1, ?)
		 ON CONFLICT(id) DO UPDATE SET password_hash = excluded.password_hash`,
		hash,
	)
	return err
}
