// Package vision 提供图像识别与 OCR 功能
package vision

import (
	"image"
)

// Version 版本号
const Version = "1.0.0"

// Point 表示二维坐标点
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// NewPoint 创建新的 Point
func NewPoint(x, y int) Point {
	return Point{X: x, Y: y}
}

// Rectangle 表示矩形区域（四个角点）
type Rectangle struct {
	TopLeft     Point `json:"top_left"`
	BottomLeft  Point `json:"bottom_left"`
	BottomRight Point `json:"bottom_right"`
	TopRight    Point `json:"top_right"`
}

// NewRectangle 从左上角坐标和宽高创建矩形
func NewRectangle(x, y, w, h int) Rectangle {
	return Rectangle{
		TopLeft:     Point{X: x, Y: y},
		BottomLeft:  Point{X: x, Y: y + h},
		BottomRight: Point{X: x + w, Y: y + h},
		TopRight:    Point{X: x + w, Y: y},
	}
}

// Center 返回矩形中心点
func (r Rectangle) Center() Point {
	return Point{
		X: (r.TopLeft.X + r.BottomRight.X) / 2,
		Y: (r.TopLeft.Y + r.BottomRight.Y) / 2,
	}
}

// Width 返回矩形宽度
func (r Rectangle) Width() int {
	return r.TopRight.X - r.TopLeft.X
}

// Height 返回矩形高度
func (r Rectangle) Height() int {
	return r.BottomLeft.Y - r.TopLeft.Y
}

// ToImageRect 转换为 image.Rectangle
func (r Rectangle) ToImageRect() image.Rectangle {
	return image.Rect(r.TopLeft.X, r.TopLeft.Y, r.BottomRight.X, r.BottomRight.Y)
}

// MatchResult 图像匹配结果
type MatchResult struct {
	// Result 匹配到的中心点坐标
	Result Point `json:"result"`
	// Rectangle 匹配区域的四个角点
	Rectangle Rectangle `json:"rectangle"`
	// Confidence 匹配置信度 (0-1)
	Confidence float64 `json:"confidence"`
	// Time 匹配耗时（毫秒）
	Time float64 `json:"time,omitempty"`
}

// OcrResult OCR 识别结果
type OcrResult struct {
	// Text 识别的文字内容
	Text string `json:"text"`
	// Confidence 识别置信度 (0-1)
	Confidence float64 `json:"confidence"`
	// Position 文字中心位置
	Position Point `json:"position"`
	// Box 文字边界框四个角点
	Box []Point `json:"box,omitempty"`
}

// ImageInput 支持的图像输入类型
// 可以是文件路径 (string)、image.Image 或 gocv.Mat
type ImageInput interface{}

// MatchMethod 匹配方法枚举
// 仅保留 SIFT 算法
type MatchMethod string

const (
	// MatchMethodSIFT SIFT 特征点匹配
	MatchMethodSIFT MatchMethod = "sift"
)

// DefaultMatchMethods 默认匹配方法列表
var DefaultMatchMethods = []MatchMethod{
	MatchMethodSIFT,
}
