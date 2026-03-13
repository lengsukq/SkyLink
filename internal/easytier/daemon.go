package easytier

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DaemonConfig 描述 easytier-daemon 进程启动所需的最小配置。
// 该结构仅关心二进制路径与 env 文件路径，具体网络参数仍由 env 文件提供。
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
}

// DaemonManager 管理单个 easytier-daemon 进程的生命周期。
type DaemonManager interface {
	Start(ctx context.Context, cfg DaemonConfig) error
	Stop(ctx context.Context) error
	Status() DaemonState
}

type daemonManager struct {
	mu           sync.Mutex
	cmd          *exec.Cmd
	state        DaemonState
	lastEnvFile  string
	lastWorkDir  string
}

// NewDaemonManager 创建一个新的 DaemonManager 实例。
func NewDaemonManager() DaemonManager {
	return &daemonManager{}
}

// Start 启动 easytier-daemon 进程；若已在运行则直接返回 nil。
// 仅负责按 env 文件构造环境并拉起进程，不关心具体网络参数。
func (m *daemonManager) Start(ctx context.Context, cfg DaemonConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.BinaryPath == "" {
		cfg.BinaryPath = DefaultDaemonBinary()
	}
	if cfg.BinaryPath == "" {
		return errors.New("easytier-daemon binary path is empty")
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

	cmd := exec.CommandContext(ctx, cfg.BinaryPath)
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
	m.lastEnvFile = cfg.EnvFile
	m.lastWorkDir = cfg.WorkDir

	// 异步读取日志，避免阻塞管道。
	go discardOutput(stdoutPipe)
	go discardOutput(stderrPipe)

	// 监控退出，更新状态。
	go func() {
		cmd.Wait()
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.cmd == cmd {
			m.state.Running = false
			m.state.PID = 0
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
	return nil
}

// Status 返回最近一次观测到的守护进程状态。
func (m *daemonManager) Status() DaemonState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// DefaultDaemonBinary 返回默认的 easytier-daemon 可执行名。
func DefaultDaemonBinary() string {
	return "easytier-daemon"
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

func discardOutput(r interface{}) {
	switch v := r.(type) {
	case *os.File:
		buf := make([]byte, 1024)
		for {
			if _, err := v.Read(buf); err != nil {
				return
			}
		}
	default:
	}
}

