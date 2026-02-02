package ocr

import (
	"fmt"
	"image"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	goocr "github.com/getcharzp/go-ocr"

	"github.com/zoeyai/zoeyworker/internal/logger"
)

// 默认文字匹配相似度阈值
const DefaultSimilarityThreshold = 0.8

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
// PP-OCRv5 优化参数说明：
// - DetMaxSideLen: 1280 (提高分辨率，更好识别小字)
// - HeatmapThreshold: 0.2 (降低阈值，检测更多文字区域，适合 UI 截图)
// - DetOutsideExpandPix: 15 (扩大检测框，避免文字被裁切)
// - RecHeight: 48 (PP-OCRv5 标准识别高度)
// - RecModelNumClasses: 6625 (PP-OCRv4 中文字典类别数)
func NewTextRecognizer(config Config) (*TextRecognizer, error) {
	ocrConfig := goocr.Config{
		OnnxRuntimeLibPath: config.OnnxRuntimeLibPath,
		DetModelPath:       config.DetModelPath,
		RecModelPath:       config.RecModelPath,
		DictPath:           config.DictPath,
		// PP-OCRv4 Mobile 优化参数 - 平衡速度与精度
		DetMaxSideLen:       1920,  // 高分辨率检测
		HeatmapThreshold:    0.12,  // 低阈值，检测小文字
		DetOutsideExpandPix: 20,    // 扩大检测框边界
		RecHeight:           48,    // PP-OCRv4 识别高度
		RecModelNumClasses:  6625,  // PP-OCRv4 中文字典类别数
	}

	// 如果配置中有自定义值，使用自定义值
	if config.CPUThreads > 0 {
		ocrConfig.NumThreads = config.CPUThreads
	}
	if config.UseGPU {
		ocrConfig.UseCuda = true
	}

	engine, err := goocr.NewPaddleOcrEngine(ocrConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 OCR 引擎失败: %w", err)
	}

	logger.Info("OCR 引擎初始化成功 (PP-OCRv5)")

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

// FindText 查找特定文字的位置（使用默认 80% 相似度阈值）
func (r *TextRecognizer) FindText(img image.Image, targetText string) (*Point, error) {
	return r.FindTextWithThreshold(img, targetText, DefaultSimilarityThreshold)
}

// FindTextWithThreshold 查找特定文字的位置，支持自定义相似度阈值
// threshold: 0.0-1.0，建议 0.8（80%）
func (r *TextRecognizer) FindTextWithThreshold(img image.Image, targetText string, threshold float64) (*Point, error) {
	startTime := time.Now()

	results, err := r.Recognize(img)
	if err != nil {
		return nil, err
	}

	target := strings.ToLower(targetText)
	var bestMatch *OcrResult
	var bestSimilarity float64

	for i, result := range results {
		text := strings.ToLower(result.Text)

		// 跳过空文本
		if len(text) == 0 {
			continue
		}

		// 1. 精确匹配（最高优先级）
		if text == target {
			elapsed := float64(time.Since(startTime).Milliseconds())
			logger.LogEvent("OCR", true, elapsed, fmt.Sprintf("精确匹配: %s", targetText))
			return &result.Position, nil
		}

		// 2. 包含匹配（次高优先级）
		// 确保两个字符串都非空，且较短的字符串至少有 2 个字符
		minLen := len(target)
		if len(text) < minLen {
			minLen = len(text)
		}
		if minLen >= 2 && (strings.Contains(text, target) || strings.Contains(target, text)) {
			elapsed := float64(time.Since(startTime).Milliseconds())
			logger.LogEvent("OCR", true, elapsed, fmt.Sprintf("包含匹配: %s -> %s", targetText, result.Text))
			return &result.Position, nil
		}

		// 3. 相似度匹配（使用阈值）
		similarity := calculateSimilarity(target, text)
		if similarity >= threshold && similarity > bestSimilarity {
			bestSimilarity = similarity
			bestMatch = &results[i]
		}
	}

	// 返回最佳相似度匹配
	if bestMatch != nil {
		elapsed := float64(time.Since(startTime).Milliseconds())
		logger.LogEvent("OCR", true, elapsed, fmt.Sprintf("相似匹配(%.0f%%): %s -> %s", bestSimilarity*100, targetText, bestMatch.Text))
		return &bestMatch.Position, nil
	}

	// 输出调试信息：所有识别到的文字
	var recognizedTexts []string
	for _, r := range results {
		if len(r.Text) > 0 {
			recognizedTexts = append(recognizedTexts, r.Text)
		}
	}
	if len(recognizedTexts) > 0 {
		logger.Debug(fmt.Sprintf("识别到的文字: %v", recognizedTexts))
	}

	elapsed := float64(time.Since(startTime).Milliseconds())
	logger.LogEvent("OCR", false, elapsed, fmt.Sprintf("未找到文字: %s (阈值: %.0f%%)", targetText, threshold*100))
	return nil, nil
}

// calculateSimilarity 计算两个字符串的相似度（Levenshtein 距离归一化）
// 返回 0.0-1.0，1.0 表示完全相同
func calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// 转换为 rune 切片以正确处理中文
	r1 := []rune(s1)
	r2 := []rune(s2)

	// 计算 Levenshtein 编辑距离
	distance := levenshteinDistance(r1, r2)

	// 归一化：1 - (距离 / 最大可能距离)
	maxLen := max(len(r1), len(r2))
	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance 计算两个 rune 切片的编辑距离
func levenshteinDistance(s1, s2 []rune) int {
	len1, len2 := len(s1), len(s2)

	// 优化：短字符串作为行
	if len1 > len2 {
		s1, s2 = s2, s1
		len1, len2 = len2, len1
	}

	// 只需要两行
	prev := make([]int, len1+1)
	curr := make([]int, len1+1)

	// 初始化第一行
	for i := range prev {
		prev[i] = i
	}

	// 填充矩阵
	for j := 1; j <= len2; j++ {
		curr[0] = j
		for i := 1; i <= len1; i++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			curr[i] = min(
				prev[i]+1,      // 删除
				curr[i-1]+1,    // 插入
				prev[i-1]+cost, // 替换
			)
		}
		prev, curr = curr, prev
	}

	return prev[len1]
}

// FindTextExact 精确查找文字（不使用相似度匹配）
func (r *TextRecognizer) FindTextExact(img image.Image, targetText string) (*Point, error) {
	startTime := time.Now()

	results, err := r.Recognize(img)
	if err != nil {
		return nil, err
	}

	target := strings.ToLower(targetText)

	for _, result := range results {
		text := strings.ToLower(result.Text)
		// 只支持精确匹配和包含匹配
		if text == target || strings.Contains(text, target) || strings.Contains(target, text) {
			elapsed := float64(time.Since(startTime).Milliseconds())
			logger.LogEvent("OCR", true, elapsed, fmt.Sprintf("找到文字: %s", targetText))
			return &result.Position, nil
		}
	}

	elapsed := float64(time.Since(startTime).Milliseconds())
	logger.LogEvent("OCR", false, elapsed, fmt.Sprintf("未找到文字: %s", targetText))
	return nil, nil
}

// GetSimilarity 计算两个文字的相似度（导出给外部使用）
func GetSimilarity(s1, s2 string) float64 {
	return calculateSimilarity(strings.ToLower(s1), strings.ToLower(s2))
}

// 用于统计识别到但未精确匹配的文字
func (r *TextRecognizer) logUnmatchedTexts(results []OcrResult, target string) {
	var texts []string
	for _, r := range results {
		if r.Text != "" && utf8.RuneCountInString(r.Text) <= 20 { // 只记录短文本
			texts = append(texts, r.Text)
		}
	}
	if len(texts) > 0 {
		logger.Debug(fmt.Sprintf("识别到的文字: %v, 目标: %s", texts, target))
	}
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
