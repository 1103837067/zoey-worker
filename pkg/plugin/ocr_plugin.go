// Package plugin 管理可选插件（如 OCR）
package plugin

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// 模型和库下载地址 - 使用 PP-OCRv5 最新模型 + ONNX Runtime 1.23
const (
	// PP-OCRv4 Mobile 模型来自 SWHL/RapidOCR (轻量高速版，仅 16MB)
	RapidOCRBase = "https://huggingface.co/SWHL/RapidOCR/resolve/main/PP-OCRv4"
	// PP-OCRv3 字典（PP-OCRv4 共用）
	DictBase = "https://huggingface.co/monkt/paddleocr-onnx/resolve/main/languages/chinese"
	// ONNX Runtime 1.23.0 官方下载
	OnnxRuntimeVersion = "1.23.0"
	OnnxRuntimeBaseURL = "https://github.com/microsoft/onnxruntime/releases/download/v" + OnnxRuntimeVersion
)

// 需要下载的文件列表
type downloadFile struct {
	name       string
	url        string
	destPath   string
	size       int64  // 预估大小（字节）
	isArchive  bool   // 是否为压缩包
	archiveLib string // 压缩包内的库文件路径
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
		var err error
		if f.isArchive {
			// 下载并解压压缩包
			err = p.downloadAndExtract(f.url, f.destPath, f.archiveLib, func(downloaded int64) {
				p.mu.Lock()
				p.progress = float64(downloadedSize+downloaded) / float64(totalSize) * 100
				if p.onProgress != nil {
					p.onProgress(p.progress)
				}
				p.mu.Unlock()
			})
		} else {
			// 直接下载文件
			err = p.downloadFile(f.url, f.destPath, func(downloaded int64) {
				p.mu.Lock()
				p.progress = float64(downloadedSize+downloaded) / float64(totalSize) * 100
				if p.onProgress != nil {
					p.onProgress(p.progress)
				}
				p.mu.Unlock()
			})
		}
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
// PP-OCRv4 Mobile 模型：检测模型 4.75MB，中文识别模型 10.9MB，字典 74KB（共约 16MB）
func (p *OCRPlugin) getDownloadFiles() []downloadFile {
	files := []downloadFile{
		// PP-OCRv4 Mobile 检测模型
		{
			name:     "det.onnx",
			url:      RapidOCRBase + "/ch_PP-OCRv4_det_infer.onnx",
			destPath: filepath.Join(p.baseDir, "paddle_weights", "det.onnx"),
			size:     5 * 1024 * 1024, // ~4.75MB
		},
		// PP-OCRv4 Mobile 中文识别模型
		{
			name:     "rec.onnx",
			url:      RapidOCRBase + "/ch_PP-OCRv4_rec_infer.onnx",
			destPath: filepath.Join(p.baseDir, "paddle_weights", "rec.onnx"),
			size:     11 * 1024 * 1024, // ~10.9MB
		},
		// PP-OCRv4 中文字典 (ppocr_keys_v1.txt, 6623 字符)
		{
			name:     "dict.txt",
			url:      "https://raw.githubusercontent.com/PaddlePaddle/PaddleOCR/main/ppocr/utils/ppocr_keys_v1.txt",
			destPath: filepath.Join(p.baseDir, "paddle_weights", "dict.txt"),
			size:     30 * 1024, // ~30KB
		},
	}

	// 根据平台添加 ONNX Runtime 1.23.0 (从 GitHub 官方下载)
	// 注意：需要解压 tgz 包，这里改为直接下载解压后的库文件
	// 由于 GitHub releases 是 tgz 格式，我们需要特殊处理
	var onnxFile downloadFile
	switch runtime.GOOS {
	case "windows":
		// Windows: onnxruntime-win-x64-1.23.0.zip
		onnxFile = downloadFile{
			name:       "onnxruntime.dll",
			url:        OnnxRuntimeBaseURL + "/onnxruntime-win-x64-" + OnnxRuntimeVersion + ".zip",
			destPath:   filepath.Join(p.baseDir, "lib", "onnxruntime.dll"),
			size:       35 * 1024 * 1024,
			isArchive:  true,
			archiveLib: "onnxruntime-win-x64-" + OnnxRuntimeVersion + "/lib/onnxruntime.dll",
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			// macOS ARM64: onnxruntime-osx-arm64-1.23.0.tgz
			onnxFile = downloadFile{
				name:       "onnxruntime_arm64.dylib",
				url:        OnnxRuntimeBaseURL + "/onnxruntime-osx-arm64-" + OnnxRuntimeVersion + ".tgz",
				destPath:   filepath.Join(p.baseDir, "lib", "onnxruntime_arm64.dylib"),
				size:       35 * 1024 * 1024,
				isArchive:  true,
				archiveLib: "onnxruntime-osx-arm64-" + OnnxRuntimeVersion + "/lib/libonnxruntime." + OnnxRuntimeVersion + ".dylib",
			}
		} else {
			// macOS x64: onnxruntime-osx-x64-1.23.0.tgz (注意: 1.24 将停止提供)
			onnxFile = downloadFile{
				name:       "onnxruntime_amd64.dylib",
				url:        OnnxRuntimeBaseURL + "/onnxruntime-osx-x64-" + OnnxRuntimeVersion + ".tgz",
				destPath:   filepath.Join(p.baseDir, "lib", "onnxruntime_amd64.dylib"),
				size:       50 * 1024 * 1024,
				isArchive:  true,
				archiveLib: "onnxruntime-osx-x64-" + OnnxRuntimeVersion + "/lib/libonnxruntime." + OnnxRuntimeVersion + ".dylib",
			}
		}
	default: // linux
		if runtime.GOARCH == "arm64" {
			// Linux ARM64: onnxruntime-linux-aarch64-1.23.0.tgz
			onnxFile = downloadFile{
				name:       "onnxruntime_arm64.so",
				url:        OnnxRuntimeBaseURL + "/onnxruntime-linux-aarch64-" + OnnxRuntimeVersion + ".tgz",
				destPath:   filepath.Join(p.baseDir, "lib", "onnxruntime_arm64.so"),
				size:       35 * 1024 * 1024,
				isArchive:  true,
				archiveLib: "onnxruntime-linux-aarch64-" + OnnxRuntimeVersion + "/lib/libonnxruntime.so." + OnnxRuntimeVersion,
			}
		} else {
			// Linux x64: onnxruntime-linux-x64-1.23.0.tgz
			onnxFile = downloadFile{
				name:       "onnxruntime_amd64.so",
				url:        OnnxRuntimeBaseURL + "/onnxruntime-linux-x64-" + OnnxRuntimeVersion + ".tgz",
				destPath:   filepath.Join(p.baseDir, "lib", "onnxruntime_amd64.so"),
				size:       35 * 1024 * 1024,
				isArchive:  true,
				archiveLib: "onnxruntime-linux-x64-" + OnnxRuntimeVersion + "/lib/libonnxruntime.so." + OnnxRuntimeVersion,
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

// downloadAndExtract 下载压缩包并解压特定文件
func (p *OCRPlugin) downloadAndExtract(url, destPath, archiveLib string, onProgress func(int64)) error {
	// 下载到临时文件
	tmpArchive := destPath + ".archive.tmp"
	defer os.Remove(tmpArchive)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// 下载压缩包
	out, err := os.Create(tmpArchive)
	if err != nil {
		return err
	}

	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				out.Close()
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
			out.Close()
			return err
		}
	}
	out.Close()

	// 根据文件类型解压
	if strings.HasSuffix(url, ".tgz") || strings.HasSuffix(url, ".tar.gz") {
		return p.extractTgz(tmpArchive, destPath, archiveLib)
	} else if strings.HasSuffix(url, ".zip") {
		return p.extractZip(tmpArchive, destPath, archiveLib)
	}

	return fmt.Errorf("不支持的压缩格式: %s", url)
}

// extractTgz 从 tgz 文件中提取特定文件
func (p *OCRPlugin) extractTgz(archivePath, destPath, targetFile string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Name == targetFile {
			// 找到目标文件，写入 destPath
			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, tr)
			return err
		}
	}

	return fmt.Errorf("在压缩包中未找到文件: %s", targetFile)
}

// extractZip 从 zip 文件中提取特定文件
func (p *OCRPlugin) extractZip(archivePath, destPath, targetFile string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == targetFile {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, rc)
			return err
		}
	}

	return fmt.Errorf("在压缩包中未找到文件: %s", targetFile)
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
