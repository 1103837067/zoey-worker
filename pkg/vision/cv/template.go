package cv

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"gocv.io/x/gocv"
)

// CV 包配置
var (
	// DefaultThreshold 默认匹配阈值
	DefaultThreshold = 0.8
	// CurrentPath 当前工作路径
	CurrentPath = ""
)

// Template 模板匹配类
type Template struct {
	// Filename 模板文件路径
	Filename string
	// Threshold 匹配阈值
	Threshold float64
	// ScaleCandidates 额外缩放候选（用于特征点匹配）
	ScaleCandidates []float64

	// 缓存的模板图像
	cachedMat *gocv.Mat
}

// TemplateOption 模板选项
type TemplateOption func(*Template)

// NewTemplate 创建新的 Template
func NewTemplate(filename string, opts ...TemplateOption) *Template {
	t := &Template{
		Filename:  filename,
		Threshold: DefaultThreshold,
		ScaleCandidates: []float64{
			0.5,
			0.75,
			1.0,
			1.25,
			1.5,
			2.0,
		},
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// WithTemplateThreshold 设置阈值
func WithTemplateThreshold(threshold float64) TemplateOption {
	return func(t *Template) {
		t.Threshold = threshold
	}
}

// WithTemplateScales 设置额外缩放候选（用于特征点匹配）
func WithTemplateScales(scales ...float64) TemplateOption {
	return func(t *Template) {
		t.ScaleCandidates = scales
	}
}

// MatchIn 在屏幕图像中匹配模板
func (t *Template) MatchIn(screen gocv.Mat) (*Point, error) {
	result, err := t.cvMatch(screen)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	pos := result.Result
	return &pos, nil
}

// MatchResultIn 在屏幕图像中匹配模板，返回完整匹配结果
func (t *Template) MatchResultIn(screen gocv.Mat) (*MatchResult, error) {
	return t.cvMatch(screen)
}

// MatchAllIn 在屏幕图像中查找所有匹配
func (t *Template) MatchAllIn(screen gocv.Mat) ([]*MatchResult, error) {
	result, err := t.cvMatch(screen)
	if err != nil || result == nil {
		return nil, err
	}
	return []*MatchResult{result}, nil
}

// cvMatch 执行 CV 匹配
func (t *Template) cvMatch(screen gocv.Mat) (*MatchResult, error) {
	image, err := t.readImage()
	if err != nil {
		return nil, err
	}
	defer image.Close()

	scaleList := t.ScaleCandidates
	if len(scaleList) == 0 {
		scaleList = []float64{1.0}
	}

	var best *MatchResult
	for _, scale := range scaleList {
		scaledImage, cleanup := scaleTemplate(image, scale)
		m := NewSIFTMatching(scaledImage, screen, t.Threshold)
		result, err := m.FindBestResult()
		m.Close()
		if cleanup != nil {
			cleanup()
		}
		if err != nil || result == nil {
			continue
		}
		if best == nil || result.Confidence > best.Confidence {
			best = result
		}
	}
	if best != nil {
		return best, nil
	}

	return nil, nil
}

// readImage 读取模板图像
func (t *Template) readImage() (gocv.Mat, error) {
	filename := t.Filename

	if t.cachedMat != nil && !t.cachedMat.Empty() {
		return t.cachedMat.Clone(), nil
	}

	// 如果是 base64 data URL，直接读取，不处理路径
	if strings.HasPrefix(filename, "data:image/") {
		mat, err := ReadImage(filename)
		if err != nil {
			return mat, err
		}
		cached := mat.Clone()
		if t.cachedMat != nil {
			t.cachedMat.Close()
		}
		t.cachedMat = &cached
		return mat, nil
	}

	// 处理相对路径
	if CurrentPath != "" && !filepath.IsAbs(filename) {
		filename = filepath.Join(CurrentPath, filename)
	}

	mat, err := ReadImage(filename)
	if err != nil {
		return mat, err
	}
	cached := mat.Clone()
	if t.cachedMat != nil {
		t.cachedMat.Close()
	}
	t.cachedMat = &cached
	return mat, nil
}

// Close 释放资源
func (t *Template) Close() {
	if t.cachedMat != nil {
		t.cachedMat.Close()
		t.cachedMat = nil
	}
}

// String 返回字符串表示
func (t *Template) String() string {
	return fmt.Sprintf("Template(%s)", t.Filename)
}

func scaleTemplate(image gocv.Mat, scale float64) (gocv.Mat, func()) {
	if scale <= 0 {
		return image, nil
	}
	if scale == 1.0 {
		return image, nil
	}
	newW := max(1, int(float64(image.Cols())*scale))
	newH := max(1, int(float64(image.Rows())*scale))
	scaled := ResizeImage(image, newW, newH)
	return scaled, func() { scaled.Close() }
}

// FindLocation 便捷函数：在源图像中查找模板位置
func FindLocation(screen, template interface{}, opts ...TemplateOption) (*Point, error) {
	// 加载源图像
	screenMat, err := LoadImageInput(screen)
	if err != nil {
		return nil, fmt.Errorf("加载源图像失败: %w", err)
	}
	defer screenMat.Close()

	// 处理模板
	var tmpl *Template
	switch v := template.(type) {
	case string:
		tmpl = NewTemplate(v, opts...)
	case *Template:
		tmpl = v
	default:
		return nil, fmt.Errorf("不支持的模板类型: %T", template)
	}

	return tmpl.MatchIn(screenMat)
}

// FindAllLocations 便捷函数：在源图像中查找所有模板位置
func FindAllLocations(screen, template interface{}, opts ...TemplateOption) ([]*MatchResult, error) {
	// 加载源图像
	screenMat, err := LoadImageInput(screen)
	if err != nil {
		return nil, fmt.Errorf("加载源图像失败: %w", err)
	}
	defer screenMat.Close()

	// 处理模板
	var tmpl *Template
	switch v := template.(type) {
	case string:
		tmpl = NewTemplate(v, opts...)
	case *Template:
		tmpl = v
	default:
		return nil, fmt.Errorf("不支持的模板类型: %T", template)
	}

	return tmpl.MatchAllIn(screenMat)
}

// MatchLoop 循环匹配直到找到或超时
func MatchLoop(screenshotFn func() (gocv.Mat, error), template string, timeout time.Duration, opts ...TemplateOption) (*Point, error) {
	tmpl := NewTemplate(template, opts...)
	startTime := time.Now()

	for {
		screen, err := screenshotFn()
		if err != nil {
			return nil, fmt.Errorf("截图失败: %w", err)
		}

		pos, err := tmpl.MatchIn(screen)
		screen.Close()

		if err != nil {
			return nil, err
		}
		if pos != nil {
			return pos, nil
		}

		if time.Since(startTime) > timeout {
			return nil, fmt.Errorf("匹配超时")
		}

		// 短暂休眠避免 CPU 占用过高
		time.Sleep(100 * time.Millisecond)
	}
}
