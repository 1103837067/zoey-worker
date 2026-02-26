//go:build !windows

package executor

import "github.com/go-vgo/robotgo"

func setCursorPos(x, y int) {
	robotgo.Move(x, y)
}

func dragSmooth(startX, startY, endX, endY int) {
	robotgo.Move(startX, startY)
	robotgo.DragSmooth(endX, endY)
}
