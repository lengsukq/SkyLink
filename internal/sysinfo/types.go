package sysinfo

// LogicalVolume is a fixed disk volume with space accounting (Windows local drives).
type LogicalVolume struct {
	Path       string `json:"path"`
	Label      string `json:"label,omitempty"`
	TotalBytes uint64 `json:"total_bytes"`
	FreeBytes  uint64 `json:"free_bytes"`
	UsedBytes  uint64 `json:"used_bytes"`
}
