package store

import "time"

// ListSMBMappings returns all SMB mapping definitions.
func (s *Store) ListSMBMappings() ([]SMBMapping, error) {
	rows, err := s.db.Query(`
		SELECT id, name, local_path, share_name, enabled, read_only, created_at, updated_at
		FROM smb_mappings
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]SMBMapping, 0)
	for rows.Next() {
		var item SMBMapping
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.LocalPath,
			&item.ShareName,
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

// GetSMBMapping returns one SMB mapping by ID.
func (s *Store) GetSMBMapping(id int64) (*SMBMapping, error) {
	var item SMBMapping
	err := s.db.QueryRow(`
		SELECT id, name, local_path, share_name, enabled, read_only, created_at, updated_at
		FROM smb_mappings
		WHERE id = ?
	`, id).Scan(
		&item.ID,
		&item.Name,
		&item.LocalPath,
		&item.ShareName,
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

// GetSMBMappingByShareName returns one SMB mapping by share name.
func (s *Store) GetSMBMappingByShareName(shareName string) (*SMBMapping, error) {
	var item SMBMapping
	err := s.db.QueryRow(`
		SELECT id, name, local_path, share_name, enabled, read_only, created_at, updated_at
		FROM smb_mappings
		WHERE share_name = ?
	`, shareName).Scan(
		&item.ID,
		&item.Name,
		&item.LocalPath,
		&item.ShareName,
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

// AddSMBMapping inserts one SMB mapping.
func (s *Store) AddSMBMapping(item *SMBMapping) (int64, error) {
	now := time.Now().Unix()
	res, err := s.db.Exec(`
		INSERT INTO smb_mappings (name, local_path, share_name, enabled, read_only, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		item.Name,
		item.LocalPath,
		item.ShareName,
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

// UpdateSMBMapping updates one SMB mapping by ID.
func (s *Store) UpdateSMBMapping(id int64, item *SMBMapping) error {
	_, err := s.db.Exec(`
		UPDATE smb_mappings
		SET name = ?, local_path = ?, share_name = ?, enabled = ?, read_only = ?, updated_at = ?
		WHERE id = ?
	`,
		item.Name,
		item.LocalPath,
		item.ShareName,
		item.Enabled,
		item.ReadOnly,
		time.Now().Unix(),
		id,
	)
	return err
}

// DeleteSMBMapping deletes one SMB mapping by ID.
func (s *Store) DeleteSMBMapping(id int64) error {
	_, err := s.db.Exec(`DELETE FROM smb_mappings WHERE id = ?`, id)
	return err
}
