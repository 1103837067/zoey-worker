// Package cv 提供图像匹配功能
package cv

// Point 表示二维坐标点
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Rectangle 表示矩形区域（四个角点）
type Rectangle struct {
	TopLeft     Point `json:"top_left"`
	BottomLeft  Point `json:"bottom_left"`
	BottomRight Point `json:"bottom_right"`
	TopRight    Point `json:"top_right"`
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

// TargetPos 目标位置枚举
type TargetPos int

const (
	TargetPosMid TargetPos = iota
	TargetPosTopLeft
	TargetPosTopRight
	TargetPosBottomLeft
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

// MatchMethod 匹配方法枚举
type MatchMethod string

const (
	MatchMethodTemplate           MatchMethod = "tpl"   // 模板匹配（最快，要求尺寸一致）
	MatchMethodMultiScaleTemplate MatchMethod = "mstpl" // 多尺度模板匹配（适应不同分辨率/DPI）
	MatchMethodKAZE               MatchMethod = "kaze"  // KAZE 特征点匹配
	MatchMethodBRISK              MatchMethod = "brisk" // BRISK 特征点匹配
	MatchMethodAKAZE              MatchMethod = "akaze" // AKAZE 特征点匹配
	MatchMethodORB                MatchMethod = "orb"   // ORB 特征点匹配（效果较差）
)

// Matcher 匹配器接口
type Matcher interface {
	// FindBestResult 查找最佳匹配结果
	FindBestResult() (*MatchResult, error)
	// FindAllResults 查找所有匹配结果
	FindAllResults() ([]*MatchResult, error)
}

// MatcherFactory 匹配器工厂函数类型
type MatcherFactory func(search, source ImageMat, opts MatcherOptions) Matcher

// MatcherOptions 匹配器选项
type MatcherOptions struct {
	Threshold  float64
	RGB        bool
	RecordPos  *Point
	Resolution [2]int
	ScaleMax   int
	ScaleStep  float64
}

// DefaultMatcherOptions 默认匹配器选项
func DefaultMatcherOptions() MatcherOptions {
	return MatcherOptions{
		Threshold: 0.8,
		RGB:       false,
		ScaleMax:  800,
		ScaleStep: 0.005,
	}
}

// ImageMat 图像矩阵接口
// 抽象 gocv.Mat，便于测试和扩展
type ImageMat interface {
	// Rows 返回图像高度
	Rows() int
	// Cols 返回图像宽度
	Cols() int
	// Empty 检查是否为空
	Empty() bool
	// Close 释放资源
	Close() error
}
