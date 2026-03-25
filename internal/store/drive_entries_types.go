package store

type DriveEntry struct {
	ID         int64  `json:"id"`
	AccountID  int64  `json:"account_id"`
	Path       string `json:"path"`
	ParentPath string `json:"parent_path"`
	Name       string `json:"name"`
	Ext        string `json:"ext"`
	Type       string `json:"type"`
	IsDir      bool   `json:"is_dir"`
	SizeBytes  int64  `json:"size_bytes"`
	ModifiedAt int64  `json:"modified_at"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}
