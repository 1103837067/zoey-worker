package auto

import (
	"fmt"

	"github.com/go-vgo/robotgo"
)

// ==================== 窗口操作 ====================

// ActivateWindow 激活窗口（支持应用名称或窗口标题）
func ActivateWindow(name string) error {
	return activateWindowPlatform(name)
}

// ActivateWindowByPID 通过 PID 激活窗口
func ActivateWindowByPID(pid int) error {
	return activateWindowByPIDPlatform(pid)
}

// ActivateWindowByTitle 通过应用名和窗口标题激活特定窗口
// appName: 应用名称，如 "Microsoft Edge"
// windowTitle: 窗口标题的部分内容，如 "GitHub"
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

	// 转换 int32 -> int
	result := make([]int, len(pids))
	for i, pid := range pids {
		result[i] = int(pid)
	}
	return result, nil
}

// ==================== 鼠标操作 ====================

// MoveTo 移动鼠标到指定位置
func MoveTo(x, y int) {
	robotgo.Move(x, y)
}

// MoveSmooth 平滑移动鼠标
func MoveSmooth(x, y int) {
	robotgo.MoveSmooth(x, y)
}

// Click 点击
func Click(button ...string) {
	btn := "left"
	if len(button) > 0 {
		btn = button[0]
	}
	robotgo.Click(btn, false)
}

// DoubleClick 双击
func DoubleClick(button ...string) {
	btn := "left"
	if len(button) > 0 {
		btn = button[0]
	}
	robotgo.Click(btn, true)
}

// RightClick 右键点击
func RightClick() {
	robotgo.Click("right", false)
}

// Scroll 滚动
func Scroll(x, y int) {
	robotgo.Scroll(x, y)
}

// ScrollUp 向上滚动
func ScrollUp(lines int) {
	robotgo.ScrollDir(lines, "up")
}

// ScrollDown 向下滚动
func ScrollDown(lines int) {
	robotgo.ScrollDir(lines, "down")
}

// Drag 拖拽
func Drag(x, y int) {
	robotgo.DragSmooth(x, y)
}

// GetMousePosition 获取鼠标位置
func GetMousePosition() (x, y int) {
	return robotgo.Location()
}

// ==================== 键盘操作 ====================

// TypeText 输入文字
func TypeText(text string) {
	robotgo.TypeStr(text)
}

// KeyTap 按键
func KeyTap(key string, modifiers ...string) {
	if len(modifiers) > 0 {
		robotgo.KeyTap(key, modifiers)
	} else {
		robotgo.KeyTap(key)
	}
}

// KeyDown 按下键
func KeyDown(key string) {
	robotgo.KeyToggle(key, "down")
}

// KeyUp 释放键
func KeyUp(key string) {
	robotgo.KeyToggle(key, "up")
}

// HotKey 组合键
func HotKey(keys ...string) {
	if len(keys) == 0 {
		return
	}
	if len(keys) == 1 {
		robotgo.KeyTap(keys[0])
		return
	}
	robotgo.KeyTap(keys[len(keys)-1], keys[:len(keys)-1])
}

// ==================== 剪贴板操作 ====================

// CopyToClipboard 复制到剪贴板
func CopyToClipboard(text string) error {
	return robotgo.WriteAll(text)
}

// ReadClipboard 读取剪贴板
func ReadClipboard() (string, error) {
	return robotgo.ReadAll()
}
