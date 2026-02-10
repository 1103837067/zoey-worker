//go:build !windows

package auto

import (
	"math"

	"github.com/go-vgo/robotgo"
)

// NormalizePointForInput 非 Windows 平台无需缩放
func NormalizePointForInput(x, y int) (int, int) {
	return x, y
}

// NormalizePointForScreen 非 Windows 平台无需缩放
func NormalizePointForScreen(x, y int) (int, int) {
	return x, y
}

// NormalizeRegionForInput 非 Windows 平台无需缩放
func NormalizeRegionForInput(x, y, width, height int) (int, int, int, int) {
	return x, y, width, height
}

// NormalizeRegionForScreen 非 Windows 平台无需缩放
func NormalizeRegionForScreen(x, y, width, height int) (int, int, int, int) {
	return x, y, width, height
}

// ResetCoordinateScaleCache 非 Windows 平台无操作
func ResetCoordinateScaleCache() {}

// GetDPIScale 非 Windows 平台返回 1.0
func GetDPIScale() float64 {
	return 1.0
}

// GetPhysicalScreenSize 获取物理屏幕尺寸
// 非 Windows 平台等同于 robotgo.GetScreenSize()（macOS Retina 由 robotgo 自行处理）
func GetPhysicalScreenSize() (width, height int) {
	return robotgo.GetScreenSize()
}

// ResetDPIScaleCache 非 Windows 平台无操作
func ResetDPIScaleCache() {}

// ScaleInt 缩放整数值
func ScaleInt(value int, factor float64) int {
	if factor <= 0 {
		return value
	}
	return int(math.Round(float64(value) * factor))
}
