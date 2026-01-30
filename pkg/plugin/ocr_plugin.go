// Package plugin 管理可选插件（如 OCR）
package plugin

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// OCRPlugin OCR 插件管理器
type OCRPlugin struct {
	baseDir     string
	mu          sync.RWMutex
	downloading bool
	progress    float64
	onProgress  func(float64)
}

// OCRPluginStatus OCR 插件状态
type OCRPluginStatus struct {
	Installed       bool    `json:"installed"`
	Downloading     bool    `json:"downloading"`
	Progress        float64 `json:"progress"` // 0-100
	OnnxRuntimePath string  `json:"onnxRuntimePath"`
	DetModelPath    string  `json:"detModelPath"`
	RecModelPath    string  `json:"recModelPath"`
	DictPath        string  `json:"dictPath"`
}

// HuggingFace 模型仓库地址
const (
	HFRepoBase = "https://huggingface.co/getcharzp/go-ocr/resolve/main"
)

// 需要下载的文件列表
type downloadFile struct {
	name     string
	url      string
	destPath string
	size     int64 // 预估大小（字节）
}

// NewOCRPlugin 创建 OCR 插件管理器
func NewOCRPlugin() *OCRPlugin {
	// 默认存储在用户目录下
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".zoey-worker", "plugins", "ocr")

	return &OCRPlugin{
		baseDir: baseDir,
	}
}

// SetProgressCallback 设置进度回调
func (p *OCRPlugin) SetProgressCallback(callback func(float64)) {
	p.mu.Lock()
	p.onProgress = callback
	p.mu.Unlock()
}

// GetStatus 获取插件状态
func (p *OCRPlugin) GetStatus() OCRPluginStatus {
	p.mu.RLock()
	downloading := p.downloading
	progress := p.progress
	p.mu.RUnlock()

	status := OCRPluginStatus{
		Downloading: downloading,
		Progress:    progress,
	}

	// 检查文件是否存在
	onnxPath := p.getOnnxRuntimePath()
	detPath := filepath.Join(p.baseDir, "paddle_weights", "det.onnx")
	recPath := filepath.Join(p.baseDir, "paddle_weights", "rec.onnx")
	dictPath := filepath.Join(p.baseDir, "paddle_weights", "dict.txt")

	status.OnnxRuntimePath = onnxPath
	status.DetModelPath = detPath
	status.RecModelPath = recPath
	status.DictPath = dictPath

	// 检查所有文件是否存在
	status.Installed = fileExists(onnxPath) &&
		fileExists(detPath) &&
		fileExists(recPath) &&
		fileExists(dictPath)

	return status
}

// IsInstalled 检查是否已安装
func (p *OCRPlugin) IsInstalled() bool {
	return p.GetStatus().Installed
}

// Install 下载并安装 OCR 插件
func (p *OCRPlugin) Install() error {
	p.mu.Lock()
	if p.downloading {
		p.mu.Unlock()
		return fmt.Errorf("正在下载中")
	}
	p.downloading = true
	p.progress = 0
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.downloading = false
		p.mu.Unlock()
	}()

	// 创建目录
	if err := os.MkdirAll(filepath.Join(p.baseDir, "lib"), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(p.baseDir, "paddle_weights"), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 需要下载的文件
	files := p.getDownloadFiles()

	// 计算总大小
	var totalSize int64
	for _, f := range files {
		totalSize += f.size
	}

	// 下载所有文件
	var downloadedSize int64
	for _, f := range files {
		err := p.downloadFile(f.url, f.destPath, func(downloaded int64) {
			p.mu.Lock()
			p.progress = float64(downloadedSize+downloaded) / float64(totalSize) * 100
			if p.onProgress != nil {
				p.onProgress(p.progress)
			}
			p.mu.Unlock()
		})
		if err != nil {
			return fmt.Errorf("下载 %s 失败: %w", f.name, err)
		}
		downloadedSize += f.size
	}

	p.mu.Lock()
	p.progress = 100
	if p.onProgress != nil {
		p.onProgress(100)
	}
	p.mu.Unlock()

	return nil
}

// Uninstall 卸载 OCR 插件
func (p *OCRPlugin) Uninstall() error {
	return os.RemoveAll(p.baseDir)
}

// GetConfig 获取 OCR 配置（供 OCR 初始化使用）
func (p *OCRPlugin) GetConfig() (onnxPath, detPath, recPath, dictPath string, err error) {
	status := p.GetStatus()
	if !status.Installed {
		return "", "", "", "", fmt.Errorf("OCR 插件未安装")
	}
	return status.OnnxRuntimePath, status.DetModelPath, status.RecModelPath, status.DictPath, nil
}

// getOnnxRuntimePath 根据平台获取 ONNX Runtime 库路径
func (p *OCRPlugin) getOnnxRuntimePath() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(p.baseDir, "lib", "onnxruntime.dll")
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return filepath.Join(p.baseDir, "lib", "onnxruntime_arm64.dylib")
		}
		return filepath.Join(p.baseDir, "lib", "onnxruntime_amd64.dylib")
	default: // linux
		if runtime.GOARCH == "arm64" {
			return filepath.Join(p.baseDir, "lib", "onnxruntime_arm64.so")
		}
		return filepath.Join(p.baseDir, "lib", "onnxruntime_amd64.so")
	}
}

// getDownloadFiles 获取需要下载的文件列表
func (p *OCRPlugin) getDownloadFiles() []downloadFile {
	files := []downloadFile{
		// 模型文件
		{
			name:     "det.onnx",
			url:      HFRepoBase + "/paddle_weights/det.onnx",
			destPath: filepath.Join(p.baseDir, "paddle_weights", "det.onnx"),
			size:     3 * 1024 * 1024, // ~3MB
		},
		{
			name:     "rec.onnx",
			url:      HFRepoBase + "/paddle_weights/rec.onnx",
			destPath: filepath.Join(p.baseDir, "paddle_weights", "rec.onnx"),
			size:     5 * 1024 * 1024, // ~5MB
		},
		{
			name:     "dict.txt",
			url:      HFRepoBase + "/paddle_weights/dict.txt",
			destPath: filepath.Join(p.baseDir, "paddle_weights", "dict.txt"),
			size:     200 * 1024, // ~200KB
		},
	}

	// 根据平台添加 ONNX Runtime
	var onnxFile downloadFile
	switch runtime.GOOS {
	case "windows":
		onnxFile = downloadFile{
			name:     "onnxruntime.dll",
			url:      HFRepoBase + "/lib/onnxruntime.dll",
			destPath: filepath.Join(p.baseDir, "lib", "onnxruntime.dll"),
			size:     50 * 1024 * 1024, // ~50MB
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			onnxFile = downloadFile{
				name:     "onnxruntime_arm64.dylib",
				url:      HFRepoBase + "/lib/onnxruntime_arm64.dylib",
				destPath: filepath.Join(p.baseDir, "lib", "onnxruntime_arm64.dylib"),
				size:     50 * 1024 * 1024,
			}
		} else {
			onnxFile = downloadFile{
				name:     "onnxruntime_amd64.dylib",
				url:      HFRepoBase + "/lib/onnxruntime_amd64.dylib",
				destPath: filepath.Join(p.baseDir, "lib", "onnxruntime_amd64.dylib"),
				size:     50 * 1024 * 1024,
			}
		}
	default: // linux
		if runtime.GOARCH == "arm64" {
			onnxFile = downloadFile{
				name:     "onnxruntime_arm64.so",
				url:      HFRepoBase + "/lib/onnxruntime_arm64.so",
				destPath: filepath.Join(p.baseDir, "lib", "onnxruntime_arm64.so"),
				size:     50 * 1024 * 1024,
			}
		} else {
			onnxFile = downloadFile{
				name:     "onnxruntime_amd64.so",
				url:      HFRepoBase + "/lib/onnxruntime_amd64.so",
				destPath: filepath.Join(p.baseDir, "lib", "onnxruntime_amd64.so"),
				size:     50 * 1024 * 1024,
			}
		}
	}
	files = append([]downloadFile{onnxFile}, files...)

	return files
}

// downloadFile 下载单个文件
func (p *OCRPlugin) downloadFile(url, destPath string, onProgress func(int64)) error {
	// 创建请求
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// 创建临时文件
	tmpPath := destPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// 下载并追踪进度
	var downloaded int64
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				os.Remove(tmpPath)
				return writeErr
			}
			downloaded += int64(n)
			if onProgress != nil {
				onProgress(downloaded)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(tmpPath)
			return err
		}
	}

	// 重命名为最终文件
	out.Close()
	return os.Rename(tmpPath, destPath)
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// 全局单例
var (
	globalOCRPlugin *OCRPlugin
	ocrPluginOnce   sync.Once
)

// GetOCRPlugin 获取全局 OCR 插件管理器
func GetOCRPlugin() *OCRPlugin {
	ocrPluginOnce.Do(func() {
		globalOCRPlugin = NewOCRPlugin()
	})
	return globalOCRPlugin
}
