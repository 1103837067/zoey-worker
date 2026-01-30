# Vision 模块 - 视觉识别

图像和文字识别的统一入口模块。

## 子模块

| 模块    | 路径             | 功能                             |
| ------- | ---------------- | -------------------------------- |
| **CV**  | `pkg/vision/cv`  | 图像匹配（模板、特征点、多尺度） |
| **OCR** | `pkg/vision/ocr` | 文字识别（PaddleOCR）            |

## 快速使用

```go
import "github.com/zoeyai/zoeyworker/pkg/vision"

// 图像匹配
pos, err := vision.FindImage(screen, "button.png")

// 文字识别
text, err := vision.RecognizeText(screen)

// 查找文字位置
pos, err := vision.FindText(screen, "登录")
```

## 详细文档

- [CV 模块](./cv/README.md) - 图像匹配详细文档
- [OCR 模块](./ocr/README.md) - 文字识别详细文档
