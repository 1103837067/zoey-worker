package cv

import (
	"image"
	"math"
	"time"

	"gocv.io/x/gocv"
)

// MultiScaleTemplateMatching 多尺度模板匹配
// 适用场景：
//   - 不同分辨率显示器（1080p vs 4K）
//   - DPI 缩放（125%/150%/200%）
//   - 响应式 UI 元素大小变化
//   - 录制和回放时分辨率不同
type MultiScaleTemplateMatching struct {
	imSearch  gocv.Mat
	imSource  gocv.Mat
	threshold float64
	rgb       bool
	scaleMax  int     // 源图像最大尺寸限制，默认 800
	scaleStep float64 // 搜索步长，默认 0.005
}

// NewMultiScaleTemplateMatching 创建多尺度模板匹配器
func NewMultiScaleTemplateMatching(search, source gocv.Mat, threshold float64, rgb bool) *MultiScaleTemplateMatching {
	return &MultiScaleTemplateMatching{
		imSearch:  search,
		imSource:  source,
		threshold: threshold,
		rgb:       rgb,
		scaleMax:  800,
		scaleStep: 0.005,
	}
}

// NewMultiScaleTemplateMatchingWithParams 创建多尺度模板匹配器（带参数）
func NewMultiScaleTemplateMatchingWithParams(search, source gocv.Mat, threshold float64, rgb bool, scaleMax int, scaleStep float64) *MultiScaleTemplateMatching {
	return &MultiScaleTemplateMatching{
		imSearch:  search,
		imSource:  source,
		threshold: threshold,
		rgb:       rgb,
		scaleMax:  scaleMax,
		scaleStep: scaleStep,
	}
}

// FindBestResult 查找最佳匹配结果
func (m *MultiScaleTemplateMatching) FindBestResult() (*MatchResult, error) {
	startTime := time.Now()

	// 校验图像输入
	if err := checkSourceLargerThanSearch(m.imSource, m.imSearch); err != nil {
		return nil, err
	}

	// 转换为灰度图
	searchGray := gocv.NewMat()
	sourceGray := gocv.NewMat()
	defer searchGray.Close()
	defer sourceGray.Close()

	gocv.CvtColor(m.imSearch, &searchGray, gocv.ColorBGRToGray)
	gocv.CvtColor(m.imSource, &sourceGray, gocv.ColorBGRToGray)

	// 多尺度搜索
	confidence, maxLoc, w, h := m.multiScaleSearch(
		sourceGray,
		searchGray,
		0.01,  // ratioMin
		0.99,  // ratioMax
		3.0,   // timeout
	)

	if confidence < m.threshold {
		return nil, nil
	}

	// 计算目标区域
	result := m.buildResult(maxLoc, w, h, confidence, startTime)
	return result, nil
}

// FindAllResults 查找所有匹配结果（多尺度匹配不支持，返回最佳结果）
func (m *MultiScaleTemplateMatching) FindAllResults() ([]*MatchResult, error) {
	result, err := m.FindBestResult()
	if err != nil || result == nil {
		return nil, err
	}
	return []*MatchResult{result}, nil
}

// multiScaleSearch 多尺度搜索核心算法
func (m *MultiScaleTemplateMatching) multiScaleSearch(
	source, search gocv.Mat,
	ratioMin, ratioMax float64,
	timeout float64,
) (confidence float64, maxLoc Point, w, h int) {

	var maxInfo *scaleSearchInfo
	maxVal := float64(0)

	startTime := time.Now()
	ratio := ratioMin

	for ratio <= ratioMax {
		// 按比例缩放
		scaledSource, scaledSearch, tr, sr := m.resizeByRatio(source, search, ratio)
		
		// 检查模板最小尺寸
		if scaledSearch.Rows() < 10 || scaledSearch.Cols() < 10 {
			scaledSource.Close()
			scaledSearch.Close()
			ratio += m.scaleStep
			continue
		}

		// 模板匹配
		result := gocv.NewMat()
		gocv.MatchTemplate(scaledSource, scaledSearch, &result, gocv.TmCcoeffNormed, gocv.NewMat())
		
		_, currentMaxVal, _, currentMaxLoc := gocv.MinMaxLoc(result)
		result.Close()

		scaledW := scaledSearch.Cols()
		scaledH := scaledSearch.Rows()

		currentMaxValF64 := float64(currentMaxVal)
		if currentMaxValF64 > maxVal {
			maxVal = currentMaxValF64
			maxInfo = &scaleSearchInfo{
				ratio:  ratio,
				maxVal: currentMaxValF64,
				maxLoc: Point{X: currentMaxLoc.X, Y: currentMaxLoc.Y},
				w:      scaledW,
				h:      scaledH,
				tr:     tr,
				sr:     sr,
			}
		}

		scaledSource.Close()
		scaledSearch.Close()

		// 超时检查
		elapsed := time.Since(startTime).Seconds()
		if elapsed > timeout && maxVal >= m.threshold {
			if maxInfo != nil {
				orgLoc, orgW, orgH := m.orgSize(maxInfo.maxLoc, maxInfo.w, maxInfo.h, maxInfo.tr, maxInfo.sr)
				conf := m.getConfidenceFromCrop(orgLoc, orgW, orgH)
				if conf >= m.threshold {
					return conf, orgLoc, orgW, orgH
				}
			}
		}

		ratio += m.scaleStep
	}

	if maxInfo == nil {
		return 0, Point{}, 0, 0
	}

	// 返回最佳结果
	orgLoc, orgW, orgH := m.orgSize(maxInfo.maxLoc, maxInfo.w, maxInfo.h, maxInfo.tr, maxInfo.sr)
	conf := m.getConfidenceFromCrop(orgLoc, orgW, orgH)
	return conf, orgLoc, orgW, orgH
}

// scaleSearchInfo 存储单次搜索的信息
type scaleSearchInfo struct {
	ratio  float64
	maxVal float64
	maxLoc Point
	w, h   int
	tr, sr float64
}

// resizeByRatio 按比例缩放图像
func (m *MultiScaleTemplateMatching) resizeByRatio(source, search gocv.Mat, ratio float64) (gocv.Mat, gocv.Mat, float64, float64) {
	// 源图像最大尺寸限制
	srcMaxDim := float64(max(source.Rows(), source.Cols()))
	sr := math.Min(float64(m.scaleMax)/srcMaxDim, 1.0)

	scaledSource := gocv.NewMat()
	if sr < 1.0 {
		gocv.Resize(source, &scaledSource, image.Point{
			X: int(float64(source.Cols()) * sr),
			Y: int(float64(source.Rows()) * sr),
		}, 0, 0, gocv.InterpolationLinear)
	} else {
		source.CopyTo(&scaledSource)
	}

	// 计算模板缩放比例
	srcH := float64(scaledSource.Rows())
	srcW := float64(scaledSource.Cols())
	searchH := float64(search.Rows())
	searchW := float64(search.Cols())

	var tr float64
	if searchH/srcH >= searchW/srcW {
		tr = (srcH * ratio) / searchH
	} else {
		tr = (srcW * ratio) / searchW
	}

	// 缩放模板
	newW := max(int(searchW*tr), 1)
	newH := max(int(searchH*tr), 1)

	scaledSearch := gocv.NewMat()
	gocv.Resize(search, &scaledSearch, image.Point{X: newW, Y: newH}, 0, 0, gocv.InterpolationLinear)

	return scaledSource, scaledSearch, tr, sr
}

// orgSize 还原到原始尺寸
func (m *MultiScaleTemplateMatching) orgSize(maxLoc Point, w, h int, tr, sr float64) (Point, int, int) {
	return Point{
		X: int(float64(maxLoc.X) / sr),
		Y: int(float64(maxLoc.Y) / sr),
	}, int(float64(w) / sr), int(float64(h) / sr)
}

// getConfidenceFromCrop 从裁剪区域计算置信度
func (m *MultiScaleTemplateMatching) getConfidenceFromCrop(loc Point, w, h int) float64 {
	// 边界检查
	if loc.X < 0 || loc.Y < 0 || loc.X+w > m.imSource.Cols() || loc.Y+h > m.imSource.Rows() {
		return 0
	}

	// 裁剪区域
	roi := m.imSource.Region(image.Rect(loc.X, loc.Y, loc.X+w, loc.Y+h))
	defer roi.Close()

	// 缩放到模板大小进行比较
	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(roi, &resized, image.Point{X: m.imSearch.Cols(), Y: m.imSearch.Rows()}, 0, 0, gocv.InterpolationLinear)

	if m.rgb {
		return CalRGBConfidence(resized, m.imSearch)
	}
	return CalCcoeffConfidence(resized, m.imSearch)
}

// buildResult 构建匹配结果
func (m *MultiScaleTemplateMatching) buildResult(loc Point, w, h int, confidence float64, startTime time.Time) *MatchResult {
	return &MatchResult{
		Result: Point{
			X: loc.X + w/2,
			Y: loc.Y + h/2,
		},
		Rectangle: Rectangle{
			TopLeft:     loc,
			TopRight:    Point{X: loc.X + w, Y: loc.Y},
			BottomLeft:  Point{X: loc.X, Y: loc.Y + h},
			BottomRight: Point{X: loc.X + w, Y: loc.Y + h},
		},
		Confidence: confidence,
		Time:       float64(time.Since(startTime).Milliseconds()),
	}
}
