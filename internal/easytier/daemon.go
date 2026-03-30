package easytier

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const daemonLogMaxBytes = 64 * 1024

// DaemonConfig 描述 easytier-daemon 进程启动所需的最小配置。
// 该结构仅关心二进制路径与 env 文件路径，具体网络参数仍由 env 文件提供。
// WorkDir 非空时作为子进程的工作目录，通常应为二进制所在目录（与官方「在安装/解压目录下启动」一致）。
type DaemonConfig struct {
	BinaryPath string
	EnvFile    string
	WorkDir    string
}

// DaemonState 表示当前守护进程状态，用于对外查询。
type DaemonState struct {
	Running      bool      `json:"running"`
	PID          int       `json:"pid,omitempty"`
	LastStartErr string    `json:"last_start_error,omitempty"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	BinaryPath   string    `json:"binary_path,omitempty"` // 本次启动使用的二进制路径，用于推导「已启动版本」
}

// DaemonManager 管理单个 easytier-daemon 进程的生命周期。
type DaemonManager interface {
	Start(ctx context.Context, cfg DaemonConfig) error
	Stop(ctx context.Context) error
	Status() DaemonState
	Logs() string
}

type daemonManager struct {
	mu          sync.Mutex
	cmd         *exec.Cmd
	state       DaemonState
	lastEnvFile string
	lastWorkDir string
	logBuf      *daemonLogBuffer
}

// daemonLogBuffer 有界缓冲区，保留守护进程 stdout/stderr 的最近输出。
type daemonLogBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
	max int
}

func (b *daemonLogBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, err = b.buf.Write(p)
	if err != nil {
		return n, err
	}
	for b.buf.Len() > b.max {
		discard := b.buf.Len() - b.max
		data := make([]byte, b.buf.Len())
		copy(data, b.buf.Bytes())
		b.buf.Reset()
		_, _ = b.buf.Write(data[discard:])
	}
	return n, nil
}

func (b *daemonLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// NewDaemonManager 创建一个新的 DaemonManager 实例。
func NewDaemonManager() DaemonManager {
	return &daemonManager{
		logBuf: &daemonLogBuffer{max: daemonLogMaxBytes},
	}
}

// Start 启动 easytier-daemon 进程；若已在运行则直接返回 nil。
// 仅负责按 env 文件构造环境并拉起进程，不关心具体网络参数。
func (m *daemonManager) Start(ctx context.Context, cfg DaemonConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.BinaryPath == "" {
		cfg.BinaryPath = DefaultDaemonBinary()
	}
	cfg.BinaryPath = normalizeDaemonBinaryPath(cfg.BinaryPath)
	if cfg.BinaryPath == "" {
		return errors.New("easytier-daemon binary path is empty")
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = ResolveDaemonWorkDir(cfg.BinaryPath)
	}

	if m.cmd != nil && m.cmd.Process != nil {
		if err := m.cmd.Process.Signal(os.Interrupt); err == nil {
			// 已经在运行，认为是幂等启动
			return nil
		}
	}

	env, err := buildEnvFromFile(cfg.EnvFile)
	if err != nil {
		return fmt.Errorf("load easytier env: %w", err)
	}

	// 使用 exec.Command 而非 CommandContext：守护进程需长期运行，不能随请求 context 取消而被杀掉。
	cmd := exec.Command(cfg.BinaryPath)
	setDaemonProcAttr(cmd)
	cmd.Env = append(os.Environ(), env...)
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		m.state.LastStartErr = err.Error()
		m.state.Running = false
		m.state.PID = 0
		return err
	}

	m.cmd = cmd
	m.state.Running = true
	m.state.PID = cmd.Process.Pid
	m.state.StartedAt = time.Now()
	m.state.LastStartErr = ""
	m.state.BinaryPath = cfg.BinaryPath
	m.lastEnvFile = cfg.EnvFile
	m.lastWorkDir = cfg.WorkDir

	// 异步读取日志到有界缓冲区，供 API 查询。
	go func() { _, _ = io.Copy(m.logBuf, stdoutPipe) }()
	go func() { _, _ = io.Copy(m.logBuf, stderrPipe) }()

	// 监控退出，更新状态（进程秒退时 LastStartErr 便于排查缺 DLL、配置错误等）。
	go func() {
		waitErr := cmd.Wait()
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.cmd == cmd {
			m.state.Running = false
			m.state.PID = 0
			m.state.BinaryPath = ""
			if waitErr != nil {
				m.state.LastStartErr = waitErr.Error()
			} else if ps := cmd.ProcessState; ps != nil && !ps.Success() {
				m.state.LastStartErr = fmt.Sprintf("process exited with code %d", ps.ExitCode())
			}
		}
	}()

	return nil
}

// Stop 尝试优雅停止进程；若未运行则为幂等操作。
func (m *daemonManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd == nil || m.cmd.Process == nil {
		return nil
	}

	// 优先尝试发送中断信号
	if err := m.cmd.Process.Signal(os.Interrupt); err != nil {
		// 回退到直接 Kill
		_ = m.cmd.Process.Kill()
	} else {
		done := make(chan struct{})
		go func() {
			_ = m.cmd.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			_ = m.cmd.Process.Kill()
		}
	}

	m.state.Running = false
	m.state.PID = 0
	m.state.BinaryPath = ""
	return nil
}

// Status 返回最近一次观测到的守护进程状态。
func (m *daemonManager) Status() DaemonState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// Logs 返回守护进程最近 stdout/stderr 输出（有界，约最近 64KB）。
func (m *daemonManager) Logs() string {
	if m.logBuf == nil {
		return ""
	}
	return m.logBuf.String()
}

// DefaultDaemonBinary 返回默认的 EasyTier 守护进程可执行名（与 GitHub 发布包内名称一致）。
func DefaultDaemonBinary() string {
	if runtime.GOOS == "windows" {
		return "easytier-core.exe"
	}
	return "easytier-core"
}

// ResolveDaemonWorkDir 返回启动 easytier-core 时应设置的进程工作目录，等价于先 cd 到解压/安装目录再执行。
// 对 RuntimeDownloader 写入的绝对路径，为二进制所在目录；对仅在 PATH 中的名称，为 LookPath 解析到的目录。
// 无法解析时返回空字符串，子进程将继承 SkyLink 的当前工作目录。
func ResolveDaemonWorkDir(binaryPath string) string {
	p := strings.TrimSpace(binaryPath)
	if p == "" {
		return ""
	}
	resolved := p
	if !filepath.IsAbs(p) {
		hasSep := strings.Contains(p, string(filepath.Separator)) || strings.Contains(p, "/")
		if hasSep {
			abs, err := filepath.Abs(p)
			if err != nil {
				return ""
			}
			resolved = abs
		} else {
			lp, err := exec.LookPath(p)
			if err != nil {
				return ""
			}
			resolved = lp
		}
	}
	return filepath.Dir(filepath.Clean(resolved))
}

// normalizeDaemonBinaryPath 将含目录的相对路径转为绝对路径，便于 Windows 上 fork/exec：
// 子进程的可执行文件路径是相对于 SkyLink 进程当前工作目录解析的，而非 cmd.Dir；服务/快捷方式启动时 CWD 常不是项目目录，会导致「已下载却找不到路径」。
// 仅文件名（依赖 PATH）时保持原样。
func normalizeDaemonBinaryPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	clean := filepath.Clean(p)
	if !strings.ContainsRune(clean, filepath.Separator) && !strings.Contains(clean, "/") {
		return clean
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return clean
	}
	return filepath.Clean(abs)
}

// NewDaemonConfig 构造守护进程配置，并自动设置 WorkDir=ResolveDaemonWorkDir(binaryPath)。
// BinaryPath 会经 normalizeDaemonBinaryPath 处理，避免相对路径在 exec 时依赖不可预期的 CWD。
func NewDaemonConfig(binaryPath, envFile string) DaemonConfig {
	norm := normalizeDaemonBinaryPath(binaryPath)
	return DaemonConfig{
		BinaryPath: norm,
		EnvFile:    envFile,
		WorkDir:    ResolveDaemonWorkDir(norm),
	}
}

// buildEnvFromFile 将简单的 KEY=VALUE env 文件解析为环境变量切片。
// 若文件不存在则返回空切片。
func buildEnvFromFile(path string) ([]string, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var env []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "=") {
			continue
		}
		key, value, ok := splitEnvLine(line)
		if !ok || key == "" {
			continue
		}
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	// 若未显式指定工作目录，默认使用 env 文件所在目录，方便 daemon 读写相对路径。
	if len(env) > 0 && path != "" {
		_ = filepath.Dir(path)
	}

	return env, nil
}

func splitEnvLine(line string) (key, value string, ok bool) {
	idx := strings.IndexByte(line, '=')
	if idx <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	value = strings.Trim(value, `"'`)
	return key, value, true
}
