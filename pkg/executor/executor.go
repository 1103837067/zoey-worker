package executor

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	"github.com/zoeyai/zoeyworker/pkg/grpc"
	pb "github.com/zoeyai/zoeyworker/pkg/grpc/pb"
	"github.com/zoeyai/zoeyworker/pkg/plugin"
	"github.com/zoeyai/zoeyworker/pkg/uia"
	"github.com/zoeyai/zoeyworker/pkg/vision/ocr"
)

// TaskType 任务类型
const (
	TaskTypeClickImage    = "click_image"
	TaskTypeClickText     = "click_text"
	TaskTypeClickNative   = "click_native"
	TaskTypeTypeText      = "type_text"
	TaskTypeKeyPress      = "key_press"
	TaskTypeScreenshot    = "screenshot"
	TaskTypeWaitImage     = "wait_image"
	TaskTypeWaitText      = "wait_text"
	TaskTypeWaitTime      = "wait_time"
	TaskTypeMouseMove     = "mouse_move"
	TaskTypeMouseClick    = "mouse_click"
	TaskTypeActivateApp   = "activate_app"
	TaskTypeCloseApp      = "close_app"
	TaskTypeGridClick     = "grid_click"
	TaskTypeImageExists   = "image_exists"
	TaskTypeTextExists    = "text_exists"
	TaskTypeAssertImage   = "assert_image"
	TaskTypeAssertText    = "assert_text"
	TaskTypeGetClipboard  = "get_clipboard"
	TaskTypeSetClipboard  = "set_clipboard"
	// 批量执行类型
	TaskTypeDebugCase    = "debug_case"
	TaskTypeExecutePlan  = "execute_plan"  // 执行测试计划
	TaskTypeExecuteCase  = "execute_case"  // 执行单个用例
)

// 使用 pb 包中的枚举类型
// TaskStatus: pb.TaskStatus_TASK_STATUS_SUCCESS, etc.
// FailureReason: pb.FailureReason_FAILURE_REASON_NOT_FOUND, etc.

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

// TaskInfo 任务信息
type TaskInfo struct {
	TaskID    string
	TaskType  string
	StartedAt int64
	CancelCh  chan struct{}
}

// Executor 任务执行器
type Executor struct {
	client        *grpc.Client
	runningTasks  map[string]*TaskInfo // 运行中的任务信息
	tasksMutex    sync.Mutex
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
	case TaskTypeClickImage:
		result, err = e.executeClickImage(payload)
	case TaskTypeClickText:
		result, err = e.executeClickText(payload)
	case TaskTypeClickNative:
		result, err = e.executeClickNative(payload)
	case TaskTypeTypeText:
		result, err = e.executeTypeText(payload)
	case TaskTypeKeyPress:
		result, err = e.executeKeyPress(payload)
	case TaskTypeScreenshot:
		result, err = e.executeScreenshot(payload)
	case TaskTypeWaitImage:
		result, err = e.executeWaitImage(payload)
	case TaskTypeWaitText:
		result, err = e.executeWaitText(payload)
	case TaskTypeWaitTime:
		result, err = e.executeWaitTime(payload)
	case TaskTypeMouseMove:
		result, err = e.executeMouseMove(payload)
	case TaskTypeMouseClick:
		result, err = e.executeMouseClick(payload)
	case TaskTypeActivateApp:
		result, err = e.executeActivateApp(payload)
	case TaskTypeCloseApp:
		result, err = e.executeCloseApp(payload)
	case TaskTypeGridClick:
		result, err = e.executeGridClick(payload)
	case TaskTypeImageExists:
		result, err = e.executeImageExists(payload)
	case TaskTypeTextExists:
		result, err = e.executeTextExists(payload)
	case TaskTypeAssertImage:
		result, err = e.executeAssertImage(payload)
	case TaskTypeAssertText:
		result, err = e.executeAssertText(payload)
	case TaskTypeGetClipboard:
		result, err = e.executeGetClipboard(payload)
	case TaskTypeSetClipboard:
		result, err = e.executeSetClipboard(payload)
	case TaskTypeDebugCase:
		// debug_case 是特殊的批量执行任务，需要单独处理
		e.executeDebugCase(taskID, payload, startTime)
		return // 直接返回，不走下面的结果发送逻辑
	case TaskTypeExecutePlan:
		// execute_plan 执行测试计划（多个用例）
		e.executeExecutePlan(taskID, payload, startTime)
		return
	case TaskTypeExecuteCase:
		// execute_case 执行单个用例
		e.executeExecuteCase(taskID, payload, startTime)
		return
	default:
		err = fmt.Errorf("未知的任务类型: %s", taskType)
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

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// executeClickImage 执行点击图像
func (e *Executor) executeClickImage(payload map[string]interface{}) (interface{}, error) {
	imagePath, ok := payload["image"].(string)
	if !ok || imagePath == "" {
		return nil, fmt.Errorf("缺少 image 参数")
	}

	// 检查是否有网格参数
	gridStr, _ := payload["grid"].(string)
	
	opts := e.parseAutoOptions(payload)
	
	if gridStr != "" {
		// 使用网格点击
		err := auto.ClickImageWithGrid(imagePath, gridStr, opts...)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"clicked": true, "grid": gridStr}, nil
	}
	
	// 普通点击
	err := auto.ClickImage(imagePath, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"clicked": true}, nil
}

// isOCRAvailable 检查 OCR 功能是否可用（插件安装或默认配置可用）
func isOCRAvailable() bool {
	// 先检查插件是否已安装
	if plugin.GetOCRPlugin().IsInstalled() {
		return true
	}
	// 再检查默认配置（打包的模型文件）是否可用
	return ocr.IsAvailable()
}

// executeClickText 执行点击文字
func (e *Executor) executeClickText(payload map[string]interface{}) (interface{}, error) {
	// 检查 OCR 是否可用（插件或默认配置）
	if !isOCRAvailable() {
		return nil, fmt.Errorf("OCR 功能未安装，请在客户端设置中下载安装 OCR 支持")
	}

	text, ok := payload["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	opts := e.parseAutoOptions(payload)
	err := auto.ClickText(text, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"clicked": true}, nil
}

// executeTypeText 执行输入文字
func (e *Executor) executeTypeText(payload map[string]interface{}) (interface{}, error) {
	text, ok := payload["text"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	auto.TypeText(text)
	return map[string]bool{"typed": true}, nil
}

// executeKeyPress 执行按键
func (e *Executor) executeKeyPress(payload map[string]interface{}) (interface{}, error) {
	// 新格式：keys 数组 (如 ["Ctrl", "C"] 或 ["Enter"])
	if keysRaw, ok := payload["keys"].([]interface{}); ok && len(keysRaw) > 0 {
		var keys []string
		for _, k := range keysRaw {
			if s, ok := k.(string); ok {
				keys = append(keys, s)
			}
		}
		
		if len(keys) == 0 {
			return nil, fmt.Errorf("keys 数组为空")
		}
		
		// 最后一个是主键，前面的是修饰键
		if len(keys) == 1 {
			// 单个按键
			auto.KeyTap(keys[0])
		} else {
			// 组合键：前面的是修饰键，最后一个是主键
			mainKey := keys[len(keys)-1]
			modifiers := keys[:len(keys)-1]
			auto.KeyTap(mainKey, modifiers...)
		}
		
		return map[string]interface{}{"pressed": true, "keys": keys}, nil
	}
	
	// 旧格式兼容：key + modifiers
	key, ok := payload["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("缺少 key 参数")
	}

	// 解析修饰键
	var modifiers []string
	if mods, ok := payload["modifiers"].([]interface{}); ok {
		for _, m := range mods {
			if s, ok := m.(string); ok {
				modifiers = append(modifiers, s)
			}
		}
	}

	auto.KeyTap(key, modifiers...)
	return map[string]bool{"pressed": true}, nil
}

// executeScreenshot 执行截屏
func (e *Executor) executeScreenshot(payload map[string]interface{}) (interface{}, error) {
	savePath, _ := payload["save_path"].(string)

	img, err := auto.CaptureScreen()
	if err != nil {
		return nil, err
	}

	if savePath != "" {
		// 保存截图
		file, err := os.Create(savePath)
		if err != nil {
			return nil, fmt.Errorf("创建文件失败: %w", err)
		}
		defer file.Close()

		if err := png.Encode(file, img); err != nil {
			return nil, fmt.Errorf("编码图片失败: %w", err)
		}
		return map[string]string{"path": savePath}, nil
	}

	// 不保存时返回截图信息
	bounds := img.Bounds()
	return map[string]interface{}{
		"width":  bounds.Dx(),
		"height": bounds.Dy(),
	}, nil
}

// executeWaitImage 执行等待图像
func (e *Executor) executeWaitImage(payload map[string]interface{}) (interface{}, error) {
	imagePath, ok := payload["image"].(string)
	if !ok || imagePath == "" {
		return nil, fmt.Errorf("缺少 image 参数")
	}

	opts := e.parseAutoOptions(payload)
	pos, err := auto.WaitForImage(imagePath, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"found": true,
		"x":     pos.X,
		"y":     pos.Y,
	}, nil
}

// executeWaitText 执行等待文字
func (e *Executor) executeWaitText(payload map[string]interface{}) (interface{}, error) {
	// 检查 OCR 是否可用（插件或默认配置）
	if !isOCRAvailable() {
		return nil, fmt.Errorf("OCR 功能未安装，请在客户端设置中下载安装 OCR 支持")
	}

	text, ok := payload["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	opts := e.parseAutoOptions(payload)
	pos, err := auto.WaitForText(text, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"found": true,
		"x":     pos.X,
		"y":     pos.Y,
	}, nil
}

// executeMouseMove 执行鼠标移动
func (e *Executor) executeMouseMove(payload map[string]interface{}) (interface{}, error) {
	x, xOk := payload["x"].(float64)
	y, yOk := payload["y"].(float64)

	if !xOk || !yOk {
		return nil, fmt.Errorf("缺少 x 或 y 参数")
	}

	auto.MoveTo(int(x), int(y))
	return map[string]bool{"moved": true}, nil
}

// executeMouseClick 执行鼠标点击
func (e *Executor) executeMouseClick(payload map[string]interface{}) (interface{}, error) {
	x, xOk := payload["x"].(float64)
	y, yOk := payload["y"].(float64)

	if !xOk || !yOk {
		return nil, fmt.Errorf("缺少 x 或 y 参数")
	}

	double, _ := payload["double"].(bool)
	right, _ := payload["right"].(bool)

	auto.MoveTo(int(x), int(y))

	if double {
		auto.DoubleClick()
	} else if right {
		auto.RightClick()
	} else {
		auto.Click()
	}

	return map[string]bool{"clicked": true}, nil
}

// executeActivateApp 执行激活应用
func (e *Executor) executeActivateApp(payload map[string]interface{}) (interface{}, error) {
	appName, _ := payload["app_name"].(string)
	windowTitle, _ := payload["window_title"].(string)

	log("DEBUG", fmt.Sprintf("executeActivateApp: app_name='%s', window_title='%s'", appName, windowTitle))

	// 如果同时有应用名和窗口标题，使用精确匹配
	if appName != "" && windowTitle != "" {
		log("DEBUG", fmt.Sprintf("Using ActivateWindowByTitle('%s', '%s')", appName, windowTitle))
		err := auto.ActivateWindowByTitle(appName, windowTitle)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"activated": true}, nil
	}

	// 只有应用名，直接激活应用
	if appName != "" {
		log("DEBUG", fmt.Sprintf("Using ActivateWindow('%s')", appName))
		err := auto.ActivateWindow(appName)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"activated": true}, nil
	}

	// 只有窗口标题，尝试通过标题查找并激活
	if windowTitle != "" {
		log("DEBUG", fmt.Sprintf("Using ActivateWindow by title: '%s'", windowTitle))
		err := auto.ActivateWindow(windowTitle)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"activated": true}, nil
	}

	return nil, fmt.Errorf("缺少 app_name 或 window_title 参数")
}

// executeGridClick 执行网格点击
func (e *Executor) executeGridClick(payload map[string]interface{}) (interface{}, error) {
	grid, ok := payload["grid"].(string)
	if !ok || grid == "" {
		return nil, fmt.Errorf("缺少 grid 参数")
	}

	// 获取区域
	var region auto.Region
	if r, ok := payload["region"].(map[string]interface{}); ok {
		region.X = int(r["x"].(float64))
		region.Y = int(r["y"].(float64))
		region.Width = int(r["width"].(float64))
		region.Height = int(r["height"].(float64))
	} else {
		// 默认使用全屏
		w, h := auto.GetScreenSize()
		region = auto.Region{X: 0, Y: 0, Width: w, Height: h}
	}

	opts := e.parseAutoOptions(payload)
	err := auto.ClickGrid(region, grid, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"clicked": true}, nil
}

// executeImageExists 执行检查图像存在
func (e *Executor) executeImageExists(payload map[string]interface{}) (interface{}, error) {
	imagePath, ok := payload["image"].(string)
	if !ok || imagePath == "" {
		return nil, fmt.Errorf("缺少 image 参数")
	}

	opts := e.parseAutoOptions(payload)
	exists := auto.ImageExists(imagePath, opts...)

	return map[string]bool{"exists": exists}, nil
}

// executeTextExists 执行检查文字存在
func (e *Executor) executeTextExists(payload map[string]interface{}) (interface{}, error) {
	text, ok := payload["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	opts := e.parseAutoOptions(payload)
	exists := auto.TextExists(text, opts...)

	return map[string]bool{"exists": exists}, nil
}

// executeGetClipboard 执行获取剪贴板
func (e *Executor) executeGetClipboard(payload map[string]interface{}) (interface{}, error) {
	text, err := auto.ReadClipboard()
	if err != nil {
		return nil, err
	}

	return map[string]string{"text": text}, nil
}

// executeSetClipboard 执行设置剪贴板
func (e *Executor) executeSetClipboard(payload map[string]interface{}) (interface{}, error) {
	text, ok := payload["text"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	err := auto.CopyToClipboard(text)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"copied": true}, nil
}

// executeClickNative 执行原生控件点击
func (e *Executor) executeClickNative(payload map[string]interface{}) (interface{}, error) {
	// 检查是否支持 UIA
	if !uia.IsSupported() {
		return nil, fmt.Errorf("原生控件点击需要 Windows + Python + pywinauto 环境")
	}

	automationID, _ := payload["automation_id"].(string)
	windowTitle, _ := payload["window_title"].(string)

	if automationID == "" {
		return nil, fmt.Errorf("缺少 automation_id 参数")
	}

	// 获取窗口句柄
	var windowHandle int
	if windowTitle != "" {
		// 通过标题查找窗口
		windows, err := auto.GetWindows(windowTitle)
		if err != nil || len(windows) == 0 {
			return nil, fmt.Errorf("未找到窗口: %s", windowTitle)
		}
		windowHandle = windows[0].PID
	} else {
		// 获取活动窗口
		windows, err := auto.GetWindows()
		if err != nil || len(windows) == 0 {
			return nil, fmt.Errorf("未找到活动窗口")
		}
		windowHandle = windows[0].PID
	}

	// 尝试使用 UIA 点击
	err := uia.ClickElement(windowHandle, automationID)
	if err != nil {
		return nil, fmt.Errorf("点击控件失败: %w", err)
	}

	return map[string]bool{"clicked": true}, nil
}

// executeWaitTime 执行等待时间
func (e *Executor) executeWaitTime(payload map[string]interface{}) (interface{}, error) {
	duration, ok := payload["duration"].(float64)
	if !ok {
		duration = 1000 // 默认 1 秒
	}

	time.Sleep(time.Duration(duration) * time.Millisecond)
	return map[string]interface{}{"waited": true, "duration_ms": duration}, nil
}

// executeCloseApp 执行关闭应用
func (e *Executor) executeCloseApp(payload map[string]interface{}) (interface{}, error) {
	appName, ok := payload["app_name"].(string)
	if !ok || appName == "" {
		return nil, fmt.Errorf("缺少 app_name 参数")
	}

	// 查找进程并终止
	processes, err := auto.GetProcesses()
	if err != nil {
		return nil, fmt.Errorf("获取进程列表失败: %w", err)
	}

	for _, proc := range processes {
		if proc.Name == appName {
			if err := auto.KillProcess(proc.PID); err != nil {
				return nil, fmt.Errorf("终止进程失败: %w", err)
			}
			return map[string]interface{}{"closed": true, "pid": proc.PID}, nil
		}
	}

	return nil, fmt.Errorf("未找到进程: %s", appName)
}

// executeAssertImage 执行图像断言
func (e *Executor) executeAssertImage(payload map[string]interface{}) (interface{}, error) {
	imagePath, ok := payload["image"].(string)
	if !ok || imagePath == "" {
		return nil, fmt.Errorf("缺少 image 参数")
	}

	opts := e.parseAutoOptions(payload)
	exists := auto.ImageExists(imagePath, opts...)

	if !exists {
		return nil, fmt.Errorf("断言失败: 未找到指定图像")
	}

	return map[string]bool{"asserted": true, "exists": true}, nil
}

// executeAssertText 执行文字断言
func (e *Executor) executeAssertText(payload map[string]interface{}) (interface{}, error) {
	text, ok := payload["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	opts := e.parseAutoOptions(payload)
	exists := auto.TextExists(text, opts...)

	if !exists {
		return nil, fmt.Errorf("断言失败: 未找到指定文字 '%s'", text)
	}

	return map[string]bool{"asserted": true, "exists": true}, nil
}

// executeDebugCase 执行调试用例（顺序执行多个步骤）
func (e *Executor) executeDebugCase(taskID string, payload map[string]interface{}, startTime time.Time) {
	// 解析步骤列表
	stepsRaw, ok := payload["steps"].([]interface{})
	if !ok || len(stepsRaw) == 0 {
		taskErr := newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_PARAM_ERROR, "缺少 steps 参数或步骤列表为空")
		e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
		return
	}

	stopOnFail, _ := payload["stop_on_fail"].(bool)
	// 是否启用截图（默认启用，可通过 capture_screenshots: false 禁用）
	captureScreenshots := true
	if cs, ok := payload["capture_screenshots"].(bool); ok {
		captureScreenshots = cs
	}
	// 截图质量（JPEG 质量 1-100，默认 60 以减小传输量）
	screenshotQuality := 60
	if sq, ok := payload["screenshot_quality"].(float64); ok && sq > 0 && sq <= 100 {
		screenshotQuality = int(sq)
	}

	totalSteps := len(stepsRaw)

	log("INFO", fmt.Sprintf("[Task:%s] debug_case 开始，共 %d 个步骤, 截图=%v, 质量=%d", taskID, totalSteps, captureScreenshots, screenshotQuality))

	var completedSteps, passedSteps, failedSteps int32

	for i, stepRaw := range stepsRaw {
		stepMap, ok := stepRaw.(map[string]interface{})
		if !ok {
			log("WARN", fmt.Sprintf("[Task:%s] 步骤 %d 格式错误", taskID, i+1))
			continue
		}

		stepID, _ := stepMap["step_id"].(string)
		stepExecutionID, _ := stepMap["step_execution_id"].(string) // 步骤执行记录 ID（后端创建后传入）
		stepTaskType, _ := stepMap["task_type"].(string)
		stepParams, _ := stepMap["params"].(map[string]interface{})

		// 构建步骤级别的 taskID（用于前端区分每个步骤的结果）
		stepTaskID := fmt.Sprintf("step_%s_%d", stepID, time.Now().UnixMilli())

		log("INFO", fmt.Sprintf("[Task:%s] 执行步骤 %d/%d: %s (type=%s)", taskID, i+1, totalSteps, stepID, stepTaskType))

		// 发送步骤进度
		e.sendTaskProgress(taskID, int32(totalSteps), completedSteps, passedSteps, failedSteps, stepTaskType, "RUNNING")

		// 1. 执行前截图
		var screenshotBefore string
		if captureScreenshots {
			if sb, err := auto.CaptureScreenToBase64(screenshotQuality); err == nil {
				screenshotBefore = sb
			} else {
				log("WARN", fmt.Sprintf("[Task:%s] 执行前截图失败: %s", taskID, err.Error()))
			}
		}

		// 2. 执行单个步骤（增强版，返回更多信息）
		stepStartTime := time.Now()
		actionResult := e.executeSingleStepV2(stepTaskType, stepParams)
		durationMs := time.Since(stepStartTime).Milliseconds()

		// 3. 执行后截图
		var screenshotAfter string
		if captureScreenshots {
			if sa, err := auto.CaptureScreenToBase64(screenshotQuality); err == nil {
				screenshotAfter = sa
			} else {
				log("WARN", fmt.Sprintf("[Task:%s] 执行后截图失败: %s", taskID, err.Error()))
			}
		}

		completedSteps++

		// 4. 构建完整的步骤执行结果
		stepResult := &StepExecutionResult{
			StepExecutionID:  stepExecutionID,
			StepID:           stepID,
			ActionType:       mapTaskTypeToActionType(stepTaskType),
			ScreenshotBefore: screenshotBefore,
			ScreenshotAfter:  screenshotAfter,
			TargetBounds:     actionResult.TargetBounds,
			ClickPosition:    actionResult.ClickPosition,
			InputText:        actionResult.InputText,
			DurationMs:       durationMs,
		}

		if !actionResult.Success {
			failedSteps++
			taskErr := classifyError(actionResult.Error)
			log("ERROR", fmt.Sprintf("[Task:%s] 步骤 %s 执行失败: %s", taskID, stepID, taskErr.Message))

			stepResult.Status = mapTaskStatusToString(taskErr.Status)
			stepResult.ErrorMessage = taskErr.Message
			stepResult.FailureReason = mapFailureReasonToString(taskErr.Reason)

			// 发送步骤失败结果（使用增强版）
			e.sendStepResultV2(stepTaskID, stepResult)

			if stopOnFail {
				log("INFO", fmt.Sprintf("[Task:%s] stop_on_fail=true，停止执行", taskID))
				// 发送整体任务失败结果
				e.sendTaskProgress(taskID, int32(totalSteps), completedSteps, passedSteps, failedSteps, stepTaskType, "FAILED")
				e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
				return
			}
		} else {
			passedSteps++
			log("INFO", fmt.Sprintf("[Task:%s] 步骤 %s 执行成功", taskID, stepID))

			stepResult.Status = "SUCCESS"

			// 发送步骤成功结果（使用增强版）
			e.sendStepResultV2(stepTaskID, stepResult)
		}
	}

	// 所有步骤执行完成
	log("INFO", fmt.Sprintf("[Task:%s] debug_case 完成: passed=%d, failed=%d", taskID, passedSteps, failedSteps))

	// 发送最终进度和结果
	finalStatus := "SUCCESS"
	if failedSteps > 0 {
		finalStatus = "PARTIAL_FAILED"
	}
	e.sendTaskProgress(taskID, int32(totalSteps), completedSteps, passedSteps, failedSteps, "", finalStatus)

	// 发送整体任务结果
	resultJSON, _ := json.Marshal(map[string]interface{}{
		"total_steps":     totalSteps,
		"completed_steps": completedSteps,
		"passed_steps":    passedSteps,
		"failed_steps":    failedSteps,
	})

	if failedSteps > 0 {
		e.sendTaskResultWithError(taskID, newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_UNSPECIFIED, fmt.Sprintf("部分步骤失败: %d/%d", failedSteps, totalSteps)), nil, startTime)
	} else {
		e.sendTaskResultSuccess(taskID, string(resultJSON), nil, startTime)
	}
}

// executeExecutePlan 执行测试计划（顺序执行多个用例）
// payload 格式:
//
//	{
//	  "plan_execution_id": "xxx",
//	  "plan_id": "xxx",
//	  "cases": [
//	    {
//	      "case_execution_id": "xxx",
//	      "case_id": "xxx",
//	      "case_name": "用例名称",
//	      "steps": [...]  // 同 debug_case 格式
//	    }
//	  ],
//	  "stop_on_fail": true/false,
//	  "capture_screenshots": true/false,
//	  "screenshot_quality": 60
//	}
func (e *Executor) executeExecutePlan(taskID string, payload map[string]interface{}, startTime time.Time) {
	planExecutionID, _ := payload["plan_execution_id"].(string)
	planID, _ := payload["plan_id"].(string)

	// 解析用例列表
	casesRaw, ok := payload["cases"].([]interface{})
	if !ok || len(casesRaw) == 0 {
		taskErr := newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_PARAM_ERROR, "缺少 cases 参数或用例列表为空")
		e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
		return
	}

	stopOnFail, _ := payload["stop_on_fail"].(bool)
	captureScreenshots := true
	if cs, ok := payload["capture_screenshots"].(bool); ok {
		captureScreenshots = cs
	}
	screenshotQuality := 60
	if sq, ok := payload["screenshot_quality"].(float64); ok && sq > 0 && sq <= 100 {
		screenshotQuality = int(sq)
	}

	totalCases := len(casesRaw)
	log("INFO", fmt.Sprintf("[Task:%s] execute_plan 开始，计划=%s，共 %d 个用例", taskID, planID, totalCases))

	var completedCases, passedCases, failedCases int32

	for caseIdx, caseRaw := range casesRaw {
		caseMap, ok := caseRaw.(map[string]interface{})
		if !ok {
			log("WARN", fmt.Sprintf("[Task:%s] 用例 %d 格式错误", taskID, caseIdx+1))
			continue
		}

		caseExecutionID, _ := caseMap["case_execution_id"].(string)
		caseID, _ := caseMap["case_id"].(string)
		caseName, _ := caseMap["case_name"].(string)
		stepsRaw, _ := caseMap["steps"].([]interface{})

		if len(stepsRaw) == 0 {
			log("WARN", fmt.Sprintf("[Task:%s] 用例 %s 没有步骤，跳过", taskID, caseName))
			continue
		}

		log("INFO", fmt.Sprintf("[Task:%s] 执行用例 %d/%d: %s (id=%s)", taskID, caseIdx+1, totalCases, caseName, caseID))

		// 执行用例中的所有步骤
		caseResult := e.executeCaseSteps(taskID, caseExecutionID, caseID, stepsRaw, stopOnFail, captureScreenshots, screenshotQuality)

		completedCases++
		if caseResult.Success {
			passedCases++
			log("INFO", fmt.Sprintf("[Task:%s] 用例 %s 执行成功", taskID, caseName))
		} else {
			failedCases++
			log("ERROR", fmt.Sprintf("[Task:%s] 用例 %s 执行失败: %s", taskID, caseName, caseResult.ErrorMessage))

			if stopOnFail {
				log("INFO", fmt.Sprintf("[Task:%s] stop_on_fail=true，停止执行计划", taskID))
				break
			}
		}
	}

	// 所有用例执行完成
	log("INFO", fmt.Sprintf("[Task:%s] execute_plan 完成: passed=%d, failed=%d", taskID, passedCases, failedCases))

	// 发送整体结果
	resultJSON, _ := json.Marshal(map[string]interface{}{
		"plan_execution_id": planExecutionID,
		"plan_id":           planID,
		"total_cases":       totalCases,
		"completed_cases":   completedCases,
		"passed_cases":      passedCases,
		"failed_cases":      failedCases,
	})

	if failedCases > 0 {
		e.sendTaskResultWithError(taskID, newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_UNSPECIFIED, fmt.Sprintf("部分用例失败: %d/%d", failedCases, totalCases)), nil, startTime)
	} else {
		e.sendTaskResultSuccess(taskID, string(resultJSON), nil, startTime)
	}
}

// CaseExecutionResult 用例执行结果
type CaseExecutionResult struct {
	Success      bool
	ErrorMessage string
	TotalSteps   int
	PassedSteps  int
	FailedSteps  int
}

// executeCaseSteps 执行用例中的所有步骤（内部方法，供 execute_plan 和 execute_case 使用）
func (e *Executor) executeCaseSteps(taskID, caseExecutionID, caseID string, stepsRaw []interface{}, stopOnFail, captureScreenshots bool, screenshotQuality int) *CaseExecutionResult {
	result := &CaseExecutionResult{
		Success:    true,
		TotalSteps: len(stepsRaw),
	}

	for i, stepRaw := range stepsRaw {
		stepMap, ok := stepRaw.(map[string]interface{})
		if !ok {
			log("WARN", fmt.Sprintf("[Task:%s] 步骤 %d 格式错误", taskID, i+1))
			continue
		}

		stepID, _ := stepMap["step_id"].(string)
		stepExecutionID, _ := stepMap["step_execution_id"].(string)
		stepTaskType, _ := stepMap["task_type"].(string)
		stepParams, _ := stepMap["params"].(map[string]interface{})

		// 构建步骤级别的 taskID
		stepTaskID := fmt.Sprintf("step_%s_%d", stepID, time.Now().UnixMilli())

		log("INFO", fmt.Sprintf("[Task:%s] 执行步骤 %d/%d: %s (type=%s)", taskID, i+1, len(stepsRaw), stepID, stepTaskType))

		// 发送步骤进度
		e.sendTaskProgress(taskID, int32(len(stepsRaw)), int32(i), int32(result.PassedSteps), int32(result.FailedSteps), stepTaskType, "RUNNING")

		// 1. 执行前截图
		var screenshotBefore string
		if captureScreenshots {
			if sb, err := auto.CaptureScreenToBase64(screenshotQuality); err == nil {
				screenshotBefore = sb
			}
		}

		// 2. 执行步骤
		stepStartTime := time.Now()
		actionResult := e.executeSingleStepV2(stepTaskType, stepParams)
		durationMs := time.Since(stepStartTime).Milliseconds()

		// 3. 执行后截图
		var screenshotAfter string
		if captureScreenshots {
			if sa, err := auto.CaptureScreenToBase64(screenshotQuality); err == nil {
				screenshotAfter = sa
			}
		}

		// 4. 构建步骤执行结果
		stepResult := &StepExecutionResult{
			StepExecutionID:  stepExecutionID,
			StepID:           stepID,
			ActionType:       mapTaskTypeToActionType(stepTaskType),
			ScreenshotBefore: screenshotBefore,
			ScreenshotAfter:  screenshotAfter,
			TargetBounds:     actionResult.TargetBounds,
			ClickPosition:    actionResult.ClickPosition,
			InputText:        actionResult.InputText,
			DurationMs:       durationMs,
		}

		if !actionResult.Success {
			result.FailedSteps++
			taskErr := classifyError(actionResult.Error)

			stepResult.Status = mapTaskStatusToString(taskErr.Status)
			stepResult.ErrorMessage = taskErr.Message
			stepResult.FailureReason = mapFailureReasonToString(taskErr.Reason)

			// 发送步骤失败结果
			e.sendStepResultV2(stepTaskID, stepResult)

			if stopOnFail {
				result.Success = false
				result.ErrorMessage = taskErr.Message
				return result
			}
		} else {
			result.PassedSteps++
			stepResult.Status = "SUCCESS"

			// 发送步骤成功结果
			e.sendStepResultV2(stepTaskID, stepResult)
		}
	}

	// 如果有失败的步骤，标记用例失败
	if result.FailedSteps > 0 {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("部分步骤失败: %d/%d", result.FailedSteps, result.TotalSteps)
	}

	return result
}

// executeExecuteCase 执行单个用例
// payload 格式同 debug_case，但会保存执行记录到数据库
func (e *Executor) executeExecuteCase(taskID string, payload map[string]interface{}, startTime time.Time) {
	caseExecutionID, _ := payload["case_execution_id"].(string)
	caseID, _ := payload["case_id"].(string)

	// 解析步骤列表
	stepsRaw, ok := payload["steps"].([]interface{})
	if !ok || len(stepsRaw) == 0 {
		taskErr := newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_PARAM_ERROR, "缺少 steps 参数或步骤列表为空")
		e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
		return
	}

	stopOnFail := true // 默认遇到失败停止
	if sf, ok := payload["stop_on_fail"].(bool); ok {
		stopOnFail = sf
	}
	captureScreenshots := true
	if cs, ok := payload["capture_screenshots"].(bool); ok {
		captureScreenshots = cs
	}
	screenshotQuality := 60
	if sq, ok := payload["screenshot_quality"].(float64); ok && sq > 0 && sq <= 100 {
		screenshotQuality = int(sq)
	}

	log("INFO", fmt.Sprintf("[Task:%s] execute_case 开始，用例=%s，共 %d 个步骤", taskID, caseID, len(stepsRaw)))

	// 执行所有步骤
	result := e.executeCaseSteps(taskID, caseExecutionID, caseID, stepsRaw, stopOnFail, captureScreenshots, screenshotQuality)

	log("INFO", fmt.Sprintf("[Task:%s] execute_case 完成: passed=%d, failed=%d", taskID, result.PassedSteps, result.FailedSteps))

	// 发送结果
	resultJSON, _ := json.Marshal(map[string]interface{}{
		"case_execution_id": caseExecutionID,
		"case_id":           caseID,
		"total_steps":       result.TotalSteps,
		"passed_steps":      result.PassedSteps,
		"failed_steps":      result.FailedSteps,
	})

	if !result.Success {
		e.sendTaskResultWithError(taskID, newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_UNSPECIFIED, result.ErrorMessage), nil, startTime)
	} else {
		e.sendTaskResultSuccess(taskID, string(resultJSON), nil, startTime)
	}
}

// executeSingleStep 执行单个步骤（内部方法，不发送确认）
func (e *Executor) executeSingleStep(taskType string, payload map[string]interface{}) (interface{}, error) {
	switch taskType {
	case TaskTypeClickImage:
		return e.executeClickImage(payload)
	case TaskTypeClickText:
		return e.executeClickText(payload)
	case TaskTypeClickNative:
		return e.executeClickNative(payload)
	case TaskTypeTypeText:
		return e.executeTypeText(payload)
	case TaskTypeKeyPress:
		return e.executeKeyPress(payload)
	case TaskTypeScreenshot:
		return e.executeScreenshot(payload)
	case TaskTypeWaitImage:
		return e.executeWaitImage(payload)
	case TaskTypeWaitText:
		return e.executeWaitText(payload)
	case TaskTypeWaitTime:
		return e.executeWaitTime(payload)
	case TaskTypeMouseMove:
		return e.executeMouseMove(payload)
	case TaskTypeMouseClick:
		return e.executeMouseClick(payload)
	case TaskTypeActivateApp:
		return e.executeActivateApp(payload)
	case TaskTypeCloseApp:
		return e.executeCloseApp(payload)
	case TaskTypeGridClick:
		return e.executeGridClick(payload)
	case TaskTypeImageExists:
		return e.executeImageExists(payload)
	case TaskTypeTextExists:
		return e.executeTextExists(payload)
	case TaskTypeAssertImage:
		return e.executeAssertImage(payload)
	case TaskTypeAssertText:
		return e.executeAssertText(payload)
	case TaskTypeGetClipboard:
		return e.executeGetClipboard(payload)
	case TaskTypeSetClipboard:
		return e.executeSetClipboard(payload)
	default:
		return nil, fmt.Errorf("未知的任务类型: %s", taskType)
	}
}

// executeSingleStepV2 执行单个步骤（增强版，返回更多信息用于回放）
func (e *Executor) executeSingleStepV2(taskType string, payload map[string]interface{}) *ActionResult {
	result := &ActionResult{Success: true}

	// 记录输入文本（用于 type_text 等操作）
	if text, ok := payload["text"].(string); ok && taskType == TaskTypeTypeText {
		result.InputText = text
	}

	// 获取鼠标当前位置（执行前），用于某些操作的位置记录
	mouseX, mouseY := auto.GetMousePosition()

	// 执行操作
	var data interface{}
	var err error

	switch taskType {
	case TaskTypeClickImage:
		data, err = e.executeClickImageV2(payload, result)
	case TaskTypeClickText:
		data, err = e.executeClickTextV2(payload, result)
	case TaskTypeMouseClick:
		data, err = e.executeMouseClickV2(payload, result)
	case TaskTypeGridClick:
		data, err = e.executeGridClickV2(payload, result)
	default:
		// 对于其他操作，使用原始方法
		data, err = e.executeSingleStep(taskType, payload)
	}

	if err != nil {
		result.Success = false
		result.Error = err
		// 记录失败时的鼠标位置（可能有助于调试）
		if result.ClickPosition == nil {
			result.ClickPosition = &PositionInfo{X: mouseX, Y: mouseY}
		}
	}

	result.Data = data
	return result
}

// executeClickImageV2 执行点击图像（增强版，记录位置信息）
func (e *Executor) executeClickImageV2(payload map[string]interface{}, result *ActionResult) (interface{}, error) {
	imagePath, ok := payload["image"].(string)
	if !ok || imagePath == "" {
		return nil, fmt.Errorf("缺少 image 参数")
	}

	gridStr, _ := payload["grid"].(string)
	opts := e.parseAutoOptions(payload)

	if gridStr != "" {
		// 使用网格点击 - 需要先获取图像位置再计算网格位置
		// 目前先执行，后续可以增强返回点击位置
		err := auto.ClickImageWithGrid(imagePath, gridStr, opts...)
		if err != nil {
			return nil, err
		}
		// 记录点击后的鼠标位置
		x, y := auto.GetMousePosition()
		result.ClickPosition = &PositionInfo{X: x, Y: y}
		return map[string]interface{}{"clicked": true, "grid": gridStr}, nil
	}

	// 普通点击 - 使用增强版获取位置信息
	pos, err := auto.WaitForImage(imagePath, opts...)
	if err != nil {
		return nil, err
	}

	// 记录目标边界（图像匹配区域的中心点，这里简化处理）
	result.ClickPosition = &PositionInfo{X: pos.X, Y: pos.Y}

	// 执行点击
	err = auto.ClickImage(imagePath, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"clicked": true}, nil
}

// executeClickTextV2 执行点击文字（增强版，记录位置信息）
func (e *Executor) executeClickTextV2(payload map[string]interface{}, result *ActionResult) (interface{}, error) {
	// 检查 OCR 是否可用（插件或默认配置）
	if !isOCRAvailable() {
		return nil, fmt.Errorf("OCR 功能未安装，请在客户端设置中下载安装 OCR 支持")
	}

	text, ok := payload["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	opts := e.parseAutoOptions(payload)

	// 先获取文字位置
	pos, err := auto.WaitForText(text, opts...)
	if err != nil {
		return nil, err
	}

	// 记录点击位置
	result.ClickPosition = &PositionInfo{X: pos.X, Y: pos.Y}

	// 执行点击
	err = auto.ClickText(text, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"clicked": true}, nil
}

// executeMouseClickV2 执行鼠标点击（增强版，记录位置信息）
func (e *Executor) executeMouseClickV2(payload map[string]interface{}, result *ActionResult) (interface{}, error) {
	x, xOk := payload["x"].(float64)
	y, yOk := payload["y"].(float64)
	if !xOk || !yOk {
		return nil, fmt.Errorf("缺少 x 或 y 参数")
	}

	// 记录点击位置
	result.ClickPosition = &PositionInfo{X: int(x), Y: int(y)}

	auto.MoveTo(int(x), int(y))

	button, _ := payload["button"].(string)
	if button == "" {
		button = "left"
	}

	double, _ := payload["double"].(bool)
	if double {
		auto.DoubleClick(button)
	} else {
		auto.Click(button)
	}

	return map[string]bool{"clicked": true}, nil
}

// executeGridClickV2 执行网格点击（增强版，记录位置信息）
func (e *Executor) executeGridClickV2(payload map[string]interface{}, result *ActionResult) (interface{}, error) {
	gridStr, ok := payload["grid"].(string)
	if !ok || gridStr == "" {
		return nil, fmt.Errorf("缺少 grid 参数")
	}

	// 计算网格位置
	screenWidth, screenHeight := auto.GetScreenSize()
	region := auto.Region{X: 0, Y: 0, Width: screenWidth, Height: screenHeight}
	
	pos, err := auto.CalculateGridCenterFromString(region, gridStr)
	if err != nil {
		return nil, err
	}

	// 记录点击位置
	result.ClickPosition = &PositionInfo{X: pos.X, Y: pos.Y}

	// 执行点击
	auto.MoveTo(pos.X, pos.Y)
	auto.Click()

	return map[string]interface{}{"clicked": true, "grid": gridStr, "x": pos.X, "y": pos.Y}, nil
}

// sendTaskProgress 发送任务进度
func (e *Executor) sendTaskProgress(taskID string, totalSteps, completedSteps, passedSteps, failedSteps int32, currentStepName, status string) {
	if e.client == nil {
		return
	}

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("progress_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
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

	e.client.SendTaskMessage(msg)
}

// sendStepResultV2 发送单个步骤的执行结果（增强版，包含完整的回放数据）
func (e *Executor) sendStepResultV2(taskID string, result *StepExecutionResult) {
	if e.client == nil {
		return
	}

	// 序列化完整的步骤执行结果
	resultJSON, _ := json.Marshal(result)

	// 确定任务状态和失败原因
	var status pb.TaskStatus
	var failureReason pb.FailureReason
	success := result.Status == "SUCCESS"

	switch result.Status {
	case "SUCCESS":
		status = pb.TaskStatus_TASK_STATUS_SUCCESS
		failureReason = pb.FailureReason_FAILURE_REASON_UNSPECIFIED
	case "FAILED":
		status = pb.TaskStatus_TASK_STATUS_FAILED
		switch result.FailureReason {
		case "NOT_FOUND":
			failureReason = pb.FailureReason_FAILURE_REASON_NOT_FOUND
		case "MULTIPLE_MATCHES":
			failureReason = pb.FailureReason_FAILURE_REASON_MULTIPLE_MATCHES
		case "ASSERTION_FAILED":
			failureReason = pb.FailureReason_FAILURE_REASON_ASSERTION_FAILED
		case "PARAM_ERROR":
			failureReason = pb.FailureReason_FAILURE_REASON_PARAM_ERROR
		case "SYSTEM_ERROR":
			failureReason = pb.FailureReason_FAILURE_REASON_SYSTEM_ERROR
		default:
			failureReason = pb.FailureReason_FAILURE_REASON_UNSPECIFIED
		}
	case "SKIPPED":
		status = pb.TaskStatus_TASK_STATUS_SKIPPED
		failureReason = pb.FailureReason_FAILURE_REASON_UNSPECIFIED
	default:
		status = pb.TaskStatus_TASK_STATUS_FAILED
		failureReason = pb.FailureReason_FAILURE_REASON_UNSPECIFIED
	}

	// 构建 MatchLocation（如果有目标边界信息）
	var matchLoc *pb.MatchLocation
	if result.TargetBounds != nil {
		matchLoc = &pb.MatchLocation{
			X:      int32(result.TargetBounds.X),
			Y:      int32(result.TargetBounds.Y),
			Width:  int32(result.TargetBounds.Width),
			Height: int32(result.TargetBounds.Height),
		}
	}

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("step_result_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		Payload: &pb.WorkerMessage_TaskResult{
			TaskResult: &pb.TaskResult{
				TaskId:        taskID,
				Success:       success,
				Status:        status,
				Message:       result.ErrorMessage,
				ResultJson:    string(resultJSON),
				DurationMs:    result.DurationMs,
				FailureReason: failureReason,
				MatchLocation: matchLoc,
			},
		},
	}

	e.client.SendTaskMessage(msg)
}

// sendStepResult 发送单个步骤的执行结果（保留旧版本兼容性）
func (e *Executor) sendStepResult(taskID, stepID string, success bool, status pb.TaskStatus, message, resultJSON string, durationMs int64, failureReason pb.FailureReason) {
	if e.client == nil {
		return
	}

	// 使用 TaskResult 发送步骤结果，但在 ResultJson 中包含 step_id 信息
	resultWithStep, _ := json.Marshal(map[string]interface{}{
		"step_id": stepID,
		"result":  json.RawMessage(resultJSON),
	})

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("step_result_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		Payload: &pb.WorkerMessage_TaskResult{
			TaskResult: &pb.TaskResult{
				TaskId:        taskID,
				Success:       success,
				Status:        status,
				Message:       message,
				ResultJson:    string(resultWithStep),
				DurationMs:    durationMs,
				FailureReason: failureReason,
			},
		},
	}

	e.client.SendTaskMessage(msg)
}

// parseAutoOptions 解析自动化选项
func (e *Executor) parseAutoOptions(payload map[string]interface{}) []auto.Option {
	var opts []auto.Option

	if timeout, ok := payload["timeout"].(float64); ok {
		opts = append(opts, auto.WithTimeout(time.Duration(timeout)*time.Second))
	}

	if threshold, ok := payload["threshold"].(float64); ok {
		opts = append(opts, auto.WithThreshold(threshold))
	}

	if double, ok := payload["double"].(bool); ok && double {
		opts = append(opts, auto.WithDoubleClick())
	}

	if right, ok := payload["right"].(bool); ok && right {
		opts = append(opts, auto.WithRightClick())
	}

	// RGB 三通道校验（默认开启，可通过 rgb: false 关闭）
	if rgb, ok := payload["rgb"].(bool); ok {
		opts = append(opts, auto.WithRGB(rgb))
	}

	return opts
}

// sendTaskAck 发送任务确认
func (e *Executor) sendTaskAck(taskID string, accepted bool, message string) {
	if e.client == nil {
		return
	}

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("ack_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		Payload: &pb.WorkerMessage_TaskAck{
			TaskAck: &pb.TaskAck{
				TaskId:   taskID,
				Accepted: accepted,
				Message:  message,
			},
		},
	}

	e.client.SendTaskMessage(msg)
}

// sendTaskResultSuccess 发送成功结果
func (e *Executor) sendTaskResultSuccess(taskID string, resultJSON string, matchLoc *pb.MatchLocation, startTime time.Time) {
	if e.client == nil {
		return
	}

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("result_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		Payload: &pb.WorkerMessage_TaskResult{
			TaskResult: &pb.TaskResult{
				TaskId:        taskID,
				Success:       true,
				Status:        pb.TaskStatus_TASK_STATUS_SUCCESS,
				Message:       "",
				ResultJson:    resultJSON,
				DurationMs:    time.Since(startTime).Milliseconds(),
				FailureReason: pb.FailureReason_FAILURE_REASON_UNSPECIFIED,
				MatchLocation: matchLoc,
			},
		},
	}

	e.client.SendTaskMessage(msg)
}

// sendTaskResultWithError 发送失败结果
func (e *Executor) sendTaskResultWithError(taskID string, taskErr *TaskError, matchLoc *pb.MatchLocation, startTime time.Time) {
	if e.client == nil {
		return
	}

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("result_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		Payload: &pb.WorkerMessage_TaskResult{
			TaskResult: &pb.TaskResult{
				TaskId:        taskID,
				Success:       false,
				Status:        taskErr.Status,
				Message:       taskErr.Message,
				ResultJson:    "{}",
				DurationMs:    time.Since(startTime).Milliseconds(),
				FailureReason: taskErr.Reason,
				MatchLocation: matchLoc,
			},
		},
	}

	e.client.SendTaskMessage(msg)
}
