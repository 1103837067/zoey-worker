// Package window 提供窗口管理功能（获取、激活、截图、操作窗口）
package window

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	"github.com/zoeyai/zoeyworker/pkg/auto/grid"
	"github.com/zoeyai/zoeyworker/pkg/auto/input"
	"github.com/zoeyai/zoeyworker/pkg/auto/screen"
)

// WindowInfo 窗口信息
type WindowInfo struct {
	PID       int         `json:"pid"`
	Title     string      `json:"title"`
	OwnerName string      `json:"owner_name"`
	Bounds    auto.Region `json:"bounds"`
}

// GetWindows 获取窗口列表
func GetWindows(filter ...string) ([]WindowInfo, error) {
	return getWindowsPlatform(filter...)
}

// getWindowsRobotgo 使用 robotgo 获取窗口（Windows/Linux）
func getWindowsRobotgo(filter ...string) ([]WindowInfo, error) {
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

		if filterStr != "" {
			name, _ := robotgo.FindName(pid)
			titleLower := strings.ToLower(title)
			nameLower := strings.ToLower(name)

			if !strings.Contains(titleLower, filterStr) && !strings.Contains(nameLower, filterStr) {
				continue
			}
		}

		x, y, w, h := robotgo.GetBounds(pid)
		x, y, w, h = auto.NormalizeRegionForScreen(x, y, w, h)

		name, _ := robotgo.FindName(pid)
		windows = append(windows, WindowInfo{
			PID:       pid,
			Title:     title,
			OwnerName: name,
			Bounds: auto.Region{
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
	x, y, w, h = auto.NormalizeRegionForScreen(x, y, w, h)

	return &WindowInfo{
		PID:   pid,
		Title: title,
		Bounds: auto.Region{
			X:      x,
			Y:      y,
			Width:  w,
			Height: h,
		},
	}, nil
}

// GetWindowClient 获取窗口客户区边界
func GetWindowClient(pid int) (*auto.Region, error) {
	x, y, w, h := robotgo.GetClient(pid)
	x, y, w, h = auto.NormalizeRegionForScreen(x, y, w, h)
	if w == 0 && h == 0 {
		return nil, fmt.Errorf("无法获取窗口客户区: PID=%d", pid)
	}

	return &auto.Region{
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
	x, y, w, h = auto.NormalizeRegionForScreen(x, y, w, h)
	if w == 0 || h == 0 {
		return nil, fmt.Errorf("无法获取窗口边界: PID=%d", pid)
	}

	return screen.CaptureRegion(x, y, w, h)
}

// ClickInWindow 在窗口内点击相对位置
func ClickInWindow(pid int, relX, relY int, opts ...auto.Option) error {
	x, y, _, _ := robotgo.GetBounds(pid)
	x, y, _, _ = auto.NormalizeRegionForScreen(x, y, 0, 0)
	if x == 0 && y == 0 {
		return fmt.Errorf("无法获取窗口位置: PID=%d", pid)
	}

	if err := robotgo.ActivePid(pid); err != nil {
		return fmt.Errorf("激活窗口失败: %w", err)
	}

	o := auto.ApplyOptions(opts...)
	return input.ClickAt(x+relX+o.ClickOffset.X, y+relY+o.ClickOffset.Y, o)
}

// ClickGridInWindow 在窗口内按网格位置点击
func ClickGridInWindow(pid int, gridStr string, opts ...auto.Option) error {
	x, y, w, h := robotgo.GetBounds(pid)
	x, y, w, h = auto.NormalizeRegionForScreen(x, y, w, h)
	if w == 0 || h == 0 {
		return fmt.Errorf("无法获取窗口边界: PID=%d", pid)
	}

	if err := robotgo.ActivePid(pid); err != nil {
		return fmt.Errorf("激活窗口失败: %w", err)
	}

	rect := auto.Region{X: x, Y: y, Width: w, Height: h}
	return grid.ClickGrid(rect, gridStr, opts...)
}

// WaitForWindow 等待窗口出现
func WaitForWindow(title string, opts ...auto.Option) (*WindowInfo, error) {
	o := auto.ApplyOptions(opts...)

	startTime := time.Now()
	for {
		w, err := GetWindowByTitle(title)
		if err == nil && w != nil {
			return w, nil
		}

		if o.Timeout == 0 || time.Since(startTime) > o.Timeout {
			return nil, fmt.Errorf("等待窗口超时: %s", title)
		}

		auto.Sleep(auto.DefaultPollInterval)
	}
}

// ==================== 窗口激活（委托给平台实现） ====================

// ActivateWindow 激活窗口（支持应用名称或窗口标题）
func ActivateWindow(name string) error {
	return activateWindowPlatform(name)
}

// ActivateWindowByPID 通过 PID 激活窗口
func ActivateWindowByPID(pid int) error {
	return activateWindowByPIDPlatform(pid)
}

// ActivateWindowByTitle 通过应用名和窗口标题激活特定窗口
func ActivateWindowByTitle(appName, windowTitle string) error {
	return activateWindowByTitlePlatform(appName, windowTitle)
}

// GetActiveWindowTitle 获取当前活动窗口标题
func GetActiveWindowTitle() string {
	return robotgo.GetTitle()
}

// FindWindowPIDs 查找窗口 PID
func FindWindowPIDs(name string) ([]int, error) {
	pids, err := robotgo.FindIds(name)
	if err != nil {
		return nil, fmt.Errorf("查找窗口失败: %w", err)
	}

	result := make([]int, len(pids))
	for i, pid := range pids {
		result[i] = int(pid)
	}
	return result, nil
}
