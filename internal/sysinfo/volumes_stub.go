//go:build !windows

package sysinfo

// ListLogicalVolumes returns no volumes on non-Windows builds.
func ListLogicalVolumes() ([]LogicalVolume, error) {
	return nil, nil
}
