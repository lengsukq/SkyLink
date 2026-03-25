//go:build !windows

package drive

func ValidateRootPath(root string) error {
	return ErrNotSupported
}

