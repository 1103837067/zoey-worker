package cmdutil

import (
	"os/exec"
	"syscall"
)

// HideWindow 在 Windows 上隐藏 exec.Command 的 cmd 黑色窗口
func HideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}
