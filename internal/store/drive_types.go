package store

// DriveAccount 网盘账号（用户名+密码登录）
type DriveAccount struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"` // never expose
	RootPath     string `json:"root_path"`
	QuotaBytes   int64  `json:"quota_bytes"`
	UsedBytes    int64  `json:"used_bytes"`
	Enabled      bool   `json:"enabled"`
	LastUsedAt   *int64 `json:"last_used_at"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

