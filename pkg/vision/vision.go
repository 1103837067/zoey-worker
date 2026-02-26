// Package vision 提供图像识别与 OCR 功能
//
// 主要功能:
//   - 图像匹配 (CV): SIFT 特征点匹配
//   - 文字识别 (OCR): 基于 PaddleOCR 的中英文识别
//
// 基本用法:
//
//	// 图像匹配
//	pos, err := vision.FindLocation("screen.png", "template.png")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("找到位置: (%d, %d)\n", pos.X, pos.Y)
//
//	// OCR 识别
//	results, err := vision.RecognizeText("image.png")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, r := range results {
//	    fmt.Printf("文字: %s\n", r.Text)
//	}
package vision

import (
	"image"
	"time"

	"gocv.io/x/gocv"

	"github.com/zoeyai/zoeyworker/pkg/vision/cv"
	"github.com/zoeyai/zoeyworker/pkg/vision/ocr"
)

// ============ CV 便捷函数 ============

// FindLocation 在源图像中查找模板位置
// screen: 源图像 (文件路径、image.Image 或 gocv.Mat)
// template: 模板 (文件路径或 *cv.Template)
// opts: 可选配置
func FindLocation(screen, template interface{}, opts ...Option) (*Point, error) {
	cfg := defaultMatchConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// 转换为 cv 选项
	cvOpts := buildCVOptions(cfg)

	cvPoint, err := cv.FindLocation(screen, template, cvOpts...)
	if err != nil {
		return nil, err
	}
	if cvPoint == nil {
		return nil, nil
	}
	return &Point{X: cvPoint.X, Y: cvPoint.Y}, nil
}

// FindAllLocations 在源图像中查找所有模板位置
func FindAllLocations(screen, template interface{}, opts ...Option) ([]*MatchResult, error) {
	cfg := defaultMatchConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	cvOpts := buildCVOptions(cfg)

	cvResults, err := cv.FindAllLocations(screen, template, cvOpts...)
	if err != nil {
		return nil, err
	}

	results := make([]*MatchResult, len(cvResults))
	for i, r := range cvResults {
		results[i] = convertCVMatchResult(r)
	}
	return results, nil
}

// MatchLoop 循环匹配直到找到或超时
func MatchLoop(screenshotFn func() (gocv.Mat, error), template string, opts ...Option) (*Point, error) {
	cfg := defaultMatchConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	cvOpts := buildCVOptions(cfg)

	cvPoint, err := cv.MatchLoop(screenshotFn, template, cfg.timeout, cvOpts...)
	if err != nil {
		return nil, err
	}
	if cvPoint == nil {
		return nil, nil
	}
	return &Point{X: cvPoint.X, Y: cvPoint.Y}, nil
}

// convertCVMatchResult 转换 cv.MatchResult 到 vision.MatchResult
func convertCVMatchResult(r *cv.MatchResult) *MatchResult {
	if r == nil {
		return nil
	}
	return &MatchResult{
		Result: Point{X: r.Result.X, Y: r.Result.Y},
		Rectangle: Rectangle{
			TopLeft:     Point{X: r.Rectangle.TopLeft.X, Y: r.Rectangle.TopLeft.Y},
			BottomLeft:  Point{X: r.Rectangle.BottomLeft.X, Y: r.Rectangle.BottomLeft.Y},
			BottomRight: Point{X: r.Rectangle.BottomRight.X, Y: r.Rectangle.BottomRight.Y},
			TopRight:    Point{X: r.Rectangle.TopRight.X, Y: r.Rectangle.TopRight.Y},
		},
		Confidence: r.Confidence,
		Time:       r.Time,
	}
}

// buildCVOptions 构建 CV 选项
func buildCVOptions(cfg *matchConfig) []cv.TemplateOption {
	var opts []cv.TemplateOption

	opts = append(opts, cv.WithTemplateThreshold(cfg.threshold))

	return opts
}

// ============ OCR 便捷函数 ============

// RecognizeText 识别图像中的所有文字
// img: 图像输入 (文件路径或 image.Image)
func RecognizeText(img interface{}) ([]OcrResult, error) {
	ocrResults, err := ocr.RecognizeText(img)
	if err != nil {
		return nil, err
	}

	results := make([]OcrResult, len(ocrResults))
	for i, r := range ocrResults {
		results[i] = convertOCRResult(r)
	}
	return results, nil
}

// FindTextPosition 查找特定文字的位置
func FindTextPosition(img interface{}, text string) (*Point, error) {
	ocrPoint, err := ocr.FindTextPosition(img, text)
	if err != nil {
		return nil, err
	}
	if ocrPoint == nil {
		return nil, nil
	}
	return &Point{X: ocrPoint.X, Y: ocrPoint.Y}, nil
}

// GetAllText 获取图像中的所有文字
func GetAllText(img interface{}) (string, error) {
	return ocr.GetAllText(img)
}

// convertOCRResult 转换 ocr.OcrResult 到 vision.OcrResult
func convertOCRResult(r ocr.OcrResult) OcrResult {
	box := make([]Point, len(r.Box))
	for i, p := range r.Box {
		box[i] = Point{X: p.X, Y: p.Y}
	}
	return OcrResult{
		Text:       r.Text,
		Confidence: r.Confidence,
		Position:   Point{X: r.Position.X, Y: r.Position.Y},
		Box:        box,
	}
}

// ============ OCR 配置 ============

// InitOCR 初始化 OCR 引擎
func InitOCR(config ocr.Config) error {
	return ocr.InitGlobalRecognizer(config)
}

// NewOCRConfig 创建 OCR 配置
func NewOCRConfig() ocr.Config {
	return ocr.DefaultConfig()
}

// ============ 工具函数 ============

// ReadImage 读取图像文件
func ReadImage(filename string) (gocv.Mat, error) {
	return cv.ReadImage(filename)
}

// LoadImage 加载图像 (支持多种输入类型)
func LoadImage(input interface{}) (gocv.Mat, error) {
	return cv.LoadImageInput(input)
}

// ImageToMat 将 image.Image 转换为 gocv.Mat
func ImageToMat(img image.Image) (gocv.Mat, error) {
	return cv.ImageToMat(img)
}

// ============ Template 快捷创建 ============

// NewTemplate 创建模板
func NewTemplate(filename string, opts ...Option) *cv.Template {
	cfg := defaultMatchConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return cv.NewTemplate(filename, buildCVOptions(cfg)...)
}

// ============ 类型别名 ============

// Template 模板类型别名
type Template = cv.Template

// TextRecognizer OCR 识别器类型别名
type TextRecognizer = ocr.TextRecognizer

// OCRConfig OCR 配置类型别名
type OCRConfig = ocr.Config

// ============ 常量 ============

// 超时常量
const (
	DefaultTimeout = 10 * time.Second
)
