//go:build windows

package easytier

import (
	"os/exec"
	"syscall"
)

// CREATE_NEW_CONSOLE：子进程使用独立控制台窗口，便于直接观察 easytier-core 输出。
// 当前默认后台隐藏窗口并捕获 stdout/stderr；仅在 captureOutput=false 时启用独立控制台。
const createNewConsoleFlag = 0x00000010

func configureDaemonCmd(cmd *exec.Cmd, captureOutput bool) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	if captureOutput {
		cmd.SysProcAttr.HideWindow = true
		cmd.SysProcAttr.CreationFlags = 0
	} else {
		cmd.SysProcAttr.HideWindow = false
		cmd.SysProcAttr.CreationFlags = createNewConsoleFlag
	}
}
