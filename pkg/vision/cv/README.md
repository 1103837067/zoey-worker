# CV 模块 - 图像匹配

计算机视觉图像匹配模块，用于在屏幕截图中定位目标图像。

## 核心功能

- **SIFT 特征点匹配** - 处理缩放、旋转等变换
- **多尺度候选** - 通过多倍率模板缩放适配不同分辨率/DPI

## 快速使用

```go
import "github.com/zoeyai/zoeyworker/pkg/vision/cv"

// 方式1: 使用 Template 类
tmpl := cv.NewTemplate("button.png")
pos, err := tmpl.MatchIn(screenMat)  // 返回中心点坐标

// 方式2: 便捷函数
pos, err := cv.FindLocation(screenMat, "button.png")

// 方式3: 循环匹配直到找到或超时
pos, err := cv.MatchLoop(screenshotFn, "button.png", 10*time.Second)
```

## 配置选项

```go
tmpl := cv.NewTemplate("button.png",
    cv.WithTemplateThreshold(0.9),           // 匹配阈值 (默认 0.8)
    cv.WithTemplateScales(0.75, 1.0, 1.25),   // 多尺度候选
)
```

## 返回结果

```go
type MatchResult struct {
    Result     Point      // 匹配中心点 (x, y)
    Rectangle  Rectangle  // 匹配区域四角坐标
    Confidence float64    // 置信度 (0-1)
    Time       float64    // 耗时 (ms)
}
```

## 依赖

- `gocv.io/x/gocv` - OpenCV Go 绑定
