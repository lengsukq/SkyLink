//go:build !windows

package main

func ensureElevatedOnWindows(_ bool) error { return nil }
