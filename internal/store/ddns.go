package store

import (
	"database/sql"
	"time"
)

// ListDDNSConfigs 返回指定 CF 账号下的所有 DDNS 配置；cfAccountID 为 0 时返回所有
func (s *Store) ListDDNSConfigs(cfAccountID int64) ([]DDNSConfig, error) {
	query := `SELECT id, cf_account_id, zone_id, record_name, record_id, interval_min, enabled, last_ip, updated_at FROM ddns_config`
	var rows *sql.Rows
	var err error
	if cfAccountID > 0 {
		query += ` WHERE cf_account_id = ? ORDER BY id`
		rows, err = s.db.Query(query, cfAccountID)
	} else {
		query += ` ORDER BY id`
		rows, err = s.db.Query(query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DDNSConfig
	for rows.Next() {
		var d DDNSConfig
		var enabled int
		if err := rows.Scan(&d.ID, &d.CFAccountID, &d.ZoneID, &d.RecordName, &d.RecordID, &d.IntervalMin, &enabled, &d.LastIP, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.Enabled = enabled != 0
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDDNSConfig 按 ID 获取
func (s *Store) GetDDNSConfig(id int64) (*DDNSConfig, error) {
	var d DDNSConfig
	var enabled int
	err := s.db.QueryRow(
		`SELECT id, cf_account_id, zone_id, record_name, record_id, interval_min, enabled, last_ip, updated_at FROM ddns_config WHERE id = ?`,
		id,
	).Scan(&d.ID, &d.CFAccountID, &d.ZoneID, &d.RecordName, &d.RecordID, &d.IntervalMin, &enabled, &d.LastIP, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.Enabled = enabled != 0
	return &d, nil
}

// AddDDNSConfig 新增 DDNS 配置
func (s *Store) AddDDNSConfig(cfAccountID int64, zoneID, recordName, recordID string, intervalMin int, enabled bool) (int64, error) {
	now := time.Now().Unix()
	en := 0
	if enabled {
		en = 1
	}
	res, err := s.db.Exec(
		`INSERT INTO ddns_config (cf_account_id, zone_id, record_name, record_id, interval_min, enabled, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		cfAccountID, zoneID, recordName, recordID, intervalMin, en, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateDDNSConfig 更新
func (s *Store) UpdateDDNSConfig(id int64, intervalMin int, enabled bool) error {
	en := 0
	if enabled {
		en = 1
	}
	_, err := s.db.Exec(
		`UPDATE ddns_config SET interval_min = ?, enabled = ? WHERE id = ?`,
		intervalMin, en, id,
	)
	return err
}

// UpdateDDNSLastResult 更新上次 IP 与时间
func (s *Store) UpdateDDNSLastResult(id int64, lastIP string) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(
		`UPDATE ddns_config SET last_ip = ?, updated_at = ? WHERE id = ?`,
		lastIP, now, id,
	)
	return err
}

// DeleteDDNSConfig 删除
func (s *Store) DeleteDDNSConfig(id int64) error {
	_, err := s.db.Exec(`DELETE FROM ddns_config WHERE id = ?`, id)
	return err
}

// ListEnabledDDNSConfigs 仅返回指定 CF 账号下已启用的配置，供 DDNS 协程使用；cfAccountID 为 0 时返回所有
func (s *Store) ListEnabledDDNSConfigs(cfAccountID int64) ([]DDNSConfig, error) {
	query := `SELECT id, cf_account_id, zone_id, record_name, record_id, interval_min, last_ip, updated_at FROM ddns_config WHERE enabled = 1`
	var rows *sql.Rows
	var err error
	if cfAccountID > 0 {
		query += ` AND cf_account_id = ? ORDER BY id`
		rows, err = s.db.Query(query, cfAccountID)
	} else {
		query += ` ORDER BY id`
		rows, err = s.db.Query(query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DDNSConfig
	for rows.Next() {
		var d DDNSConfig
		if err := rows.Scan(&d.ID, &d.CFAccountID, &d.ZoneID, &d.RecordName, &d.RecordID, &d.IntervalMin, &d.LastIP, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.Enabled = true
		out = append(out, d)
	}
	return out, rows.Err()
}
