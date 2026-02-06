package grpc

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	pb "github.com/zoeyai/zoeyworker/pkg/grpc/pb"
)

// ==================== WebSocket 消息类型 ====================

// WsConnectMessage 认证消息
type WsConnectMessage struct {
	Type       string          `json:"type"`
	AccessKey  string          `json:"accessKey"`
	SecretKey  string          `json:"secretKey"`
	SystemInfo *WsSystemInfo   `json:"systemInfo,omitempty"`
}

// WsSystemInfo 系统信息（JSON）
type WsSystemInfo struct {
	Hostname     string          `json:"hostname,omitempty"`
	Platform     string          `json:"platform,omitempty"`
	OsVersion    string          `json:"osVersion,omitempty"`
	AgentVersion string          `json:"agentVersion,omitempty"`
	IpAddress    string          `json:"ipAddress,omitempty"`
	Capabilities *WsCapabilities `json:"capabilities,omitempty"`
}

// WsCapabilities 能力信息
type WsCapabilities struct {
	PythonAvailable bool   `json:"pythonAvailable"`
	PythonVersion   string `json:"pythonVersion,omitempty"`
	PythonPath      string `json:"pythonPath,omitempty"`
}

// WsConnectResponse 认证响应
type WsConnectResponse struct {
	Type      string `json:"type"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	AgentId   string `json:"agentId"`
	AgentName string `json:"agentName"`
}

// WsServerMessage 服务端消息
type WsServerMessage struct {
	MessageId   string              `json:"messageId"`
	Timestamp   int64               `json:"timestamp"`
	ExecuteTask *WsExecuteTask      `json:"executeTask,omitempty"`
	CancelTask  *WsCancelTask       `json:"cancelTask,omitempty"`
	Ping        *WsPing             `json:"ping,omitempty"`
	DataRequest *WsDataRequest      `json:"dataRequest,omitempty"`
}

// WsExecuteTask 执行任务命令
type WsExecuteTask struct {
	TaskId      string `json:"taskId"`
	TaskType    string `json:"taskType"`
	PayloadJson string `json:"payloadJson"`
}

// WsCancelTask 取消任务命令
type WsCancelTask struct {
	TaskId string `json:"taskId"`
	Reason string `json:"reason"`
}

// WsPing Ping 命令
type WsPing struct {
	Timestamp int64 `json:"timestamp"`
}

// WsDataRequest 数据查询请求
type WsDataRequest struct {
	RequestType string `json:"requestType"`
	PayloadJson string `json:"payloadJson"`
}

// WsWorkerMessage Worker 消息
type WsWorkerMessage struct {
	MessageId    string              `json:"messageId"`
	Timestamp    int64               `json:"timestamp"`
	AgentId      string              `json:"agentId,omitempty"`
	TaskAck      *WsTaskAck          `json:"taskAck,omitempty"`
	TaskProgress *WsTaskProgress     `json:"taskProgress,omitempty"`
	TaskResult   *WsTaskResult       `json:"taskResult,omitempty"`
	Pong         *WsPong             `json:"pong,omitempty"`
	DataResponse *WsDataResponse     `json:"dataResponse,omitempty"`
	Heartbeat    *WsHeartbeat        `json:"heartbeat,omitempty"`
}

// WsTaskAck 任务确认
type WsTaskAck struct {
	TaskId   string `json:"taskId"`
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
}

// WsTaskProgress 任务进度
type WsTaskProgress struct {
	TaskId          string `json:"taskId"`
	TotalSteps      int32  `json:"totalSteps"`
	CompletedSteps  int32  `json:"completedSteps"`
	PassedSteps     int32  `json:"passedSteps"`
	FailedSteps     int32  `json:"failedSteps"`
	CurrentStepName string `json:"currentStepName"`
	Status          string `json:"status"`
}

// WsTaskResult 任务结果
type WsTaskResult struct {
	TaskId        string           `json:"taskId"`
	Success       bool             `json:"success"`
	Status        int32            `json:"status"`
	Message       string           `json:"message"`
	ResultJson    string           `json:"resultJson"`
	DurationMs    int64            `json:"durationMs"`
	FailureReason int32            `json:"failureReason,omitempty"`
	MatchLocation *WsMatchLocation `json:"matchLocation,omitempty"`
}

// WsMatchLocation 匹配位置
type WsMatchLocation struct {
	X          int32   `json:"x"`
	Y          int32   `json:"y"`
	Width      int32   `json:"width"`
	Height     int32   `json:"height"`
	Confidence float32 `json:"confidence"`
}

// WsPong Pong 响应
type WsPong struct {
	ClientTimestamp int64 `json:"clientTimestamp"`
	ServerTimestamp int64 `json:"serverTimestamp"`
}

// WsDataResponse 数据响应
type WsDataResponse struct {
	RequestType string `json:"requestType"`
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	PayloadJson string `json:"payloadJson"`
}

// WsHeartbeat 心跳消息
type WsHeartbeat struct {
	ResourceInfo *WsResourceInfo `json:"resourceInfo,omitempty"`
	AgentStatus  *WsAgentStatus  `json:"agentStatus,omitempty"`
}

// WsResourceInfo 资源信息
type WsResourceInfo struct {
	CpuUsage    float32 `json:"cpuUsage"`
	MemoryUsage float32 `json:"memoryUsage"`
	DiskUsage   float32 `json:"diskUsage"`
}

// WsAgentStatus Agent 状态
type WsAgentStatus struct {
	Status            string `json:"status"`
	CurrentTaskId     string `json:"currentTaskId,omitempty"`
	CurrentTaskType   string `json:"currentTaskType,omitempty"`
	TaskStartedAt     int64  `json:"taskStartedAt,omitempty"`
	RunningTasksCount int32  `json:"runningTasksCount"`
}

// ==================== Client ====================

// Client WebSocket 客户端
type Client struct {
	config *ClientConfig
	conn   *websocket.Conn

	agentID     string
	agentName   string
	isConnected bool

	outgoing chan *WsWorkerMessage
	stopCh   chan struct{}
	wg       sync.WaitGroup

	onStatusChange   StatusCallback
	onTask           TaskCallback
	onCancel         CancelCallback
	onExecutorStatus ExecutorStatusCallback

	logs   []LogEntry
	logsMu sync.Mutex

	mu sync.RWMutex
}

// NewClient 创建新的 WebSocket 客户端
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = DefaultConfig()
	}
	c := &Client{
		config:   config,
		outgoing: make(chan *WsWorkerMessage, 100),
		stopCh:   make(chan struct{}),
		logs:     make([]LogEntry, 0, 500),
	}

	// 设置全局日志函数，让 data_handler 也能输出日志
	SetLogFunc(func(level, message string) {
		c.log(level, message)
	})

	return c
}

// Connect 连接到服务端
func (c *Client) Connect(serverURL, accessKey, secretKey string) error {
	c.mu.Lock()
	c.config.ServerURL = serverURL
	c.config.AccessKey = accessKey
	c.config.SecretKey = secretKey
	c.mu.Unlock()

	return c.doConnect()
}

// buildWsURL 根据 serverURL 构建 WebSocket URL
// 支持多种输入格式：
//   - localhost:3001 → ws://localhost:3001/ws/agent
//   - http://localhost:3001 → ws://localhost:3001/ws/agent
//   - https://example.com → wss://example.com/ws/agent
//   - wss://example.com → wss://example.com/ws/agent
//   - example.com → wss://example.com/ws/agent（域名默认 wss）
func buildWsURL(serverURL string) string {
	// 已有 ws:// 或 wss:// 前缀
	if len(serverURL) > 5 && (serverURL[:5] == "ws://" || serverURL[:6] == "wss://") {
		u, err := url.Parse(serverURL)
		if err == nil {
			if u.Path == "" || u.Path == "/" {
				u.Path = "/ws/agent"
			}
			return u.String()
		}
		return serverURL
	}

	// http:// → ws://，https:// → wss://
	if len(serverURL) > 7 && serverURL[:7] == "http://" {
		return "ws://" + serverURL[7:] + "/ws/agent"
	}
	if len(serverURL) > 8 && serverURL[:8] == "https://" {
		return "wss://" + serverURL[8:] + "/ws/agent"
	}

	// 裸地址：判断是否为域名（含 .）还是 localhost
	host := serverURL
	// 如果是 localhost 或 IP:port 格式，用 ws://
	if isLocalAddress(host) {
		return "ws://" + host + "/ws/agent"
	}
	// 域名默认用 wss://
	return "wss://" + host + "/ws/agent"
}

// isLocalAddress 判断是否为本地地址
func isLocalAddress(addr string) bool {
	// 去掉端口部分
	host := addr
	if idx := len(host) - 1; idx >= 0 {
		for i := len(host) - 1; i >= 0; i-- {
			if host[i] == ':' {
				host = host[:i]
				break
			}
		}
	}
	return host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" || host == "::1"
}

// doConnect 执行连接
func (c *Client) doConnect() error {
	c.mu.Lock()
	serverURL := c.config.ServerURL
	accessKey := c.config.AccessKey
	secretKey := c.config.SecretKey
	c.mu.Unlock()

	wsURL := buildWsURL(serverURL)
	c.log("INFO", fmt.Sprintf("Connecting to %s...", wsURL))
	c.setStatus(StatusConnecting)

	// 创建 WebSocket 连接
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		c.log("ERROR", fmt.Sprintf("WebSocket connection failed: %v", err))
		c.setStatus(StatusDisconnected)
		return fmt.Errorf("连接失败: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// 发送认证消息
	sysInfo := GetSystemInfo()
	connectMsg := WsConnectMessage{
		Type:      "connect",
		AccessKey: accessKey,
		SecretKey: secretKey,
		SystemInfo: &WsSystemInfo{
			Hostname:     sysInfo.Hostname,
			Platform:     sysInfo.Platform,
			OsVersion:    sysInfo.OSVersion,
			AgentVersion: sysInfo.AgentVersion,
			IpAddress:    sysInfo.IPAddress,
		},
	}
	if sysInfo.Capabilities != nil {
		connectMsg.SystemInfo.Capabilities = &WsCapabilities{
			PythonAvailable: sysInfo.Capabilities.PythonAvailable,
			PythonVersion:   sysInfo.Capabilities.PythonVersion,
			PythonPath:      sysInfo.Capabilities.PythonPath,
		}
	}

	data, err := json.Marshal(connectMsg)
	if err != nil {
		c.log("ERROR", fmt.Sprintf("Failed to marshal connect message: %v", err))
		conn.Close()
		c.setStatus(StatusDisconnected)
		return fmt.Errorf("序列化认证消息失败: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.log("ERROR", fmt.Sprintf("Failed to send connect message: %v", err))
		conn.Close()
		c.setStatus(StatusDisconnected)
		return fmt.Errorf("发送认证消息失败: %w", err)
	}

	// 等待认证响应
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, respData, err := conn.ReadMessage()
	conn.SetReadDeadline(time.Time{}) // 清除 deadline

	if err != nil {
		c.log("ERROR", fmt.Sprintf("Failed to read connect response: %v", err))
		conn.Close()
		c.setStatus(StatusDisconnected)
		return fmt.Errorf("读取认证响应失败: %w", err)
	}

	var resp WsConnectResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		c.log("ERROR", fmt.Sprintf("Failed to parse connect response: %v", err))
		conn.Close()
		c.setStatus(StatusDisconnected)
		return fmt.Errorf("解析认证响应失败: %w", err)
	}

	if !resp.Success {
		c.log("ERROR", fmt.Sprintf("Connect rejected: %s", resp.Message))
		conn.Close()
		c.setStatus(StatusDisconnected)
		return fmt.Errorf("认证被拒绝: %s", resp.Message)
	}

	c.mu.Lock()
	c.agentID = resp.AgentId
	c.agentName = resp.AgentName
	c.isConnected = true
	c.stopCh = make(chan struct{})
	c.outgoing = make(chan *WsWorkerMessage, 100)
	c.mu.Unlock()

	c.log("INFO", fmt.Sprintf("Connected as %s (%s)", c.agentName, c.agentID))
	c.setStatus(StatusConnected)

	// 启动消息循环
	c.wg.Add(2)
	go c.sendLoop()
	go c.receiveLoop()

	// 启动心跳
	c.wg.Add(1)
	go c.heartbeatLoop()

	return nil
}

// sendLoop 发送消息循环
func (c *Client) sendLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopCh:
			return
		case msg := <-c.outgoing:
			data, err := json.Marshal(msg)
			if err != nil {
				c.log("ERROR", fmt.Sprintf("Failed to marshal message: %v", err))
				continue
			}

			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn != nil {
				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					c.log("ERROR", fmt.Sprintf("Failed to send message: %v", err))
					return
				}
			}
		}
	}
}

// receiveLoop 接收消息循环
func (c *Client) receiveLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-c.stopCh:
				return
			default:
				c.log("ERROR", fmt.Sprintf("WebSocket read error: %v", err))
				go c.attemptReconnect()
				return
			}
		}

		var msg WsServerMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			c.log("WARN", fmt.Sprintf("Failed to parse server message: %v", err))
			continue
		}

		c.handleServerMessage(&msg)
	}
}

// handleServerMessage 处理服务端消息
func (c *Client) handleServerMessage(msg *WsServerMessage) {
	switch {
	case msg.Ping != nil:
		c.handlePing(msg.MessageId, msg.Ping)
	case msg.ExecuteTask != nil:
		c.handleExecuteTask(msg.ExecuteTask)
	case msg.DataRequest != nil:
		c.handleDataRequest(msg.MessageId, msg.DataRequest)
	case msg.CancelTask != nil:
		c.handleCancelTask(msg.CancelTask)
	}
}

// handlePing 处理 Ping
func (c *Client) handlePing(msgID string, ping *WsPing) {
	c.log("DEBUG", "Received ping, sending pong")
	c.sendMessage(&WsWorkerMessage{
		MessageId: msgID,
		Timestamp: time.Now().UnixMilli(),
		AgentId:   c.agentID,
		Pong: &WsPong{
			ClientTimestamp: time.Now().UnixMilli(),
			ServerTimestamp: ping.Timestamp,
		},
	})
}

// handleExecuteTask 处理任务执行
func (c *Client) handleExecuteTask(task *WsExecuteTask) {
	c.log("INFO", fmt.Sprintf("Received task: %s", task.TaskId))

	c.mu.RLock()
	callback := c.onTask
	c.mu.RUnlock()

	if callback != nil {
		callback(task.TaskId, task.TaskType, task.PayloadJson)
	}
}

// handleDataRequest 处理数据请求
func (c *Client) handleDataRequest(msgID string, req *WsDataRequest) {
	c.log("INFO", fmt.Sprintf("Received data request: %s", req.RequestType))

	response := HandleDataRequest(req.RequestType, req.PayloadJson)

	c.sendMessage(&WsWorkerMessage{
		MessageId: msgID,
		Timestamp: time.Now().UnixMilli(),
		AgentId:   c.agentID,
		DataResponse: &WsDataResponse{
			RequestType: response.RequestType,
			Success:     response.Success,
			Message:     response.Message,
			PayloadJson: response.PayloadJSON,
		},
	})
}

// handleCancelTask 处理取消任务
func (c *Client) handleCancelTask(cmd *WsCancelTask) {
	c.log("INFO", fmt.Sprintf("Received cancel task: %s, reason: %s", cmd.TaskId, cmd.Reason))

	c.mu.RLock()
	callback := c.onCancel
	c.mu.RUnlock()

	success := false
	if callback != nil {
		success = callback(cmd.TaskId)
	}

	// TaskStatus_TASK_STATUS_CANCELLED = 3
	c.sendMessage(&WsWorkerMessage{
		MessageId: fmt.Sprintf("cancel_ack_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		AgentId:   c.agentID,
		TaskResult: &WsTaskResult{
			TaskId:  cmd.TaskId,
			Success: success,
			Status:  3, // TASK_STATUS_CANCELLED
			Message: cmd.Reason,
		},
	})

	if success {
		c.log("INFO", fmt.Sprintf("Task cancelled successfully: %s", cmd.TaskId))
	} else {
		c.log("WARN", fmt.Sprintf("Task not found or already completed: %s", cmd.TaskId))
	}
}

// heartbeatLoop 心跳循环
func (c *Client) heartbeatLoop() {
	defer c.wg.Done()

	c.mu.RLock()
	interval := c.config.HeartbeatInterval
	c.mu.RUnlock()

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.sendHeartbeat()
		}
	}
}

// sendHeartbeat 发送心跳
func (c *Client) sendHeartbeat() {
	c.mu.RLock()
	callback := c.onExecutorStatus
	c.mu.RUnlock()

	var agentStatus *WsAgentStatus
	if callback != nil {
		status, taskID, taskType, startedAt, count := callback()
		agentStatus = &WsAgentStatus{
			Status:            status,
			CurrentTaskId:     taskID,
			CurrentTaskType:   taskType,
			TaskStartedAt:     startedAt,
			RunningTasksCount: int32(count),
		}
	} else {
		agentStatus = &WsAgentStatus{
			Status:            "IDLE",
			RunningTasksCount: 0,
		}
	}

	c.sendMessage(&WsWorkerMessage{
		MessageId: fmt.Sprintf("heartbeat_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		AgentId:   c.agentID,
		Heartbeat: &WsHeartbeat{
			AgentStatus: agentStatus,
		},
	})
	c.log("DEBUG", "Heartbeat sent")
}

// sendMessage 发送消息到队列
func (c *Client) sendMessage(msg *WsWorkerMessage) {
	select {
	case c.outgoing <- msg:
	default:
		c.log("WARN", "Outgoing message queue full, dropping message")
	}
}

// Disconnect 断开连接
func (c *Client) Disconnect() error {
	c.mu.Lock()
	if !c.isConnected {
		c.mu.Unlock()
		return nil
	}
	c.isConnected = false
	c.mu.Unlock()

	// 发送停止信号
	close(c.stopCh)

	// 等待 goroutine 结束
	c.wg.Wait()

	// 关闭连接
	c.mu.Lock()
	if c.conn != nil {
		c.conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
		c.conn = nil
	}
	c.agentID = ""
	c.agentName = ""
	c.mu.Unlock()

	c.log("INFO", "Disconnected")
	c.setStatus(StatusDisconnected)

	return nil
}

// attemptReconnect 尝试重连
func (c *Client) attemptReconnect() {
	c.mu.Lock()
	if !c.isConnected {
		c.mu.Unlock()
		return
	}
	c.isConnected = false
	c.mu.Unlock()

	// 关闭旧连接
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()

	c.setStatus(StatusReconnecting)

	// 指数退避重连
	for i, delay := range c.config.ReconnectDelays {
		select {
		case <-c.stopCh:
			return
		default:
		}

		c.log("INFO", fmt.Sprintf("Reconnect attempt %d/%d in %ds...", i+1, len(c.config.ReconnectDelays), delay))
		time.Sleep(time.Duration(delay) * time.Second)

		select {
		case <-c.stopCh:
			return
		default:
		}

		if err := c.doConnect(); err == nil {
			c.log("INFO", "Reconnected successfully!")
			return
		}
	}

	c.log("ERROR", "Failed to reconnect after all attempts")
	c.setStatus(StatusDisconnected)
}

// ==================== 供 executor 调用的方法 ====================

// SendTaskMessage 发送任务消息（兼容 executor 原有接口）
// 接受 *pb.WorkerMessage，手动转换为 WsWorkerMessage 发送
func (c *Client) SendTaskMessage(msg *pb.WorkerMessage) {
	wsMsg := &WsWorkerMessage{
		MessageId: msg.MessageId,
		Timestamp: msg.Timestamp,
		AgentId:   msg.AgentId,
	}

	switch payload := msg.Payload.(type) {
	case *pb.WorkerMessage_TaskAck:
		if ack := payload.TaskAck; ack != nil {
			wsMsg.TaskAck = &WsTaskAck{
				TaskId:   ack.TaskId,
				Accepted: ack.Accepted,
				Message:  ack.Message,
			}
		}
	case *pb.WorkerMessage_TaskProgress:
		if p := payload.TaskProgress; p != nil {
			wsMsg.TaskProgress = &WsTaskProgress{
				TaskId:          p.TaskId,
				TotalSteps:      p.TotalSteps,
				CompletedSteps:  p.CompletedSteps,
				PassedSteps:     p.PassedSteps,
				FailedSteps:     p.FailedSteps,
				CurrentStepName: p.CurrentStepName,
				Status:          p.Status,
			}
		}
	case *pb.WorkerMessage_TaskResult:
		if r := payload.TaskResult; r != nil {
			wsResult := &WsTaskResult{
				TaskId:        r.TaskId,
				Success:       r.Success,
				Status:        int32(r.Status),
				Message:       r.Message,
				ResultJson:    r.ResultJson,
				DurationMs:    r.DurationMs,
				FailureReason: int32(r.FailureReason),
			}
			if r.MatchLocation != nil {
				wsResult.MatchLocation = &WsMatchLocation{
					X:          r.MatchLocation.X,
					Y:          r.MatchLocation.Y,
					Width:      r.MatchLocation.Width,
					Height:     r.MatchLocation.Height,
					Confidence: r.MatchLocation.Confidence,
				}
			}
			wsMsg.TaskResult = wsResult
		}
	case *pb.WorkerMessage_Pong:
		if p := payload.Pong; p != nil {
			wsMsg.Pong = &WsPong{
				ClientTimestamp: p.ClientTimestamp,
				ServerTimestamp: p.ServerTimestamp,
			}
		}
	case *pb.WorkerMessage_DataResponse:
		if d := payload.DataResponse; d != nil {
			wsMsg.DataResponse = &WsDataResponse{
				RequestType: d.RequestType,
				Success:     d.Success,
				Message:     d.Message,
				PayloadJson: d.PayloadJson,
			}
		}
	case *pb.WorkerMessage_Heartbeat:
		if h := payload.Heartbeat; h != nil {
			hb := &WsHeartbeat{}
			if h.AgentStatus != nil {
				hb.AgentStatus = &WsAgentStatus{
					Status:            h.AgentStatus.Status,
					CurrentTaskId:     h.AgentStatus.CurrentTaskId,
					CurrentTaskType:   h.AgentStatus.CurrentTaskType,
					TaskStartedAt:     h.AgentStatus.TaskStartedAt,
					RunningTasksCount: h.AgentStatus.RunningTasksCount,
				}
			}
			wsMsg.Heartbeat = hb
		}
	}

	c.sendMessage(wsMsg)
}

// GetStatus 获取当前状态
func (c *Client) GetStatus() (ClientStatus, string, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := StatusDisconnected
	if c.isConnected {
		status = StatusConnected
	}
	return status, c.agentID, c.agentName
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// SetStatusCallback 设置状态变更回调
func (c *Client) SetStatusCallback(callback StatusCallback) {
	c.mu.Lock()
	c.onStatusChange = callback
	c.mu.Unlock()
}

// SetTaskCallback 设置任务回调
func (c *Client) SetTaskCallback(callback TaskCallback) {
	c.mu.Lock()
	c.onTask = callback
	c.mu.Unlock()
}

// SetCancelCallback 设置取消任务回调
func (c *Client) SetCancelCallback(callback CancelCallback) {
	c.mu.Lock()
	c.onCancel = callback
	c.mu.Unlock()
}

// SetExecutorStatusCallback 设置执行器状态回调
func (c *Client) SetExecutorStatusCallback(callback ExecutorStatusCallback) {
	c.mu.Lock()
	c.onExecutorStatus = callback
	c.mu.Unlock()
}

// setStatus 设置状态并触发回调
func (c *Client) setStatus(status ClientStatus) {
	c.mu.RLock()
	callback := c.onStatusChange
	c.mu.RUnlock()

	if callback != nil {
		callback(status)
	}
}

// Log 记录日志（公开方法）
func (c *Client) Log(level, message string) {
	c.log(level, message)
}

// log 记录日志（内部方法）
func (c *Client) log(level, message string) {
	entry := LogEntry{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Level:     level,
		Message:   message,
	}

	c.logsMu.Lock()
	c.logs = append(c.logs, entry)
	if len(c.logs) > 500 {
		c.logs = c.logs[len(c.logs)-500:]
	}
	c.logsMu.Unlock()

	fmt.Printf("[%s] %s\n", level, message)
}

// GetLogs 获取日志
func (c *Client) GetLogs(limit int) []LogEntry {
	c.logsMu.Lock()
	defer c.logsMu.Unlock()

	if limit <= 0 || limit > len(c.logs) {
		limit = len(c.logs)
	}

	result := make([]LogEntry, limit)
	copy(result, c.logs[len(c.logs)-limit:])
	return result
}
