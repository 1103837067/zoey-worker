package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	"github.com/zoeyai/zoeyworker/pkg/config"
	"github.com/zoeyai/zoeyworker/pkg/executor"
	"github.com/zoeyai/zoeyworker/pkg/grpc"
)

// App 应用结构体
type App struct {
	ctx        context.Context
	grpcClient *grpc.Client
	configMgr  *config.Manager
	executor   *executor.Executor
}

// NewApp 创建应用实例
func NewApp() *App {
	return &App{
		configMgr: config.GetDefaultManager(),
	}
}

// startup 应用启动时调用
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.grpcClient = grpc.NewClient(nil)
	a.executor = executor.NewExecutor(a.grpcClient)

	// 设置任务回调
	a.grpcClient.SetTaskCallback(func(taskID, taskType, payloadJSON string) {
		go a.executor.Execute(taskID, taskType, payloadJSON)
	})

	// 设置取消任务回调
	a.grpcClient.SetCancelCallback(func(taskID string) bool {
		return a.executor.CancelTask(taskID)
	})
}

// shutdown 应用关闭时调用
func (a *App) shutdown(ctx context.Context) {
	if a.grpcClient != nil && a.grpcClient.IsConnected() {
		a.grpcClient.Disconnect()
	}
}

// ==================== 配置管理 ====================

// ConfigData 配置数据
type ConfigData struct {
	// 连接设置
	ServerURL   string `json:"server_url"`
	AccessKey   string `json:"access_key"`
	SecretKey   string `json:"secret_key"`
	AutoConnect bool   `json:"auto_connect"`

	// 重连设置
	AutoReconnect     bool `json:"auto_reconnect"`
	ReconnectInterval int  `json:"reconnect_interval"`

	// 日志设置
	LogLevel string `json:"log_level"`

	// GUI 设置
	MinimizeToTray bool `json:"minimize_to_tray"`
	StartMinimized bool `json:"start_minimized"`
}

// LoadConfig 加载配置
func (a *App) LoadConfig() ConfigData {
	cfg, err := a.configMgr.Load()
	if err != nil {
		defaults := config.DefaultConnectionConfig()
		return ConfigData{
			ServerURL:         defaults.ServerURL,
			AutoConnect:       defaults.AutoConnect,
			AutoReconnect:     defaults.AutoReconnect,
			ReconnectInterval: defaults.ReconnectInterval,
			LogLevel:          defaults.LogLevel,
			MinimizeToTray:    defaults.MinimizeToTray,
			StartMinimized:    defaults.StartMinimized,
		}
	}
	return ConfigData{
		ServerURL:         cfg.ServerURL,
		AccessKey:         cfg.AccessKey,
		SecretKey:         cfg.SecretKey,
		AutoConnect:       cfg.AutoConnect,
		AutoReconnect:     cfg.AutoReconnect,
		ReconnectInterval: cfg.ReconnectInterval,
		LogLevel:          cfg.LogLevel,
		MinimizeToTray:    cfg.MinimizeToTray,
		StartMinimized:    cfg.StartMinimized,
	}
}

// SaveConfig 保存配置
func (a *App) SaveConfig(data ConfigData) error {
	cfg := &config.ConnectionConfig{
		ServerURL:         data.ServerURL,
		AccessKey:         data.AccessKey,
		SecretKey:         data.SecretKey,
		AutoConnect:       data.AutoConnect,
		AutoReconnect:     data.AutoReconnect,
		ReconnectInterval: data.ReconnectInterval,
		LogLevel:          data.LogLevel,
		MinimizeToTray:    data.MinimizeToTray,
		StartMinimized:    data.StartMinimized,
	}
	return a.configMgr.Save(cfg)
}

// ==================== 连接管理 ====================

// ConnectResult 连接结果
type ConnectResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
}

// Connect 连接服务端
func (a *App) Connect(serverURL, accessKey, secretKey string) ConnectResult {
	if a.grpcClient == nil {
		return ConnectResult{Success: false, Message: "客户端未初始化"}
	}

	err := a.grpcClient.Connect(serverURL, accessKey, secretKey)
	if err != nil {
		return ConnectResult{Success: false, Message: err.Error()}
	}

	status, agentID, agentName := a.grpcClient.GetStatus()
	if status == grpc.StatusConnected {
		// 连接成功后保存配置
		a.SaveConfig(ConfigData{
			ServerURL:   serverURL,
			AccessKey:   accessKey,
			SecretKey:   secretKey,
			AutoConnect: false,
		})
	}

	return ConnectResult{
		Success:   status == grpc.StatusConnected,
		Message:   "",
		AgentID:   agentID,
		AgentName: agentName,
	}
}

// Disconnect 断开连接
func (a *App) Disconnect() error {
	if a.grpcClient == nil {
		return nil
	}
	return a.grpcClient.Disconnect()
}

// StatusInfo 状态信息
type StatusInfo struct {
	Connected bool   `json:"connected"`
	Status    string `json:"status"`
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
}

// GetStatus 获取连接状态
func (a *App) GetStatus() StatusInfo {
	if a.grpcClient == nil {
		return StatusInfo{Connected: false, Status: "disconnected"}
	}

	status, agentID, agentName := a.grpcClient.GetStatus()
	return StatusInfo{
		Connected: a.grpcClient.IsConnected(),
		Status:    string(status),
		AgentID:   agentID,
		AgentName: agentName,
	}
}

// ==================== 日志 ====================

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// GetLogs 获取日志
func (a *App) GetLogs(limit int) []LogEntry {
	if a.grpcClient == nil {
		return []LogEntry{}
	}

	logs := a.grpcClient.GetLogs(limit)
	result := make([]LogEntry, len(logs))
	for i, log := range logs {
		result[i] = LogEntry{
			Timestamp: log.Timestamp,
			Level:     log.Level,
			Message:   log.Message,
		}
	}
	return result
}

// ==================== 系统信息 ====================

// SystemInfo 系统信息
type SystemInfo struct {
	Hostname string `json:"hostname"`
	Platform string `json:"platform"`
	Version  string `json:"version"`
}

// GetSystemInfo 获取系统信息
func (a *App) GetSystemInfo() SystemInfo {
	hostname, _ := os.Hostname()
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macOS"
	} else if platform == "windows" {
		platform = "Windows"
	}

	return SystemInfo{
		Hostname: hostname,
		Platform: platform,
		Version:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// ==================== 权限管理 (macOS) ====================

// PermissionInfo 权限信息
type PermissionInfo struct {
	Accessibility   bool   `json:"accessibility"`
	ScreenRecording bool   `json:"screen_recording"`
	AllGranted      bool   `json:"all_granted"`
	Message         string `json:"message"`
}

// CheckPermissions 检查权限状态
func (a *App) CheckPermissions() PermissionInfo {
	status := auto.CheckPermissions()
	return PermissionInfo{
		Accessibility:   status.Accessibility,
		ScreenRecording: status.ScreenRecording,
		AllGranted:      status.AllGranted,
		Message:         auto.GetPermissionInstructions(status),
	}
}

// RequestAccessibilityPermission 请求辅助功能权限
func (a *App) RequestAccessibilityPermission() bool {
	return auto.RequestAccessibilityPermission()
}

// OpenAccessibilitySettings 打开辅助功能设置
func (a *App) OpenAccessibilitySettings() {
	auto.OpenAccessibilitySettings()
}

// OpenScreenRecordingSettings 打开屏幕录制设置
func (a *App) OpenScreenRecordingSettings() {
	auto.OpenScreenRecordingSettings()
}
