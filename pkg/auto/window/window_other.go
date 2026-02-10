//go:build !darwin && !windows

package window

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"
)

// getWindowsPlatform Linux 等其他平台实现
func getWindowsPlatform(filter ...string) ([]WindowInfo, error) {
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

// activateWindowByTitlePlatform 非 macOS 系统（Linux）
func activateWindowByTitlePlatform(appName, windowTitle string) error {
	allWindows, err := getWindowsRobotgo()
	if err != nil {
		return fmt.Errorf("获取窗口列表失败: %w", err)
	}

	var targetWindow *WindowInfo
	appNameLower := strings.ToLower(appName)
	windowTitleLower := strings.ToLower(windowTitle)

	for i := range allWindows {
		w := &allWindows[i]
		ownerMatch := strings.Contains(strings.ToLower(w.OwnerName), appNameLower)
		titleMatch := strings.Contains(strings.ToLower(w.Title), windowTitleLower)
		if ownerMatch && titleMatch {
			targetWindow = w
			break
		}
	}

	if targetWindow == nil {
		for i := range allWindows {
			w := &allWindows[i]
			if strings.Contains(strings.ToLower(w.OwnerName), appNameLower) {
				targetWindow = w
				break
			}
		}
	}

	if targetWindow == nil {
		for i := range allWindows {
			w := &allWindows[i]
			if strings.Contains(strings.ToLower(w.Title), windowTitleLower) {
				targetWindow = w
				break
			}
		}
	}

	if targetWindow == nil {
		return fmt.Errorf("未找到匹配的窗口: appName=%s, windowTitle=%s", appName, windowTitle)
	}

	robotgo.MaxWindow(targetWindow.PID)

	if err := robotgo.ActivePid(targetWindow.PID); err != nil {
		return fmt.Errorf("激活窗口失败: %w", err)
	}

	time.Sleep(100 * time.Millisecond)

	currentTitle := robotgo.GetTitle()
	if !strings.Contains(strings.ToLower(currentTitle), windowTitleLower) &&
		!strings.Contains(strings.ToLower(currentTitle), appNameLower) {
		robotgo.ActivePid(targetWindow.PID)
	}

	return nil
}
