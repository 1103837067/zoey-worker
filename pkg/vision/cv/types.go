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

// MatchMethod 匹配方法枚举
// 仅保留 SIFT 算法
type MatchMethod string

const (
	MatchMethodSIFT MatchMethod = "sift" // SIFT 特征点匹配（更稳但更慢）
)
