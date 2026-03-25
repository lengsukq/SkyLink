//go:build !windows

package easytier

import (
	"os/exec"
	"strconv"
	"strings"
)

func platformPIDsOnPort(port int) ([]int, error) {
	cmd := exec.Command("lsof", "-ti", ":"+strconv.Itoa(port))
	out, cmdErr := cmd.Output()
	if cmdErr != nil && len(out) == 0 {
		return nil, nil
	}
	if cmdErr != nil {
		return nil, cmdErr
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

func platformKillPID(pid int) error {
	return exec.Command("kill", "-9", strconv.Itoa(pid)).Run()
}
