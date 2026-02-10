// Package text 提供 OCR 文字识别和文字匹配功能
package text

import (
	"fmt"
	"image"
	"time"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	"github.com/zoeyai/zoeyworker/pkg/auto/input"
	"github.com/zoeyai/zoeyworker/pkg/auto/screen"
)

// ClickText 点击文字位置
func ClickText(text string, opts ...auto.Option) error {
	o := auto.ApplyOptions(opts...)

	pos, err := waitForTextInternal(text, o)
	if err != nil {
		return err
	}

	return input.ClickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
}

// WaitForText 等待文字出现
func WaitForText(text string, opts ...auto.Option) (*auto.Point, error) {
	o := auto.ApplyOptions(opts...)
	return waitForTextInternal(text, o)
}

// TextExists 检查文字是否存在
func TextExists(text string, opts ...auto.Option) bool {
	o := auto.ApplyOptions(opts...)
	o.Timeout = 0
	pos, _ := waitForTextInternal(text, o)
	return pos != nil
}

// waitForTextInternal 内部等待文字函数
func waitForTextInternal(text string, o *auto.Options) (*auto.Point, error) {
	recognizer, err := getTextRecognizer()
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	for {
		// 截图
		var img image.Image
		var captureErr error
		if o.Region != nil {
			img, captureErr = screen.CaptureRegion(o.Region.X, o.Region.Y, o.Region.Width, o.Region.Height)
		} else {
			img, captureErr = screen.CaptureScreen()
		}
		if captureErr != nil {
			return nil, captureErr
		}

		// OCR 查找文字
		result, err := recognizer.FindText(img, text)
		if err != nil {
			return nil, fmt.Errorf("OCR 识别失败: %w", err)
		}

		if result != nil {
			meta := screen.BuildCaptureMeta(o, img)
			adjusted := screen.AdjustPoint(auto.Point{X: result.X, Y: result.Y}, meta)
			return &adjusted, nil
		}

		if o.Timeout == 0 || time.Since(startTime) > o.Timeout {
			return nil, fmt.Errorf("等待文字超时: %s", text)
		}

		time.Sleep(auto.DefaultPollInterval)
	}
}
