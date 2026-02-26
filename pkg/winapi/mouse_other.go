//go:build !windows

package winapi

import "github.com/go-vgo/robotgo"

func SetCursorPos(x, y int) {
	robotgo.Move(x, y)
}

func DragSmooth(startX, startY, endX, endY int) {
	robotgo.Move(startX, startY)
	robotgo.DragSmooth(endX, endY)
}
