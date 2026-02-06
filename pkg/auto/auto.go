package auto

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"time"

	"github.com/go-vgo/robotgo"
	"gocv.io/x/gocv"

	"github.com/zoeyai/zoeyworker/pkg/vision/cv"
)

// ==================== 截图操作 ====================

// CaptureScreen 截取屏幕
func CaptureScreen() (image.Image, error) {
	img, err := robotgo.CaptureImg()
	if err != nil {
		return nil, fmt.Errorf("截屏失败: %w", err)
	}
	return img, nil
}

// CaptureRegion 截取屏幕区域
func CaptureRegion(x, y, width, height int) (image.Image, error) {
	img, err := robotgo.CaptureImg(x, y, width, height)
	if err != nil {
		return nil, fmt.Errorf("截取区域失败: %w", err)
	}
	return img, nil
}

// GetScreenSize 获取屏幕尺寸
func GetScreenSize() (width, height int) {
	return robotgo.GetScreenSize()
}

// GetDisplayCount 获取显示器数量
func GetDisplayCount() int {
	return robotgo.DisplaysNum()
}

// ImageToBase64 将图像转换为 Base64 字符串
// format: "png" 或 "jpeg"，默认 "jpeg"（更小的体积）
// quality: JPEG 质量 1-100，默认 80
func ImageToBase64(img image.Image, format string, quality int) (string, error) {
	if img == nil {
		return "", fmt.Errorf("图像为空")
	}

	var buf bytes.Buffer
	var mimeType string

	if format == "" {
		format = "jpeg" // 默认使用 JPEG 以减小体积
	}
	if quality <= 0 || quality > 100 {
		quality = 80 // 默认质量
	}

	switch format {
	case "png":
		err := png.Encode(&buf, img)
		if err != nil {
			return "", fmt.Errorf("PNG 编码失败: %w", err)
		}
		mimeType = "image/png"
	case "jpeg", "jpg":
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		if err != nil {
			return "", fmt.Errorf("JPEG 编码失败: %w", err)
		}
		mimeType = "image/jpeg"
	default:
		return "", fmt.Errorf("不支持的图像格式: %s", format)
	}

	// 编码为 Base64
	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())

	// 返回带 Data URI 前缀的字符串
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str), nil
}

// CaptureScreenToBase64 截取屏幕并转换为 Base64
// 返回 JPEG 格式的 Base64 字符串（更小的体积）
func CaptureScreenToBase64(quality int) (string, error) {
	img, err := CaptureScreen()
	if err != nil {
		return "", err
	}
	return ImageToBase64(img, "jpeg", quality)
}

// ==================== 内部辅助函数 ====================
// 以下函数被 auto_image.go 和 auto_text.go 共用

// captureForMatch 截图用于匹配
func captureForMatch(o *Options) (gocv.Mat, captureMeta, error) {
	var img image.Image
	var err error

	if o.Region != nil {
		img, err = robotgo.CaptureImg(o.Region.X, o.Region.Y, o.Region.Width, o.Region.Height)
	} else {
		img, err = robotgo.CaptureImg()
	}

	if err != nil {
		return gocv.Mat{}, captureMeta{}, fmt.Errorf("截屏失败: %w", err)
	}

	mat, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return gocv.Mat{}, captureMeta{}, fmt.Errorf("转换图像失败: %w", err)
	}

	meta := buildCaptureMeta(o, img)
	return mat, meta, nil
}

type captureMeta struct {
	scaleX  float64
	scaleY  float64
	offsetX int
	offsetY int
}

func buildCaptureMeta(o *Options, img image.Image) captureMeta {
	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()

	expectedW, expectedH := robotgo.GetScreenSize()
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

	return captureMeta{
		scaleX:  scaleX,
		scaleY:  scaleY,
		offsetX: offsetX,
		offsetY: offsetY,
	}
}

func adjustMatchResult(result *cv.MatchResult, meta captureMeta) *cv.MatchResult {
	if result == nil {
		return nil
	}

	adjusted := *result
	adjusted.Result = adjustCVPoint(result.Result, meta)
	adjusted.Rectangle = cv.Rectangle{
		TopLeft:     adjustCVPoint(result.Rectangle.TopLeft, meta),
		BottomLeft:  adjustCVPoint(result.Rectangle.BottomLeft, meta),
		BottomRight: adjustCVPoint(result.Rectangle.BottomRight, meta),
		TopRight:    adjustCVPoint(result.Rectangle.TopRight, meta),
	}

	return &adjusted
}

func adjustPoint(p Point, meta captureMeta) Point {
	return Point{
		X: scaleCoord(p.X, meta.scaleX) + meta.offsetX,
		Y: scaleCoord(p.Y, meta.scaleY) + meta.offsetY,
	}
}

func adjustCVPoint(p cv.Point, meta captureMeta) cv.Point {
	return cv.Point{
		X: scaleCoord(p.X, meta.scaleX) + meta.offsetX,
		Y: scaleCoord(p.Y, meta.scaleY) + meta.offsetY,
	}
}

func scaleCoord(value int, scale float64) int {
	if scale <= 0 {
		return value
	}
	return int(math.Round(float64(value) / scale))
}

func minInt(values ...int) int {
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func maxInt(values ...int) int {
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// clickAt 在指定位置点击
func clickAt(x, y int, o *Options) error {
	robotgo.Move(x, y)
	time.Sleep(50 * time.Millisecond) // 短暂延迟确保鼠标到位

	if o.RightClick {
		robotgo.Click("right", false)
	} else if o.DoubleClick {
		robotgo.Click("left", true)
	} else {
		robotgo.Click("left", false)
	}

	return nil
}

// Sleep 休眠
func Sleep(d time.Duration) {
	time.Sleep(d)
}

// MilliSleep 毫秒休眠
func MilliSleep(ms int) {
	robotgo.MilliSleep(ms)
}
