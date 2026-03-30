//go:build !windows

package easytier

import "os/exec"

func configureDaemonCmd(cmd *exec.Cmd, _ bool) {}