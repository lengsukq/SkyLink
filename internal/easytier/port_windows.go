//go:build windows

package easytier

import (
	"os/exec"
	"strconv"
	"strings"
)

func platformPIDsOnPort(port int) ([]int, error) {
	cmd := exec.Command("cmd", "/c", "netstat", "-ano", "-p", "tcp")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	portSuffix := ":" + strconv.Itoa(port)
	lines := strings.Split(string(out), "\n")
	seen := make(map[int]bool)
	var pids []int
	for _, raw := range lines {
		line := strings.TrimSpace(strings.ReplaceAll(raw, "\r", ""))
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		if !strings.EqualFold(fields[0], "tcp") {
			continue
		}
		localAddr := fields[1]
		state := fields[3]
		if !strings.EqualFold(state, "LISTENING") {
			continue
		}
		if !strings.HasSuffix(localAddr, portSuffix) {
			continue
		}
		pid, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil || pid <= 0 || seen[pid] {
			continue
		}
		seen[pid] = true
		pids = append(pids, pid)
	}
	return pids, nil
}

func platformKillPID(pid int) error {
	return exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F").Run()
}
