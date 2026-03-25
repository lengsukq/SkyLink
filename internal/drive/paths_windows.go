//go:build windows

package drive

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// ensureNoReparsePoints prevents escaping root through junctions/symlinks (reparse points).
func ensureNoReparsePoints(root string, full string) error {
	root = filepath.Clean(root)
	full = filepath.Clean(full)
	if root == "" || full == "" {
		return ErrInvalidPath
	}

	// Walk from root to full path.
	rel, err := filepath.Rel(root, full)
	if err != nil {
		return ErrInvalidPath
	}
	rel = filepath.Clean(rel)
	if rel == "." {
		return nil
	}

	cur := root
	for _, part := range splitPathParts(rel) {
		cur = filepath.Join(cur, part)
		isReparse, err := pathIsReparsePoint(cur)
		if err != nil {
			return ErrInvalidPath
		}
		if isReparse {
			return ErrPathOutsideRoot
		}
	}
	return nil
}

func splitPathParts(rel string) []string {
	rel = strings.Trim(rel, `\/`)
	if rel == "" {
		return nil
	}
	rel = strings.ReplaceAll(rel, "/", `\`)
	return strings.Split(rel, `\`)
}

func pathIsReparsePoint(p string) (bool, error) {
	ptr, err := windows.UTF16PtrFromString(p)
	if err != nil {
		return false, err
	}
	attrs, err := windows.GetFileAttributes(ptr)
	if err != nil {
		return false, err
	}
	return attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0, nil
}
