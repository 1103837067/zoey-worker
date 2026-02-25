package executor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zoeyai/zoeyworker/pkg/auto/input"
	"github.com/zoeyai/zoeyworker/pkg/auto/screen"
	pb "github.com/zoeyai/zoeyworker/pkg/grpc/pb"
)

const TaskTypeAIAction = "ai_action"

// aiActionPayload AI 动作的 payload 结构
type aiActionPayload struct {
	Action            string                 `json:"action"`
	Parameters        map[string]interface{} `json:"parameters"`
	ScreenshotQuality int                    `json:"screenshot_quality"`
}

// AIActionResult AI 动作的返回结构
type AIActionResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	Screenshot   string `json:"screenshot"`
	ScreenWidth  int    `json:"screen_width"`
	ScreenHeight int    `json:"screen_height"`
}

// mapNormalizedCoord 将 0-1000 归一化坐标映射到实际屏幕分辨率
func mapNormalizedCoord(nx, ny float64) (int, int) {
	sw, sh := screen.GetScreenSize()
	realX := int(nx / 1000.0 * float64(sw))
	realY := int(ny / 1000.0 * float64(sh))
	return realX, realY
}

func getFloat(params map[string]interface{}, key string, fallback float64) float64 {
	if v, ok := params[key].(float64); ok {
		return v
	}
	return fallback
}

func getString(params map[string]interface{}, key string, fallback string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return fallback
}

func getStringSlice(params map[string]interface{}, key string) []string {
	raw, ok := params[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case string:
		return []string{v}
	}
	return nil
}

// executeAIAction 执行 AI 动作并自动截屏返回
func (e *Executor) executeAIAction(taskID string, payload map[string]interface{}, startTime time.Time) {
	var ap aiActionPayload
	raw, _ := json.Marshal(payload)
	if err := json.Unmarshal(raw, &ap); err != nil {
		taskErr := newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_PARAM_ERROR, fmt.Sprintf("解析 ai_action payload 失败: %v", err))
		e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
		return
	}

	if ap.ScreenshotQuality <= 0 || ap.ScreenshotQuality > 100 {
		ap.ScreenshotQuality = 60
	}
	if ap.Parameters == nil {
		ap.Parameters = map[string]interface{}{}
	}

	msg, err := executeAIOperation(ap.Action, ap.Parameters)
	if err != nil {
		taskErr := newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_SYSTEM_ERROR, fmt.Sprintf("AI 动作执行失败: %v", err))
		e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
		return
	}

	screenshot, screenshotErr := screen.CaptureScreenToBase64(ap.ScreenshotQuality)
	if screenshotErr != nil {
		log("WARN", fmt.Sprintf("[Task:%s] 截屏失败: %v", taskID, screenshotErr))
	}

	sw, sh := screen.GetScreenSize()
	result := AIActionResult{
		Success:      true,
		Message:      msg,
		Screenshot:   screenshot,
		ScreenWidth:  sw,
		ScreenHeight: sh,
	}

	resultJSON, _ := json.Marshal(result)
	e.sendTaskResultSuccess(taskID, string(resultJSON), nil, startTime)
}

// executeAIOperation 根据动作类型执行对应操作
func executeAIOperation(action string, params map[string]interface{}) (string, error) {
	switch action {
	case "click":
		x, y := mapNormalizedCoord(getFloat(params, "x", 500), getFloat(params, "y", 500))
		input.MoveTo(x, y)
		time.Sleep(50 * time.Millisecond)
		input.Click()
		return fmt.Sprintf("Clicked at (%d, %d)", x, y), nil

	case "double_click":
		x, y := mapNormalizedCoord(getFloat(params, "x", 500), getFloat(params, "y", 500))
		input.MoveTo(x, y)
		time.Sleep(50 * time.Millisecond)
		input.DoubleClick()
		return fmt.Sprintf("Double clicked at (%d, %d)", x, y), nil

	case "right_click":
		x, y := mapNormalizedCoord(getFloat(params, "x", 500), getFloat(params, "y", 500))
		input.MoveTo(x, y)
		time.Sleep(50 * time.Millisecond)
		input.RightClick()
		return fmt.Sprintf("Right clicked at (%d, %d)", x, y), nil

	case "type":
		text := getString(params, "text", "")
		if text == "" {
			return "", fmt.Errorf("缺少 text 参数")
		}
		input.TypeText(text)
		return fmt.Sprintf("Typed: %s", truncateString(text, 50)), nil

	case "press":
		keys := getStringSlice(params, "keys")
		if len(keys) == 0 {
			return "", fmt.Errorf("缺少 keys 参数")
		}
		input.HotKey(keys...)
		return fmt.Sprintf("Pressed: %v", keys), nil

	case "scroll":
		amount := int(getFloat(params, "amount", 3))
		if nx, ok := params["x"]; ok {
			if ny, ok := params["y"]; ok {
				x, y := mapNormalizedCoord(nx.(float64), ny.(float64))
				input.MoveTo(x, y)
				time.Sleep(50 * time.Millisecond)
			}
		}
		input.Scroll(0, amount)
		return fmt.Sprintf("Scrolled: %d", amount), nil

	case "drag":
		startX, startY := mapNormalizedCoord(getFloat(params, "start_x", 500), getFloat(params, "start_y", 500))
		endX, endY := mapNormalizedCoord(getFloat(params, "end_x", 500), getFloat(params, "end_y", 500))
		input.MoveTo(startX, startY)
		time.Sleep(100 * time.Millisecond)
		input.Drag(endX, endY)
		return fmt.Sprintf("Dragged from (%d, %d) to (%d, %d)", startX, startY, endX, endY), nil

	case "move":
		x, y := mapNormalizedCoord(getFloat(params, "x", 500), getFloat(params, "y", 500))
		input.MoveSmooth(x, y)
		return fmt.Sprintf("Moved to (%d, %d)", x, y), nil

	case "wait":
		seconds := getFloat(params, "seconds", 1.0)
		time.Sleep(time.Duration(seconds*1000) * time.Millisecond)
		return fmt.Sprintf("Waited for %.1f seconds", seconds), nil

	case "screenshot":
		return "Screenshot captured", nil

	case "task_complete":
		result := getString(params, "result", "Task completed")
		return fmt.Sprintf("Task completed: %s", result), nil

	default:
		return "", fmt.Errorf("未知动作类型: %s", action)
	}
}
