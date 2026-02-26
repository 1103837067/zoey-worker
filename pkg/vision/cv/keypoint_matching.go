package cv

import (
	"fmt"
	"math"
	"sort"
	"time"

	"gocv.io/x/gocv"
)

const (
	defaultKeypointMinInliers    = 4
	defaultKeypointMinInlierRate = 0.3
	defaultCornerTolRatio        = 0.02
	defaultCornerTolPx           = 8.0
)

// KeypointMatcher 特征点匹配器接口
type KeypointMatcher interface {
	// Detect 检测特征点
	Detect(img gocv.Mat) ([]gocv.KeyPoint, gocv.Mat)
	// Close 释放资源
	Close()
}

// keypointMatchingBase 特征点匹配基类
type keypointMatchingBase struct {
	imSearch   gocv.Mat
	imSource   gocv.Mat
	threshold  float64
	detector   KeypointMatcher
	normType   gocv.NormType
	methodName string
	minInliers int
	minInRate  float64
}

// FindBestResult 查找最佳匹配结果
func (k *keypointMatchingBase) FindBestResult() (*MatchResult, error) {
	startTime := time.Now()

	// 检查图像
	if k.imSearch.Empty() || k.imSource.Empty() {
		return nil, fmt.Errorf("图像为空")
	}

	// 检测特征点
	kpSearch, descSearch := k.detector.Detect(k.imSearch)
	kpSource, descSource := k.detector.Detect(k.imSource)
	defer descSearch.Close()
	defer descSource.Close()

	if len(kpSearch) < 2 || len(kpSource) < 2 {
		return nil, nil
	}

	// 创建匹配器（使用匹配器对应的距离类型）
	matcher := gocv.NewBFMatcherWithParams(k.normType, false)
	defer matcher.Close()

	// 进行 KNN 匹配
	matches := matcher.KnnMatch(descSearch, descSource, 2)

	// 筛选好的匹配点（比率测试）
	goodMatches := filterGoodMatches(matches, 0.75)
	if len(goodMatches) < 4 {
		return nil, nil
	}

	// 计算匹配结果
	result, err := k.computeResult(kpSearch, kpSource, goodMatches)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	result.Time = float64(time.Since(startTime).Milliseconds())

	// 置信度校验
	if result.Confidence < k.threshold {
		return nil, nil
	}

	return result, nil
}

// FindAllResults 查找所有匹配结果（特征点匹配通常只返回一个结果）
func (k *keypointMatchingBase) FindAllResults() ([]*MatchResult, error) {
	result, err := k.FindBestResult()
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return []*MatchResult{result}, nil
}

// computeResult 计算匹配结果
func (k *keypointMatchingBase) computeResult(kpSearch, kpSource []gocv.KeyPoint, matches []gocv.DMatch) (*MatchResult, error) {
	// 提取匹配点坐标
	srcPts := make([]gocv.Point2f, len(matches))
	dstPts := make([]gocv.Point2f, len(matches))

	for i, m := range matches {
		srcPts[i] = gocv.Point2f{
			X: float32(kpSearch[m.QueryIdx].X),
			Y: float32(kpSearch[m.QueryIdx].Y),
		}
		dstPts[i] = gocv.Point2f{
			X: float32(kpSource[m.TrainIdx].X),
			Y: float32(kpSource[m.TrainIdx].Y),
		}
	}

	if len(matches) >= 4 {
		return k.computeWithHomography(srcPts, dstPts, matches)
	}
	if len(matches) == 3 {
		return k.computeWithThreePoints(srcPts, dstPts, matches)
	}
	if len(matches) == 2 {
		return k.computeWithTwoPoints(srcPts, dstPts, matches)
	}
	return nil, nil
}

// computeWithHomography 使用单应性矩阵计算（4+点）
func (k *keypointMatchingBase) computeWithHomography(srcPts, dstPts []gocv.Point2f, matches []gocv.DMatch) (*MatchResult, error) {
	// 创建点向量
	srcMat := gocv.NewMatWithSize(len(srcPts), 1, gocv.MatTypeCV32FC2)
	dstMat := gocv.NewMatWithSize(len(dstPts), 1, gocv.MatTypeCV32FC2)
	defer srcMat.Close()
	defer dstMat.Close()

	for i := range srcPts {
		srcMat.SetFloatAt(i, 0, srcPts[i].X)
		srcMat.SetFloatAt(i, 1, srcPts[i].Y)
		dstMat.SetFloatAt(i, 0, dstPts[i].X)
		dstMat.SetFloatAt(i, 1, dstPts[i].Y)
	}

	// 计算单应性矩阵
	mask := gocv.NewMat()
	defer mask.Close()
	H := gocv.FindHomography(srcMat, dstMat, gocv.HomographyMethodRANSAC, 5.0, &mask, 2000, 0.995)
	defer H.Close()

	if H.Empty() {
		return nil, nil
	}

	inliers, inlierRate := countInliers(mask, len(matches))
	if inliers < k.minInliers || inlierRate < k.minInRate {
		return nil, nil
	}

	// 获取搜索图像的四个角点
	h, w := k.imSearch.Rows(), k.imSearch.Cols()
	corners := []gocv.Point2f{
		{X: 0, Y: 0},
		{X: 0, Y: float32(h)},
		{X: float32(w), Y: float32(h)},
		{X: float32(w), Y: 0},
	}

	// 透视变换
	transformedCorners := perspectiveTransform(corners, H)

	if !validateCorners(transformedCorners, k.imSource.Cols(), k.imSource.Rows()) {
		return nil, nil
	}

	// 计算中心点
	centerX := (transformedCorners[0].X + transformedCorners[2].X) / 2
	centerY := (transformedCorners[0].Y + transformedCorners[2].Y) / 2

	// 计算置信度
	confidence := k.calculateConfidence(matches, mask)

	return &MatchResult{
		Result: Point{X: int(centerX), Y: int(centerY)},
		Rectangle: Rectangle{
			TopLeft:     Point{X: int(transformedCorners[0].X), Y: int(transformedCorners[0].Y)},
			BottomLeft:  Point{X: int(transformedCorners[1].X), Y: int(transformedCorners[1].Y)},
			BottomRight: Point{X: int(transformedCorners[2].X), Y: int(transformedCorners[2].Y)},
			TopRight:    Point{X: int(transformedCorners[3].X), Y: int(transformedCorners[3].Y)},
		},
		Confidence: confidence,
	}, nil
}

// computeWithThreePoints 使用三个点计算
func (k *keypointMatchingBase) computeWithThreePoints(srcPts, dstPts []gocv.Point2f, matches []gocv.DMatch) (*MatchResult, error) {
	// 计算中心点
	centerX := (dstPts[0].X + dstPts[1].X + dstPts[2].X) / 3
	centerY := (dstPts[0].Y + dstPts[1].Y + dstPts[2].Y) / 3

	// 估算矩形区域
	h, w := k.imSearch.Rows(), k.imSearch.Cols()
	halfW, halfH := float32(w)/2, float32(h)/2

	confidence := k.calculateSimpleConfidence(matches)

	result := &MatchResult{
		Result: Point{X: int(centerX), Y: int(centerY)},
		Rectangle: Rectangle{
			TopLeft:     Point{X: int(centerX - halfW), Y: int(centerY - halfH)},
			BottomLeft:  Point{X: int(centerX - halfW), Y: int(centerY + halfH)},
			BottomRight: Point{X: int(centerX + halfW), Y: int(centerY + halfH)},
			TopRight:    Point{X: int(centerX + halfW), Y: int(centerY - halfH)},
		},
		Confidence: confidence,
	}

	if !validateCorners(rectToCorners(result.Rectangle), k.imSource.Cols(), k.imSource.Rows()) {
		return nil, nil
	}

	return result, nil
}

// computeWithTwoPoints 使用两个点计算
func (k *keypointMatchingBase) computeWithTwoPoints(srcPts, dstPts []gocv.Point2f, matches []gocv.DMatch) (*MatchResult, error) {
	// 计算中心点
	centerX := (dstPts[0].X + dstPts[1].X) / 2
	centerY := (dstPts[0].Y + dstPts[1].Y) / 2

	h, w := k.imSearch.Rows(), k.imSearch.Cols()
	halfW, halfH := float32(w)/2, float32(h)/2

	confidence := k.calculateSimpleConfidence(matches)

	result := &MatchResult{
		Result: Point{X: int(centerX), Y: int(centerY)},
		Rectangle: Rectangle{
			TopLeft:     Point{X: int(centerX - halfW), Y: int(centerY - halfH)},
			BottomLeft:  Point{X: int(centerX - halfW), Y: int(centerY + halfH)},
			BottomRight: Point{X: int(centerX + halfW), Y: int(centerY + halfH)},
			TopRight:    Point{X: int(centerX + halfW), Y: int(centerY - halfH)},
		},
		Confidence: confidence,
	}

	if !validateCorners(rectToCorners(result.Rectangle), k.imSource.Cols(), k.imSource.Rows()) {
		return nil, nil
	}

	return result, nil
}

// calculateConfidence 计算置信度
func (k *keypointMatchingBase) calculateConfidence(matches []gocv.DMatch, mask gocv.Mat) float64 {
	if mask.Empty() {
		return k.calculateSimpleConfidence(matches)
	}

	// 统计内点数量
	inliers, _ := countInliers(mask, len(matches))

	// 置信度 = 内点比例，然后做修正 (1 + confidence) / 2
	confidence := float64(inliers) / float64(len(matches))
	return (1 + confidence) / 2
}

// calculateSimpleConfidence 简单置信度计算
func (k *keypointMatchingBase) calculateSimpleConfidence(matches []gocv.DMatch) float64 {
	if len(matches) == 0 {
		return 0
	}

	// 基于匹配点距离计算置信度
	totalDist := float64(0)
	for _, m := range matches {
		totalDist += float64(m.Distance)
	}
	avgDist := totalDist / float64(len(matches))

	// 距离越小置信度越高
	confidence := math.Max(0, 1-avgDist/100)
	return (1 + confidence) / 2
}

// filterGoodMatches 筛选好的匹配点
func filterGoodMatches(matches [][]gocv.DMatch, ratio float64) []gocv.DMatch {
	var good []gocv.DMatch
	for _, m := range matches {
		if len(m) >= 2 && float64(m[0].Distance) < ratio*float64(m[1].Distance) {
			good = append(good, m[0])
		}
	}

	// 按距离排序
	sort.Slice(good, func(i, j int) bool {
		return good[i].Distance < good[j].Distance
	})

	return good
}

func countInliers(mask gocv.Mat, total int) (int, float64) {
	if total == 0 || mask.Empty() {
		return 0, 0
	}
	inliers := 0
	for i := 0; i < mask.Rows(); i++ {
		if mask.GetUCharAt(i, 0) > 0 {
			inliers++
		}
	}
	return inliers, float64(inliers) / float64(total)
}

func validateCorners(corners []gocv.Point2f, width, height int) bool {
	if len(corners) != 4 || width <= 0 || height <= 0 {
		return false
	}

	w := float64(width)
	h := float64(height)
	tolX := math.Max(defaultCornerTolPx, w*defaultCornerTolRatio)
	tolY := math.Max(defaultCornerTolPx, h*defaultCornerTolRatio)

	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

	for _, pt := range corners {
		x := float64(pt.X)
		y := float64(pt.Y)
		if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
			return false
		}
		if x < -tolX || x > (w-1)+tolX || y < -tolY || y > (h-1)+tolY {
			return false
		}
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}

	if maxX-minX < 2 || maxY-minY < 2 {
		return false
	}

	if polygonArea(corners) < 1 {
		return false
	}

	return true
}

func polygonArea(pts []gocv.Point2f) float64 {
	if len(pts) < 3 {
		return 0
	}
	area := 0.0
	for i := 0; i < len(pts); i++ {
		j := (i + 1) % len(pts)
		area += float64(pts[i].X*pts[j].Y - pts[j].X*pts[i].Y)
	}
	if area < 0 {
		area = -area
	}
	return area * 0.5
}

func rectToCorners(rect Rectangle) []gocv.Point2f {
	return []gocv.Point2f{
		{X: float32(rect.TopLeft.X), Y: float32(rect.TopLeft.Y)},
		{X: float32(rect.BottomLeft.X), Y: float32(rect.BottomLeft.Y)},
		{X: float32(rect.BottomRight.X), Y: float32(rect.BottomRight.Y)},
		{X: float32(rect.TopRight.X), Y: float32(rect.TopRight.Y)},
	}
}

// perspectiveTransform 透视变换
func perspectiveTransform(pts []gocv.Point2f, H gocv.Mat) []gocv.Point2f {
	result := make([]gocv.Point2f, len(pts))

	for i, pt := range pts {
		// 齐次坐标
		x := float64(pt.X)
		y := float64(pt.Y)

		// H * [x, y, 1]^T
		h00 := H.GetDoubleAt(0, 0)
		h01 := H.GetDoubleAt(0, 1)
		h02 := H.GetDoubleAt(0, 2)
		h10 := H.GetDoubleAt(1, 0)
		h11 := H.GetDoubleAt(1, 1)
		h12 := H.GetDoubleAt(1, 2)
		h20 := H.GetDoubleAt(2, 0)
		h21 := H.GetDoubleAt(2, 1)
		h22 := H.GetDoubleAt(2, 2)

		w := h20*x + h21*y + h22
		if w != 0 {
			result[i].X = float32((h00*x + h01*y + h02) / w)
			result[i].Y = float32((h10*x + h11*y + h12) / w)
		}
	}

	return result
}

// SIFTMatching SIFT 特征点匹配
type SIFTMatching struct {
	*keypointMatchingBase
	sift gocv.SIFT
}

// NewSIFTMatching 创建 SIFT 匹配器
func NewSIFTMatching(search, source gocv.Mat, threshold float64) *SIFTMatching {
	sift := gocv.NewSIFT()
	m := &SIFTMatching{
		keypointMatchingBase: &keypointMatchingBase{
			imSearch:   search,
			imSource:   source,
			threshold:  threshold,
			normType:   gocv.NormL2,
			methodName: "SIFT",
			minInliers: defaultKeypointMinInliers,
			minInRate:  defaultKeypointMinInlierRate,
		},
		sift: sift,
	}
	m.detector = m
	return m
}

// Detect 检测特征点
func (s *SIFTMatching) Detect(img gocv.Mat) ([]gocv.KeyPoint, gocv.Mat) {
	return s.sift.DetectAndCompute(img, gocv.NewMat())
}

// Close 释放资源
func (s *SIFTMatching) Close() {
	s.sift.Close()
}
