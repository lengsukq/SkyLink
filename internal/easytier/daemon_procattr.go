//go:build !windows

package easytier

import "os/exec"

func setDaemonProcAttr(cmd *exec.Cmd) {}