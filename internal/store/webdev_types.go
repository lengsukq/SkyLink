package store

// WebDAVMapping 定义一个本地目录的 WebDAV 映射配置。
type WebDAVMapping struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	LocalPath string `json:"local_path"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Enabled   bool   `json:"enabled"`
	ReadOnly  bool   `json:"read_only"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}
