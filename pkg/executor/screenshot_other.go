//go:build !windows

package executor

import (
	"fmt"
	"image"

	"github.com/go-vgo/robotgo"
)

func captureScreenWithCursor() (image.Image, error) {
	w, h := robotgo.GetScreenSize()
	img, err := robotgo.CaptureImg(0, 0, w, h)
	if err != nil {
		return nil, fmt.Errorf("截屏失败: %w", err)
	}
	return img, nil
}
