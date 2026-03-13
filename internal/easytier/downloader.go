package easytier

import (
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

	// 真正开始下载。
	if err := d.downloadDaemon(ctx, version, platform, targetPath); err != nil {
		return "", err
	}
	return targetPath, nil
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

func (d *RuntimeDownloader) downloadDaemon(ctx context.Context, version string, platform Platform, targetPath string) error {
	tag := strings.TrimSpace(version)
	if tag == "" {
		tag = "latest"
	}

	assetURL, resolvedTag, err := d.resolveAssetURL(ctx, tag, platform)
	if err != nil {
		return err
	}

	// 若 resolve 过程中调整了 tag，则更新路径，避免目录名与实际版本不符。
	targetPath = d.binaryPath(resolvedTag, platform)

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("mkdir runtime dir: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return err
	}
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download easytier-daemon: %s", resp.Status)
	}

	tmpPath := targetPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if platform.OS != "windows" {
		if err := os.Chmod(tmpPath, 0o755); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
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
		archHints = append(archHints, "x86_64")
	case "arm64":
		archHints = append(archHints, "aarch64")
	}

	for i := range assets {
		name := strings.ToLower(assets[i].Name)
		if !strings.Contains(name, "core") {
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
	// 回退：若找不到严格匹配的，就返回第一个包含 core 的 asset。
	for i := range assets {
		if strings.Contains(strings.ToLower(assets[i].Name), "core") {
			return &assets[i]
		}
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

