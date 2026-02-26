// Package screen 提供屏幕截图和编码功能
package screen

import (
	"fmt"
	"image"
	"sync"

	"github.com/go-vgo/robotgo"
	"github.com/zoeyai/zoeyworker/pkg/auto"
)

// 最近一次截图的实际像素尺寸
// robotgo.Move() 使用的坐标系与 CaptureImg() 返回的像素尺寸一致，
// 因此归一化坐标应基于截图尺寸映射，无需考虑 DPI 缩放。
var (
	lastCaptureW, lastCaptureH int
	captureSizeMu              sync.RWMutex
)

// CaptureScreen 截取全屏（物理像素）
func CaptureScreen() (image.Image, error) {
	img, err := robotgo.CaptureImg()
	if err != nil {
		return nil, fmt.Errorf("截屏失败: %w", err)
	}

	bounds := img.Bounds()
	captureSizeMu.Lock()
	lastCaptureW = bounds.Dx()
	lastCaptureH = bounds.Dy()
	captureSizeMu.Unlock()

	return img, nil
}

// CaptureRegion 截取屏幕区域
func CaptureRegion(x, y, width, height int) (image.Image, error) {
	inputX, inputY, inputW, inputH := auto.NormalizeRegionForInput(x, y, width, height)
	img, err := robotgo.CaptureImg(inputX, inputY, inputW, inputH)
	if err != nil {
		return nil, fmt.Errorf("截取区域失败: %w", err)
	}
	return img, nil
}

// GetScreenSize 返回截图的实际像素尺寸（与 robotgo.Move 坐标系一致）
func GetScreenSize() (width, height int) {
	captureSizeMu.RLock()
	w, h := lastCaptureW, lastCaptureH
	captureSizeMu.RUnlock()
	if w > 0 && h > 0 {
		return w, h
	}
	return auto.GetPhysicalScreenSize()
}

// GetDisplayCount 获取显示器数量
func GetDisplayCount() int {
	return robotgo.DisplaysNum()
}
