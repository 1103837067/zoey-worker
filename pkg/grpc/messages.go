package grpc

// WebSocket 消息类型定义
// 客户端与服务端通信的所有 JSON 消息结构

// WsConnectMessage 认证消息
type WsConnectMessage struct {
	Type       string        `json:"type"`
	AccessKey  string        `json:"accessKey"`
	SecretKey  string        `json:"secretKey"`
	SystemInfo *WsSystemInfo `json:"systemInfo,omitempty"`
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
	MessageId   string         `json:"messageId"`
	Timestamp   int64          `json:"timestamp"`
	ExecuteTask *WsExecuteTask `json:"executeTask,omitempty"`
	CancelTask  *WsCancelTask  `json:"cancelTask,omitempty"`
	Ping        *WsPing        `json:"ping,omitempty"`
	DataRequest *WsDataRequest `json:"dataRequest,omitempty"`
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
	MessageId    string          `json:"messageId"`
	Timestamp    int64           `json:"timestamp"`
	AgentId      string          `json:"agentId,omitempty"`
	TaskAck      *WsTaskAck      `json:"taskAck,omitempty"`
	TaskProgress *WsTaskProgress `json:"taskProgress,omitempty"`
	TaskResult   *WsTaskResult   `json:"taskResult,omitempty"`
	Pong         *WsPong         `json:"pong,omitempty"`
	DataResponse *WsDataResponse `json:"dataResponse,omitempty"`
	Heartbeat    *WsHeartbeat    `json:"heartbeat,omitempty"`
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
