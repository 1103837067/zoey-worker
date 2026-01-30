# Config 模块 - 配置管理

管理 Worker 的连接配置，支持持久化存储。

## 功能

- 加载/保存连接配置
- 配置文件加密存储 (权限 0600)
- 默认配置支持

## 快速使用

```go
import "github.com/zoeyai/zoeyworker/pkg/config"

// 加载配置
cfg, err := config.Load()
if err != nil {
    log.Printf("加载配置失败: %v", err)
}

// 保存配置
cfg := &config.ConnectionConfig{
    ServerURL:   "localhost:50051",
    AccessKey:   "your_access_key",
    SecretKey:   "your_secret_key",
    AutoConnect: true,
}
err := config.Save(cfg)

// 清除配置
config.Clear()
```

## 配置结构

```go
type ConnectionConfig struct {
    ServerURL   string `json:"server_url"`   // 服务端地址
    AccessKey   string `json:"access_key"`   // 访问密钥
    SecretKey   string `json:"secret_key"`   // 秘密密钥
    AutoConnect bool   `json:"auto_connect"` // 自动连接
}
```

## 配置文件位置

默认位置: `~/.zoey-worker/config.json`

## 自定义配置目录

```go
manager := config.NewManagerWithDir("/custom/path")
cfg, _ := manager.Load()
manager.Save(cfg)
```
