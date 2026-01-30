package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/zoeyai/zoeyworker/pkg/grpc/pb"
)

// Client gRPC 客户端
type Client struct {
	config *ClientConfig
	conn   *grpc.ClientConn
	client pb.AgentServiceClient

	agentID     string
	agentName   string
	isConnected bool

	taskStream *TaskStreamHandler
	stopCh     chan struct{}
	wg         sync.WaitGroup

	heartbeatFailures int

	onStatusChange   StatusCallback
	onTask           TaskCallback
	onCancel         CancelCallback
	onExecutorStatus ExecutorStatusCallback

	logs   []LogEntry
	logsMu sync.Mutex

	mu sync.RWMutex
}

// NewClient 创建新的 gRPC 客户端
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = DefaultConfig()
	}
	c := &Client{
		config: config,
		stopCh: make(chan struct{}),
		logs:   make([]LogEntry, 0, 500),
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

// doConnect 执行连接
func (c *Client) doConnect() error {
	c.mu.Lock()
	serverURL := c.config.ServerURL
	accessKey := c.config.AccessKey
	secretKey := c.config.SecretKey
	c.mu.Unlock()

	c.log("INFO", fmt.Sprintf("Connecting to %s...", serverURL))
	c.setStatus(StatusConnecting)

	// 创建 gRPC 连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, serverURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		c.log("ERROR", fmt.Sprintf("Connection failed: %v", err))
		c.setStatus(StatusDisconnected)
		return fmt.Errorf("连接失败: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.client = pb.NewAgentServiceClient(conn)
	c.mu.Unlock()

	// 调用 Connect RPC
	sysInfo := GetSystemInfo()
	req := &pb.ConnectRequest{
		AccessKey: accessKey,
		SecretKey: secretKey,
		SystemInfo: &pb.SystemInfo{
			Hostname:     sysInfo.Hostname,
			Platform:     sysInfo.Platform,
			OsVersion:    sysInfo.OSVersion,
			AgentVersion: sysInfo.AgentVersion,
			IpAddress:    sysInfo.IPAddress,
		},
	}

	resp, err := c.client.Connect(context.Background(), req)
	if err != nil {
		c.log("ERROR", fmt.Sprintf("Connect RPC failed: %v", err))
		conn.Close()
		c.setStatus(StatusDisconnected)
		return fmt.Errorf("认证失败: %w", err)
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
	c.heartbeatFailures = 0
	c.stopCh = make(chan struct{})
	c.mu.Unlock()

	c.log("INFO", fmt.Sprintf("Connected as %s (%s)", c.agentName, c.agentID))
	c.setStatus(StatusConnected)

	// 启动心跳
	c.wg.Add(1)
	go c.heartbeatLoop()

	// 启动任务流
	c.startTaskStream()

	return nil
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

	// 停止任务流
	if c.taskStream != nil {
		c.taskStream.Stop()
	}

	// 等待 goroutine 结束
	c.wg.Wait()

	// 关闭连接
	c.mu.Lock()
	if c.conn != nil {
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

// heartbeatLoop 心跳循环
func (c *Client) heartbeatLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Duration(c.config.HeartbeatInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.mu.RLock()
			if !c.isConnected || c.client == nil {
				c.mu.RUnlock()
				continue
			}
			agentID := c.agentID
			executorStatusCallback := c.onExecutorStatus
			c.mu.RUnlock()

			// 构建心跳请求
			req := &pb.HeartbeatRequest{
				AgentId:   agentID,
				Timestamp: time.Now().UnixMilli(),
			}

			// 添加执行器状态
			if executorStatusCallback != nil {
				status, taskID, taskType, startedAt, count := executorStatusCallback()
				req.AgentStatus = &pb.AgentStatus{
					Status:            status,
					CurrentTaskId:     taskID,
					CurrentTaskType:   taskType,
					TaskStartedAt:     startedAt,
					RunningTasksCount: int32(count),
				}
			} else {
				// 默认空闲状态
				req.AgentStatus = &pb.AgentStatus{
					Status:            "IDLE",
					RunningTasksCount: 0,
				}
			}

			// 发送心跳
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := c.client.Heartbeat(ctx, req)
			cancel()

			if err != nil {
				c.mu.Lock()
				c.heartbeatFailures++
				failures := c.heartbeatFailures
				maxFailures := c.config.MaxHeartbeatFailures
				c.mu.Unlock()

				c.log("WARN", fmt.Sprintf("Heartbeat failed (%d/%d): %v", failures, maxFailures, err))

				if failures >= maxFailures {
					c.log("INFO", "Too many heartbeat failures, attempting reconnect...")
					go c.attemptReconnect()
					return
				}
			} else {
				c.mu.Lock()
				c.heartbeatFailures = 0
				c.mu.Unlock()
				c.log("DEBUG", "Heartbeat sent")
			}
		}
	}
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

	// 停止任务流
	if c.taskStream != nil {
		c.taskStream.Stop()
		c.taskStream = nil
	}

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

// startTaskStream 启动任务流
func (c *Client) startTaskStream() {
	c.mu.RLock()
	client := c.client
	agentID := c.agentID
	c.mu.RUnlock()

	c.taskStream = NewTaskStreamHandler(c, agentID)
	c.taskStream.SetTaskCallback(c.onTask)
	c.taskStream.SetCancelCallback(c.onCancel)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.taskStream.Start(client, c.stopCh)
	}()
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

	// 同时输出到控制台
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

// SendTaskMessage 发送任务消息
func (c *Client) SendTaskMessage(msg *pb.WorkerMessage) {
	if c.taskStream != nil {
		c.taskStream.SendMessage(msg)
	}
}
