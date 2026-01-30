// Package ocr 提供 OCR 文字识别功能
package ocr

import "os"

func init() {
	// 初始化 statFile 函数
	statFile = func(path string) (interface{}, error) {
		return os.Stat(path)
	}
}

// Point 表示二维坐标点
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// OcrResult OCR 识别结果
type OcrResult struct {
	// Text 识别的文字内容
	Text string `json:"text"`
	// Confidence 识别置信度 (0-1)
	Confidence float64 `json:"confidence"`
	// Position 文字中心位置
	Position Point `json:"position"`
	// Box 文字边界框四个角点
	Box []Point `json:"box,omitempty"`
}

// Config OCR 配置
type Config struct {
	// OnnxRuntimeLibPath ONNX Runtime 动态库路径
	OnnxRuntimeLibPath string
	// DetModelPath 检测模型路径
	DetModelPath string
	// RecModelPath 识别模型路径
	RecModelPath string
	// DictPath 字典文件路径
	DictPath string
	// Language 语言 (ch, en)
	Language string
	// UseGPU 是否使用 GPU
	UseGPU bool
	// CPUThreads CPU 线程数
	CPUThreads int
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		OnnxRuntimeLibPath: getDefaultOnnxRuntimePath(),
		DetModelPath:       getDefaultModelPath("det.onnx"),
		RecModelPath:       getDefaultModelPath("rec.onnx"),
		DictPath:           getDefaultModelPath("dict.txt"),
		Language:           "ch",
		UseGPU:             false,
		CPUThreads:         4,
	}
}

// getDefaultOnnxRuntimePath 获取默认的 ONNX Runtime 库路径
func getDefaultOnnxRuntimePath() string {
	// 根据操作系统和架构选择正确的库文件
	// 优先查找项目内的 models/lib 目录
	paths := []string{
		"models/lib/onnxruntime_arm64.dylib", // macOS ARM64
		"models/lib/onnxruntime_amd64.dylib", // macOS AMD64
		"models/lib/onnxruntime_arm64.so",    // Linux ARM64
		"models/lib/onnxruntime_amd64.so",    // Linux AMD64
		"models/lib/onnxruntime.dll",         // Windows
		"./lib/onnxruntime.so",               // 默认位置
	}

	for _, p := range paths {
		if fileExists(p) {
			return p
		}
	}

	return paths[len(paths)-1]
}

// getDefaultModelPath 获取默认的模型路径
func getDefaultModelPath(filename string) string {
	paths := []string{
		"models/paddle_weights/" + filename,
		"./models/paddle_weights/" + filename,
	}

	for _, p := range paths {
		if fileExists(p) {
			return p
		}
	}

	return paths[0]
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := statFile(path)
	return err == nil
}

// statFile 包装 os.Stat 以便测试
var statFile = func(path string) (interface{}, error) {
	return nil, nil // 将在 init 中设置
}

// Box 文字边界框
type Box struct {
	// Points 四个角点坐标
	Points [4]Point
}

// Center 返回边界框中心点
func (b Box) Center() Point {
	x := (b.Points[0].X + b.Points[1].X + b.Points[2].X + b.Points[3].X) / 4
	y := (b.Points[0].Y + b.Points[1].Y + b.Points[2].Y + b.Points[3].Y) / 4
	return Point{X: x, Y: y}
}

// DetectResult 检测结果
type DetectResult struct {
	Box   Box
	Score float64
}

// RecognizeResult 识别结果
type RecognizeResult struct {
	Text       string
	Confidence float64
}

// OCRResult 完整 OCR 结果（检测 + 识别）
type OCRResult struct {
	Box        Box
	Text       string
	Confidence float64
}

// ToOcrResult 转换为 vision.OcrResult
func (r OCRResult) ToOcrResult() OcrResult {
	center := r.Box.Center()
	return OcrResult{
		Text:       r.Text,
		Confidence: r.Confidence,
		Position:   center,
		Box: []Point{
			r.Box.Points[0],
			r.Box.Points[1],
			r.Box.Points[2],
			r.Box.Points[3],
		},
	}
}
