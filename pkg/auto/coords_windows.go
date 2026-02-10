//go:build windows

package auto

import (
	"fmt"
	"math"
	"sync"
	"syscall"

	"github.com/go-vgo/robotgo"
)

// =====================================================================
// Windows 坐标系统设计说明
// =====================================================================
//
// 三个坐标空间：
//   1. 物理像素 (physical)  — 截图的实际像素，OpenCV 匹配结果在此空间
//   2. 逻辑坐标 (logical)   — Windows 逻辑像素 = 物理 / DPI_scale
//   3. robotgo 坐标 (input)  — robotgo.Move/Click 需要的坐标
//
// 在 DPI Aware 进程 (manifest: permonitorv2) 中：
//   - robotgo.GetScreenSize() 在不同版本/环境下可能返回物理或逻辑尺寸
//   - robotgo.CaptureImg()    始终返回物理像素
//   - robotgo.Move(x,y)       在不同版本下可能期望物理或逻辑坐标
//
// 因此我们不做假设，而是在初始化时通过对比截图尺寸与 GetScreenSize()
// 的返回值来自动探测 robotgo 使用的坐标空间。
//
// coordScale = 截图像素尺寸 / robotgo输入坐标空间尺寸
//   - 若 GetScreenSize 返回逻辑尺寸: coordScale = physical/logical = DPI_scale
//   - 若 GetScreenSize 返回物理尺寸: coordScale = physical/physical = 1.0
//     但此时 Move 也使用物理坐标，所以 1.0 是正确的。
//
// NormalizePointForInput:  截图坐标 → robotgo坐标 = x / coordScale
// NormalizePointForScreen: robotgo坐标 → 截图坐标 = x * coordScale
// =====================================================================

var (
	coordinateScaleMu sync.Mutex
	cachedScaleX      float64
	cachedScaleY      float64
	coordsDetected    bool
	debugLogOnce      sync.Once
)

// DPI 相关
var (
	user32DPI              = syscall.NewLazyDLL("user32.dll")
	gdi32DPI               = syscall.NewLazyDLL("gdi32.dll")
	procGetDpiForWindowDPI = user32DPI.NewProc("GetDpiForWindow")
	procGetDeviceCapsDPI   = gdi32DPI.NewProc("GetDeviceCaps")
	procGetDCDPI           = user32DPI.NewProc("GetDC")
	procReleaseDCDPI       = user32DPI.NewProc("ReleaseDC")
	procGetForegroundDPI   = user32DPI.NewProc("GetForegroundWindow")
	procGetDesktopDPI      = user32DPI.NewProc("GetDesktopWindow")

	cachedDPIScale float64 = 0
)

const (
	logpixelsX = 88
	logpixelsY = 90
)

// GetDPIScale 获取 Windows DPI 缩放比例
// 1.0 = 100%, 1.25 = 125%, 1.5 = 150%, 2.0 = 200%
func GetDPIScale() float64 {
	if cachedDPIScale > 0 {
		return cachedDPIScale
	}

	var dpi int

	// 方法1: 使用 GetDpiForWindow (Windows 10 1607+)
	if procGetDpiForWindowDPI.Find() == nil {
		hwnd, _, _ := procGetForegroundDPI.Call()
		if hwnd == 0 {
			hwnd, _, _ = procGetDesktopDPI.Call()
		}
		if hwnd != 0 {
			d, _, _ := procGetDpiForWindowDPI.Call(hwnd)
			if d > 0 {
				dpi = int(d)
			}
		}
	}

	// 方法2: 使用 GDI GetDeviceCaps
	if dpi == 0 && procGetDCDPI.Find() == nil && procGetDeviceCapsDPI.Find() == nil {
		dc, _, _ := procGetDCDPI.Call(0)
		if dc != 0 {
			d, _, _ := procGetDeviceCapsDPI.Call(dc, uintptr(logpixelsX))
			if d > 0 {
				dpi = int(d)
			}
			procReleaseDCDPI.Call(0, dc)
		}
	}

	if dpi <= 0 {
		dpi = 96
	}

	scale := float64(dpi) / 96.0
	if scale < 0.5 || scale > 4.0 {
		scale = 1.0
	}

	cachedDPIScale = scale
	return scale
}

// GetPhysicalScreenSize 获取物理屏幕尺寸（与截图分辨率一致）
// 利用 coordScale: 物理 = robotgo尺寸 * coordScale
func GetPhysicalScreenSize() (width, height int) {
	w, h := robotgo.GetScreenSize()
	scaleX, scaleY := getCoordinateScale()
	return ScaleInt(w, scaleX), ScaleInt(h, scaleY)
}

// ResetDPIScaleCache 重置 DPI 缩放缓存
func ResetDPIScaleCache() {
	cachedDPIScale = 0
	ResetCoordinateScaleCache()
}

// getCoordinateScale 获取 截图像素 → robotgo输入坐标 之间的缩放比。
//
// 通过对比 robotgo.CaptureImg() 的尺寸与 robotgo.GetScreenSize() 的返回值
// 来自动检测 robotgo 在当前环境下的坐标空间。
func getCoordinateScale() (float64, float64) {
	coordinateScaleMu.Lock()
	defer coordinateScaleMu.Unlock()

	if coordsDetected {
		return cachedScaleX, cachedScaleY
	}

	scaleX, scaleY := detectCoordinateScale()
	cachedScaleX = scaleX
	cachedScaleY = scaleY
	coordsDetected = true

	// 只在首次探测时输出调试日志
	debugLogOnce.Do(func() {
		dpi := GetDPIScale()
		rw, rh := robotgo.GetScreenSize()
		pw, ph := GetPhysicalScreenSize()
		fmt.Printf("[auto/coords] DPI=%.0f%% robotgo_screen=%dx%d physical=%dx%d coordScale=%.3f\n",
			dpi*100, rw, rh, pw, ph, scaleX)
	})

	return cachedScaleX, cachedScaleY
}

func detectCoordinateScale() (float64, float64) {
	reportedW, reportedH := robotgo.GetScreenSize()
	if reportedW <= 0 || reportedH <= 0 {
		return 1.0, 1.0
	}

	img, err := robotgo.CaptureImg()
	if err != nil || img == nil {
		// 截图失败，用 DPI scale 兜底
		s := GetDPIScale()
		return s, s
	}

	captureW := img.Bounds().Dx()
	captureH := img.Bounds().Dy()
	if captureW <= 0 || captureH <= 0 {
		return 1.0, 1.0
	}

	ratioX := float64(captureW) / float64(reportedW)
	ratioY := float64(captureH) / float64(reportedH)

	// 如果截图尺寸明显大于 GetScreenSize 报告的尺寸，
	// 说明 GetScreenSize 返回的是逻辑尺寸，robotgo.Move 期望逻辑坐标。
	// coordScale = 截图/逻辑 = ratio
	//
	// 如果截图尺寸 ≈ GetScreenSize，说明两者都在同一空间（通常是物理），
	// robotgo.Move 也使用物理坐标，coordScale = 1.0。
	scaleX := normalizeScale(ratioX)
	scaleY := normalizeScale(ratioY)
	return scaleX, scaleY
}

func normalizeScale(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 1.0
	}
	if v < 0.5 || v > 4.0 {
		return 1.0
	}
	if math.Abs(v-1.0) < 0.05 {
		return 1.0
	}
	return v
}

// ResetCoordinateScaleCache 重置坐标缩放缓存
func ResetCoordinateScaleCache() {
	coordinateScaleMu.Lock()
	defer coordinateScaleMu.Unlock()
	cachedScaleX = 0
	cachedScaleY = 0
	coordsDetected = false
}

// NormalizePointForInput 将截图物理坐标转换为 robotgo 输入坐标
// 截图坐标 / coordScale = robotgo坐标
func NormalizePointForInput(x, y int) (int, int) {
	scaleX, scaleY := getCoordinateScale()
	if scaleX <= 0 {
		scaleX = 1.0
	}
	if scaleY <= 0 {
		scaleY = 1.0
	}
	return ScaleInt(x, 1.0/scaleX), ScaleInt(y, 1.0/scaleY)
}

// NormalizePointForScreen 将 robotgo 坐标转换为截图物理坐标
// robotgo坐标 * coordScale = 截图坐标
func NormalizePointForScreen(x, y int) (int, int) {
	scaleX, scaleY := getCoordinateScale()
	return ScaleInt(x, scaleX), ScaleInt(y, scaleY)
}

// NormalizeRegionForInput 将截图物理区域转换为 robotgo 输入区域
func NormalizeRegionForInput(x, y, width, height int) (int, int, int, int) {
	scaleX, scaleY := getCoordinateScale()
	if scaleX <= 0 {
		scaleX = 1.0
	}
	if scaleY <= 0 {
		scaleY = 1.0
	}

	nx := ScaleInt(x, 1.0/scaleX)
	ny := ScaleInt(y, 1.0/scaleY)
	nw := ScaleInt(width, 1.0/scaleX)
	nh := ScaleInt(height, 1.0/scaleY)

	if width > 0 && nw < 1 {
		nw = 1
	}
	if height > 0 && nh < 1 {
		nh = 1
	}

	return nx, ny, nw, nh
}

// NormalizeRegionForScreen 将 robotgo 区域转换为截图物理区域
func NormalizeRegionForScreen(x, y, width, height int) (int, int, int, int) {
	scaleX, scaleY := getCoordinateScale()
	return ScaleInt(x, scaleX), ScaleInt(y, scaleY), ScaleInt(width, scaleX), ScaleInt(height, scaleY)
}

// ScaleInt 缩放整数值
func ScaleInt(value int, factor float64) int {
	if factor <= 0 {
		return value
	}
	return int(math.Round(float64(value) * factor))
}
