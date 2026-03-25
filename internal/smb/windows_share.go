//go:build windows

package smb
import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	errShareNameEmpty = errors.New("share_name is required")
	errPathEmpty      = errors.New("local_path is required")
)

type LocalShare struct {
	Name string
	Path string
}

func CreateOrUpdateShare(shareName string, localPath string, readOnly bool, grantAccount string) error {
	normalizedName := strings.TrimSpace(shareName)
	normalizedPath := filepath.Clean(strings.TrimSpace(localPath))
	if normalizedName == "" {
		return errShareNameEmpty
	}
	if normalizedPath == "" || normalizedPath == "." {
		return errPathEmpty
	}
	if err := validateDirectory(normalizedPath); err != nil {
		return err
	}

	grant := strings.TrimSpace(grantAccount)
	if strings.Contains(grant, ",") {
		return errors.New("grant_account must not contain commas")
	}

	// Make updates idempotent by deleting existing share first.
	_ = DeleteShare(normalizedName)
	args := []string{"share", fmt.Sprintf("%s=%s", normalizedName, normalizedPath)}
	if grant != "" {
		perm := "CHANGE"
		if readOnly {
			perm = "READ"
		}
		args = append(args, fmt.Sprintf("/GRANT:%s,%s", grant, perm))
	} else if readOnly {
		args = append(args, "/GRANT:Everyone,READ")
	}
	if err := runNetCommand(args...); err != nil {
		return err
	}
	return nil
}

func DeleteShare(shareName string) error {
	normalizedName := strings.TrimSpace(shareName)
	if normalizedName == "" {
		return errShareNameEmpty
	}
	if err := runNetCommand("share", normalizedName, "/DELETE", "/Y"); err != nil {
		lowered := strings.ToLower(err.Error())
		if strings.Contains(lowered, "2118") || strings.Contains(lowered, "cannot find the file specified") ||
			strings.Contains(lowered, "2310") {
			return nil
		}
		return err
	}
	return nil
}

func HealthCheck(shareName string, localPath string) error {
	normalizedName := strings.TrimSpace(shareName)
	normalizedPath := filepath.Clean(strings.TrimSpace(localPath))
	if normalizedName == "" {
		return errShareNameEmpty
	}
	if normalizedPath == "" || normalizedPath == "." {
		return errPathEmpty
	}
	if err := validateDirectory(normalizedPath); err != nil {
		return err
	}

	out, err := runNetCommandOutput("share", normalizedName)
	if err != nil {
		lowered := strings.ToLower(err.Error())
		if strings.Contains(lowered, "2310") {
			return errors.New("share not found")
		}
		return err
	}
	if !strings.Contains(strings.ToLower(out), strings.ToLower(normalizedPath)) {
		return errors.New("share exists but target path does not match")
	}
	return nil
}

func ListLocalShares() ([]LocalShare, error) {
	out, err := runNetCommandOutput("share")
	if err != nil {
		return nil, err
	}
	return parseNetShareList(out), nil
}

func validateDirectory(pathValue string) error {
	info, err := os.Stat(pathValue)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("local_path not found")
		}
		return errors.New("local_path is not accessible")
	}
	if !info.IsDir() {
		return errors.New("local_path must be a directory")
	}
	return nil
}

func runNetCommand(args ...string) error {
	_, err := runNetCommandOutput(args...)
	return err
}

func runNetCommandOutput(args ...string) (string, error) {
	cmd := exec.Command("net", args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			return "", fmt.Errorf("net %s failed: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("net %s failed: %s", strings.Join(args, " "), text)
	}
	return text, nil
}

var spacesRegexp = regexp.MustCompile(`\s{2,}`)

func parseNetShareList(text string) []LocalShare {
	lines := strings.Split(text, "\n")
	shares := make([]LocalShare, 0)
	inTable := false
	for _, raw := range lines {
		line := strings.TrimSpace(strings.ReplaceAll(raw, "\r", ""))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "---") {
			inTable = true
			continue
		}
		if !inTable {
			continue
		}
		if strings.Contains(line, "The command completed successfully.") {
			break
		}
		name, path, ok := parseShareLine(line)
		if !ok || name == "" || path == "" {
			continue
		}
		shares = append(shares, LocalShare{Name: name, Path: path})
	}
	return shares
}

func parseShareLine(line string) (string, string, bool) {
	if line == "" {
		return "", "", false
	}
	parts := spacesRegexp.Split(line, -1)
	if len(parts) < 2 {
		return "", "", false
	}
	name := strings.TrimSpace(parts[0])
	path := ""
	for _, p := range parts[1:] {
		trimmed := strings.TrimSpace(p)
		if strings.Contains(trimmed, `:\`) || strings.HasPrefix(trimmed, `\\`) {
			path = trimmed
			break
		}
	}
	return name, path, path != ""
}
