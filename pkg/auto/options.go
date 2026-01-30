// Package auto 提供 UI 自动化功能
// 组合 vision 模块和 robotgo 实现高级自动化操作
package auto

import (
	"time"

	"github.com/zoeyai/zoeyworker/pkg/vision/cv"
)

// Option 配置选项函数类型
type Option func(*Options)

// Options 自动化操作配置
type Options struct {
	// Timeout 操作超时时间
	Timeout time.Duration
	// Interval 重试间隔
	Interval time.Duration
	// Threshold 图像匹配阈值 (0-1)
	Threshold float64
	// ClickOffset 点击偏移量
	ClickOffset Point
	// DoubleClick 是否双击
	DoubleClick bool
	// RightClick 是否右键点击
	RightClick bool
	// Methods 图像匹配方法
	Methods []cv.MatchMethod
	// Region 搜索区域 (nil 表示全屏)
	Region *Region
	// DisplayID 显示器 ID (-1 表示当前)
	DisplayID int
}

// Point 表示二维坐标点
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Region 表示矩形区域
type Region struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DefaultOptions 默认配置
func DefaultOptions() *Options {
	return &Options{
		Timeout:     3 * time.Second, // 默认 3 秒超时
		Interval:    200 * time.Millisecond,
		Threshold:   0.8,
		ClickOffset: Point{X: 0, Y: 0},
		DoubleClick: false,
		RightClick:  false,
		Methods:     cv.DefaultMethods,
		Region:      nil,
		DisplayID:   -1,
	}
}

// applyOptions 应用配置选项
func applyOptions(opts ...Option) *Options {
	o := DefaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithTimeout 设置超时时间
func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.Timeout = d
	}
}

// WithInterval 设置重试间隔
func WithInterval(d time.Duration) Option {
	return func(o *Options) {
		o.Interval = d
	}
}

// WithThreshold 设置匹配阈值
func WithThreshold(t float64) Option {
	return func(o *Options) {
		o.Threshold = t
	}
}

// WithClickOffset 设置点击偏移量
func WithClickOffset(x, y int) Option {
	return func(o *Options) {
		o.ClickOffset = Point{X: x, Y: y}
	}
}

// WithDoubleClick 设置双击
func WithDoubleClick() Option {
	return func(o *Options) {
		o.DoubleClick = true
	}
}

// WithRightClick 设置右键点击
func WithRightClick() Option {
	return func(o *Options) {
		o.RightClick = true
	}
}

// WithMethods 设置匹配方法
func WithMethods(methods ...cv.MatchMethod) Option {
	return func(o *Options) {
		o.Methods = methods
	}
}

// WithMultiScale 启用多尺度匹配
// 适用于不同分辨率/DPI 的场景
func WithMultiScale() Option {
	return func(o *Options) {
		o.Methods = []cv.MatchMethod{cv.MatchMethodMultiScaleTemplate}
	}
}

// WithMultiScaleFallback 多尺度匹配 + 普通匹配降级策略
// 先尝试快速的普通匹配，失败则用多尺度
func WithMultiScaleFallback() Option {
	return func(o *Options) {
		o.Methods = []cv.MatchMethod{cv.MatchMethodTemplate, cv.MatchMethodMultiScaleTemplate}
	}
}

// WithRegion 设置搜索区域
func WithRegion(x, y, width, height int) Option {
	return func(o *Options) {
		o.Region = &Region{X: x, Y: y, Width: width, Height: height}
	}
}

// WithDisplayID 设置显示器 ID
func WithDisplayID(id int) Option {
	return func(o *Options) {
		o.DisplayID = id
	}
}
