// Package auto 提供 UI 自动化功能的共享类型和工具函数。
// 具体功能分布在子包中：screen, input, window, image, text, grid。
package auto

import (
	"math"
	"time"

	"github.com/go-vgo/robotgo"
)

// Sleep 休眠
func Sleep(d time.Duration) {
	time.Sleep(d)
}

// MilliSleep 毫秒休眠
func MilliSleep(ms int) {
	robotgo.MilliSleep(ms)
}

// ScaleCoord 按比例缩放坐标值
func ScaleCoord(value int, scale float64) int {
	if scale <= 0 {
		return value
	}
	return int(math.Round(float64(value) / scale))
}

// MinInt 返回最小值
func MinInt(values ...int) int {
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// MaxInt 返回最大值
func MaxInt(values ...int) int {
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}
