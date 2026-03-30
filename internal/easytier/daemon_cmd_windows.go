//go:build windows

package easytier

import (
	"os/exec"
	"syscall"
)

// CREATE_NEW_CONSOLE：子进程使用独立控制台窗口，easytier-core 可持续展示节点/路由等 TUI 输出。
// 与管道捕获 stdout 互斥，故默认不捕获；需要 Web「守护进程日志」时设 SKYLINK_EASYTIER_DAEMON_CAPTURE_OUTPUT=1。
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
