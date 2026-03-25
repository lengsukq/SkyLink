package easytier

import (
	"fmt"
	"strconv"
	"strings"
)

// DefaultEasyTierPorts 为 EasyTier 默认占用的端口：RPC(15888) + 各类 listener(11010-11013)。
// 解除端口占用时会依次检查并结束这些端口上的进程。
var DefaultEasyTierPorts = []int{15888, 11010, 11011, 11012, 11013}

// KillProcessOnPort 结束占用指定端口的进程（Unix：lsof + kill -9；Windows：netstat + taskkill）。
func KillProcessOnPort(port int) (killed int, err error) {
	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid port %d", port)
	}
	pids, err := platformPIDsOnPort(port)
	if err != nil {
		return 0, err
	}
	for _, pid := range pids {
		if platformKillPID(pid) == nil {
			killed++
		}
	}
	return killed, nil
}

// PIDsOnPort 返回占用指定端口的进程 PID 列表（监听该端口的进程），无占用时返回 nil。
func PIDsOnPort(port int) ([]int, error) {
	if port <= 0 || port > 65535 {
		return nil, nil
	}
	return platformPIDsOnPort(port)
}

// KillProcessOnPorts 结束占用给定端口列表的进程；同一 PID 只结束一次。返回被结束的进程数和涉及端口。
func KillProcessOnPorts(ports []int) (totalKilled int, portsFreed []int, err error) {
	seenPID := make(map[int]bool)
	for _, port := range ports {
		if port <= 0 || port > 65535 {
			continue
		}
		pids, _ := platformPIDsOnPort(port)
		if len(pids) == 0 {
			continue
		}
		portsFreed = append(portsFreed, port)
		for _, pid := range pids {
			if seenPID[pid] {
				continue
			}
			seenPID[pid] = true
			if platformKillPID(pid) == nil {
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
		// [::1]:port — 最后一个冒号后是端口
		if port, err := strconv.Atoi(p); err == nil && port > 0 && port <= 65535 {
			return port
		}
	}
	if port, err := strconv.Atoi(addr); err == nil && port > 0 && port <= 65535 {
		return port
	}
	return 0
}
