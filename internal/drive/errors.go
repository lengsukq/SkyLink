package drive

import "errors"

var (
	ErrInvalidPath      = errors.New("invalid path")
	ErrPathOutsideRoot  = errors.New("path outside root")
	ErrRootNotFound     = errors.New("root path not found")
	ErrRootNotDirectory = errors.New("root path must be a directory")
	ErrQuotaExceeded    = errors.New("quota exceeded")
	ErrNotSupported     = errors.New("not supported")
)

