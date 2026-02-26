package executor

import "github.com/zoeyai/zoeyworker/pkg/winapi"

func setCursorPos(x, y int) {
	winapi.SetCursorPos(x, y)
}

func dragSmooth(startX, startY, endX, endY int) {
	winapi.DragSmooth(startX, startY, endX, endY)
}
