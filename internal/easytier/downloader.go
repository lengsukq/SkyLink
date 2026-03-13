package easytier

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Platform 描述后端运行平台（仅关心 easytier-daemon 需要的维度）。
type Platform struct {
	OS   string
	Arch string
}

// RuntimeDownloader 负责按版本 + 平台下载并缓存 easytier-daemon。
type RuntimeDownloader struct {
	runtimeDir string
	httpClient *http.Client

	mu        sync.Mutex
	inFlight  map[string]*sync.Cond
}

// NewRuntimeDownloader 创建一个新的 RuntimeDownloader。
func NewRuntimeDownloader(runtimeDir string) *RuntimeDownloader {
	if strings.TrimSpace(runtimeDir) == "" {
		runtimeDir = "./data/easytier-bin"
	}
	return &RuntimeDownloader{
		runtimeDir: runtimeDir,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		inFlight:   make(map[string]*sync.Cond),
	}
}

// CurrentPlatform 返回当前进程所在平台。
func CurrentPlatform() Platform {
	return Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

// EnsureDaemon 确保指定版本和平台的 easytier-daemon 已下载，返回本地二进制路径。
// version 形如 "v2.4.4" 或 "latest"；platform 为空则自动取当前平台。
func (d *RuntimeDownloader) EnsureDaemon(ctx context.Context, version string, platform Platform) (string, error) {
	if platform.OS == "" || platform.Arch == "" {
		platform = CurrentPlatform()
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = "latest"
	}

	key := fmt.Sprintf("%s-%s-%s", version, platform.OS, platform.Arch)
	targetPath := d.binaryPath(version, platform)

	// 快速路径：已存在则直接返回。
	if fi, err := os.Stat(targetPath); err == nil && fi.Mode().IsRegular() && fi.Size() > 0 {
		return targetPath, nil
	}
	// 当 version 为 "latest" 时，可能之前已下载到 resolvedTag 目录（如 v2.4.5），先检查该路径避免重复下载。
	if version == "latest" {
		if _, resolvedTag, err := d.resolveAssetURL(ctx, "latest", platform); err == nil && resolvedTag != "" {
			resolvedPath := d.binaryPath(resolvedTag, platform)
			if fi, err := os.Stat(resolvedPath); err == nil && fi.Mode().IsRegular() && fi.Size() > 0 {
				return resolvedPath, nil
			}
		}
	}

	// 避免并发重复下载。
	d.mu.Lock()
	if cond, ok := d.inFlight[key]; ok {
		// 已有下载在进行中，等待其完成。
		for ok {
			cond.Wait()
			if fi, err := os.Stat(targetPath); err == nil && fi.Mode().IsRegular() && fi.Size() > 0 {
				d.mu.Unlock()
				return targetPath, nil
			}
			_, ok = d.inFlight[key]
			if !ok {
				break
			}
		}
		d.mu.Unlock()
		// 若仍未成功，则视作失败。
		if fi, err := os.Stat(targetPath); err == nil && fi.Mode().IsRegular() && fi.Size() > 0 {
			return targetPath, nil
		}
		return "", fmt.Errorf("failed to download easytier-daemon for %s", key)
	}

	cond := sync.NewCond(&d.mu)
	d.inFlight[key] = cond
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		delete(d.inFlight, key)
		cond.Broadcast()
		d.mu.Unlock()
	}()

	// 真正开始下载；downloadDaemon 会按 resolvedTag 写入，返回实际路径。
	writtenPath, err := d.downloadDaemon(ctx, version, platform)
	if err != nil {
		return "", err
	}
	return writtenPath, nil
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

// InstalledRuntime 表示已下载的某一版本+平台的运行时
type InstalledRuntime struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Path    string `json:"path"`
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

func (d *RuntimeDownloader) binaryPath(version string, platform Platform) string {
	safeVersion := strings.TrimSpace(version)
	if safeVersion == "" {
		safeVersion = "latest"
	}
	label := fmt.Sprintf("%s-%s", platform.OS, platform.Arch)
	binName := "easytier-core"
	if platform.OS == "windows" {
		binName += ".exe"
	}
	return filepath.Join(d.runtimeDir, safeVersion, label, binName)
}

// VersionFromBinaryPath 从绝对路径推导版本：若路径在 runtimeDir 下，返回相对路径的第一段（即版本目录名）；否则返回空。
func (d *RuntimeDownloader) VersionFromBinaryPath(absPath string) string {
	if d == nil || absPath == "" {
		return ""
	}
	absDir, err := filepath.Abs(d.runtimeDir)
	if err != nil {
		return ""
	}
	absPathClean, err := filepath.Abs(absPath)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(absDir, absPathClean)
	if err != nil {
		return ""
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 1 {
		return ""
	}
	return parts[0]
}

// downloadDaemon 下载指定版本与平台的守护进程，写入 resolvedTag 对应目录，返回实际写入的二进制路径。
func (d *RuntimeDownloader) downloadDaemon(ctx context.Context, version string, platform Platform) (string, error) {
	tag := strings.TrimSpace(version)
	if tag == "" {
		tag = "latest"
	}

	assetURL, resolvedTag, err := d.resolveAssetURL(ctx, tag, platform)
	if err != nil {
		return "", err
	}

	// 统一按 resolvedTag 路径写入，避免 "latest" 与真实 tag 不一致导致启动时找不到文件。
	targetPath := d.binaryPath(resolvedTag, platform)

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", fmt.Errorf("mkdir runtime dir: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download easytier-daemon: %s", resp.Status)
	}

	if strings.HasSuffix(strings.ToLower(assetURL), ".zip") {
		if err := d.downloadAndExtractZip(resp.Body, platform, targetPath); err != nil {
			return "", err
		}
		return targetPath, nil
	}

	tmpPath := targetPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	if platform.OS != "windows" {
		if err := os.Chmod(tmpPath, 0o755); err != nil {
			_ = os.Remove(tmpPath)
			return "", err
		}
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	return targetPath, nil
}

// downloadAndExtractZip reads a zip from r, finds easytier-core or easytier-core.exe, and writes it to targetPath.
func (d *RuntimeDownloader) downloadAndExtractZip(r io.Reader, platform Platform, targetPath string) error {
	tmpZip, err := os.CreateTemp(filepath.Dir(targetPath), "easytier-*.zip")
	if err != nil {
		return fmt.Errorf("create temp zip: %w", err)
	}
	tmpZipPath := tmpZip.Name()
	defer func() { _ = os.Remove(tmpZipPath) }()

	if _, err := io.Copy(tmpZip, r); err != nil {
		_ = tmpZip.Close()
		return fmt.Errorf("download zip: %w", err)
	}
	if err := tmpZip.Close(); err != nil {
		return err
	}

	zr, err := zip.OpenReader(tmpZipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	wantName := "easytier-core"
	if platform.OS == "windows" {
		wantName = "easytier-core.exe"
	}
	var found *zip.File
	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		if base == wantName || strings.ToLower(base) == "easytier-core.exe" || strings.ToLower(base) == "easytier-core" {
			found = f
			break
		}
	}
	if found == nil {
		return fmt.Errorf("zip does not contain %s", wantName)
	}

	rc, err := found.Open()
	if err != nil {
		return fmt.Errorf("open zip entry: %w", err)
	}
	defer rc.Close()

	out, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	if _, err := io.Copy(out, rc); err != nil {
		_ = out.Close()
		_ = os.Remove(targetPath)
		return fmt.Errorf("extract: %w", err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(targetPath)
		return err
	}

	if platform.OS != "windows" {
		if err := os.Chmod(targetPath, 0o755); err != nil {
			_ = os.Remove(targetPath)
			return err
		}
	}
	return nil
}

// resolveAssetURL 调用 GitHub Releases API，根据 tag 和平台找到合适的 daemon 资产下载 URL。
func (d *RuntimeDownloader) resolveAssetURL(ctx context.Context, tag string, platform Platform) (assetURL string, resolvedTag string, err error) {
	base := "https://api.github.com/repos/EasyTier/EasyTier/releases"
	url := ""
	if tag == "" || tag == "latest" {
		url = base + "/latest"
	} else {
		url = fmt.Sprintf("%s/tags/%s", base, tag)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github releases api: %s", resp.Status)
	}

	var v struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", "", err
	}

	if len(v.Assets) == 0 {
		return "", "", errors.New("no assets in EasyTier release")
	}

	asset := selectDaemonAsset(v.Assets, platform)
	if asset == nil {
		return "", "", fmt.Errorf("no suitable easytier-daemon asset for %s/%s", platform.OS, platform.Arch)
	}

	return asset.BrowserDownloadURL, v.TagName, nil
}

// excludeAssetNames: release assets that are not the CLI daemon zip (e.g. GUI installers).
var excludeAssetNames = []string{"gui", "magisk", "web-dashboard", "apk", "dmg", "appimage", ".deb", "setup.exe"}

func selectDaemonAsset(assets []struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}, platform Platform) *struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
} {
	osHints := []string{platform.OS}
	switch platform.OS {
	case "darwin":
		osHints = append(osHints, "macos", "mac")
	case "windows":
		osHints = append(osHints, "win")
	}
	archHints := []string{platform.Arch}
	switch platform.Arch {
	case "amd64":
		archHints = append(archHints, "x86_64", "x64")
	case "arm64":
		archHints = append(archHints, "aarch64")
	}

	for i := range assets {
		name := strings.ToLower(assets[i].Name)
		if !strings.Contains(name, "easytier") {
			continue
		}
		if containsAny(name, excludeAssetNames) {
			continue
		}
		if !containsAny(name, osHints) {
			continue
		}
		if !containsAny(name, archHints) {
			continue
		}
		return &assets[i]
	}
	return nil
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if sub != "" && strings.Contains(s, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

// PlatformLabel 供 API 返回给前端的平台项（os + arch + label）
type PlatformLabel struct {
	OS    string `json:"os"`
	Arch  string `json:"arch"`
	Label string `json:"label"`
}

// supportedPlatforms 与 GitHub EasyTier releases 的 CLI zip 资产一致（easytier-<os>-<arch>-*.zip）
var supportedPlatforms = []Platform{
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "darwin", Arch: "amd64"},
	{OS: "darwin", Arch: "arm64"},
	{OS: "windows", Arch: "amd64"},
	{OS: "windows", Arch: "arm64"},
}

// SupportedPlatformsWithLabels 返回支持的平台列表（供版本下拉与下载用）
func SupportedPlatformsWithLabels() []PlatformLabel {
	out := make([]PlatformLabel, 0, len(supportedPlatforms))
	for _, p := range supportedPlatforms {
		out = append(out, PlatformLabel{OS: p.OS, Arch: p.Arch, Label: p.OS + "/" + p.Arch})
	}
	return out
}

