// Package ocr 提供 OCR 文字识别功能
package ocr

import (
	"os"
	"path/filepath"
	"runtime"
)

func init() {
	// 初始化 statFile 函数
	statFile = func(path string) (interface{}, error) {
		return os.Stat(path)
	}
}

// getExecutableDir 获取可执行文件所在目录
func getExecutableDir() string {
	execPath, err := os.Executable()
	if err != nil {
		return "."
	}
	// 解析符号链接
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "."
	}
	return filepath.Dir(execPath)
}

// getResourcesDir 获取资源目录（跨平台）
func getResourcesDir() string {
	execDir := getExecutableDir()

	if runtime.GOOS == "darwin" {
		// macOS: 检查是否在 .app bundle 中
		// 结构: ZoeyWorker.app/Contents/MacOS/ZoeyWorker
		//       ZoeyWorker.app/Contents/Resources/models
		resourcesDir := filepath.Join(execDir, "..", "Resources")
		if fileExists(resourcesDir) {
			return resourcesDir
		}
	}

	// Windows/Linux 或非 bundle 模式: 资源与可执行文件同目录
	return execDir
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
	execDir := getExecutableDir()
	resourcesDir := getResourcesDir()

	// 根据操作系统和架构选择正确的库文件
	var paths []string

	switch runtime.GOOS {
	case "darwin":
		// macOS: 先查找 Frameworks 目录（.app bundle），再查找相对路径
		frameworksDir := filepath.Join(execDir, "..", "Frameworks")
		paths = []string{
			filepath.Join(frameworksDir, "libonnxruntime.dylib"),
			filepath.Join(frameworksDir, "onnxruntime.dylib"),
			filepath.Join(execDir, "libonnxruntime.dylib"),
			filepath.Join(resourcesDir, "lib", "onnxruntime_arm64.dylib"),
			filepath.Join(resourcesDir, "lib", "onnxruntime_amd64.dylib"),
			"models/lib/onnxruntime_arm64.dylib",
			"models/lib/onnxruntime_amd64.dylib",
		}
	case "windows":
		// Windows: 与 exe 同目录
		paths = []string{
			filepath.Join(execDir, "onnxruntime.dll"),
			filepath.Join(execDir, "onnxruntime_providers_shared.dll"),
			filepath.Join(resourcesDir, "onnxruntime.dll"),
			"models/lib/onnxruntime.dll",
			"onnxruntime.dll",
		}
	default:
		// Linux
		paths = []string{
			filepath.Join(execDir, "libonnxruntime.so"),
			filepath.Join(resourcesDir, "lib", "onnxruntime_arm64.so"),
			filepath.Join(resourcesDir, "lib", "onnxruntime_amd64.so"),
			"models/lib/onnxruntime_arm64.so",
			"models/lib/onnxruntime_amd64.so",
			"./lib/onnxruntime.so",
		}
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
	execDir := getExecutableDir()
	resourcesDir := getResourcesDir()

	paths := []string{
		// 打包后的路径
		filepath.Join(resourcesDir, "models", "paddle_weights", filename),
		filepath.Join(execDir, "models", "paddle_weights", filename),
		// 开发时的相对路径
		filepath.Join("models", "paddle_weights", filename),
		filepath.Join(".", "models", "paddle_weights", filename),
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

// IsAvailable 检查 OCR 功能是否可用（模型文件是否存在）
func IsAvailable() bool {
	config := DefaultConfig()
	return fileExists(config.OnnxRuntimeLibPath) &&
		fileExists(config.DetModelPath) &&
		fileExists(config.RecModelPath) &&
		fileExists(config.DictPath)
}
