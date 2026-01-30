package executor

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"time"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	"github.com/zoeyai/zoeyworker/pkg/grpc"
	pb "github.com/zoeyai/zoeyworker/pkg/grpc/pb"
)

// TaskType 任务类型
const (
	TaskTypeClickImage    = "click_image"
	TaskTypeClickText     = "click_text"
	TaskTypeTypeText      = "type_text"
	TaskTypeKeyPress      = "key_press"
	TaskTypeScreenshot    = "screenshot"
	TaskTypeWaitImage     = "wait_image"
	TaskTypeWaitText      = "wait_text"
	TaskTypeMouseMove     = "mouse_move"
	TaskTypeMouseClick    = "mouse_click"
	TaskTypeActivateApp   = "activate_app"
	TaskTypeGridClick     = "grid_click"
	TaskTypeImageExists   = "image_exists"
	TaskTypeTextExists    = "text_exists"
	TaskTypeGetClipboard  = "get_clipboard"
	TaskTypeSetClipboard  = "set_clipboard"
)

// Executor 任务执行器
type Executor struct {
	client *grpc.Client
}

// NewExecutor 创建任务执行器
func NewExecutor(client *grpc.Client) *Executor {
	return &Executor{
		client: client,
	}
}

// Execute 执行任务
func (e *Executor) Execute(taskID, taskType, payloadJSON string) {
	startTime := time.Now()

	// 发送任务确认
	e.sendTaskAck(taskID, true, "任务已接收")

	// 解析 payload
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		e.sendTaskResult(taskID, false, "FAILED", fmt.Sprintf("解析 payload 失败: %v", err), "{}", startTime)
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
	case TaskTypeMouseMove:
		result, err = e.executeMouseMove(payload)
	case TaskTypeMouseClick:
		result, err = e.executeMouseClick(payload)
	case TaskTypeActivateApp:
		result, err = e.executeActivateApp(payload)
	case TaskTypeGridClick:
		result, err = e.executeGridClick(payload)
	case TaskTypeImageExists:
		result, err = e.executeImageExists(payload)
	case TaskTypeTextExists:
		result, err = e.executeTextExists(payload)
	case TaskTypeGetClipboard:
		result, err = e.executeGetClipboard(payload)
	case TaskTypeSetClipboard:
		result, err = e.executeSetClipboard(payload)
	default:
		err = fmt.Errorf("未知的任务类型: %s", taskType)
	}

	// 发送结果
	if err != nil {
		e.sendTaskResult(taskID, false, "FAILED", err.Error(), "{}", startTime)
	} else {
		resultJSON, _ := json.Marshal(result)
		e.sendTaskResult(taskID, true, "SUCCESS", "", string(resultJSON), startTime)
	}
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
