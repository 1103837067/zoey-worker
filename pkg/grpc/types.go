package grpc

import (
	"os"
	"runtime"
	"strings"
)

// ClientStatus 客户端状态
type ClientStatus string

const (
	StatusDisconnected ClientStatus = "disconnected"
	StatusConnecting   ClientStatus = "connecting"
	StatusConnected    ClientStatus = "connected"
	StatusReconnecting ClientStatus = "reconnecting"
)

// SystemInfo 系统信息
type SystemInfo struct {
	Hostname     string `json:"hostname"`
	Platform     string `json:"platform"`
	OSVersion    string `json:"os_version"`
	AgentVersion string `json:"agent_version"`
	IPAddress    string `json:"ip_address"`
}

// GetSystemInfo 获取当前系统信息
func GetSystemInfo() *SystemInfo {
	hostname, _ := os.Hostname()

	platform := strings.ToUpper(runtime.GOOS)
	if platform == "DARWIN" {
		platform = "MACOS"
	}

	return &SystemInfo{
		Hostname:     hostname,
		Platform:     platform,
		OSVersion:    runtime.GOOS + "/" + runtime.GOARCH,
		AgentVersion: Version,
		IPAddress:    getLocalIP(),
	}
}

// getLocalIP 获取本地 IP 地址
func getLocalIP() string {
	// 简化实现，返回默认值
	// 实际生产环境应该遍历网络接口获取
	return "127.0.0.1"
}

// ClientConfig 客户端配置
type ClientConfig struct {
	// ServerURL 服务端地址 (host:port)
	ServerURL string
	// AccessKey 访问密钥
	AccessKey string
	// SecretKey 秘密密钥
	SecretKey string
	// HeartbeatInterval 心跳间隔（秒）
	HeartbeatInterval int
	// MaxHeartbeatFailures 最大心跳失败次数
	MaxHeartbeatFailures int
	// ReconnectDelays 重连延迟序列（秒）
	ReconnectDelays []int
}

// DefaultConfig 默认配置
func DefaultConfig() *ClientConfig {
	return &ClientConfig{
		HeartbeatInterval:    5,
		MaxHeartbeatFailures: 3,
		ReconnectDelays:      []int{2, 5, 10, 30, 60},
	}
}

// StatusCallback 状态变更回调函数
type StatusCallback func(status ClientStatus)

// TaskCallback 任务回调函数
type TaskCallback func(taskID, taskType, payloadJSON string)

// CancelCallback 取消任务回调函数
type CancelCallback func(taskID string) bool

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// Version 版本号
const Version = "1.0.0"
