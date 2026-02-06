package auto

import (
	"fmt"
	"image"
	"time"

	"gocv.io/x/gocv"

	"github.com/zoeyai/zoeyworker/pkg/vision/cv"
)

// ==================== 图像匹配操作 ====================

// ClickImage 点击图像位置
func ClickImage(templatePath string, opts ...Option) error {
	o := applyOptions(opts...)

	pos, err := waitForImageInternal(templatePath, o)
	if err != nil {
		return err
	}

	return clickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
}

// ClickImageWithGrid 点击图像匹配区域内的网格位置
// gridStr: 网格位置字符串 (如 "2.2.1.1" 表示 2x2 网格的第1行第1列)
func ClickImageWithGrid(templatePath string, gridStr string, opts ...Option) error {
	o := applyOptions(opts...)

	// 等待图像出现并获取完整匹配结果
	result, err := waitForImageResultInternal(templatePath, o)
	if err != nil {
		return err
	}

	// 计算匹配区域
	rect := result.Rectangle
	minX := minInt(rect.TopLeft.X, rect.TopRight.X, rect.BottomLeft.X, rect.BottomRight.X)
	maxX := maxInt(rect.TopLeft.X, rect.TopRight.X, rect.BottomLeft.X, rect.BottomRight.X)
	minY := minInt(rect.TopLeft.Y, rect.TopRight.Y, rect.BottomLeft.Y, rect.BottomRight.Y)
	maxY := maxInt(rect.TopLeft.Y, rect.TopRight.Y, rect.BottomLeft.Y, rect.BottomRight.Y)
	matchRegion := Region{
		X:      minX,
		Y:      minY,
		Width:  maxInt(1, maxX-minX),
		Height: maxInt(1, maxY-minY),
	}

	// 计算网格位置的点击坐标
	clickPos, err := CalculateGridCenterFromString(matchRegion, gridStr)
	if err != nil {
		return fmt.Errorf("计算网格位置失败: %w", err)
	}

	return clickAt(clickPos.X+o.ClickOffset.X, clickPos.Y+o.ClickOffset.Y, o)
}

// ClickImageData 点击图像位置（使用图像数据）
func ClickImageData(template image.Image, opts ...Option) error {
	o := applyOptions(opts...)

	pos, err := waitForImageDataInternal(template, o)
	if err != nil {
		return err
	}

	return clickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
}

// WaitForImage 等待图像出现
func WaitForImage(templatePath string, opts ...Option) (*Point, error) {
	o := applyOptions(opts...)
	return waitForImageInternal(templatePath, o)
}

// WaitForImageData 等待图像出现（使用图像数据）
func WaitForImageData(template image.Image, opts ...Option) (*Point, error) {
	o := applyOptions(opts...)
	return waitForImageDataInternal(template, o)
}

// ImageExists 检查图像是否存在
func ImageExists(templatePath string, opts ...Option) bool {
	o := applyOptions(opts...)
	o.Timeout = 0 // 不等待
	pos, _ := waitForImageInternal(templatePath, o)
	return pos != nil
}

// ImageExistsData 检查图像是否存在（使用图像数据）
func ImageExistsData(template image.Image, opts ...Option) bool {
	o := applyOptions(opts...)
	o.Timeout = 0
	pos, _ := waitForImageDataInternal(template, o)
	return pos != nil
}

// ==================== 内部图像匹配函数 ====================

// waitForImageInternal 内部等待图像函数
func waitForImageInternal(templatePath string, o *Options) (*Point, error) {
	result, err := waitForImageResultInternal(templatePath, o)
	if err != nil {
		return nil, err
	}

	pos := result.Result
	return &Point{X: pos.X, Y: pos.Y}, nil
}

// waitForImageResultInternal 内部等待图像函数（返回完整匹配结果）
func waitForImageResultInternal(templatePath string, o *Options) (*cv.MatchResult, error) {
	tmpl := cv.NewTemplate(templatePath,
		cv.WithTemplateThreshold(o.Threshold),
	)

	startTime := time.Now()
	for {
		screen, meta, err := captureForMatch(o)
		if err != nil {
			return nil, err
		}

		result, err := tmpl.MatchResultIn(screen)
		screen.Close()

		if err != nil {
			return nil, fmt.Errorf("匹配失败: %w", err)
		}
		if result != nil {
			return adjustMatchResult(result, meta), nil
		}

		if o.Timeout == 0 || time.Since(startTime) > o.Timeout {
			return nil, fmt.Errorf("等待图像超时: %s", templatePath)
		}

		time.Sleep(defaultPollInterval)
	}
}

// waitForImageDataInternal 内部等待图像函数（使用图像数据）
func waitForImageDataInternal(template image.Image, o *Options) (*Point, error) {
	// 转换 image.Image 为 gocv.Mat
	templateMat, err := gocv.ImageToMatRGB(template)
	if err != nil {
		return nil, fmt.Errorf("转换模板图像失败: %w", err)
	}
	defer templateMat.Close()

	startTime := time.Now()
	for {
		screen, meta, err := captureForMatch(o)
		if err != nil {
			return nil, err
		}

		matcher := cv.NewSIFTMatching(templateMat, screen, o.Threshold)
		result, err := matcher.FindBestResult()
		matcher.Close()
		screen.Close()

		if err != nil {
			return nil, fmt.Errorf("匹配失败: %w", err)
		}
		if result != nil {
			adjusted := adjustMatchResult(result, meta)
			return &Point{X: adjusted.Result.X, Y: adjusted.Result.Y}, nil
		}

		if o.Timeout == 0 || time.Since(startTime) > o.Timeout {
			return nil, fmt.Errorf("等待图像超时")
		}

		time.Sleep(defaultPollInterval)
	}
}
