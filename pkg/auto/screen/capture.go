// Package screen 提供屏幕截图和编码功能
package screen

import (
	"fmt"
	"image"

	"github.com/go-vgo/robotgo"
	"github.com/zoeyai/zoeyworker/pkg/auto"
)

// CaptureScreen 截取主显示器全屏
// 使用 robotgo.GetScreenSize() 获取主显示器尺寸，指定区域截图避免多显示器黑边
func CaptureScreen() (image.Image, error) {
	w, h := robotgo.GetScreenSize()
	if w > 0 && h > 0 {
		img, err := robotgo.CaptureImg(0, 0, w, h)
		if err != nil {
			return nil, fmt.Errorf("截屏失败: %w", err)
		}
		return img, nil
	}

	img, err := robotgo.CaptureImg()
	if err != nil {
		return nil, fmt.Errorf("截屏失败: %w", err)
	}
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

// GetScreenSize 获取屏幕尺寸（物理像素，与截图分辨率一致）
func GetScreenSize() (width, height int) {
	return auto.GetPhysicalScreenSize()
}

// GetDisplayCount 获取显示器数量
func GetDisplayCount() int {
	return robotgo.DisplaysNum()
}
