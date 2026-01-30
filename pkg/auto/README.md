# Auto 模块 - UI 自动化

高级 UI 自动化模块，整合图像识别、OCR、鼠标键盘操作。

## 核心功能

| 分类         | 功能                               |
| ------------ | ---------------------------------- |
| **截图**     | 全屏截图、区域截图、窗口截图       |
| **图像操作** | 点击图像、等待图像、检查图像存在   |
| **文字操作** | 点击文字、等待文字、检查文字存在   |
| **鼠标**     | 移动、点击、双击、右键、拖拽、滚动 |
| **键盘**     | 输入文字、按键、组合键             |
| **窗口**     | 激活、最小化、最大化、获取列表     |
| **进程**     | 获取列表、查找、终止               |
| **网格点击** | 将区域分割成网格并点击指定格子     |

## 快速使用

```go
import "github.com/zoeyai/zoeyworker/pkg/auto"

// 点击图像
auto.ClickImage("login_button.png")

// 等待图像出现并点击
auto.ClickImage("submit.png", auto.WithTimeout(10*time.Second))

// 点击文字
auto.ClickText("确定")

// 组合操作
auto.ActivateWindow("Chrome")
auto.ClickImage("search_box.png")
auto.TypeText("hello world")
auto.KeyTap("enter")
```

## 配置选项

```go
auto.ClickImage("button.png",
    auto.WithTimeout(10*time.Second),    // 超时时间
    auto.WithThreshold(0.9),             // 匹配阈值
    auto.WithClickOffset(10, 5),         // 点击偏移
    auto.WithDoubleClick(),              // 双击
    auto.WithRightClick(),               // 右键
    auto.WithRegion(0, 0, 800, 600),     // 搜索区域
    auto.WithMultiScale(),               // 强制使用多尺度匹配
)
```

## 网格点击

将矩形区域分割成网格，点击指定格子：

```go
// 格式: rows.cols.row.col
// 例如 "2.2.1.1" 表示 2x2 网格的第1行第1列

rect := auto.Region{X: 100, Y: 100, Width: 200, Height: 200}
auto.ClickGrid(rect, "2.2.1.1")  // 点击左上角

// 在窗口内按网格点击
auto.ClickGridInWindow(pid, "3.3.2.2")  // 点击中心
```

## 窗口操作

```go
// 获取窗口列表
windows, _ := auto.GetWindows()
windows, _ := auto.GetWindows("Chrome")  // 按名称过滤

// 激活窗口
auto.ActivateWindow("Chrome")
auto.BringWindowToFront(pid)

// 窗口操作
auto.MinimizeWindow(pid)
auto.MaximizeWindow(pid)
auto.CloseWindowByPID(pid)

// 等待窗口出现
window, _ := auto.WaitForWindow("登录", auto.WithTimeout(10*time.Second))
```

## 进程操作

```go
// 获取所有进程
processes, _ := auto.GetProcesses()

// 按名称查找
processes, _ := auto.FindProcess("chrome")

// 检查进程状态
running := auto.IsProcessRunning(pid)

// 终止进程
auto.KillProcess(pid)
```

## 依赖

- `github.com/go-vgo/robotgo` - 跨平台桌面自动化
- `github.com/shirou/gopsutil/v4` - 系统/进程信息
- `pkg/vision/cv` - 图像匹配
- `pkg/vision/ocr` - 文字识别
