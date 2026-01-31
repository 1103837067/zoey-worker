//go:build !darwin && !windows

package auto

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

// activateWindowByTitlePlatform 非 macOS 系统（Windows/Linux）
// 增强版：先查找匹配窗口，再确保激活并置顶
//
// 统一逻辑：获取所有窗口 → 按进程名/窗口标题匹配 → 恢复最小化 → 激活置顶
func activateWindowByTitlePlatform(appName, windowTitle string) error {
	// 1. 获取所有窗口
	allWindows, err := getWindowsRobotgo()
	if err != nil {
		return fmt.Errorf("获取窗口列表失败: %w", err)
	}

	// 2. 查找匹配的窗口
	var targetWindow *WindowInfo
	appNameLower := strings.ToLower(appName)
	windowTitleLower := strings.ToLower(windowTitle)

	// 优先级 1: 进程名 + 窗口标题都匹配
	for i := range allWindows {
		w := &allWindows[i]
		ownerMatch := strings.Contains(strings.ToLower(w.OwnerName), appNameLower)
		titleMatch := strings.Contains(strings.ToLower(w.Title), windowTitleLower)
		if ownerMatch && titleMatch {
			targetWindow = w
			break
		}
	}

	// 优先级 2: 只匹配进程名（OwnerName）
	if targetWindow == nil {
		for i := range allWindows {
			w := &allWindows[i]
			if strings.Contains(strings.ToLower(w.OwnerName), appNameLower) {
				targetWindow = w
				break
			}
		}
	}

	// 优先级 3: 只匹配窗口标题
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

	// 3. 先尝试恢复最小化窗口
	// robotgo.MaxWindow 在 Windows 上会恢复最小化窗口（相当于 SW_RESTORE）
	// 如果窗口已经正常显示，这个操作不会有副作用
	robotgo.MaxWindow(targetWindow.PID)

	// 4. 激活窗口并置顶
	// robotgo.ActivePid 在 Windows 上调用 SetForegroundWindow
	if err := robotgo.ActivePid(targetWindow.PID); err != nil {
		return fmt.Errorf("激活窗口失败: %w", err)
	}

	// 5. 额外尝试：如果第一次激活没成功，等一小会再试一次
	// 有些情况下 Windows 会阻止后台程序抢夺焦点
	time.Sleep(100 * time.Millisecond)

	// 检查是否激活成功（通过获取当前活动窗口标题）
	// 如果没成功，再尝试一次
	currentTitle := robotgo.GetTitle()
	if !strings.Contains(strings.ToLower(currentTitle), windowTitleLower) &&
		!strings.Contains(strings.ToLower(currentTitle), appNameLower) {
		// 再次尝试激活
		robotgo.ActivePid(targetWindow.PID)
	}

	return nil
}
