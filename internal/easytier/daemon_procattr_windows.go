//go:build windows

package easytier

import (
	"os/exec"
	"syscall"
)

func setDaemonProcAttr(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// 无控制台宿主时避免部分 Windows 下控制台子进程异常退出；后台运行不抢焦点。
	cmd.SysProcAttr.HideWindow = true
}
