package ocr

import (
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	goocr "github.com/getcharzp/go-ocr"

	"github.com/zoeyai/zoeyworker/internal/logger"
)

// TextRecognizer OCR 识别器
type TextRecognizer struct {
	engine goocr.Engine
	config Config
	mu     sync.Mutex
}

// 全局单例实例
var (
	globalRecognizer *TextRecognizer
	globalOnce       sync.Once
	globalErr        error
)

// NewTextRecognizer 创建新的 OCR 识别器
func NewTextRecognizer(config Config) (*TextRecognizer, error) {
	ocrConfig := goocr.Config{
		OnnxRuntimeLibPath: config.OnnxRuntimeLibPath,
		DetModelPath:       config.DetModelPath,
		RecModelPath:       config.RecModelPath,
		DictPath:           config.DictPath,
	}

	engine, err := goocr.NewPaddleOcrEngine(ocrConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 OCR 引擎失败: %w", err)
	}

	logger.Info("OCR 引擎初始化成功")

	return &TextRecognizer{
		engine: engine,
		config: config,
	}, nil
}

// GetGlobalRecognizer 获取全局 OCR 识别器
func GetGlobalRecognizer() (*TextRecognizer, error) {
	globalOnce.Do(func() {
		globalRecognizer, globalErr = NewTextRecognizer(DefaultConfig())
	})
	return globalRecognizer, globalErr
}

// InitGlobalRecognizer 使用指定配置初始化全局识别器
func InitGlobalRecognizer(config Config) error {
	var err error
	globalOnce.Do(func() {
		globalRecognizer, err = NewTextRecognizer(config)
		globalErr = err
	})
	return err
}

// Recognize 识别图像中的所有文字
func (r *TextRecognizer) Recognize(img image.Image) ([]OcrResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	startTime := time.Now()

	results, err := r.engine.RunOCR(img)
	if err != nil {
		elapsed := float64(time.Since(startTime).Milliseconds())
		logger.LogEvent("OCR", false, elapsed, "识别失败")
		return nil, fmt.Errorf("OCR 识别失败: %w", err)
	}

	// 转换结果
	ocrResults := make([]OcrResult, 0, len(results))
	for _, result := range results {
		ocrResult := convertResult(result)
		ocrResults = append(ocrResults, ocrResult)
	}

	elapsed := float64(time.Since(startTime).Milliseconds())
	logger.LogEvent("OCR", true, elapsed, fmt.Sprintf("识别到 %d 个文本", len(ocrResults)))

	return ocrResults, nil
}

// FindText 查找特定文字的位置
func (r *TextRecognizer) FindText(img image.Image, targetText string) (*Point, error) {
	startTime := time.Now()

	results, err := r.Recognize(img)
	if err != nil {
		return nil, err
	}

	target := strings.ToLower(targetText)

	for _, result := range results {
		text := strings.ToLower(result.Text)
		// 支持部分匹配
		if strings.Contains(text, target) || strings.Contains(target, text) {
			elapsed := float64(time.Since(startTime).Milliseconds())
			logger.LogEvent("OCR", true, elapsed, fmt.Sprintf("找到文字: %s", targetText))
			return &result.Position, nil
		}
	}

	elapsed := float64(time.Since(startTime).Milliseconds())
	logger.LogEvent("OCR", false, elapsed, fmt.Sprintf("未找到文字: %s", targetText))
	return nil, nil
}

// GetAllText 获取图像中的所有文字（拼接）
func (r *TextRecognizer) GetAllText(img image.Image) (string, error) {
	results, err := r.Recognize(img)
	if err != nil {
		return "", err
	}

	var texts []string
	for _, result := range results {
		if result.Text != "" {
			texts = append(texts, result.Text)
		}
	}

	return strings.Join(texts, " "), nil
}

// Close 释放资源
func (r *TextRecognizer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.engine != nil {
		r.engine.Destroy()
		r.engine = nil
	}
	return nil
}

// convertResult 转换 go-ocr 结果为 OcrResult
func convertResult(result goocr.RecResult) OcrResult {
	// go-ocr RecResult: Box [4]int{x1, y1, x2, y2}, Text string, Score float32
	box := result.Box
	center := Point{
		X: (box[0] + box[2]) / 2,
		Y: (box[1] + box[3]) / 2,
	}

	return OcrResult{
		Text:       result.Text,
		Confidence: float64(result.Score),
		Position:   center,
		Box: []Point{
			{X: box[0], Y: box[1]},
			{X: box[0], Y: box[3]},
			{X: box[2], Y: box[3]},
			{X: box[2], Y: box[1]},
		},
	}
}

// ClearCache 清除全局识别器缓存
func ClearCache() {
	if globalRecognizer != nil {
		globalRecognizer.Close()
		globalRecognizer = nil
	}
	globalOnce = sync.Once{}
	globalErr = nil
}
