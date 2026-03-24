//go:build !windows

package smb

import "errors"

var errWindowsOnly = errors.New("smb is only supported on windows")

type LocalShare struct {
	Name string
	Path string
}

func CreateOrUpdateShare(_ string, _ string, _ bool) error { return errWindowsOnly }

func DeleteShare(_ string) error { return errWindowsOnly }

func HealthCheck(_ string, _ string) error { return errWindowsOnly }

func ListLocalShares() ([]LocalShare, error) { return nil, errWindowsOnly }
