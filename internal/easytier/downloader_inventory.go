package easytier

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InstalledRuntime 表示已下载的某一版本+平台的运行时
type InstalledRuntime struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Path    string `json:"path"`
}

// HasDaemon 返回指定版本和平台的 easytier-daemon 是否已在本地准备就绪。
// 该方法不会触发下载，仅检查缓存目录中是否存在可执行文件。
func (d *RuntimeDownloader) HasDaemon(version string, platform Platform) bool {
	if d == nil {
		return false
	}
	if platform.OS == "" || platform.Arch == "" {
		platform = CurrentPlatform()
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = "latest"
	}
	targetPath := d.binaryPath(version, platform)
	if fi, err := os.Stat(targetPath); err == nil && fi.Mode().IsRegular() && fi.Size() > 0 {
		return true
	}
	return false
}

// ListInstalled 扫描 runtimeDir，返回所有已下载的版本+平台列表。
func (d *RuntimeDownloader) ListInstalled() ([]InstalledRuntime, error) {
	if d == nil {
		return nil, nil
	}
	entries, err := os.ReadDir(d.runtimeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []InstalledRuntime
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		version := e.Name()
		versionDir := filepath.Join(d.runtimeDir, version)
		subs, err := os.ReadDir(versionDir)
		if err != nil {
			continue
		}
		for _, sub := range subs {
			if !sub.IsDir() {
				continue
			}
			label := sub.Name()
			parts := strings.SplitN(label, "-", 2)
			if len(parts) != 2 {
				continue
			}
			platform := Platform{OS: parts[0], Arch: parts[1]}
			targetPath := d.binaryPath(version, platform)
			fi, err := os.Stat(targetPath)
			if err != nil || !fi.Mode().IsRegular() || fi.Size() == 0 {
				continue
			}
			absPath, _ := filepath.Abs(targetPath)
			out = append(out, InstalledRuntime{
				Version: version,
				OS:      platform.OS,
				Arch:    platform.Arch,
				Path:    absPath,
			})
		}
	}
	return out, nil
}

// DaemonPath 返回指定版本和平台对应的本地二进制路径（不检查是否存在）
func (d *RuntimeDownloader) DaemonPath(version string, platform Platform) string {
	if d == nil {
		return ""
	}
	if platform.OS == "" || platform.Arch == "" {
		platform = CurrentPlatform()
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = "latest"
	}
	return d.binaryPath(version, platform)
}

// RemoveDaemon 移除已下载的指定版本+平台的二进制文件；仅当路径在 runtimeDir 内时执行。
func (d *RuntimeDownloader) RemoveDaemon(version string, platform Platform) error {
	if d == nil {
		return fmt.Errorf("runtime downloader is nil")
	}
	if platform.OS == "" || platform.Arch == "" {
		platform = CurrentPlatform()
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = "latest"
	}
	targetPath := d.binaryPath(version, platform)
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}
	absDir, err := filepath.Abs(d.runtimeDir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absDir, absTarget)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path outside runtime dir")
	}
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	parentDir := filepath.Dir(targetPath)
	if dirEntries, err := os.ReadDir(parentDir); err == nil && len(dirEntries) == 0 {
		_ = os.Remove(parentDir)
	}
	return nil
}
