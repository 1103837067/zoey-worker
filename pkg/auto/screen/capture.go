// Package screen 提供屏幕截图和编码功能
//
// 坐标原则：截图多大，屏幕就是多大。
// CaptureImg 返回的 Bounds 就是 robotgo.Move 的坐标范围，不需要额外缩放。
package screen

import (
	"fmt"
	"image"
	"sync"

	"github.com/go-vgo/robotgo"
)

var (
	lastCaptureW, lastCaptureH int
	captureSizeMu              sync.RWMutex
)

// CaptureScreen 截取主显示器全屏，缓存 Bounds 尺寸
func CaptureScreen() (image.Image, error) {
	w, h := robotgo.GetScreenSize()
	img, err := robotgo.CaptureImg(0, 0, w, h)
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

// CaptureRegion 截取屏幕指定区域
func CaptureRegion(x, y, width, height int) (image.Image, error) {
	img, err := robotgo.CaptureImg(x, y, width, height)
	if err != nil {
		return nil, fmt.Errorf("截取区域失败: %w", err)
	}
	return img, nil
}

// GetScreenSize 返回截图的实际像素尺寸（= robotgo.Move 的坐标范围）
func GetScreenSize() (width, height int) {
	captureSizeMu.RLock()
	w, h := lastCaptureW, lastCaptureH
	captureSizeMu.RUnlock()
	if w > 0 && h > 0 {
		return w, h
	}
	// 首次调用还没截过图，先截一次拿真实尺寸
	if _, err := CaptureScreen(); err == nil {
		captureSizeMu.RLock()
		w, h = lastCaptureW, lastCaptureH
		captureSizeMu.RUnlock()
		return w, h
	}
	return robotgo.GetScreenSize()
}

// GetDisplayCount 获取显示器数量
func GetDisplayCount() int {
	return robotgo.DisplaysNum()
}
