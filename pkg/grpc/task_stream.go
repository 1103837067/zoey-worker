package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/zoeyai/zoeyworker/pkg/grpc/pb"
)

// TaskStreamHandler 任务流处理器
type TaskStreamHandler struct {
	client   *Client
	agentID  string
	outgoing chan *pb.WorkerMessage
	stopCh   chan struct{}
	stream   pb.AgentService_TaskStreamClient

	onTask   TaskCallback
	onCancel CancelCallback

	mu      sync.Mutex
	running bool
}

// NewTaskStreamHandler 创建任务流处理器
func NewTaskStreamHandler(client *Client, agentID string) *TaskStreamHandler {
	return &TaskStreamHandler{
		client:   client,
		agentID:  agentID,
		outgoing: make(chan *pb.WorkerMessage, 100),
		stopCh:   make(chan struct{}),
	}
}

// SetTaskCallback 设置任务回调
func (h *TaskStreamHandler) SetTaskCallback(callback TaskCallback) {
	h.mu.Lock()
	h.onTask = callback
	h.mu.Unlock()
}

// SetCancelCallback 设置取消回调
func (h *TaskStreamHandler) SetCancelCallback(callback CancelCallback) {
	h.mu.Lock()
	h.onCancel = callback
	h.mu.Unlock()
}

// Start 启动任务流
func (h *TaskStreamHandler) Start(client pb.AgentServiceClient, stopCh chan struct{}) {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.stopCh = stopCh
	h.mu.Unlock()

	h.client.log("INFO", "Task stream starting...")

	// 创建双向流
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.TaskStream(ctx)
	if err != nil {
		h.client.log("ERROR", fmt.Sprintf("Failed to create task stream: %v", err))
		return
	}

	h.mu.Lock()
	h.stream = stream
	h.mu.Unlock()

	// 发送绑定消息
	bindMsg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("bind_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		AgentId:   h.agentID,
	}
	if err := stream.Send(bindMsg); err != nil {
		h.client.log("ERROR", fmt.Sprintf("Failed to send bind message: %v", err))
		return
	}

	h.client.log("INFO", "Task stream started")

	// 启动发送协程
	go h.sendLoop(stream)

	// 接收消息循环
	h.receiveLoop(stream)
}

// sendLoop 发送消息循环
func (h *TaskStreamHandler) sendLoop(stream pb.AgentService_TaskStreamClient) {
	for {
		select {
		case <-h.stopCh:
			return
		case msg := <-h.outgoing:
			if err := stream.Send(msg); err != nil {
				h.client.log("ERROR", fmt.Sprintf("Failed to send message: %v", err))
				return
			}
		}
	}
}

// receiveLoop 接收消息循环
func (h *TaskStreamHandler) receiveLoop(stream pb.AgentService_TaskStreamClient) {
	for {
		select {
		case <-h.stopCh:
			return
		default:
		}

		msg, err := stream.Recv()
		if err != nil {
			select {
			case <-h.stopCh:
				// 正常停止
				return
			default:
				h.client.log("ERROR", fmt.Sprintf("Task stream error: %v", err))
				return
			}
		}

		h.handleServerMessage(msg)
	}
}

// handleServerMessage 处理服务端消息
func (h *TaskStreamHandler) handleServerMessage(msg *pb.ServerMessage) {
	msgID := msg.MessageId

	switch {
	case msg.GetPing() != nil:
		h.handlePing(msgID, msg.GetPing())

	case msg.GetExecuteTask() != nil:
		h.handleExecuteTask(msg.GetExecuteTask())

	case msg.GetDataRequest() != nil:
		h.handleDataRequest(msgID, msg.GetDataRequest())

	case msg.GetCancelTask() != nil:
		h.handleCancelTask(msg.GetCancelTask())
	}
}

// handlePing 处理 Ping 消息
func (h *TaskStreamHandler) handlePing(msgID string, ping *pb.PingCommand) {
	h.client.log("DEBUG", "Received ping, sending pong")

	pong := &pb.WorkerMessage{
		MessageId: msgID,
		Timestamp: time.Now().UnixMilli(),
		AgentId:   h.agentID,
		Payload: &pb.WorkerMessage_Pong{
			Pong: &pb.PongResponse{
				ClientTimestamp: time.Now().UnixMilli(),
				ServerTimestamp: ping.Timestamp,
			},
		},
	}

	h.SendMessage(pong)
}

// handleExecuteTask 处理任务执行请求
func (h *TaskStreamHandler) handleExecuteTask(task *pb.ExecuteTaskCommand) {
	h.client.log("INFO", fmt.Sprintf("Received task: %s", task.TaskId))

	h.mu.Lock()
	callback := h.onTask
	h.mu.Unlock()

	if callback != nil {
		callback(task.TaskId, task.TaskType, task.PayloadJson)
	}
}

// handleDataRequest 处理数据请求
func (h *TaskStreamHandler) handleDataRequest(msgID string, req *pb.DataRequest) {
	h.client.log("INFO", fmt.Sprintf("Received data request: %s", req.RequestType))

	// 调用数据处理器
	response := HandleDataRequest(req.RequestType, req.PayloadJson)

	h.client.log("DEBUG", fmt.Sprintf("Data response: success=%v, msg=%s, payload_len=%d",
		response.Success, response.Message, len(response.PayloadJSON)))

	// 发送响应
	msg := &pb.WorkerMessage{
		MessageId: msgID,
		Timestamp: time.Now().UnixMilli(),
		AgentId:   h.agentID,
		Payload: &pb.WorkerMessage_DataResponse{
			DataResponse: &pb.DataResponse{
				RequestType: response.RequestType,
				Success:     response.Success,
				Message:     response.Message,
				PayloadJson: response.PayloadJSON,
			},
		},
	}

	h.SendMessage(msg)
	h.client.log("DEBUG", "Data response sent to queue")
}

// handleCancelTask 处理取消任务请求
func (h *TaskStreamHandler) handleCancelTask(cmd *pb.CancelTaskCommand) {
	h.client.log("INFO", fmt.Sprintf("Received cancel task: %s, reason: %s", cmd.TaskId, cmd.Reason))

	h.mu.Lock()
	callback := h.onCancel
	h.mu.Unlock()

	success := false
	if callback != nil {
		success = callback(cmd.TaskId)
	}

	// 发送取消确认
	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("cancel_ack_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		AgentId:   h.agentID,
		Payload: &pb.WorkerMessage_TaskResult{
			TaskResult: &pb.TaskResult{
				TaskId:  cmd.TaskId,
				Success: success,
				Status:  "CANCELLED",
				Message: cmd.Reason,
			},
		},
	}

	h.SendMessage(msg)

	if success {
		h.client.log("INFO", fmt.Sprintf("Task cancelled successfully: %s", cmd.TaskId))
	} else {
		h.client.log("WARN", fmt.Sprintf("Task not found or already completed: %s", cmd.TaskId))
	}
}

// SendMessage 发送消息
func (h *TaskStreamHandler) SendMessage(msg *pb.WorkerMessage) {
	select {
	case h.outgoing <- msg:
	default:
		h.client.log("WARN", "Outgoing message queue full, dropping message")
	}
}

// Stop 停止任务流
func (h *TaskStreamHandler) Stop() {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return
	}
	h.running = false
	h.mu.Unlock()

	// 关闭流
	if h.stream != nil {
		h.stream.CloseSend()
	}

	h.client.log("INFO", "Task stream stopped")
}

// SendTaskAck 发送任务确认
func (h *TaskStreamHandler) SendTaskAck(taskID string, accepted bool, message string) {
	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("ack_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		AgentId:   h.agentID,
		Payload: &pb.WorkerMessage_TaskAck{
			TaskAck: &pb.TaskAck{
				TaskId:   taskID,
				Accepted: accepted,
				Message:  message,
			},
		},
	}
	h.SendMessage(msg)
}

// SendTaskProgress 发送任务进度
func (h *TaskStreamHandler) SendTaskProgress(taskID string, totalSteps, completedSteps, passedSteps, failedSteps int32, currentStepName, status string) {
	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("progress_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		AgentId:   h.agentID,
		Payload: &pb.WorkerMessage_TaskProgress{
			TaskProgress: &pb.TaskProgress{
				TaskId:          taskID,
				TotalSteps:      totalSteps,
				CompletedSteps:  completedSteps,
				PassedSteps:     passedSteps,
				FailedSteps:     failedSteps,
				CurrentStepName: currentStepName,
				Status:          status,
			},
		},
	}
	h.SendMessage(msg)
}

// SendTaskResult 发送任务结果
func (h *TaskStreamHandler) SendTaskResult(taskID string, success bool, status, message, resultJSON string, durationMs int64) {
	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("result_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		AgentId:   h.agentID,
		Payload: &pb.WorkerMessage_TaskResult{
			TaskResult: &pb.TaskResult{
				TaskId:     taskID,
				Success:    success,
				Status:     status,
				Message:    message,
				ResultJson: resultJSON,
				DurationMs: durationMs,
			},
		},
	}
	h.SendMessage(msg)
}
