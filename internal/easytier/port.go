package easytier

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// DefaultEasyTierPorts 为 EasyTier 默认占用的端口：RPC(15888) + 各类 listener(11010-11013)。
// 解除端口占用时会依次检查并结束这些端口上的进程。
var DefaultEasyTierPorts = []int{15888, 11010, 11011, 11012, 11013}

// KillProcessOnPort 结束占用指定端口的进程（Unix：lsof + kill -9；Windows 未实现返回错误）。
func KillProcessOnPort(port int) (killed int, err error) {
	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid port %d", port)
	}
	if runtime.GOOS == "windows" {
		return 0, fmt.Errorf("release port on Windows is not supported, please close the process manually")
	}
	// lsof -ti :PORT 输出占用该端口的 PID，每行一个；无占用时无输出且退出码非 0
	cmd := exec.Command("lsof", "-ti", ":"+strconv.Itoa(port))
	out, cmdErr := cmd.Output()
	if cmdErr != nil && len(out) == 0 {
		// 无输出表示没有进程占用该端口
		return 0, nil
	}
	if cmdErr != nil {
		return 0, fmt.Errorf("lsof: %w", cmdErr)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		killCmd := exec.Command("kill", "-9", strconv.Itoa(pid))
		if killErr := killCmd.Run(); killErr == nil {
			killed++
		}
	}
	return killed, nil
}

// PIDsOnPort 返回占用指定端口的进程 PID 列表，无占用时返回 nil。
func PIDsOnPort(port int) ([]int, error) {
	if port <= 0 || port > 65535 || runtime.GOOS == "windows" {
		return nil, nil
	}
	cmd := exec.Command("lsof", "-ti", ":"+strconv.Itoa(port))
	out, cmdErr := cmd.Output()
	if cmdErr != nil || len(out) == 0 {
		return nil, nil
	}
	var pids []int
	seen := make(map[int]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil || seen[pid] {
			continue
		}
		seen[pid] = true
		pids = append(pids, pid)
	}
	return pids, nil
}

// KillProcessOnPorts 结束占用给定端口列表的进程；同一 PID 只结束一次。返回被结束的进程数和涉及端口。
func KillProcessOnPorts(ports []int) (totalKilled int, portsFreed []int, err error) {
	if runtime.GOOS == "windows" {
		return 0, nil, fmt.Errorf("release port on Windows is not supported")
	}
	seenPID := make(map[int]bool)
	for _, port := range ports {
		if port <= 0 || port > 65535 {
			continue
		}
		pids, _ := PIDsOnPort(port)
		if len(pids) == 0 {
			continue
		}
		portsFreed = append(portsFreed, port)
		for _, pid := range pids {
			if seenPID[pid] {
				continue
			}
			seenPID[pid] = true
			killCmd := exec.Command("kill", "-9", strconv.Itoa(pid))
			if killCmd.Run() == nil {
				totalKilled++
			}
		}
	}
	return totalKilled, portsFreed, nil
}

// ParsePortFromAddress 从 "host:port" 或 ":port" 或 "port" 中解析出端口号，无法解析时返回 0。
func ParsePortFromAddress(addr string) int {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return 0
	}
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		p := strings.TrimSpace(addr[idx+1:])
		if port, err := strconv.Atoi(p); err == nil && port > 0 && port <= 65535 {
			return port
		}
	}
	if port, err := strconv.Atoi(addr); err == nil && port > 0 && port <= 65535 {
		return port
	}
	return 0
}
