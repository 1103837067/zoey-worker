# CV 模块 - 图像匹配

计算机视觉图像匹配模块，用于在屏幕截图中定位目标图像。

## 核心功能

- **模板匹配** - 在大图中查找小图位置
- **多尺度匹配** - 自动适应不同分辨率/DPI
- **特征点匹配** - 处理缩放、旋转等变换

## 匹配算法

| 算法   | 方法名                          | 速度  | 缩放鲁棒性 | 适用场景     |
| ------ | ------------------------------- | ----- | ---------- | ------------ |
| BRISK  | `MatchMethodBRISK`              | 77ms  | 83%        | 默认首选     |
| 多尺度 | `MatchMethodMultiScaleTemplate` | 757ms | 100%       | 跨分辨率/DPI |
| 模板   | `MatchMethodTemplate`           | 10ms  | 仅100%     | 尺寸完全一致 |
| AKAZE  | `MatchMethodAKAZE`              | 120ms | 50%        | 复杂变换     |

## 默认策略

**BRISK → 多尺度** 降级策略：先快速尝试 BRISK，失败则用多尺度兜底。

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
    cv.WithTemplateRGB(true),                // RGB 三通道校验
    cv.WithTemplateMethods(cv.MatchMethodBRISK, cv.MatchMethodMultiScaleTemplate),
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
