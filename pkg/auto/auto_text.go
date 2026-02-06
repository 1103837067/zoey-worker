package auto

import (
	"fmt"
	"image"
	"time"

	"github.com/zoeyai/zoeyworker/pkg/vision/ocr"
)

// ==================== OCR/文字操作 ====================

// 全局 OCR 识别器
var globalTextRecognizer *ocr.TextRecognizer

// InitOCR 初始化 OCR（可选，不调用会自动初始化）
func InitOCR(config ocr.Config) error {
	var err error
	globalTextRecognizer, err = ocr.NewTextRecognizer(config)
	return err
}

// getTextRecognizer 获取或创建 OCR 识别器
func getTextRecognizer() (*ocr.TextRecognizer, error) {
	if globalTextRecognizer == nil {
		// 尝试使用插件提供的配置
		ocrPlugin := getOCRPlugin()
		if ocrPlugin != nil && ocrPlugin.IsInstalled() {
			onnxPath, detPath, recPath, dictPath, err := ocrPlugin.GetConfig()
			if err == nil {
				config := ocr.Config{
					OnnxRuntimeLibPath: onnxPath,
					DetModelPath:       detPath,
					RecModelPath:       recPath,
					DictPath:           dictPath,
				}
				recognizer, err := ocr.NewTextRecognizer(config)
				if err == nil {
					globalTextRecognizer = recognizer
					return globalTextRecognizer, nil
				}
			}
		}

		// 回退到默认配置
		recognizer, err := ocr.GetGlobalRecognizer()
		if err != nil {
			return nil, fmt.Errorf("初始化 OCR 失败: %w", err)
		}
		globalTextRecognizer = recognizer
	}
	return globalTextRecognizer, nil
}

// OCRPluginInterface OCR 插件接口
type OCRPluginInterface interface {
	IsInstalled() bool
	GetConfig() (onnxPath, detPath, recPath, dictPath string, err error)
}

var ocrPluginInstance OCRPluginInterface

// SetOCRPlugin 设置 OCR 插件实例（避免循环导入）
func SetOCRPlugin(p OCRPluginInterface) {
	ocrPluginInstance = p
}

func getOCRPlugin() OCRPluginInterface {
	return ocrPluginInstance
}

// ClickText 点击文字位置
func ClickText(text string, opts ...Option) error {
	o := applyOptions(opts...)

	pos, err := waitForTextInternal(text, o)
	if err != nil {
		return err
	}

	return clickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
}

// WaitForText 等待文字出现
func WaitForText(text string, opts ...Option) (*Point, error) {
	o := applyOptions(opts...)
	return waitForTextInternal(text, o)
}

// TextExists 检查文字是否存在
func TextExists(text string, opts ...Option) bool {
	o := applyOptions(opts...)
	o.Timeout = 0
	pos, _ := waitForTextInternal(text, o)
	return pos != nil
}

// waitForTextInternal 内部等待文字函数
func waitForTextInternal(text string, o *Options) (*Point, error) {
	recognizer, err := getTextRecognizer()
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	for {
		// 截图
		var img image.Image
		var captureErr error
		if o.Region != nil {
			img, captureErr = CaptureRegion(o.Region.X, o.Region.Y, o.Region.Width, o.Region.Height)
		} else {
			img, captureErr = CaptureScreen()
		}
		if captureErr != nil {
			return nil, captureErr
		}

		// OCR 查找文字
		result, err := recognizer.FindText(img, text)
		if err != nil {
			return nil, fmt.Errorf("OCR 识别失败: %w", err)
		}

		if result != nil {
			meta := buildCaptureMeta(o, img)
			adjusted := adjustPoint(Point{X: result.X, Y: result.Y}, meta)
			return &adjusted, nil
		}

		if o.Timeout == 0 || time.Since(startTime) > o.Timeout {
			return nil, fmt.Errorf("等待文字超时: %s", text)
		}

		time.Sleep(defaultPollInterval)
	}
}
