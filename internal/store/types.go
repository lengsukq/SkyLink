package store

// Mapping 域名 -> 后端地址的映射（反代路由）
type Mapping struct {
	ID        int64  `json:"id"`
	Host      string `json:"host"`       // 如 xx.yyy.com
	Backend   string `json:"backend"`    // 如 http://127.0.0.1:3000
	CreatedAt int64  `json:"created_at"`
}

// DDNSConfig DDNS 配置：定时更新某条 A 记录为当前公网 IP
type DDNSConfig struct {
	ID          int64  `json:"id"`
	ZoneID      string `json:"zone_id"`
	RecordName  string `json:"record_name"`  // 如 @ 或 sub
	RecordID    string `json:"record_id"`   // CF 记录 ID，更新时用
	IntervalMin int    `json:"interval_min"` // 更新间隔（分钟）
	Enabled     bool   `json:"enabled"`
	LastIP      string `json:"last_ip"`
	UpdatedAt   int64  `json:"updated_at"`
}
