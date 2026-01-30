# gRPC 模块 - 服务端通信

gRPC 客户端模块，用于 Worker 与服务端的通信。

## 核心功能

- **连接认证** - 使用 AccessKey/SecretKey 连接服务端
- **心跳保活** - 定期发送心跳，检测连接状态
- **自动重连** - 连接断开时自动重连（指数退避）
- **双向流** - TaskStream 双向流，接收任务、发送结果
- **数据请求** - 处理服务端的数据查询请求

## 快速使用

```go
import "github.com/zoeyai/zoeyworker/pkg/grpc"

// 创建客户端
client := grpc.NewClient(nil)

// 设置回调
client.SetStatusCallback(func(status grpc.ClientStatus) {
    fmt.Printf("状态变更: %s\n", status)
})

client.SetTaskCallback(func(taskID, taskType, payload string) {
    fmt.Printf("收到任务: %s\n", taskID)
})

// 连接服务端
err := client.Connect("localhost:50051", "access_key", "secret_key")
if err != nil {
    log.Fatal(err)
}

// 断开连接
defer client.Disconnect()
```

## 配置选项

```go
config := &grpc.ClientConfig{
    ServerURL:            "localhost:50051",
    AccessKey:            "your_access_key",
    SecretKey:            "your_secret_key",
    HeartbeatInterval:    5,              // 心跳间隔（秒）
    MaxHeartbeatFailures: 3,              // 最大心跳失败次数
    ReconnectDelays:      []int{2, 5, 10, 30, 60}, // 重连延迟序列
}

client := grpc.NewClient(config)
```

## 状态

| 状态           | 说明   |
| -------------- | ------ |
| `disconnected` | 未连接 |
| `connecting`   | 连接中 |
| `connected`    | 已连接 |
| `reconnecting` | 重连中 |

## 数据请求

支持处理服务端发来的数据查询请求：

| 请求类型           | 功能         | 调用模块              |
| ------------------ | ------------ | --------------------- |
| `GET_APPLICATIONS` | 获取进程列表 | `auto.GetProcesses()` |
| `GET_WINDOWS`      | 获取窗口列表 | `auto.GetWindows()`   |
| `GET_ELEMENTS`     | 获取 UI 元素 | 暂不支持              |

## 任务消息

```go
// 发送任务确认
client.taskStream.SendTaskAck(taskID, true, "已接收")

// 发送任务进度
client.taskStream.SendTaskProgress(taskID, 10, 5, 4, 1, "步骤5", "RUNNING")

// 发送任务结果
client.taskStream.SendTaskResult(taskID, true, "SUCCESS", "", resultJSON, 1234)
```

## Protobuf

基于 `packages/proto/src/agent.proto` 生成的 Go 代码位于 `pb/` 目录。

生成命令：

```bash
protoc --go_out=pkg/grpc/pb --go_opt=paths=source_relative \
       --go-grpc_out=pkg/grpc/pb --go-grpc_opt=paths=source_relative \
       -I packages/proto/src packages/proto/src/agent.proto
```

## 依赖

- `google.golang.org/grpc` - gRPC 客户端
- `google.golang.org/protobuf` - Protobuf 序列化
- `pkg/auto` - 进程/窗口信息获取
