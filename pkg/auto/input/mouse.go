// Package input 提供鼠标、键盘和剪贴板操作
package input

import (
	"github.com/go-vgo/robotgo"

	"github.com/zoeyai/zoeyworker/pkg/auto"
)

// MoveTo 移动鼠标到指定位置
func MoveTo(x, y int) {
	inputX, inputY := auto.NormalizePointForInput(x, y)
	robotgo.Move(inputX, inputY)
}

// MoveSmooth 平滑移动鼠标
func MoveSmooth(x, y int) {
	inputX, inputY := auto.NormalizePointForInput(x, y)
	robotgo.MoveSmooth(inputX, inputY)
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
	inputX, inputY := auto.NormalizePointForInput(x, y)
	robotgo.DragSmooth(inputX, inputY)
}

// GetMousePosition 获取鼠标位置
func GetMousePosition() (x, y int) {
	inputX, inputY := robotgo.Location()
	return auto.NormalizePointForScreen(inputX, inputY)
}
