# OCR 模块 - 文字识别

基于 PaddleOCR + ONNX 的文字识别模块，支持中英文混合识别。

## 核心功能

- **文字检测** - 定位图像中的文字区域
- **文字识别** - 识别文字内容
- **文字查找** - 在图像中查找指定文字的位置

## 快速使用

```go
import "github.com/zoeyai/zoeyworker/pkg/vision/ocr"

// 获取全局识别器（自动初始化）
recognizer, err := ocr.GetGlobalRecognizer()

// 识别所有文字
results, err := recognizer.RecognizeAll(img)
for _, r := range results {
    fmt.Printf("文字: %s, 位置: (%d,%d), 置信度: %.2f\n",
        r.Text, r.X, r.Y, r.Confidence)
}

// 查找指定文字
result, err := recognizer.FindText(img, "登录")
if result != nil {
    fmt.Printf("找到 '登录' 在 (%d, %d)\n", result.X, result.Y)
}
```

## 配置选项

```go
config := ocr.Config{
    ModelDir:  "/path/to/models",  // 模型目录
    UseGPU:    false,              // 是否使用 GPU
    NumThread: 4,                  // CPU 线程数
}
recognizer, err := ocr.NewTextRecognizer(config)
```

## 返回结果

```go
type TextResult struct {
    Text       string  // 识别的文字
    X, Y       int     // 文字中心坐标
    Confidence float64 // 置信度 (0-1)
    Box        Box     // 文字边界框
}
```

## 模型

使用 PP-OCRv4 模型，支持：

- 中文简体/繁体
- 英文
- 数字和符号

## 依赖

- `github.com/nicoxiang/go-ocr` - PaddleOCR ONNX 推理
