package vision

import (
	"time"
)

// Options 全局配置选项
type Options struct {
	// CV 配置
	CVThreshold float64       // 匹配阈值，默认 0.8
	FindTimeout time.Duration // 查找超时时间，默认 10s
	CVStrategy  []MatchMethod // 匹配策略列表

	// OCR 配置
	OCRLanguage     string // OCR 语言，默认 "ch"
	OCRModelDir     string // OCR 模型目录
	OCRUseGPU       bool   // 是否使用 GPU
	OCRUseAngleCls  bool   // 是否使用方向分类器
	OCRCPUThreads   int    // CPU 线程数
	OCRUseTensorRT  bool   // 是否使用 TensorRT
	OCRPrecision    string // 精度 (fp32, fp16)

	// 日志配置
	LogEnabled bool   // 是否启用日志
	LogLevel   string // 日志级别
	LogConsole bool   // 是否输出到控制台
	LogFile    bool   // 是否输出到文件
	LogPath    string // 日志文件路径

	// 路径配置
	CurrentPath string // 当前工作路径
}

// DefaultOptions 默认配置
var DefaultOptions = Options{
	// CV 默认配置
	CVThreshold: 0.8,
	FindTimeout: 10 * time.Second,
	CVStrategy:  DefaultMatchMethods,

	// OCR 默认配置
	OCRLanguage:     "ch",
	OCRModelDir:     "",
	OCRUseGPU:       true,
	OCRUseAngleCls:  false,
	OCRCPUThreads:   4,
	OCRUseTensorRT:  false,
	OCRPrecision:    "fp32",

	// 日志默认配置
	LogEnabled: true,
	LogLevel:   "INFO",
	LogConsole: true,
	LogFile:    false,
	LogPath:    "test/test.log",

	// 路径配置
	CurrentPath: "",
}

// globalOptions 全局配置实例
var globalOptions = DefaultOptions

// GetOptions 获取当前全局配置
func GetOptions() *Options {
	return &globalOptions
}

// SetOptions 设置全局配置
func SetOptions(opts Options) {
	globalOptions = opts
}

// ResetOptions 重置为默认配置
func ResetOptions() {
	globalOptions = DefaultOptions
}

// Option 配置选项函数类型
type Option func(*matchConfig)

// matchConfig 匹配时的临时配置
type matchConfig struct {
	threshold  float64
	rgb        bool
	targetPos  TargetPos
	timeout    time.Duration
	methods    []MatchMethod
	resolution [2]int
	recordPos  *Point
	scaleMax   int
	scaleStep  float64
}

// defaultMatchConfig 默认匹配配置
func defaultMatchConfig() *matchConfig {
	return &matchConfig{
		threshold: globalOptions.CVThreshold,
		rgb:       false,
		targetPos: TargetPosMid,
		timeout:   globalOptions.FindTimeout,
		methods:   globalOptions.CVStrategy,
		scaleMax:  800,
		scaleStep: 0.005,
	}
}

// WithThreshold 设置匹配阈值
func WithThreshold(threshold float64) Option {
	return func(c *matchConfig) {
		c.threshold = threshold
	}
}

// WithRGB 设置是否使用 RGB 三通道校验
func WithRGB(rgb bool) Option {
	return func(c *matchConfig) {
		c.rgb = rgb
	}
}

// WithTargetPos 设置目标位置
func WithTargetPos(pos TargetPos) Option {
	return func(c *matchConfig) {
		c.targetPos = pos
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(c *matchConfig) {
		c.timeout = timeout
	}
}

// WithMethods 设置匹配方法
func WithMethods(methods ...MatchMethod) Option {
	return func(c *matchConfig) {
		c.methods = methods
	}
}

// WithResolution 设置录制分辨率
func WithResolution(width, height int) Option {
	return func(c *matchConfig) {
		c.resolution = [2]int{width, height}
	}
}

// WithRecordPos 设置录制位置
func WithRecordPos(x, y int) Option {
	return func(c *matchConfig) {
		c.recordPos = &Point{X: x, Y: y}
	}
}

// WithScaleMax 设置多尺度匹配最大范围
func WithScaleMax(max int) Option {
	return func(c *matchConfig) {
		c.scaleMax = max
	}
}

// WithScaleStep 设置多尺度匹配步长
func WithScaleStep(step float64) Option {
	return func(c *matchConfig) {
		c.scaleStep = step
	}
}
