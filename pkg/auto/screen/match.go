package screen

import (
	"fmt"
	"image"

	"github.com/go-vgo/robotgo"
	"gocv.io/x/gocv"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	"github.com/zoeyai/zoeyworker/pkg/vision/cv"
)

// CaptureMeta 截图元信息（缩放和偏移量）
type CaptureMeta struct {
	ScaleX  float64
	ScaleY  float64
	OffsetX int
	OffsetY int
}

// CaptureForMatch 截图用于匹配，返回 gocv.Mat 和元信息
func CaptureForMatch(o *auto.Options) (gocv.Mat, CaptureMeta, error) {
	var img image.Image
	var err error

	if o.Region != nil {
		inputX, inputY, inputW, inputH := auto.NormalizeRegionForInput(o.Region.X, o.Region.Y, o.Region.Width, o.Region.Height)
		img, err = robotgo.CaptureImg(inputX, inputY, inputW, inputH)
	} else {
		img, err = robotgo.CaptureImg()
	}

	if err != nil {
		return gocv.Mat{}, CaptureMeta{}, fmt.Errorf("截屏失败: %w", err)
	}

	mat, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return gocv.Mat{}, CaptureMeta{}, fmt.Errorf("转换图像失败: %w", err)
	}

	meta := BuildCaptureMeta(o, img)
	return mat, meta, nil
}

// BuildCaptureMeta 构建截图元信息
func BuildCaptureMeta(o *auto.Options, img image.Image) CaptureMeta {
	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()

	expectedW, expectedH := GetScreenSize()
	offsetX, offsetY := 0, 0
	if o.Region != nil {
		expectedW = o.Region.Width
		expectedH = o.Region.Height
		offsetX = o.Region.X
		offsetY = o.Region.Y
	}

	scaleX := 1.0
	if expectedW > 0 && imgW > 0 {
		scaleX = float64(imgW) / float64(expectedW)
	}
	scaleY := 1.0
	if expectedH > 0 && imgH > 0 {
		scaleY = float64(imgH) / float64(expectedH)
	}

	return CaptureMeta{
		ScaleX:  scaleX,
		ScaleY:  scaleY,
		OffsetX: offsetX,
		OffsetY: offsetY,
	}
}

// AdjustMatchResult 调整匹配结果坐标（反向缩放 + 偏移）
func AdjustMatchResult(result *cv.MatchResult, meta CaptureMeta) *cv.MatchResult {
	if result == nil {
		return nil
	}

	adjusted := *result
	adjusted.Result = AdjustCVPoint(result.Result, meta)
	adjusted.Rectangle = cv.Rectangle{
		TopLeft:     AdjustCVPoint(result.Rectangle.TopLeft, meta),
		BottomLeft:  AdjustCVPoint(result.Rectangle.BottomLeft, meta),
		BottomRight: AdjustCVPoint(result.Rectangle.BottomRight, meta),
		TopRight:    AdjustCVPoint(result.Rectangle.TopRight, meta),
	}

	return &adjusted
}

// AdjustPoint 调整点坐标
func AdjustPoint(p auto.Point, meta CaptureMeta) auto.Point {
	return auto.Point{
		X: auto.ScaleCoord(p.X, meta.ScaleX) + meta.OffsetX,
		Y: auto.ScaleCoord(p.Y, meta.ScaleY) + meta.OffsetY,
	}
}

// AdjustCVPoint 调整 cv.Point 坐标
func AdjustCVPoint(p cv.Point, meta CaptureMeta) cv.Point {
	return cv.Point{
		X: auto.ScaleCoord(p.X, meta.ScaleX) + meta.OffsetX,
		Y: auto.ScaleCoord(p.Y, meta.ScaleY) + meta.OffsetY,
	}
}
