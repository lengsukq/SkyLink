package store

import (
	"database/sql"
	"time"
)

// ListMappings 返回所有映射
func (s *Store) ListMappings() ([]Mapping, error) {
	rows, err := s.db.Query(
		`SELECT id, host, backend, created_at FROM mappings ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Mapping
	for rows.Next() {
		var m Mapping
		if err := rows.Scan(&m.ID, &m.Host, &m.Backend, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetMappingByHost 按 Host 查一条
func (s *Store) GetMappingByHost(host string) (*Mapping, error) {
	var m Mapping
	err := s.db.QueryRow(
		`SELECT id, host, backend, created_at FROM mappings WHERE host = ?`,
		host,
	).Scan(&m.ID, &m.Host, &m.Backend, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// AddMapping 新增映射
func (s *Store) AddMapping(host, backend string) (int64, error) {
	now := time.Now().Unix()
	res, err := s.db.Exec(
		`INSERT INTO mappings (host, backend, created_at) VALUES (?, ?, ?)`,
		host, backend, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateMapping 更新后端地址
func (s *Store) UpdateMapping(id int64, backend string) error {
	_, err := s.db.Exec(`UPDATE mappings SET backend = ? WHERE id = ?`, backend, id)
	return err
}

// DeleteMapping 删除映射
func (s *Store) DeleteMapping(id int64) error {
	_, err := s.db.Exec(`DELETE FROM mappings WHERE id = ?`, id)
	return err
}

// HostToBackend 返回 host -> backend 的映射表（供反代查表）
func (s *Store) HostToBackend() (map[string]string, error) {
	list, err := s.ListMappings()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(list))
	for _, m := range list {
		out[m.Host] = m.Backend
	}
	return out, nil
}
