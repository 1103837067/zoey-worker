package auto

import (
	"fmt"
	"image"
	"runtime"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"
)

// WindowInfo 窗口信息
// 注意：Go 使用 PID 作为窗口标识符，与 Python 的 hwnd 不同
// robotgo 跨平台实现：macOS 用 PID，Windows 内部转换为 hwnd
type WindowInfo struct {
	PID    int    `json:"pid"`
	Title  string `json:"title"`
	Bounds Region `json:"bounds"`
}

// GetWindows 获取窗口列表
// filter: 可选的过滤条件，按窗口标题或进程名称筛选
func GetWindows(filter ...string) ([]WindowInfo, error) {
	// macOS 使用原生 API 避免 robotgo 的权限弹窗问题
	if runtime.GOOS == "darwin" {
		return getWindowsDarwin(filter...)
	}
	return getWindowsRobotgo(filter...)
}

// getWindowsRobotgo 使用 robotgo 获取窗口（Windows/Linux）
func getWindowsRobotgo(filter ...string) ([]WindowInfo, error) {
	// 获取所有进程
	pids, err := robotgo.Pids()
	if err != nil {
		return nil, fmt.Errorf("获取进程列表失败: %w", err)
	}

	filterStr := ""
	if len(filter) > 0 {
		filterStr = strings.ToLower(filter[0])
	}

	var windows []WindowInfo

	for _, pid := range pids {
		title := robotgo.GetTitle(pid)
		if title == "" {
			continue
		}

		// 如果有过滤条件，检查标题或进程名是否匹配
		if filterStr != "" {
			name, _ := robotgo.FindName(pid)
			titleLower := strings.ToLower(title)
			nameLower := strings.ToLower(name)

			if !strings.Contains(titleLower, filterStr) && !strings.Contains(nameLower, filterStr) {
				continue
			}
		}

		// 获取窗口边界
		x, y, w, h := robotgo.GetBounds(pid)

		windows = append(windows, WindowInfo{
			PID:   pid,
			Title: title,
			Bounds: Region{
				X:      x,
				Y:      y,
				Width:  w,
				Height: h,
			},
		})
	}

	return windows, nil
}

// GetWindowByTitle 按标题查找窗口 (部分匹配)
func GetWindowByTitle(title string) (*WindowInfo, error) {
	windows, err := GetWindows(title)
	if err != nil {
		return nil, err
	}

	if len(windows) == 0 {
		return nil, fmt.Errorf("未找到标题包含 %q 的窗口", title)
	}

	return &windows[0], nil
}

// GetWindowByPID 按 PID 获取窗口信息
func GetWindowByPID(pid int) (*WindowInfo, error) {
	title := robotgo.GetTitle(pid)
	if title == "" {
		return nil, fmt.Errorf("未找到 PID=%d 的窗口", pid)
	}

	x, y, w, h := robotgo.GetBounds(pid)

	return &WindowInfo{
		PID:   pid,
		Title: title,
		Bounds: Region{
			X:      x,
			Y:      y,
			Width:  w,
			Height: h,
		},
	}, nil
}

// GetWindowClient 获取窗口客户区边界
func GetWindowClient(pid int) (*Region, error) {
	x, y, w, h := robotgo.GetClient(pid)
	if w == 0 && h == 0 {
		return nil, fmt.Errorf("无法获取窗口客户区: PID=%d", pid)
	}

	return &Region{
		X:      x,
		Y:      y,
		Width:  w,
		Height: h,
	}, nil
}

// MinimizeWindow 最小化窗口
func MinimizeWindow(pid int) {
	robotgo.MinWindow(pid)
}

// MaximizeWindow 最大化窗口
func MaximizeWindow(pid int) {
	robotgo.MaxWindow(pid)
}

// CloseWindowByPID 关闭窗口
func CloseWindowByPID(pid int) {
	robotgo.CloseWindow(pid)
}

// BringWindowToFront 将窗口置于前台
func BringWindowToFront(pid int) error {
	return robotgo.ActivePid(pid)
}

// CaptureWindow 截取窗口截图
func CaptureWindow(pid int) (image.Image, error) {
	x, y, w, h := robotgo.GetBounds(pid)
	if w == 0 || h == 0 {
		return nil, fmt.Errorf("无法获取窗口边界: PID=%d", pid)
	}

	return CaptureRegion(x, y, w, h)
}

// ClickInWindow 在窗口内点击相对位置
// relX, relY: 相对于窗口左上角的偏移
func ClickInWindow(pid int, relX, relY int, opts ...Option) error {
	x, y, _, _ := robotgo.GetBounds(pid)
	if x == 0 && y == 0 {
		return fmt.Errorf("无法获取窗口位置: PID=%d", pid)
	}

	// 先激活窗口
	if err := robotgo.ActivePid(pid); err != nil {
		return fmt.Errorf("激活窗口失败: %w", err)
	}

	o := applyOptions(opts...)
	return clickAt(x+relX+o.ClickOffset.X, y+relY+o.ClickOffset.Y, o)
}

// ClickGridInWindow 在窗口内按网格位置点击
func ClickGridInWindow(pid int, gridStr string, opts ...Option) error {
	x, y, w, h := robotgo.GetBounds(pid)
	if w == 0 || h == 0 {
		return fmt.Errorf("无法获取窗口边界: PID=%d", pid)
	}

	// 先激活窗口
	if err := robotgo.ActivePid(pid); err != nil {
		return fmt.Errorf("激活窗口失败: %w", err)
	}

	rect := Region{X: x, Y: y, Width: w, Height: h}
	return ClickGrid(rect, gridStr, opts...)
}

// WaitForWindow 等待窗口出现
func WaitForWindow(title string, opts ...Option) (*WindowInfo, error) {
	o := applyOptions(opts...)

	startTime := time.Now()
	for {
		window, err := GetWindowByTitle(title)
		if err == nil && window != nil {
			return window, nil
		}

		if o.Timeout == 0 || time.Since(startTime) > o.Timeout {
			return nil, fmt.Errorf("等待窗口超时: %s", title)
		}

		Sleep(o.Interval)
	}
}
