//go:build windows

package drive

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/skylink/skylink/internal/sysinfo"
)

func ValidateRootPath(root string) error {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." {
		return ErrInvalidPath
	}
	if !filepath.IsAbs(root) {
		return ErrInvalidPath
	}
	if !isOnFixedDrive(root) {
		return ErrInvalidPath
	}
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if e := os.MkdirAll(root, 0755); e != nil {
				return ErrRootNotFound
			}
			info, err = os.Stat(root)
			if err != nil {
				return ErrRootNotFound
			}
		}
		if err != nil {
			return ErrInvalidPath
		}
	}
	if !info.IsDir() {
		return ErrRootNotDirectory
	}
	return nil
}

func isOnFixedDrive(path string) bool {
	vol := filepath.VolumeName(path) // e.g. "C:"
	if vol == "" {
		return false
	}
	driveRoot := strings.ToUpper(vol) + `\`
	list, err := sysinfo.ListLogicalVolumes()
	if err != nil {
		return false
	}
	for _, v := range list {
		if strings.EqualFold(strings.TrimSpace(v.Path), driveRoot) {
			return true
		}
	}
	return false
}

