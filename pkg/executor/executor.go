package executor

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zoeyai/zoeyworker/pkg/grpc"
	pb "github.com/zoeyai/zoeyworker/pkg/grpc/pb"
)

// ==================== 任务类型常量 ====================

// TaskType 任务类型
const (
	TaskTypeClickImage   = "click_image"
	TaskTypeClickText    = "click_text"
	TaskTypeClickNative  = "click_native"
	TaskTypeTypeText     = "type_text"
	TaskTypeKeyPress     = "key_press"
	TaskTypeScreenshot   = "screenshot"
	TaskTypeWaitImage    = "wait_image"
	TaskTypeWaitText     = "wait_text"
	TaskTypeWaitTime     = "wait_time"
	TaskTypeMouseMove    = "mouse_move"
	TaskTypeMouseClick   = "mouse_click"
	TaskTypeActivateApp  = "activate_app"
	TaskTypeCloseApp     = "close_app"
	TaskTypeGridClick    = "grid_click"
	TaskTypeImageExists  = "image_exists"
	TaskTypeTextExists   = "text_exists"
	TaskTypeAssertImage  = "assert_image"
	TaskTypeAssertText   = "assert_text"
	TaskTypeGetClipboard = "get_clipboard"
	TaskTypeSetClipboard = "set_clipboard"
	TaskTypeRunPython    = "run_python"
	// 批量执行类型
	TaskTypeDebugCase   = "debug_case"
	TaskTypeExecutePlan = "execute_plan" // 执行测试计划
	TaskTypeExecuteCase = "execute_case" // 执行单个用例
)

// ==================== 类型定义 ====================

// 使用 pb 包中的枚举类型
// TaskStatus: pb.TaskStatus_TASK_STATUS_SUCCESS, etc.
// FailureReason: pb.FailureReason_FAILURE_REASON_NOT_FOUND, etc.

// DebugMatchData 调试匹配数据（用于发送到前端调试面板）
type DebugMatchData struct {
	TaskID         string  `json:"task_id"`
	ActionType     string  `json:"action_type"`
	Status         string  `json:"status"`          // searching, found, not_found, error
	TemplateBase64 string  `json:"template_base64"` // 目标图片 base64
	ScreenBase64   string  `json:"screen_base64"`   // 截图 base64
	Matched        bool    `json:"matched"`
	Confidence     float64 `json:"confidence"`
	X              int     `json:"x"`
	Y              int     `json:"y"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Duration       int64   `json:"duration_ms"`
	Error          string  `json:"error,omitempty"`
	Timestamp      int64   `json:"timestamp"` // 时间戳，用于前端判断是否有新数据
}

// 调试数据存储
var (
	latestDebugData  *DebugMatchData
	debugDataMutex   sync.RWMutex
	debugDataVersion int64 // 版本号，每次更新时递增
)

// GetLatestDebugData 获取最新的调试数据（供前端轮询）
func GetLatestDebugData() *DebugMatchData {
	debugDataMutex.RLock()
	defer debugDataMutex.RUnlock()
	return latestDebugData
}

// GetDebugDataVersion 获取调试数据版本号
func GetDebugDataVersion() int64 {
	debugDataMutex.RLock()
	defer debugDataMutex.RUnlock()
	return debugDataVersion
}

// emitDebugMatch 保存调试匹配数据（供前端轮询获取）
func emitDebugMatch(data DebugMatchData) {
	debugDataMutex.Lock()
	defer debugDataMutex.Unlock()

	data.Timestamp = time.Now().UnixMilli()
	debugDataVersion++
	latestDebugData = &data
}

// TaskError 任务错误
type TaskError struct {
	Status  pb.TaskStatus
	Reason  pb.FailureReason
	Message string
}

func (e *TaskError) Error() string {
	return e.Message
}

// newTaskError 创建任务错误
func newTaskError(status pb.TaskStatus, reason pb.FailureReason, message string) *TaskError {
	return &TaskError{Status: status, Reason: reason, Message: message}
}

// classifyError 对错误进行分类
func classifyError(err error) *TaskError {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	errLower := strings.ToLower(errStr)

	// 超时单独作为状态
	if strings.Contains(errLower, "timeout") || strings.Contains(errLower, "超时") {
		return newTaskError(pb.TaskStatus_TASK_STATUS_TIMEOUT, pb.FailureReason_FAILURE_REASON_UNSPECIFIED, errStr)
	}

	// 其他错误归类为 FAILED + 具体原因
	var reason pb.FailureReason
	switch {
	case strings.Contains(errLower, "not found") || strings.Contains(errLower, "未找到") ||
		strings.Contains(errLower, "找不到") || strings.Contains(errLower, "匹配失败") ||
		strings.Contains(errLower, "无法在屏幕中找到"):
		reason = pb.FailureReason_FAILURE_REASON_NOT_FOUND
	case strings.Contains(errLower, "multiple") || strings.Contains(errLower, "多个"):
		reason = pb.FailureReason_FAILURE_REASON_MULTIPLE_MATCHES
	case strings.Contains(errLower, "断言") || strings.Contains(errLower, "assert"):
		reason = pb.FailureReason_FAILURE_REASON_ASSERTION_FAILED
	case strings.Contains(errLower, "参数") || strings.Contains(errLower, "param") || strings.Contains(errLower, "缺少"):
		reason = pb.FailureReason_FAILURE_REASON_PARAM_ERROR
	default:
		reason = pb.FailureReason_FAILURE_REASON_SYSTEM_ERROR
	}

	return newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, reason, errStr)
}

// StepExecutionResult 步骤执行结果（用于前端回放）
type StepExecutionResult struct {
	StepExecutionID string `json:"stepExecutionId,omitempty"` // 步骤执行记录 ID
	StepID          string `json:"stepId"`                    // 步骤 ID
	Status          string `json:"status"`                    // SUCCESS, FAILED, SKIPPED

	// 截图（Base64 格式）
	ScreenshotBefore string `json:"screenshotBefore,omitempty"` // 执行前截图
	ScreenshotAfter  string `json:"screenshotAfter,omitempty"`  // 执行后截图

	// 操作信息
	ActionType string `json:"actionType"` // click, long_press, double_click, input, swipe, assert, wait

	// 目标元素边框（用于回放时高亮显示）
	TargetBounds *BoundsInfo `json:"targetBounds,omitempty"`

	// 实际点击位置（用于回放时显示点击动画）
	ClickPosition *PositionInfo `json:"clickPosition,omitempty"`

	// 滑动轨迹（仅 swipe 操作）
	SwipePath *SwipePathInfo `json:"swipePath,omitempty"`

	// 输入内容（仅 input 操作）
	InputText string `json:"inputText,omitempty"`

	// 执行耗时（毫秒）
	DurationMs int64 `json:"durationMs"`

	// 错误信息（仅失败时）
	ErrorMessage  string `json:"errorMessage,omitempty"`
	FailureReason string `json:"failureReason,omitempty"` // NOT_FOUND, MULTIPLE_MATCHES, ASSERTION_FAILED, PARAM_ERROR, SYSTEM_ERROR
}

// BoundsInfo 边界信息
type BoundsInfo struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// PositionInfo 位置信息
type PositionInfo struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// SwipePathInfo 滑动轨迹信息
type SwipePathInfo struct {
	StartX int `json:"startX"`
	StartY int `json:"startY"`
	EndX   int `json:"endX"`
	EndY   int `json:"endY"`
}

// ActionResult 操作执行结果（各执行函数返回）
type ActionResult struct {
	Success       bool          // 是否成功
	Error         error         // 错误信息
	Data          interface{}   // 原始返回数据
	ClickPosition *PositionInfo // 点击位置
	TargetBounds  *BoundsInfo   // 目标边界
	InputText     string        // 输入的文本
}

// CaseExecutionResult 用例执行结果
type CaseExecutionResult struct {
	Success      bool
	ErrorMessage string
	TotalSteps   int
	PassedSteps  int
	FailedSteps  int
}

// ==================== 映射函数 ====================

// mapTaskTypeToActionType 将任务类型映射为操作类型
func mapTaskTypeToActionType(taskType string) string {
	switch taskType {
	case TaskTypeClickImage, TaskTypeClickText, TaskTypeClickNative, TaskTypeMouseClick, TaskTypeGridClick:
		return "click"
	case TaskTypeTypeText:
		return "input"
	case TaskTypeKeyPress:
		return "input"
	case TaskTypeWaitImage, TaskTypeWaitText, TaskTypeWaitTime:
		return "wait"
	case TaskTypeAssertImage, TaskTypeAssertText, TaskTypeImageExists, TaskTypeTextExists:
		return "assert"
	case TaskTypeRunPython:
		return "script"
	default:
		return "other"
	}
}

// mapFailureReasonToString 将失败原因枚举映射为字符串
func mapFailureReasonToString(reason pb.FailureReason) string {
	switch reason {
	case pb.FailureReason_FAILURE_REASON_NOT_FOUND:
		return "NOT_FOUND"
	case pb.FailureReason_FAILURE_REASON_MULTIPLE_MATCHES:
		return "MULTIPLE_MATCHES"
	case pb.FailureReason_FAILURE_REASON_ASSERTION_FAILED:
		return "ASSERTION_FAILED"
	case pb.FailureReason_FAILURE_REASON_PARAM_ERROR:
		return "PARAM_ERROR"
	case pb.FailureReason_FAILURE_REASON_SYSTEM_ERROR:
		return "SYSTEM_ERROR"
	default:
		return ""
	}
}

// mapTaskStatusToString 将任务状态枚举映射为字符串
func mapTaskStatusToString(status pb.TaskStatus) string {
	switch status {
	case pb.TaskStatus_TASK_STATUS_SUCCESS:
		return "SUCCESS"
	case pb.TaskStatus_TASK_STATUS_FAILED:
		return "FAILED"
	case pb.TaskStatus_TASK_STATUS_SKIPPED:
		return "SKIPPED"
	case pb.TaskStatus_TASK_STATUS_CANCELLED:
		return "CANCELLED"
	case pb.TaskStatus_TASK_STATUS_TIMEOUT:
		return "FAILED" // 超时也算失败
	default:
		return "UNKNOWN"
	}
}

// ==================== 日志 ====================

// LogFunc 日志函数类型
type LogFunc func(level, message string)

// 全局日志函数
var globalLogFunc LogFunc

// SetLogFunc 设置日志函数
func SetLogFunc(fn LogFunc) {
	globalLogFunc = fn
}

// log 输出日志
func log(level, message string) {
	if globalLogFunc != nil {
		globalLogFunc(level, message)
	} else {
		fmt.Printf("[%s] %s\n", level, message)
	}
}

// ==================== 执行器核心 ====================

// TaskInfo 任务信息
type TaskInfo struct {
	TaskID    string
	TaskType  string
	StartedAt int64
	CancelCh  chan struct{}
}

// Executor 任务执行器
type Executor struct {
	client       *grpc.Client
	runningTasks map[string]*TaskInfo // 运行中的任务信息
	tasksMutex   sync.Mutex
}

// NewExecutor 创建任务执行器
func NewExecutor(client *grpc.Client) *Executor {
	return &Executor{
		client:       client,
		runningTasks: make(map[string]*TaskInfo),
	}
}

// CancelTask 取消任务
func (e *Executor) CancelTask(taskID string) bool {
	e.tasksMutex.Lock()
	defer e.tasksMutex.Unlock()

	if taskInfo, exists := e.runningTasks[taskID]; exists {
		close(taskInfo.CancelCh)
		delete(e.runningTasks, taskID)
		return true
	}
	return false
}

// registerTask 注册运行中的任务
func (e *Executor) registerTask(taskID, taskType string) chan struct{} {
	e.tasksMutex.Lock()
	defer e.tasksMutex.Unlock()

	cancelCh := make(chan struct{})
	e.runningTasks[taskID] = &TaskInfo{
		TaskID:    taskID,
		TaskType:  taskType,
		StartedAt: time.Now().UnixMilli(),
		CancelCh:  cancelCh,
	}
	return cancelCh
}

// unregisterTask 注销任务
func (e *Executor) unregisterTask(taskID string) {
	e.tasksMutex.Lock()
	defer e.tasksMutex.Unlock()

	delete(e.runningTasks, taskID)
}

// GetStatus 获取执行器状态
func (e *Executor) GetStatus() (status string, currentTaskID string, currentTaskType string, taskStartedAt int64, runningCount int) {
	e.tasksMutex.Lock()
	defer e.tasksMutex.Unlock()

	runningCount = len(e.runningTasks)
	if runningCount == 0 {
		status = "IDLE"
		return
	}

	status = "BUSY"
	// 返回第一个任务的信息
	for _, info := range e.runningTasks {
		currentTaskID = info.TaskID
		currentTaskType = info.TaskType
		taskStartedAt = info.StartedAt
		break
	}
	return
}

// Execute 执行任务
func (e *Executor) Execute(taskID, taskType, payloadJSON string) {
	startTime := time.Now()

	// 日志：任务开始
	log("INFO", fmt.Sprintf("[Task:%s] 开始执行 type=%s", taskID, taskType))
	log("DEBUG", fmt.Sprintf("[Task:%s] payload=%s", taskID, truncateString(payloadJSON, 500)))

	// 注册任务，获取取消通道
	cancelCh := e.registerTask(taskID, taskType)
	defer func() {
		e.unregisterTask(taskID)
		duration := time.Since(startTime)
		log("INFO", fmt.Sprintf("[Task:%s] 执行完成 duration=%v", taskID, duration))
	}()

	// 发送任务确认
	e.sendTaskAck(taskID, true, "任务已接收")

	// 检查是否已被取消
	select {
	case <-cancelCh:
		log("WARN", fmt.Sprintf("[Task:%s] 任务在开始前被取消", taskID))
		e.sendTaskResultWithError(taskID, newTaskError(pb.TaskStatus_TASK_STATUS_CANCELLED, pb.FailureReason_FAILURE_REASON_UNSPECIFIED, "任务在开始前被取消"), nil, startTime)
		return
	default:
	}

	// 解析 payload
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		taskErr := newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_PARAM_ERROR, fmt.Sprintf("解析 payload 失败: %v", err))
		log("ERROR", fmt.Sprintf("[Task:%s] %s", taskID, taskErr.Error()))
		e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
		return
	}

	// 根据任务类型执行
	var result interface{}
	var err error

	switch taskType {
	// 批量执行类型：有自己的进度上报和结果发送逻辑，直接返回
	case TaskTypeDebugCase:
		e.executeDebugCase(taskID, payload, startTime)
		return
	case TaskTypeExecutePlan:
		e.executeExecutePlan(taskID, payload, startTime)
		return
	case TaskTypeExecuteCase:
		e.executeExecuteCase(taskID, payload, startTime)
		return
	default:
		// 单步任务：复用 executeSingleStep 统一分发
		result, err = e.executeSingleStep(taskType, payload)
	}

	// 发送结果
	if err != nil {
		taskErr := classifyError(err)
		log("ERROR", fmt.Sprintf("[Task:%s] 执行失败 status=%s reason=%s", taskID, taskErr.Status, taskErr.Reason))
		log("DEBUG", fmt.Sprintf("[Task:%s] 详细错误: %s", taskID, taskErr.Message))
		e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
	} else {
		// 尝试提取匹配位置
		var matchLoc *pb.MatchLocation
		if resultMap, ok := result.(map[string]interface{}); ok {
			if x, xOk := resultMap["x"].(int); xOk {
				if y, yOk := resultMap["y"].(int); yOk {
					matchLoc = &pb.MatchLocation{
						X: int32(x),
						Y: int32(y),
					}
					if conf, ok := resultMap["confidence"].(float64); ok {
						matchLoc.Confidence = float32(conf)
					}
				}
			}
		}

		resultJSON, _ := json.Marshal(result)
		log("INFO", fmt.Sprintf("[Task:%s] 执行成功 result=%s", taskID, truncateString(string(resultJSON), 200)))
		e.sendTaskResultSuccess(taskID, string(resultJSON), matchLoc, startTime)
	}
}

// ==================== 工具函数 ====================

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
