package executor

import (
	"math"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procSetCursorPos     = user32.NewProc("SetCursorPos")
	procSendInput        = user32.NewProc("SendInput")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
)

const (
	inputMouse     = 0
	mousefMove     = 0x0001
	mousefLeftDown = 0x0002
	mousefLeftUp   = 0x0004
	mousefAbsolute = 0x8000
	smCxscreen     = 0
	smCyscreen     = 1
)

// Win32 INPUT structure — must match C memory layout exactly.
// The union (ki/mi/hi) starts at offset 8 on 64-bit due to alignment of the
// pointer-sized fields in KEYBDINPUT. We use a fixed-size byte array and
// manually marshal the MOUSEINPUT into it.
const inputStructSize = 40 // sizeof(INPUT) on 64-bit Windows

func toAbsCoord(x, y int) (int32, int32) {
	sw, _, _ := procGetSystemMetrics.Call(smCxscreen)
	sh, _, _ := procGetSystemMetrics.Call(smCyscreen)
	ax := int32(float64(x)*65535.0/float64(sw-1) + 0.5)
	ay := int32(float64(y)*65535.0/float64(sh-1) + 0.5)
	return ax, ay
}

func sendMouseEvent(flags uint32, x, y int) {
	ax, ay := toAbsCoord(x, y)

	// Manually build INPUT struct as raw bytes to guarantee layout.
	// Layout (64-bit):
	//   [0:4]   type       = INPUT_MOUSE (0)
	//   [4:8]   padding
	//   [8:12]  dx         int32
	//   [12:16] dy         int32
	//   [16:20] mouseData  uint32
	//   [20:24] dwFlags    uint32
	//   [24:28] time       uint32
	//   [28:32] padding
	//   [32:40] dwExtraInfo uintptr
	var buf [inputStructSize]byte
	*(*uint32)(unsafe.Pointer(&buf[0])) = inputMouse
	*(*int32)(unsafe.Pointer(&buf[8])) = ax
	*(*int32)(unsafe.Pointer(&buf[12])) = ay
	*(*uint32)(unsafe.Pointer(&buf[20])) = flags | mousefAbsolute | mousefMove
	procSendInput.Call(1, uintptr(unsafe.Pointer(&buf[0])), inputStructSize)
}

func setCursorPos(x, y int) {
	procSetCursorPos.Call(uintptr(x), uintptr(y))
}

func dragSmooth(startX, startY, endX, endY int) {
	// 1. Move to start
	setCursorPos(startX, startY)
	sendMouseEvent(mousefMove, startX, startY)
	time.Sleep(120 * time.Millisecond)

	// 2. Press left button
	sendMouseEvent(mousefLeftDown, startX, startY)
	time.Sleep(120 * time.Millisecond)

	// 3. Interpolate movement — 100px per 1s
	dx := float64(endX - startX)
	dy := float64(endY - startY)
	dist := math.Sqrt(dx*dx + dy*dy)
	totalMs := dist / 100.0 * 1000.0
	if totalMs < 300 {
		totalMs = 300
	}
	steps := int(totalMs / 16)

	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		t = t * t * (3 - 2*t) // ease-in-out
		cx := startX + int(dx*t)
		cy := startY + int(dy*t)
		sendMouseEvent(mousefMove, cx, cy)
		time.Sleep(16 * time.Millisecond)
	}

	// 4. Ensure final position
	sendMouseEvent(mousefMove, endX, endY)
	time.Sleep(80 * time.Millisecond)

	// 5. Release
	sendMouseEvent(mousefLeftUp, endX, endY)
}
