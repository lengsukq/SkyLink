package drive

import (
	"path/filepath"
	"strings"
)

// CleanUserPath normalizes user provided relative paths.
// Accepts "", "/", "a/b", "\\a\\b" and returns a clean, slash-based relative path without leading slash.
func CleanUserPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/" || p == `\` || p == "." {
		return ""
	}
	p = strings.ReplaceAll(p, `\`, `/`)
	p = strings.TrimPrefix(p, "/")
	p = filepath.ToSlash(filepath.Clean(filepath.FromSlash(p)))
	p = strings.TrimPrefix(p, "./")
	if p == "." {
		return ""
	}
	return p
}

// ResolveWithinRoot joins root and userPath, returning the absolute filesystem path.
// It rejects paths escaping root (e.g. "../").
func ResolveWithinRoot(root string, userPath string) (string, error) {
	rootClean := filepath.Clean(strings.TrimSpace(root))
	if rootClean == "" || rootClean == "." {
		return "", ErrInvalidPath
	}

	rel := CleanUserPath(userPath)
	full := filepath.Join(rootClean, filepath.FromSlash(rel))
	full = filepath.Clean(full)

	if !isPathWithinRoot(rootClean, full) {
		return "", ErrPathOutsideRoot
	}
	if err := ensureNoReparsePoints(rootClean, full); err != nil {
		return "", err
	}
	return full, nil
}

func isPathWithinRoot(root string, full string) bool {
	root = filepath.Clean(root)
	full = filepath.Clean(full)

	// Ensure root has a trailing separator for prefix check.
	rootWithSep := root
	if !strings.HasSuffix(rootWithSep, string(filepath.Separator)) {
		rootWithSep += string(filepath.Separator)
	}

	// Windows is case-insensitive.
	rootLower := strings.ToLower(rootWithSep)
	fullLower := strings.ToLower(full)

	if fullLower == strings.ToLower(root) {
		return true
	}
	return strings.HasPrefix(fullLower, rootLower)
}
