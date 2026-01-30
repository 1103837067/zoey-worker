package cv

import (
	"image"
	"image/color"
	"time"

	"gocv.io/x/gocv"
)

const (
	// MaxResultCount 最大匹配结果数量
	MaxResultCount = 10
)

// TemplateMatching 模板匹配器
type TemplateMatching struct {
	imSearch  gocv.Mat
	imSource  gocv.Mat
	threshold float64
	rgb       bool
}

// NewTemplateMatching 创建模板匹配器
func NewTemplateMatching(search, source gocv.Mat, threshold float64, rgb bool) *TemplateMatching {
	return &TemplateMatching{
		imSearch:  search,
		imSource:  source,
		threshold: threshold,
		rgb:       rgb,
	}
}

// FindBestResult 查找最佳匹配结果
func (t *TemplateMatching) FindBestResult() (*MatchResult, error) {
	startTime := time.Now()

	// 检查图像尺寸
	if err := checkSourceLargerThanSearch(t.imSource, t.imSearch); err != nil {
		return nil, err
	}

	// 计算模板匹配结果矩阵
	result := t.getTemplateResultMatrix()
	defer result.Close()

	// 获取最佳匹配位置
	_, maxVal, _, maxLoc := gocv.MinMaxLoc(result)

	h, w := t.imSearch.Rows(), t.imSearch.Cols()

	// 计算置信度
	confidence := t.getConfidence(maxLoc, maxVal, w, h)

	// 计算匹配区域
	middlePoint, rectangle := t.getTargetRectangle(maxLoc, w, h)

	elapsed := float64(time.Since(startTime).Milliseconds())

	matchResult := &MatchResult{
		Result:     middlePoint,
		Rectangle:  rectangle,
		Confidence: confidence,
		Time:       elapsed,
	}

	if confidence >= t.threshold {
		return matchResult, nil
	}
	return nil, nil
}

// FindAllResults 查找所有匹配结果
func (t *TemplateMatching) FindAllResults() ([]*MatchResult, error) {
	startTime := time.Now()

	// 检查图像尺寸
	if err := checkSourceLargerThanSearch(t.imSource, t.imSearch); err != nil {
		return nil, err
	}

	// 计算模板匹配结果矩阵
	result := t.getTemplateResultMatrix()
	defer result.Close()

	h, w := t.imSearch.Rows(), t.imSearch.Cols()
	var results []*MatchResult

	for len(results) < MaxResultCount {
		_, maxVal, _, maxLoc := gocv.MinMaxLoc(result)

		confidence := t.getConfidence(maxLoc, maxVal, w, h)
		if confidence < t.threshold {
			break
		}

		middlePoint, rectangle := t.getTargetRectangle(maxLoc, w, h)
		elapsed := float64(time.Since(startTime).Milliseconds())

		results = append(results, &MatchResult{
			Result:     middlePoint,
			Rectangle:  rectangle,
			Confidence: confidence,
			Time:       elapsed,
		})

		// 屏蔽已匹配区域
		gocv.Rectangle(&result,
			image.Rect(maxLoc.X-w/2, maxLoc.Y-h/2, maxLoc.X+w/2, maxLoc.Y+h/2),
			color.RGBA{0, 0, 0, 255}, -1)
	}

	return results, nil
}

// getTemplateResultMatrix 计算模板匹配结果矩阵
func (t *TemplateMatching) getTemplateResultMatrix() gocv.Mat {
	// 转换为灰度图
	srcGray := ToGray(t.imSource)
	searchGray := ToGray(t.imSearch)
	defer srcGray.Close()
	defer searchGray.Close()

	result := gocv.NewMat()
	gocv.MatchTemplate(srcGray, searchGray, &result, gocv.TmCcoeffNormed, gocv.NewMat())

	return result
}

// getConfidence 计算置信度
func (t *TemplateMatching) getConfidence(maxLoc image.Point, maxVal float32, w, h int) float64 {
	if t.rgb {
		// RGB 三通道校验
		imgCrop := t.imSource.Region(image.Rect(maxLoc.X, maxLoc.Y, maxLoc.X+w, maxLoc.Y+h))
		defer imgCrop.Close()
		return CalRGBConfidence(imgCrop, t.imSearch)
	}
	return float64(maxVal)
}

// getTargetRectangle 计算目标区域
func (t *TemplateMatching) getTargetRectangle(leftTopPos image.Point, w, h int) (Point, Rectangle) {
	xMin, yMin := leftTopPos.X, leftTopPos.Y

	// 中心位置
	xMiddle := xMin + w/2
	yMiddle := yMin + h/2

	middlePoint := Point{X: xMiddle, Y: yMiddle}

	// 四个角点: 左上 -> 左下 -> 右下 -> 右上
	rectangle := Rectangle{
		TopLeft:     Point{X: xMin, Y: yMin},
		BottomLeft:  Point{X: xMin, Y: yMin + h},
		BottomRight: Point{X: xMin + w, Y: yMin + h},
		TopRight:    Point{X: xMin + w, Y: yMin},
	}

	return middlePoint, rectangle
}

// checkSourceLargerThanSearch 检查源图像是否大于搜索图像
func checkSourceLargerThanSearch(source, search gocv.Mat) error {
	if source.Rows() < search.Rows() || source.Cols() < search.Cols() {
		return &ImageSizeError{
			SourceSize: [2]int{source.Cols(), source.Rows()},
			SearchSize: [2]int{search.Cols(), search.Rows()},
		}
	}
	return nil
}

// ImageSizeError 图像尺寸错误
type ImageSizeError struct {
	SourceSize [2]int
	SearchSize [2]int
}

func (e *ImageSizeError) Error() string {
	return "搜索图像尺寸大于源图像"
}
