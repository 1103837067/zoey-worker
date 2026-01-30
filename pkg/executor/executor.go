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
	"github.com/zoeyai/zoeyworker/pkg/uia"
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
)

// ErrorType 错误类型
type ErrorType string

const (
	ErrorTypeNone           ErrorType = ""
	ErrorTypeInvalidParam   ErrorType = "INVALID_PARAM"   // 参数错误
	ErrorTypeTimeout        ErrorType = "TIMEOUT"         // 超时
	ErrorTypeNotFound       ErrorType = "NOT_FOUND"       // 未找到目标
	ErrorTypePermission     ErrorType = "PERMISSION"      // 权限不足
	ErrorTypeIO             ErrorType = "IO_ERROR"        // IO 错误
	ErrorTypeImageMatch     ErrorType = "IMAGE_MATCH"     // 图像匹配错误
	ErrorTypeOCR            ErrorType = "OCR_ERROR"       // OCR 识别错误
	ErrorTypeSystem         ErrorType = "SYSTEM_ERROR"    // 系统错误
	ErrorTypeUnsupported    ErrorType = "UNSUPPORTED"     // 不支持的操作
	ErrorTypeAssertion      ErrorType = "ASSERTION"       // 断言失败
	ErrorTypeCancelled      ErrorType = "CANCELLED"       // 任务取消
	ErrorTypeUnknown        ErrorType = "UNKNOWN"         // 未知错误
)

// TaskError 任务错误
type TaskError struct {
	Type    ErrorType
	Message string
	Detail  string
}

func (e *TaskError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Type, e.Message, e.Detail)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// newTaskError 创建任务错误
func newTaskError(errType ErrorType, message string, detail string) *TaskError {
	return &TaskError{Type: errType, Message: message, Detail: detail}
}

// classifyError 对错误进行分类
func classifyError(err error) *TaskError {
	if err == nil {
		return nil
	}
	
	errStr := err.Error()
	errLower := strings.ToLower(errStr)
	
	// 根据错误信息分类
	switch {
	case strings.Contains(errLower, "timeout") || strings.Contains(errLower, "超时"):
		return newTaskError(ErrorTypeTimeout, "操作超时", errStr)
	case strings.Contains(errLower, "not found") || strings.Contains(errLower, "未找到") || strings.Contains(errLower, "找不到"):
		return newTaskError(ErrorTypeNotFound, "目标未找到", errStr)
	case strings.Contains(errLower, "permission") || strings.Contains(errLower, "权限"):
		return newTaskError(ErrorTypePermission, "权限不足", errStr)
	case strings.Contains(errLower, "无法读取图像") || strings.Contains(errLower, "image") && strings.Contains(errLower, "read"):
		return newTaskError(ErrorTypeIO, "图像读取失败", errStr)
	case strings.Contains(errLower, "匹配失败") || strings.Contains(errLower, "match"):
		return newTaskError(ErrorTypeImageMatch, "图像匹配失败", errStr)
	case strings.Contains(errLower, "ocr") || strings.Contains(errLower, "识别"):
		return newTaskError(ErrorTypeOCR, "OCR 识别失败", errStr)
	case strings.Contains(errLower, "断言") || strings.Contains(errLower, "assert"):
		return newTaskError(ErrorTypeAssertion, "断言失败", errStr)
	case strings.Contains(errLower, "unsupported") || strings.Contains(errLower, "不支持"):
		return newTaskError(ErrorTypeUnsupported, "不支持的操作", errStr)
	case strings.Contains(errLower, "参数") || strings.Contains(errLower, "param") || strings.Contains(errLower, "缺少"):
		return newTaskError(ErrorTypeInvalidParam, "参数错误", errStr)
	default:
		return newTaskError(ErrorTypeUnknown, "执行失败", errStr)
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
		e.sendTaskResultWithError(taskID, newTaskError(ErrorTypeCancelled, "任务在开始前被取消", ""), "{}", startTime)
		return
	default:
	}

	// 解析 payload
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		taskErr := newTaskError(ErrorTypeInvalidParam, "解析 payload 失败", err.Error())
		log("ERROR", fmt.Sprintf("[Task:%s] %s", taskID, taskErr.Error()))
		e.sendTaskResultWithError(taskID, taskErr, "{}", startTime)
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
	default:
		err = fmt.Errorf("未知的任务类型: %s", taskType)
	}

	// 发送结果
	if err != nil {
		taskErr := classifyError(err)
		log("ERROR", fmt.Sprintf("[Task:%s] 执行失败 error_type=%s error=%s", taskID, taskErr.Type, taskErr.Message))
		log("DEBUG", fmt.Sprintf("[Task:%s] 详细错误: %s", taskID, taskErr.Detail))
		e.sendTaskResultWithError(taskID, taskErr, "{}", startTime)
	} else {
		resultJSON, _ := json.Marshal(result)
		log("INFO", fmt.Sprintf("[Task:%s] 执行成功 result=%s", taskID, truncateString(string(resultJSON), 200)))
		e.sendTaskResult(taskID, true, "SUCCESS", "", string(resultJSON), startTime)
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

	opts := e.parseAutoOptions(payload)
	err := auto.ClickImage(imagePath, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"clicked": true}, nil
}

// executeClickText 执行点击文字
func (e *Executor) executeClickText(payload map[string]interface{}) (interface{}, error) {
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
	appName, ok := payload["app_name"].(string)
	if !ok || appName == "" {
		return nil, fmt.Errorf("缺少 app_name 参数")
	}

	err := auto.ActivateWindow(appName)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"activated": true}, nil
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

// sendTaskResult 发送任务结果
func (e *Executor) sendTaskResult(taskID string, success bool, status, message, resultJSON string, startTime time.Time) {
	if e.client == nil {
		return
	}

	durationMs := time.Since(startTime).Milliseconds()

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("result_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
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

	e.client.SendTaskMessage(msg)
}

// sendTaskResultWithError 发送带错误类型的任务结果
func (e *Executor) sendTaskResultWithError(taskID string, taskErr *TaskError, resultJSON string, startTime time.Time) {
	if e.client == nil {
		return
	}

	durationMs := time.Since(startTime).Milliseconds()
	
	// 构建包含错误类型的结果 JSON
	errorResult := map[string]interface{}{
		"error_type": string(taskErr.Type),
		"message":    taskErr.Message,
		"detail":     taskErr.Detail,
	}
	errorJSON, _ := json.Marshal(errorResult)

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("result_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		Payload: &pb.WorkerMessage_TaskResult{
			TaskResult: &pb.TaskResult{
				TaskId:     taskID,
				Success:    false,
				Status:     string(taskErr.Type), // 使用错误类型作为状态
				Message:    taskErr.Error(),
				ResultJson: string(errorJSON),
				DurationMs: durationMs,
			},
		},
	}

	e.client.SendTaskMessage(msg)
}
