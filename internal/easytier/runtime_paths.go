package easytier

import (
	"path/filepath"
	"strings"
)

// RuntimePaths 表示某一 easytier-core 所在「版本运行时目录」下的路径布局，
// 用于在该目录上下文中执行 easytier-cli、或启动 core（WorkDir 与 core 同目录便于加载 DLL）。
type RuntimePaths struct {
	CorePath string // 规范化后的 easytier-core 路径（可能为仅文件名，依赖 PATH）
	CLIPath  string // easytier-cli：优先同目录，否则回退 CLICommand
	WorkDir  string // 建议子进程工作目录；空表示由调用方不设置或沿用 runCLI 默认逻辑
}

// ResolveRuntimePaths 根据 easytier-core 路径解析同版本目录下的 CLI 与工作目录。
// corePath 通常来自 RuntimeDownloader 或配置中的守护进程绝对路径。
func ResolveRuntimePaths(corePath string) RuntimePaths {
	corePath = strings.TrimSpace(corePath)
	norm := normalizeDaemonBinaryPath(corePath)
	cli := ResolveCLIPathAdjacentToCore(norm)
	if cli == "" {
		cli = CLICommand
	}
	workDir := ResolveDaemonWorkDir(norm)
	if workDir == "" && filepath.IsAbs(cli) {
		workDir = filepath.Dir(cli)
	}
	return RuntimePaths{
		CorePath: norm,
		CLIPath:  cli,
		WorkDir:  workDir,
	}
}

// NewRPCClient 基于 RuntimePaths 创建用于 RPC 的 CLI 客户端（在同目录 / WorkDir 下执行 easytier-cli）。
func NewRPCClient(paths RuntimePaths, rpcPortal string) *Client {
	cli := strings.TrimSpace(paths.CLIPath)
	if cli == "" {
		cli = CLICommand
	}
	return &Client{
		cliPath:   cli,
		rpcPortal: rpcPortal,
		workDir:   strings.TrimSpace(paths.WorkDir),
	}
}
