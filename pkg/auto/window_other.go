//go:build !darwin

package auto

// getWindowsDarwin 非 macOS 系统使用 robotgo
func getWindowsDarwin(filter ...string) ([]WindowInfo, error) {
	return getWindowsRobotgo(filter...)
}
