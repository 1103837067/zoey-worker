//go:build !windows

package cmdutil

import "os/exec"

// HideWindow 非 Windows 平台无需隐藏窗口，空实现
func HideWindow(_ *exec.Cmd) {}
