package text

import (
	"fmt"

	"github.com/zoeyai/zoeyworker/pkg/vision/ocr"
)

// OCRPluginInterface OCR 插件接口
type OCRPluginInterface interface {
	IsInstalled() bool
	GetConfig() (onnxPath, detPath, recPath, dictPath string, err error)
}

// 全局 OCR 识别器和插件
var (
	globalTextRecognizer *ocr.TextRecognizer
	ocrPluginInstance    OCRPluginInterface
)

// InitOCR 初始化 OCR（可选，不调用会自动初始化）
func InitOCR(config ocr.Config) error {
	var err error
	globalTextRecognizer, err = ocr.NewTextRecognizer(config)
	return err
}

// SetOCRPlugin 设置 OCR 插件实例（避免循环导入）
func SetOCRPlugin(p OCRPluginInterface) {
	ocrPluginInstance = p
}

// getOCRPlugin 获取 OCR 插件
func getOCRPlugin() OCRPluginInterface {
	return ocrPluginInstance
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
