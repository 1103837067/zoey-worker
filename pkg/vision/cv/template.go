package cv

import (
	"fmt"
	"path/filepath"
	"time"

	"gocv.io/x/gocv"
)

// CV 包配置
var (
	// DefaultThreshold 默认匹配阈值
	DefaultThreshold = 0.8
	// DefaultMethods 默认匹配方法
	// 策略: BRISK(快速, 83%成功率) -> 多尺度(兜底, 100%成功率)
	DefaultMethods = []MatchMethod{MatchMethodBRISK, MatchMethodMultiScaleTemplate}
	// CurrentPath 当前工作路径
	CurrentPath = ""
)

// Template 模板匹配类
type Template struct {
	// Filename 模板文件路径
	Filename string
	// Threshold 匹配阈值
	Threshold float64
	// TargetPos 目标位置
	TargetPos TargetPos
	// RecordPos 录制位置
	RecordPos *Point
	// Resolution 录制分辨率
	Resolution [2]int
	// RGB 是否使用 RGB 三通道校验
	RGB bool
	// ScaleMax 多尺度匹配最大范围
	ScaleMax int
	// ScaleStep 多尺度匹配步长
	ScaleStep float64
	// Methods 匹配方法列表
	Methods []MatchMethod

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
		TargetPos: TargetPosMid,
		RGB:       false,
		ScaleMax:  800,
		ScaleStep: 0.005,
		Methods:   DefaultMethods,
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

// WithTemplateRGB 设置 RGB 校验
func WithTemplateRGB(rgb bool) TemplateOption {
	return func(t *Template) {
		t.RGB = rgb
	}
}

// WithTemplateTargetPos 设置目标位置
func WithTemplateTargetPos(pos TargetPos) TemplateOption {
	return func(t *Template) {
		t.TargetPos = pos
	}
}

// WithTemplateResolution 设置录制分辨率
func WithTemplateResolution(width, height int) TemplateOption {
	return func(t *Template) {
		t.Resolution = [2]int{width, height}
	}
}

// WithTemplateRecordPos 设置录制位置
func WithTemplateRecordPos(x, y int) TemplateOption {
	return func(t *Template) {
		t.RecordPos = &Point{X: x, Y: y}
	}
}

// WithTemplateMethods 设置匹配方法
func WithTemplateMethods(methods ...MatchMethod) TemplateOption {
	return func(t *Template) {
		t.Methods = methods
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

	// 根据 TargetPos 返回对应位置
	pos := t.TargetPos.GetPosition(result)
	return &pos, nil
}

// MatchAllIn 在屏幕图像中查找所有匹配
func (t *Template) MatchAllIn(screen gocv.Mat) ([]*MatchResult, error) {
	image, err := t.readImage()
	if err != nil {
		return nil, err
	}
	defer image.Close()

	// 调整图像大小
	resizedImage := t.resizeImage(image, screen)
	if resizedImage.Ptr() != image.Ptr() {
		defer resizedImage.Close()
	}

	matcher := NewTemplateMatching(resizedImage, screen, t.Threshold, t.RGB)
	return matcher.FindAllResults()
}

// cvMatch 执行 CV 匹配
func (t *Template) cvMatch(screen gocv.Mat) (*MatchResult, error) {
	image, err := t.readImage()
	if err != nil {
		return nil, err
	}
	defer image.Close()

	// 调整图像大小
	resizedImage := t.resizeImage(image, screen)
	if resizedImage.Ptr() != image.Ptr() {
		defer resizedImage.Close()
	}

	// 尝试不同的匹配方法
	for _, method := range t.Methods {
		result, err := t.tryMatch(method, resizedImage, screen)
		if err != nil {
			continue
		}
		if result != nil {
			return result, nil
		}
	}

	return nil, nil
}

// tryMatch 尝试使用指定方法匹配
func (t *Template) tryMatch(method MatchMethod, image, screen gocv.Mat) (*MatchResult, error) {
	var matcher interface {
		FindBestResult() (*MatchResult, error)
	}

	switch method {
	case MatchMethodTemplate:
		matcher = NewTemplateMatching(image, screen, t.Threshold, t.RGB)
	case MatchMethodMultiScaleTemplate:
		// 多尺度模板匹配，使用配置的参数
		matcher = NewMultiScaleTemplateMatchingWithParams(image, screen, t.Threshold, t.RGB, t.ScaleMax, t.ScaleStep)
	case MatchMethodKAZE:
		m := NewKAZEMatching(image, screen, t.Threshold, t.RGB)
		defer m.Close()
		matcher = m
	case MatchMethodBRISK:
		m := NewBRISKMatching(image, screen, t.Threshold, t.RGB)
		defer m.Close()
		matcher = m
	case MatchMethodAKAZE:
		m := NewAKAZEMatching(image, screen, t.Threshold, t.RGB)
		defer m.Close()
		matcher = m
	case MatchMethodORB:
		m := NewORBMatching(image, screen, t.Threshold, t.RGB)
		defer m.Close()
		matcher = m
	default:
		return nil, fmt.Errorf("不支持的匹配方法: %s", method)
	}

	return matcher.FindBestResult()
}

// readImage 读取模板图像
func (t *Template) readImage() (gocv.Mat, error) {
	// 处理相对路径
	filename := t.Filename
	if CurrentPath != "" && !filepath.IsAbs(filename) {
		filename = filepath.Join(CurrentPath, filename)
	}

	return ReadImage(filename)
}

// resizeImage 调整图像大小以适配屏幕分辨率
func (t *Template) resizeImage(image, screen gocv.Mat) gocv.Mat {
	// 未记录分辨率，不调整
	if t.Resolution[0] == 0 || t.Resolution[1] == 0 {
		return image
	}

	screenW, screenH := screen.Cols(), screen.Rows()

	// 分辨率一致，不调整
	if t.Resolution[0] == screenW && t.Resolution[1] == screenH {
		return image
	}

	// 计算缩放后的尺寸
	h, w := image.Rows(), image.Cols()
	scaleX := float64(screenW) / float64(t.Resolution[0])
	scaleY := float64(screenH) / float64(t.Resolution[1])

	// 使用较小的缩放比例
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newW := max(1, int(float64(w)*scale))
	newH := max(1, int(float64(h)*scale))

	return ResizeImage(image, newW, newH)
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
