package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/zoeyai/zoeyworker/pkg/auto/text"
	"github.com/zoeyai/zoeyworker/pkg/config"
	"github.com/zoeyai/zoeyworker/pkg/executor"
	"github.com/zoeyai/zoeyworker/pkg/grpc"
	"github.com/zoeyai/zoeyworker/pkg/permissions"
	"github.com/zoeyai/zoeyworker/pkg/plugin"
)

// App 应用结构体（作为 Wails v3 Service）
type App struct {
	ctx                      context.Context
	grpcClient               *grpc.Client
	configMgr                *config.Manager
	executor                 *executor.Executor
	hasShownTrayNotification bool // 是否已显示过托盘通知
}

// NewApp 创建应用实例
func NewApp() *App {
	return &App{
		configMgr: config.GetDefaultManager(),
	}
}

// ServiceStartup Wails v3 服务启动时调用
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	a.ctx = ctx
	a.grpcClient = grpc.NewClient(nil)
	a.executor = executor.NewExecutor(a.grpcClient)

	// 预热系统信息（异步检测 Python 环境等耗时操作）
	grpc.WarmupSystemInfo()

	// 设置 OCR 插件
	text.SetOCRPlugin(plugin.GetOCRPlugin())

	// 调试数据通过轮询 GetDebugData 方法获取，不再使用事件

	// 设置 executor 日志函数，将日志路由到 grpcClient
	executor.SetLogFunc(func(level, message string) {
		if a.grpcClient != nil {
			// 通过 grpcClient 的日志系统输出，这样 GUI 可以看到
			a.grpcClient.Log(level, message)
		} else {
			fmt.Printf("[%s] %s\n", level, message)
		}
	})

	// 设置任务回调
	a.grpcClient.SetTaskCallback(func(taskID, taskType, payloadJSON string) {
		go a.executor.Execute(taskID, taskType, payloadJSON)
	})

	// 设置取消任务回调
	a.grpcClient.SetCancelCallback(func(taskID string) bool {
		return a.executor.CancelTask(taskID)
	})

	// 设置执行器状态回调（用于心跳上报）
	a.grpcClient.SetExecutorStatusCallback(func() (string, string, string, int64, int) {
		return a.executor.GetStatus()
	})

	return nil
}

// ServiceShutdown Wails v3 服务关闭时调用
func (a *App) ServiceShutdown() error {
	if a.grpcClient != nil && a.grpcClient.IsConnected() {
		a.grpcClient.Disconnect()
	}
	return nil
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
	ReconnectInterval int  `json:"reconnect_interval"` // 秒
	// 日志设置
	LogLevel string `json:"log_level"`
	// 界面设置
	MinimizeToTray bool `json:"minimize_to_tray"`
	StartMinimized bool `json:"start_minimized"`
}

// LoadConfig 加载配置
func (a *App) LoadConfig() ConfigData {
	cfg, err := a.configMgr.Load()
	if err != nil {
		cfg = config.DefaultConnectionConfig()
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
	cfg, err := a.configMgr.Load()
	if err != nil {
		cfg = config.DefaultConnectionConfig()
	}
	cfg.ServerURL = data.ServerURL
	cfg.AccessKey = data.AccessKey
	cfg.SecretKey = data.SecretKey
	cfg.AutoConnect = data.AutoConnect
	cfg.AutoReconnect = data.AutoReconnect
	cfg.ReconnectInterval = data.ReconnectInterval
	cfg.LogLevel = data.LogLevel
	cfg.MinimizeToTray = data.MinimizeToTray
	cfg.StartMinimized = data.StartMinimized
	return a.configMgr.Save(cfg)
}

// ==================== gRPC 连接管理 ====================

// ConnectResult 连接结果
type ConnectResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
}

// Connect 连接到服务器
func (a *App) Connect(serverURL, accessKey, secretKey string) ConnectResult {
	// 保存配置
	cfg, _ := a.configMgr.Load()
	if cfg == nil {
		cfg = config.DefaultConnectionConfig()
	}
	cfg.ServerURL = serverURL
	cfg.AccessKey = accessKey
	cfg.SecretKey = secretKey
	_ = a.configMgr.Save(cfg)

	// 连接（Connect 方法会自动启动 TaskStream）
	err := a.grpcClient.Connect(serverURL, accessKey, secretKey)
	if err != nil {
		return ConnectResult{
			Success: false,
			Message: fmt.Sprintf("连接失败: %v", err),
		}
	}

	// 获取状态
	_, agentID, agentName := a.grpcClient.GetStatus()

	return ConnectResult{
		Success:   true,
		Message:   "连接成功",
		AgentID:   agentID,
		AgentName: agentName,
	}
}

// Disconnect 断开连接
func (a *App) Disconnect() error {
	if a.grpcClient != nil {
		a.grpcClient.Disconnect()
	}
	return nil
}

// StatusResult 状态结果
type StatusResult struct {
	Connected bool   `json:"connected"`
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
}

// GetStatus 获取连接状态
func (a *App) GetStatus() StatusResult {
	if a.grpcClient == nil {
		return StatusResult{Connected: false}
	}
	_, agentID, agentName := a.grpcClient.GetStatus()
	return StatusResult{
		Connected: a.grpcClient.IsConnected(),
		AgentID:   agentID,
		AgentName: agentName,
	}
}

// ==================== 日志 ====================

// LogEntry 日志条目
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// GetLogs 获取日志
func (a *App) GetLogs(count int) []LogEntry {
	if a.grpcClient == nil {
		return []LogEntry{}
	}

	logs := a.grpcClient.GetLogs(count)
	result := make([]LogEntry, len(logs))
	for i, log := range logs {
		result[i] = LogEntry{
			Time:    log.Timestamp,
			Level:   log.Level,
			Message: log.Message,
		}
	}
	return result
}

// ==================== 系统信息 ====================

// SystemInfo 系统信息
type SystemInfo struct {
	Platform string `json:"platform"`
	Hostname string `json:"hostname"`
	Arch     string `json:"arch"`
}

// GetSystemInfo 获取系统信息
func (a *App) GetSystemInfo() SystemInfo {
	hostname, _ := os.Hostname()
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macOS"
	} else if platform == "windows" {
		platform = "Windows"
	} else if platform == "linux" {
		platform = "Linux"
	}

	return SystemInfo{
		Platform: platform,
		Hostname: hostname,
		Arch:     runtime.GOARCH,
	}
}

// ==================== 权限管理 ====================

// PermissionsInfo 权限信息
type PermissionsInfo struct {
	Accessibility    bool `json:"accessibility"`
	ScreenRecording  bool `json:"screen_recording"`
}

// CheckPermissions 检查系统权限
func (a *App) CheckPermissions() PermissionsInfo {
	status := permissions.CheckPermissions()
	if status == nil {
		return PermissionsInfo{
			Accessibility:   true,
			ScreenRecording: true,
		}
	}
	return PermissionsInfo{
		Accessibility:   status.Accessibility,
		ScreenRecording: status.ScreenRecording,
	}
}

// RequestPermissions 请求权限（触发系统弹窗）
func (a *App) RequestPermissions() PermissionsInfo {
	// 请求辅助功能权限（会触发系统弹窗）
	permissions.RequestAccessibilityPermission()
	
	// 重新检查权限状态
	return a.CheckPermissions()
}

// OpenAccessibilitySettings 打开辅助功能设置
func (a *App) OpenAccessibilitySettings() {
	permissions.OpenAccessibilitySettings()
}

// OpenScreenRecordingSettings 打开屏幕录制设置
func (a *App) OpenScreenRecordingSettings() {
	permissions.OpenScreenRecordingSettings()
}

// ResetPermissions 重置权限状态（需要用户重新授权）
func (a *App) ResetPermissions() error {
	return permissions.ResetPermissions()
}

// ==================== Python 环境检测 ====================

// PythonInfo Python 环境信息
type PythonInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Path      string `json:"path"`
}

// GetPythonInfo 获取 Python 环境信息（使用缓存）
func (a *App) GetPythonInfo() PythonInfo {
	caps := grpc.GetCachedPythonInfo()
	if caps == nil {
		return PythonInfo{}
	}
	return PythonInfo{
		Available: caps.PythonAvailable,
		Version:   caps.PythonVersion,
		Path:      caps.PythonPath,
	}
}

// RefreshPythonInfo 重新检测 Python 环境（用户手动刷新）
func (a *App) RefreshPythonInfo() PythonInfo {
	caps := grpc.RefreshPythonInfo()
	if caps == nil {
		return PythonInfo{}
	}
	return PythonInfo{
		Available: caps.PythonAvailable,
		Version:   caps.PythonVersion,
		Path:      caps.PythonPath,
	}
}

// ==================== OCR 插件管理 ====================

// OCRPluginStatusResult OCR 插件状态
type OCRPluginStatusResult struct {
	Installed bool `json:"installed"`
}

// GetOCRPluginStatus 获取 OCR 插件状态
func (a *App) GetOCRPluginStatus() OCRPluginStatusResult {
	p := plugin.GetOCRPlugin()
	return OCRPluginStatusResult{
		Installed: p.IsInstalled(),
	}
}

// InstallOCRPlugin 安装 OCR 插件
func (a *App) InstallOCRPlugin() error {
	p := plugin.GetOCRPlugin()

	// 设置进度回调
	p.SetProgressCallback(func(progress float64) {
		// Wails v3 暂时不使用事件系统，简化处理
		fmt.Printf("OCR Install progress: %.0f%%\n", progress*100)
	})

	// 开始安装
	return p.Install()
}

// UninstallOCRPlugin 卸载 OCR 插件
func (a *App) UninstallOCRPlugin() error {
	p := plugin.GetOCRPlugin()
	return p.Uninstall()
}

// ==================== 窗口控制 ====================

// ShowWindow 显示窗口
func (a *App) ShowWindow() {
	if mainWindow != nil {
		mainWindow.Show()
		mainWindow.Focus()
	}
}

// HideWindow 隐藏窗口
func (a *App) HideWindow() {
	if mainWindow != nil {
		mainWindow.Hide()
	}
}

// QuitApp 退出应用
func (a *App) QuitApp() {
	if mainApp != nil {
		mainApp.Quit()
	}
}

// ==================== 调试功能 ====================

// DebugData 调试数据（返回给前端）
type DebugData struct {
	TaskID         string  `json:"task_id"`
	ActionType     string  `json:"action_type"`
	Status         string  `json:"status"`
	TemplateBase64 string  `json:"template_base64"`
	ScreenBase64   string  `json:"screen_base64"`
	Matched        bool    `json:"matched"`
	Confidence     float64 `json:"confidence"`
	X              int     `json:"x"`
	Y              int     `json:"y"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	DurationMs     int64   `json:"duration_ms"`
	Error          string  `json:"error"`
	Timestamp      int64   `json:"timestamp"`
	Version        int64   `json:"version"`
}

// GetDebugData 获取最新的调试数据（供前端轮询）
func (a *App) GetDebugData(lastVersion int64) *DebugData {
	currentVersion := executor.GetDebugDataVersion()
	
	// 如果版本号没变，返回 nil 表示没有新数据
	if currentVersion <= lastVersion {
		return nil
	}
	
	data := executor.GetLatestDebugData()
	if data == nil {
		return nil
	}
	
	return &DebugData{
		TaskID:         data.TaskID,
		ActionType:     data.ActionType,
		Status:         data.Status,
		TemplateBase64: data.TemplateBase64,
		ScreenBase64:   data.ScreenBase64,
		Matched:        data.Matched,
		Confidence:     data.Confidence,
		X:              data.X,
		Y:              data.Y,
		Width:          data.Width,
		Height:         data.Height,
		DurationMs:     data.Duration,
		Error:          data.Error,
		Timestamp:      data.Timestamp,
		Version:        currentVersion,
	}
}

