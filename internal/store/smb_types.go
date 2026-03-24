package store

// SMBMapping defines one local directory SMB share mapping.
type SMBMapping struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	LocalPath string `json:"local_path"`
	ShareName string `json:"share_name"`
	Enabled   bool   `json:"enabled"`
	ReadOnly  bool   `json:"read_only"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}
