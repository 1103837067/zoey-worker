# Auto 模块 - UI 自动化

高级 UI 自动化模块，整合图像识别、OCR、鼠标键盘操作。

## 包结构

```
pkg/auto/              # 共享类型（Options, Point, Region）+ 坐标缩放
  screen/              # 截图、编码、匹配辅助
  input/               # 鼠标、键盘、剪贴板
  image/               # 图像匹配（模板、SIFT）
  text/                # OCR 文字识别与匹配
  grid/                # 网格计算与网格点击
  window/              # 窗口管理（获取、激活、截图）

pkg/process/           # 进程管理（独立包）
pkg/permissions/       # 系统权限检查（独立包）
pkg/python/            # Python 环境检测（独立包）
```

## 快速使用

```go
import (
    "github.com/zoeyai/zoeyworker/pkg/auto"
    "github.com/zoeyai/zoeyworker/pkg/auto/image"
    "github.com/zoeyai/zoeyworker/pkg/auto/input"
    "github.com/zoeyai/zoeyworker/pkg/auto/screen"
    "github.com/zoeyai/zoeyworker/pkg/auto/text"
    "github.com/zoeyai/zoeyworker/pkg/auto/window"
)

// 点击图像
image.ClickImage("login_button.png")

// 等待图像出现并点击
image.ClickImage("submit.png", auto.WithTimeout(10*time.Second))

// 点击文字
text.ClickText("确定")

// 组合操作
window.ActivateWindow("Chrome")
image.ClickImage("search_box.png")
input.TypeText("hello world")
input.KeyTap("enter")
```

## 配置选项

```go
image.ClickImage("button.png",
    auto.WithTimeout(10*time.Second),    // 超时时间
    auto.WithThreshold(0.9),             // 匹配阈值
    auto.WithClickOffset(10, 5),         // 点击偏移
    auto.WithDoubleClick(),              // 双击
    auto.WithRightClick(),               // 右键
    auto.WithRegion(0, 0, 800, 600),     // 搜索区域
)
```

## 网格点击

```go
import "github.com/zoeyai/zoeyworker/pkg/auto/grid"

// 格式: rows.cols.row.col
rect := auto.Region{X: 100, Y: 100, Width: 200, Height: 200}
grid.ClickGrid(rect, "2.2.1.1")  // 点击左上角

// 在窗口内按网格点击
window.ClickGridInWindow(pid, "3.3.2.2")  // 点击中心
```

## 窗口操作

```go
import "github.com/zoeyai/zoeyworker/pkg/auto/window"

// 获取窗口列表
windows, _ := window.GetWindows()
windows, _ := window.GetWindows("Chrome")  // 按名称过滤

// 激活窗口
window.ActivateWindow("Chrome")

// 等待窗口出现
w, _ := window.WaitForWindow("登录", auto.WithTimeout(10*time.Second))
```

## 进程操作

```go
import "github.com/zoeyai/zoeyworker/pkg/process"

processes, _ := process.GetProcesses()
processes, _ := process.FindProcess("chrome")
process.KillProcess(pid)
```

## 依赖

- `github.com/go-vgo/robotgo` - 跨平台桌面自动化
- `github.com/shirou/gopsutil/v4` - 系统/进程信息
- `pkg/vision/cv` - 图像匹配
- `pkg/vision/ocr` - 文字识别
