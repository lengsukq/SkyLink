//go:build !windows

package drive

func ensureNoReparsePoints(root string, full string) error {
	return nil
}
