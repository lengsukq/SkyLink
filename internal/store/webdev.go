package store

import (
	"time"
)

// ListWebDAVMappings 返回所有 WebDAV 映射定义。
func (s *Store) ListWebDAVMappings() ([]WebDAVMapping, error) {
	rows, err := s.db.Query(`
		SELECT id, name, local_path, username, password, enabled, read_only, created_at, updated_at
		FROM webdav_mappings
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]WebDAVMapping, 0)
	for rows.Next() {
		var item WebDAVMapping
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.LocalPath,
			&item.Username,
			&item.Password,
			&item.Enabled,
			&item.ReadOnly,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// GetWebDAVMapping 按 ID 获取映射定义。
func (s *Store) GetWebDAVMapping(id int64) (*WebDAVMapping, error) {
	var item WebDAVMapping
	err := s.db.QueryRow(`
		SELECT id, name, local_path, username, password, enabled, read_only, created_at, updated_at
		FROM webdav_mappings
		WHERE id = ?
	`, id).Scan(
		&item.ID,
		&item.Name,
		&item.LocalPath,
		&item.Username,
		&item.Password,
		&item.Enabled,
		&item.ReadOnly,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// AddWebDAVMapping 新增 WebDAV 映射定义。
func (s *Store) AddWebDAVMapping(item *WebDAVMapping) (int64, error) {
	now := time.Now().Unix()
	res, err := s.db.Exec(`
		INSERT INTO webdav_mappings (name, local_path, username, password, enabled, read_only, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.Name,
		item.LocalPath,
		item.Username,
		item.Password,
		item.Enabled,
		item.ReadOnly,
		now,
		now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateWebDAVMapping 更新 WebDAV 映射定义。
func (s *Store) UpdateWebDAVMapping(id int64, item *WebDAVMapping) error {
	_, err := s.db.Exec(`
		UPDATE webdav_mappings
		SET name = ?, local_path = ?, username = ?, password = ?, enabled = ?, read_only = ?, updated_at = ?
		WHERE id = ?
	`,
		item.Name,
		item.LocalPath,
		item.Username,
		item.Password,
		item.Enabled,
		item.ReadOnly,
		time.Now().Unix(),
		id,
	)
	return err
}

// DeleteWebDAVMapping 删除 WebDAV 映射定义。
func (s *Store) DeleteWebDAVMapping(id int64) error {
	_, err := s.db.Exec(`DELETE FROM webdav_mappings WHERE id = ?`, id)
	return err
}
