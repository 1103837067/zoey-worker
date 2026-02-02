package vision

import (
	"time"
)

// Options 全局配置选项
type Options struct {
	// CV 配置
	CVThreshold float64       // 匹配阈值，默认 0.8
	FindTimeout time.Duration // 查找超时时间，默认 10s

	// OCR 配置
	OCRLanguage    string // OCR 语言，默认 "ch"
	OCRModelDir    string // OCR 模型目录
	OCRUseGPU      bool   // 是否使用 GPU
	OCRUseAngleCls bool   // 是否使用方向分类器
	OCRCPUThreads  int    // CPU 线程数
	OCRUseTensorRT bool   // 是否使用 TensorRT
	OCRPrecision   string // 精度 (fp32, fp16)

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

	// OCR 默认配置
	OCRLanguage:    "ch",
	OCRModelDir:    "",
	OCRUseGPU:      true,
	OCRUseAngleCls: false,
	OCRCPUThreads:  4,
	OCRUseTensorRT: false,
	OCRPrecision:   "fp32",

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
	threshold float64
	timeout   time.Duration
}

// defaultMatchConfig 默认匹配配置
func defaultMatchConfig() *matchConfig {
	return &matchConfig{
		threshold: globalOptions.CVThreshold,
		timeout:   globalOptions.FindTimeout,
	}
}

// WithThreshold 设置匹配阈值
func WithThreshold(threshold float64) Option {
	return func(c *matchConfig) {
		c.threshold = threshold
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(c *matchConfig) {
		c.timeout = timeout
	}
}
