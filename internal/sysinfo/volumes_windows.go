//go:build windows

package sysinfo

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// ListLogicalVolumes enumerates fixed local drives and reports total / free space.
func ListLogicalVolumes() ([]LogicalVolume, error) {
	buf := make([]uint16, 256)
	n, err := windows.GetLogicalDriveStrings(uint32(len(buf)), &buf[0])
	if err != nil {
		return nil, fmt.Errorf("GetLogicalDriveStrings: %w", err)
	}
	if n == 0 {
		return nil, nil
	}
	if n > uint32(len(buf)) {
		buf = make([]uint16, n)
		if _, err := windows.GetLogicalDriveStrings(uint32(len(buf)), &buf[0]); err != nil {
			return nil, fmt.Errorf("GetLogicalDriveStrings (resize): %w", err)
		}
	}

	drives := parseNullSeparatedUTF16(buf[:n])
	out := make([]LogicalVolume, 0, len(drives))
	for _, drive := range drives {
		if drive == "" {
			continue
		}
		rootPtr, err := windows.UTF16PtrFromString(drive)
		if err != nil {
			continue
		}
		if windows.GetDriveType(rootPtr) != windows.DRIVE_FIXED {
			continue
		}
		var free, total, _ uint64
		if err := windows.GetDiskFreeSpaceEx(rootPtr, &free, &total, nil); err != nil {
			continue
		}
		var label string
		var volName [256]uint16
		if e := windows.GetVolumeInformation(rootPtr, &volName[0], uint32(len(volName)), nil, nil, nil, nil, 0); e == nil {
			label = windows.UTF16ToString(volName[:])
		}
		used := uint64(0)
		if total >= free {
			used = total - free
		}
		out = append(out, LogicalVolume{
			Path:       drive,
			Label:      label,
			TotalBytes: total,
			FreeBytes:  free,
			UsedBytes:  used,
		})
	}
	return out, nil
}

func parseNullSeparatedUTF16(buf []uint16) []string {
	var out []string
	for i := 0; i < len(buf); {
		if buf[i] == 0 {
			break
		}
		end := i
		for end < len(buf) && buf[end] != 0 {
			end++
		}
		s := windows.UTF16ToString(buf[i:end])
		if s != "" {
			out = append(out, s)
		}
		i = end + 1
	}
	return out
}
