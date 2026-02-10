// Package image 提供图像匹配功能（模板匹配、SIFT 匹配）
package image

import (
	"fmt"
	stdimage "image"
	"time"

	"gocv.io/x/gocv"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	"github.com/zoeyai/zoeyworker/pkg/auto/grid"
	"github.com/zoeyai/zoeyworker/pkg/auto/input"
	"github.com/zoeyai/zoeyworker/pkg/auto/screen"
	"github.com/zoeyai/zoeyworker/pkg/vision/cv"
)

// ClickImage 点击图像位置
func ClickImage(templatePath string, opts ...auto.Option) error {
	o := auto.ApplyOptions(opts...)

	pos, err := waitForImageInternal(templatePath, o)
	if err != nil {
		return err
	}

	return input.ClickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
}

// ClickImageWithGrid 点击图像匹配区域内的网格位置
// gridStr: 网格位置字符串 (如 "2.2.1.1" 表示 2x2 网格的第1行第1列)
func ClickImageWithGrid(templatePath string, gridStr string, opts ...auto.Option) error {
	o := auto.ApplyOptions(opts...)

	result, err := waitForImageResultInternal(templatePath, o)
	if err != nil {
		return err
	}

	rect := result.Rectangle
	minX := auto.MinInt(rect.TopLeft.X, rect.TopRight.X, rect.BottomLeft.X, rect.BottomRight.X)
	maxX := auto.MaxInt(rect.TopLeft.X, rect.TopRight.X, rect.BottomLeft.X, rect.BottomRight.X)
	minY := auto.MinInt(rect.TopLeft.Y, rect.TopRight.Y, rect.BottomLeft.Y, rect.BottomRight.Y)
	maxY := auto.MaxInt(rect.TopLeft.Y, rect.TopRight.Y, rect.BottomLeft.Y, rect.BottomRight.Y)
	matchRegion := auto.Region{
		X:      minX,
		Y:      minY,
		Width:  auto.MaxInt(1, maxX-minX),
		Height: auto.MaxInt(1, maxY-minY),
	}

	clickPos, err := grid.CalculateGridCenterFromString(matchRegion, gridStr)
	if err != nil {
		return fmt.Errorf("计算网格位置失败: %w", err)
	}

	return input.ClickAt(clickPos.X+o.ClickOffset.X, clickPos.Y+o.ClickOffset.Y, o)
}

// ClickImageGrid 点击图像匹配结果的网格位置（ClickImageWithGrid 的别名）
func ClickImageGrid(templatePath string, gridStr string, opts ...auto.Option) error {
	return ClickImageWithGrid(templatePath, gridStr, opts...)
}

// ClickImageData 点击图像位置（使用图像数据）
func ClickImageData(template stdimage.Image, opts ...auto.Option) error {
	o := auto.ApplyOptions(opts...)

	pos, err := waitForImageDataInternal(template, o)
	if err != nil {
		return err
	}

	return input.ClickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
}

// WaitForImage 等待图像出现
func WaitForImage(templatePath string, opts ...auto.Option) (*auto.Point, error) {
	o := auto.ApplyOptions(opts...)
	return waitForImageInternal(templatePath, o)
}

// WaitForImageData 等待图像出现（使用图像数据）
func WaitForImageData(template stdimage.Image, opts ...auto.Option) (*auto.Point, error) {
	o := auto.ApplyOptions(opts...)
	return waitForImageDataInternal(template, o)
}

// ImageExists 检查图像是否存在
func ImageExists(templatePath string, opts ...auto.Option) bool {
	o := auto.ApplyOptions(opts...)
	o.Timeout = 0
	pos, _ := waitForImageInternal(templatePath, o)
	return pos != nil
}

// ImageExistsData 检查图像是否存在（使用图像数据）
func ImageExistsData(template stdimage.Image, opts ...auto.Option) bool {
	o := auto.ApplyOptions(opts...)
	o.Timeout = 0
	pos, _ := waitForImageDataInternal(template, o)
	return pos != nil
}

// ==================== 内部函数 ====================

func waitForImageInternal(templatePath string, o *auto.Options) (*auto.Point, error) {
	result, err := waitForImageResultInternal(templatePath, o)
	if err != nil {
		return nil, err
	}

	pos := result.Result
	return &auto.Point{X: pos.X, Y: pos.Y}, nil
}

func waitForImageResultInternal(templatePath string, o *auto.Options) (*cv.MatchResult, error) {
	tmpl := cv.NewTemplate(templatePath,
		cv.WithTemplateThreshold(o.Threshold),
	)

	startTime := time.Now()
	for {
		screenMat, meta, err := screen.CaptureForMatch(o)
		if err != nil {
			return nil, err
		}

		result, err := tmpl.MatchResultIn(screenMat)
		screenMat.Close()

		if err != nil {
			return nil, fmt.Errorf("匹配失败: %w", err)
		}
		if result != nil {
			return screen.AdjustMatchResult(result, meta), nil
		}

		if o.Timeout == 0 || time.Since(startTime) > o.Timeout {
			return nil, fmt.Errorf("等待图像超时: %s", templatePath)
		}

		time.Sleep(auto.DefaultPollInterval)
	}
}

func waitForImageDataInternal(template stdimage.Image, o *auto.Options) (*auto.Point, error) {
	templateMat, err := gocv.ImageToMatRGB(template)
	if err != nil {
		return nil, fmt.Errorf("转换模板图像失败: %w", err)
	}
	defer templateMat.Close()

	startTime := time.Now()
	for {
		screenMat, meta, err := screen.CaptureForMatch(o)
		if err != nil {
			return nil, err
		}

		matcher := cv.NewSIFTMatching(templateMat, screenMat, o.Threshold)
		result, err := matcher.FindBestResult()
		matcher.Close()
		screenMat.Close()

		if err != nil {
			return nil, fmt.Errorf("匹配失败: %w", err)
		}
		if result != nil {
			adjusted := screen.AdjustMatchResult(result, meta)
			return &auto.Point{X: adjusted.Result.X, Y: adjusted.Result.Y}, nil
		}

		if o.Timeout == 0 || time.Since(startTime) > o.Timeout {
			return nil, fmt.Errorf("等待图像超时")
		}

		time.Sleep(auto.DefaultPollInterval)
	}
}
