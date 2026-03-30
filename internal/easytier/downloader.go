package easytier

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	mu       sync.Mutex
	inFlight map[string]*sync.Cond
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

	// 先完整下载到临时文件，再按类型解压或落盘，避免 URL 带 query 时漏判 zip、或把压缩包误当裸二进制写入。
	partPath := targetPath + ".part"
	if err := writeHTTPBodyToFile(resp.Body, partPath); err != nil {
		_ = os.Remove(partPath)
		return "", fmt.Errorf("download to temp: %w", err)
	}

	if err := d.installDaemonFromDownloadedPart(partPath, assetURL, platform, targetPath); err != nil {
		_ = os.Remove(partPath)
		_ = os.Remove(targetPath)
		return "", err
	}
	return targetPath, nil
}

func writeHTTPBodyToFile(r io.Reader, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// fileLooksLikeZip 通过魔数判断是否为 zip（本地文件头 PK\x03\x04 等）。
func fileLooksLikeZip(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var hdr [4]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		return false
	}
	return hdr[0] == 'P' && hdr[1] == 'K'
}

func assetURLPathLooksLikeZip(assetURL string) bool {
	u, err := url.Parse(assetURL)
	if err != nil || u.Path == "" {
		return false
	}
	return strings.HasSuffix(strings.ToLower(u.Path), ".zip")
}

// installDaemonFromDownloadedPart 下载完成后立即处理：zip 则解压出 easytier-core；否则将整文件作为可执行文件落盘。
func (d *RuntimeDownloader) installDaemonFromDownloadedPart(partPath, assetURL string, platform Platform, targetPath string) error {
	byMagic := fileLooksLikeZip(partPath)
	byURL := assetURLPathLooksLikeZip(assetURL)
	if byMagic || byURL {
		err := d.extractZipFileToBinary(partPath, platform, targetPath)
		if err == nil {
			_ = os.Remove(partPath)
			return nil
		}
		// 魔数为 zip 或已解压出部分文件：视为致命错误
		if byMagic {
			return fmt.Errorf("extract zip: %w", err)
		}
		// 仅 URL 路径像 zip、魔数不像时，回退为裸二进制（兼容异常响应体）
	}

	if platform.OS != "windows" {
		if err := os.Chmod(partPath, 0o755); err != nil {
			return err
		}
	}
	if err := os.Rename(partPath, targetPath); err != nil {
		return err
	}
	return nil
}

func (d *RuntimeDownloader) extractZipFileToBinary(zipPath string, platform Platform, targetPath string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	return d.extractEasyTierCoreFromZipReader(&zr.Reader, platform, targetPath)
}

// extractEasyTierCoreFromZipReader 将 zip 内文件解压到 targetPath 所在目录。
// Windows 官方包除 easytier-core.exe 外常含 wintun.dll、Packet.dll 等，必须与 exe 同目录，否则进程会立即退出。
// 策略：把所有非目录条目按文件名扁平解压到同一目录（仅用 filepath.Base，避免 zip slip）。
func (d *RuntimeDownloader) extractEasyTierCoreFromZipReader(zr *zip.Reader, platform Platform, targetPath string) error {
	destDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("mkdir extract dir: %w", err)
	}

	wantCore := "easytier-core"
	if platform.OS == "windows" {
		wantCore = "easytier-core.exe"
	}

	var sawCore bool
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if strings.Contains(f.Name, "..") {
			continue
		}
		base := filepath.Base(f.Name)
		if base == "" || base == "." {
			continue
		}
		outPath := filepath.Join(destDir, base)
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %q: %w", f.Name, err)
		}
		out, err := os.Create(outPath)
		if err != nil {
			_ = rc.Close()
			return fmt.Errorf("create %s: %w", outPath, err)
		}
		_, copyErr := io.Copy(out, rc)
		_ = rc.Close()
		if closeErr := out.Close(); closeErr != nil && copyErr == nil {
			copyErr = closeErr
		}
		if copyErr != nil {
			_ = os.Remove(outPath)
			return fmt.Errorf("extract %s: %w", base, copyErr)
		}
		lower := strings.ToLower(base)
		if base == wantCore || lower == "easytier-core.exe" || lower == "easytier-core" {
			sawCore = true
		}
		if platform.OS != "windows" && (lower == "easytier-core" || lower == "easytier-cli") {
			_ = os.Chmod(outPath, 0o755)
		}
	}

	if !sawCore {
		return fmt.Errorf("zip does not contain %s", wantCore)
	}
	if _, err := os.Stat(targetPath); err != nil {
		return fmt.Errorf("after extract, missing %s: %w", filepath.Base(targetPath), err)
	}
	if platform.OS != "windows" {
		if err := os.Chmod(targetPath, 0o755); err != nil {
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

// supportedPlatforms：本系统集成仅支持 Windows 运行时（与官方 easytier-windows-* zip 一致）。
var supportedPlatforms = []Platform{
	{OS: "windows", Arch: "amd64"},
	{OS: "windows", Arch: "arm64"},
}

// IsSupportedRuntimePlatform 是否为本产品允许的下载/缓存维度（仅 Windows）。
func IsSupportedRuntimePlatform(p Platform) bool {
	o := strings.ToLower(strings.TrimSpace(p.OS))
	a := strings.ToLower(strings.TrimSpace(p.Arch))
	return o == "windows" && (a == "amd64" || a == "arm64")
}

// SupportedPlatformsWithLabels 返回支持的平台列表（供版本下拉与下载用）
func SupportedPlatformsWithLabels() []PlatformLabel {
	out := make([]PlatformLabel, 0, len(supportedPlatforms))
	for _, p := range supportedPlatforms {
		label := "Windows amd64"
		if p.Arch == "arm64" {
			label = "Windows arm64"
		}
		out = append(out, PlatformLabel{OS: p.OS, Arch: p.Arch, Label: label})
	}
	return out
}
