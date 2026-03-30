package easytier

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolveCLIPathAdjacentToCore 若与 easytier-core 同目录存在 easytier-cli（Windows 为 .exe），返回其绝对路径。
// 用于 SkyLink 从 RuntimeDownloader 目录启动时无需把 CLI 加入系统 PATH。
// corePath 仅为可执行文件名且无目录时返回空字符串。
func ResolveCLIPathAdjacentToCore(corePath string) string {
	corePath = strings.TrimSpace(corePath)
	if corePath == "" {
		return ""
	}
	clean := filepath.Clean(corePath)
	if !strings.ContainsRune(clean, filepath.Separator) && !strings.Contains(clean, "/") {
		return ""
	}
	if !filepath.IsAbs(clean) {
		abs, err := filepath.Abs(clean)
		if err != nil {
			return ""
		}
		clean = abs
	}
	dir := filepath.Dir(clean)
	name := "easytier-cli"
	if runtime.GOOS == "windows" {
		name = "easytier-cli.exe"
	}
	candidate := filepath.Join(dir, name)
	if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() && fi.Size() > 0 {
		return filepath.Clean(candidate)
	}
	return ""
}
