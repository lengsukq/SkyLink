//go:build !windows

package main

func ensureElevatedOnWindows() error { return nil }
