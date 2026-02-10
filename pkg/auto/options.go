// Package auto 提供 UI 自动化功能
// 组合 vision 模块和 robotgo 实现高级自动化操作
package auto

import "time"

// Option 配置选项函数类型
type Option func(*Options)

// Options 自动化操作配置
type Options struct {
	// Timeout 操作超时时间
	Timeout time.Duration
	// Threshold 图像匹配阈值 (0-1)
	Threshold float64
	// ClickOffset 点击偏移量
	ClickOffset Point
	// DoubleClick 是否双击
	DoubleClick bool
	// RightClick 是否右键点击
	RightClick bool
	// Region 搜索区域 (nil 表示全屏)
	Region *Region
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
		Threshold:   0.8,
		ClickOffset: Point{X: 0, Y: 0},
		DoubleClick: false,
		RightClick:  false,
		Region:      nil,
	}
}

// ApplyOptions 应用配置选项
func ApplyOptions(opts ...Option) *Options {
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

// WithRegion 设置搜索区域
func WithRegion(x, y, width, height int) Option {
	return func(o *Options) {
		o.Region = &Region{X: x, Y: y, Width: width, Height: height}
	}
}

// DefaultPollInterval 默认轮询间隔
const DefaultPollInterval = 200 * time.Millisecond
