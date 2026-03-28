//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func ensureElevatedOnWindows(requireAdmin bool) error {
	if !requireAdmin {
		return nil
	}

	isAdmin, err := isProcessElevated()
	if err != nil {
		return fmt.Errorf("check admin privilege: %w", err)
	}
	if isAdmin {
		return nil
	}
	return relaunchAsAdministrator()
}

func isProcessElevated() (bool, error) {
	adminSID, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return false, err
	}
	token := windows.Token(0)
	isMember, err := token.IsMember(adminSID)
	if err != nil {
		return false, err
	}
	return isMember, nil
}

func relaunchAsAdministrator() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	quotedArgs := make([]string, 0, len(os.Args)-1)
	for _, a := range os.Args[1:] {
		quotedArgs = append(quotedArgs, syscall.EscapeArg(a))
	}
	argString := strings.Join(quotedArgs, " ")
	psScript := fmt.Sprintf(`Start-Process -FilePath '%s' -ArgumentList '%s' -Verb RunAs`, strings.ReplaceAll(exePath, "'", "''"), strings.ReplaceAll(argString, "'", "''"))
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("request UAC elevation failed: %w", err)
	}
	os.Exit(0)
	return nil
}
