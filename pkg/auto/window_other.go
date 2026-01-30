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
