# ZoeyWorker

Go 语言实现的 UI 自动化 Worker，支持跨平台（Windows/macOS）。

## 功能概览

| 模块           | 功能            | 路径              |
| -------------- | --------------- | ----------------- |
| **Auto**       | UI 自动化主模块 | `pkg/auto/`       |
| **Vision/CV**  | 图像匹配        | `pkg/vision/cv/`  |
| **Vision/OCR** | 文字识别        | `pkg/vision/ocr/` |
| **gRPC**       | 服务端通信      | `pkg/grpc/`       |
| **Config**     | 配置管理        | `pkg/config/`     |
| **Executor**   | 任务执行器      | `pkg/executor/`   |

## 安装

```bash
go get github.com/zoeyai/zoeyworker
```

## 命令行工具

```bash
# 编译
go build -o zoeyworker ./cmd/zoeyworker

# 运行
./zoeyworker -server localhost:50051 -access-key KEY -secret-key SECRET

# 保存配置后运行
./zoeyworker -server localhost:50051 -access-key KEY -secret-key SECRET -save
./zoeyworker  # 使用保存的配置

# 帮助
./zoeyworker -help
```

### 依赖

- **OpenCV 4.x** - 图像处理
- **ONNX Runtime** - OCR 推理

macOS:

```bash
brew install opencv
```

Windows:

```bash
# 下载 OpenCV 并设置环境变量
```

## 快速开始

```go
package main

import (
    "github.com/zoeyai/zoeyworker/pkg/auto"
)

func main() {
    // 点击图像
    auto.ClickImage("login_button.png")

    // 输入文字
    auto.TypeText("username")
    auto.KeyTap("tab")
    auto.TypeText("password")

    // 点击文字
    auto.ClickText("登录")
}
```

## 模块文档

- [Auto 模块](./pkg/auto/README.md) - UI 自动化操作
- [Vision 模块](./pkg/vision/README.md) - 图像和文字识别
  - [CV 模块](./pkg/vision/cv/README.md) - 图像匹配
  - [OCR 模块](./pkg/vision/ocr/README.md) - 文字识别
- [gRPC 模块](./pkg/grpc/README.md) - 服务端通信
- [Config 模块](./pkg/config/README.md) - 配置管理
- [Executor 模块](./pkg/executor/README.md) - 任务执行器

## 匹配算法

默认使用 **BRISK → 多尺度** 降级策略：

| 算法   | 速度  | 缩放适应 | 说明       |
| ------ | ----- | -------- | ---------- |
| BRISK  | 77ms  | 83%      | 首选，快速 |
| 多尺度 | 757ms | 100%     | 兜底，稳定 |

## 跨平台支持

| 功能     | Windows | macOS |
| -------- | ------- | ----- |
| 截图     | ✅      | ✅    |
| 鼠标操作 | ✅      | ✅    |
| 键盘操作 | ✅      | ✅    |
| 窗口管理 | ✅      | ✅    |
| 进程管理 | ✅      | ✅    |
| 图像匹配 | ✅      | ✅    |
| OCR      | ✅      | ✅    |

## 许可证

MIT
