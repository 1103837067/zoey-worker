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

// TargetPos 目标位置枚举，用于指定返回匹配结果的哪个位置
type TargetPos int

const (
	// TargetPosMid 中心点（默认）
	TargetPosMid TargetPos = iota
	// TargetPosTopLeft 左上角
	TargetPosTopLeft
	// TargetPosTopRight 右上角
	TargetPosTopRight
	// TargetPosBottomLeft 左下角
	TargetPosBottomLeft
	// TargetPosBottomRight 右下角
	TargetPosBottomRight
)

// GetPosition 根据 TargetPos 从 MatchResult 获取对应位置
func (t TargetPos) GetPosition(result *MatchResult) Point {
	if result == nil {
		return Point{}
	}
	switch t {
	case TargetPosTopLeft:
		return result.Rectangle.TopLeft
	case TargetPosTopRight:
		return result.Rectangle.TopRight
	case TargetPosBottomLeft:
		return result.Rectangle.BottomLeft
	case TargetPosBottomRight:
		return result.Rectangle.BottomRight
	default:
		return result.Result
	}
}

// ImageInput 支持的图像输入类型
// 可以是文件路径 (string)、image.Image 或 gocv.Mat
type ImageInput interface{}

// MatchMethod 匹配方法枚举
type MatchMethod string

const (
	// MatchMethodTemplate 模板匹配
	MatchMethodTemplate MatchMethod = "tpl"
	// MatchMethodMultiScaleTemplate 多尺度模板匹配
	MatchMethodMultiScaleTemplate MatchMethod = "mstpl"
	// MatchMethodKAZE KAZE 特征点匹配
	MatchMethodKAZE MatchMethod = "kaze"
	// MatchMethodBRISK BRISK 特征点匹配
	MatchMethodBRISK MatchMethod = "brisk"
	// MatchMethodAKAZE AKAZE 特征点匹配
	MatchMethodAKAZE MatchMethod = "akaze"
	// MatchMethodORB ORB 特征点匹配
	MatchMethodORB MatchMethod = "orb"
)

// DefaultMatchMethods 默认匹配方法列表
var DefaultMatchMethods = []MatchMethod{
	MatchMethodTemplate,
	MatchMethodKAZE,
}
