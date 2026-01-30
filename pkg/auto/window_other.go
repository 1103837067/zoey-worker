//go:build !darwin

package auto

import "github.com/go-vgo/robotgo"

// getWindowsDarwin 非 macOS 系统使用 robotgo
func getWindowsDarwin(filter ...string) ([]WindowInfo, error) {
	return getWindowsRobotgo(filter...)
}

// activateWindowPlatform 非 macOS 系统使用 robotgo
func activateWindowPlatform(name string) error {
	robotgo.ActiveName(name)
	return nil
}

// activateWindowByPIDPlatform 非 macOS 系统使用 robotgo
func activateWindowByPIDPlatform(pid int) error {
	robotgo.ActivePid(pid)
	return nil
}

// activateWindowByTitlePlatform 非 macOS 系统：先激活应用，再通过标题查找
func activateWindowByTitlePlatform(appName, windowTitle string) error {
	// Windows 上 robotgo.ActiveName 支持窗口标题
	robotgo.ActiveName(windowTitle)
	return nil
}
