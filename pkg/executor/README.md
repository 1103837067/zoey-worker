# Executor 模块 - 任务执行器

解析并执行服务端下发的任务。

## 功能

- 解析任务 payload
- 调用 `pkg/auto` 执行 UI 操作
- 通过 `pkg/grpc` 上报结果

## 支持的任务类型

| 任务类型        | 说明         | 必需参数                      |
| --------------- | ------------ | ----------------------------- |
| `click_image`   | 点击图像     | `image`                       |
| `click_text`    | 点击文字     | `text`                        |
| `type_text`     | 输入文字     | `text`                        |
| `key_press`     | 按键         | `key`, `modifiers?`           |
| `screenshot`    | 截屏         | `save_path?`                  |
| `wait_image`    | 等待图像出现 | `image`                       |
| `wait_text`     | 等待文字出现 | `text`                        |
| `mouse_move`    | 移动鼠标     | `x`, `y`                      |
| `mouse_click`   | 鼠标点击     | `x`, `y`, `double?`, `right?` |
| `activate_app`  | 激活应用     | `app_name`                    |
| `grid_click`    | 网格点击     | `grid`, `region?`             |
| `image_exists`  | 检查图像存在 | `image`                       |
| `text_exists`   | 检查文字存在 | `text`                        |
| `get_clipboard` | 获取剪贴板   | -                             |
| `set_clipboard` | 设置剪贴板   | `text`                        |

## 使用方法

```go
import (
    "github.com/zoeyai/zoeyworker/pkg/executor"
    "github.com/zoeyai/zoeyworker/pkg/grpc"
)

client := grpc.NewClient(nil)
exec := executor.NewExecutor(client)

// 执行任务 (异步)
go exec.Execute(taskID, "click_image", `{"image": "/path/to/template.png"}`)
```

## 任务 Payload 示例

### click_image

```json
{
  "image": "/path/to/template.png",
  "timeout": 10,
  "threshold": 0.8
}
```

### type_text

```json
{
  "text": "Hello World"
}
```

### key_press

```json
{
  "key": "c",
  "modifiers": ["command"]
}
```

### grid_click

```json
{
  "grid": "3.3.2.2",
  "region": { "x": 0, "y": 0, "width": 1920, "height": 1080 }
}
```

## 任务结果

执行完成后自动通过 gRPC 发送结果：

```json
{
  "task_id": "xxx",
  "success": true,
  "status": "SUCCESS",
  "result_json": "{\"clicked\": true}",
  "duration_ms": 1234
}
```
