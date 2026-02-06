package grpc

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/zoeyai/zoeyworker/pkg/cmdutil"
)

// Python 检测缓存：启动时检测一次，后续直接使用
var (
	cachedPythonInfo *Capabilities
	pythonDetectOnce sync.Once
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
	Hostname     string       `json:"hostname"`
	Platform     string       `json:"platform"`
	OSVersion    string       `json:"os_version"`
	AgentVersion string       `json:"agent_version"`
	IPAddress    string       `json:"ip_address"`
	Capabilities *Capabilities `json:"capabilities,omitempty"`
}

// Capabilities 环境能力信息
type Capabilities struct {
	PythonAvailable bool   `json:"python_available"`
	PythonVersion   string `json:"python_version,omitempty"`
	PythonPath      string `json:"python_path,omitempty"`
}

// WarmupSystemInfo 预热系统信息检测（启动时调用，异步执行耗时操作）
// 在后台完成 Python 检测等耗时操作，连接时直接使用缓存
func WarmupSystemInfo() {
	go func() {
		pythonDetectOnce.Do(func() {
			cachedPythonInfo = detectPythonEnv()
		})
	}()
}

// GetCachedPythonInfo 获取缓存的 Python 检测结果（不会触发重新检测）
func GetCachedPythonInfo() *Capabilities {
	pythonDetectOnce.Do(func() {
		cachedPythonInfo = detectPythonEnv()
	})
	return cachedPythonInfo
}

// RefreshPythonInfo 强制重新检测 Python 环境并更新缓存
func RefreshPythonInfo() *Capabilities {
	cachedPythonInfo = detectPythonEnv()
	return cachedPythonInfo
}

// GetSystemInfo 获取当前系统信息（使用缓存的 Python 检测结果）
func GetSystemInfo() *SystemInfo {
	hostname, _ := os.Hostname()

	platform := strings.ToUpper(runtime.GOOS)
	if platform == "DARWIN" {
		platform = "MACOS"
	}

	// 使用缓存的 Python 检测结果（如果 Warmup 还没完成，这里会同步等待）
	pythonDetectOnce.Do(func() {
		cachedPythonInfo = detectPythonEnv()
	})

	return &SystemInfo{
		Hostname:     hostname,
		Platform:     platform,
		OSVersion:    runtime.GOOS + "/" + runtime.GOARCH,
		AgentVersion: Version,
		IPAddress:    getLocalIP(),
		Capabilities: cachedPythonInfo,
	}
}

// detectPythonEnv 检测 Python 环境（内部实现，避免循环依赖 auto 包）
func detectPythonEnv() *Capabilities {
	caps := &Capabilities{}
	candidates := []string{"python3", "python"}

	for _, name := range candidates {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}

		// 获取版本号
		cmd := exec.Command(path, "--version")
		cmdutil.HideWindow(cmd) // Windows 上隐藏 cmd 黑色窗口
		output, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}

		line := strings.TrimSpace(string(output))
		parts := strings.SplitN(line, " ", 2)
		version := line
		if len(parts) == 2 {
			version = parts[1]
		}

		// 排除 Python 2.x
		if strings.HasPrefix(version, "2.") {
			continue
		}

		caps.PythonAvailable = true
		caps.PythonVersion = version
		caps.PythonPath = path
		return caps
	}

	return caps
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

// ExecutorStatusCallback 执行器状态回调函数
// 返回: status, currentTaskID, currentTaskType, taskStartedAt, runningCount
type ExecutorStatusCallback func() (string, string, string, int64, int)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// Version 版本号
const Version = "1.0.0"
