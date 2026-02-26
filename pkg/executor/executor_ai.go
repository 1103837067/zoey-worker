package executor

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"time"

	"github.com/go-vgo/robotgo"
	pb "github.com/zoeyai/zoeyworker/pkg/grpc/pb"
)

const TaskTypeAIAction = "ai_action"

type aiActionPayload struct {
	Action             string                 `json:"action"`
	Parameters         map[string]interface{} `json:"parameters"`
	ScreenshotQuality  int                    `json:"screenshot_quality"`
	CoordinateStrategy string                 `json:"coordinate_strategy"`
}

type AIActionResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	Screenshot   string `json:"screenshot"`
	ScreenWidth  int    `json:"screen_width"`
	ScreenHeight int    `json:"screen_height"`
}

// 截图尺寸，由 captureScreenBase64 更新，getCoord 直接用
var screenW, screenH int

func captureScreenBase64(quality int) (string, int, int, error) {
	img, err := captureScreenWithCursor()
	if err != nil {
		return "", 0, 0, err
	}

	bounds := img.Bounds()
	screenW = bounds.Dx()
	screenH = bounds.Dy()
	log("DEBUG", fmt.Sprintf("[截图] %dx%d", screenW, screenH))

	if quality <= 0 || quality > 100 {
		quality = 80
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return "", 0, 0, fmt.Errorf("JPEG 编码失败: %w", err)
	}

	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "data:image/jpeg;base64," + b64, screenW, screenH, nil
}

func getCoord(params map[string]interface{}, strategy string) (int, int) {
	switch strategy {
	case "qwen-vl":
		// 绝对像素坐标，直接使用
		x := int(getFloat(params, "x", 0))
		y := int(getFloat(params, "y", 0))
		log("DEBUG", fmt.Sprintf("[坐标:qwen-vl] 像素(%d,%d)", x, y))
		return x, y

	case "ui-tars":
		// 0-1 归一化
		nx := getFloat(params, "x", 0.5)
		ny := getFloat(params, "y", 0.5)
		x := int(nx * float64(screenW))
		y := int(ny * float64(screenH))
		log("DEBUG", fmt.Sprintf("[坐标:ui-tars] (%.3f,%.3f) -> (%d,%d) 屏幕=%dx%d", nx, ny, x, y, screenW, screenH))
		return x, y

	case "gemini":
		// bbox [y1, x1, y2, x2] 0-1000，取中心点
		bbox := getFloatSlice(params, "bbox")
		if len(bbox) >= 4 {
			cy := (bbox[0] + bbox[2]) / 2.0
			cx := (bbox[1] + bbox[3]) / 2.0
			x := int(cx / 1000.0 * float64(screenW))
			y := int(cy / 1000.0 * float64(screenH))
			log("DEBUG", fmt.Sprintf("[坐标:gemini] bbox[%.0f,%.0f,%.0f,%.0f] -> (%d,%d) 屏幕=%dx%d", bbox[0], bbox[1], bbox[2], bbox[3], x, y, screenW, screenH))
			return x, y
		}
		log("WARN", "[坐标:gemini] bbox 格式错误，使用屏幕中心")
		return screenW / 2, screenH / 2

	case "qwen3-vl":
		// bbox [x1, y1, x2, y2] 0-1000，取中心点
		bbox := getFloatSlice(params, "bbox")
		if len(bbox) >= 4 {
			cx := (bbox[0] + bbox[2]) / 2.0
			cy := (bbox[1] + bbox[3]) / 2.0
			x := int(cx / 1000.0 * float64(screenW))
			y := int(cy / 1000.0 * float64(screenH))
			log("DEBUG", fmt.Sprintf("[坐标:qwen3-vl] bbox[%.0f,%.0f,%.0f,%.0f] -> (%d,%d) 屏幕=%dx%d", bbox[0], bbox[1], bbox[2], bbox[3], x, y, screenW, screenH))
			return x, y
		}
		log("WARN", "[坐标:qwen3-vl] bbox 格式错误，使用屏幕中心")
		return screenW / 2, screenH / 2

	default:
		// normalized-1000 / doubao: 0-1000 归一化
		nx := getFloat(params, "x", 500)
		ny := getFloat(params, "y", 500)
		x := int(nx / 1000.0 * float64(screenW))
		y := int(ny / 1000.0 * float64(screenH))
		log("DEBUG", fmt.Sprintf("[坐标] (%.0f,%.0f) -> (%d,%d) 屏幕=%dx%d", nx, ny, x, y, screenW, screenH))
		return x, y
	}
}

func getFloat(params map[string]interface{}, key string, fallback float64) float64 {
	if v, ok := params[key].(float64); ok {
		return v
	}
	return fallback
}

func getFloatSlice(params map[string]interface{}, key string) []float64 {
	raw, ok := params[key]
	if !ok {
		return nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	result := make([]float64, 0, len(arr))
	for _, item := range arr {
		if v, ok := item.(float64); ok {
			result = append(result, v)
		}
	}
	return result
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

func executeAIOperation(action string, params map[string]interface{}, strategy string) (string, error) {
	switch action {
	case "click":
		x, y := getCoord(params, strategy)
		setCursorPos(x, y)
		time.Sleep(50 * time.Millisecond)
		robotgo.Click()
		return fmt.Sprintf("Clicked at (%d, %d)", x, y), nil

	case "double_click":
		x, y := getCoord(params, strategy)
		setCursorPos(x, y)
		time.Sleep(50 * time.Millisecond)
		robotgo.Click("left", true)
		return fmt.Sprintf("Double clicked at (%d, %d)", x, y), nil

	case "right_click":
		x, y := getCoord(params, strategy)
		setCursorPos(x, y)
		time.Sleep(50 * time.Millisecond)
		robotgo.Click("right")
		return fmt.Sprintf("Right clicked at (%d, %d)", x, y), nil

	case "type":
		text, _ := params["text"].(string)
		if text == "" {
			return "", fmt.Errorf("缺少 text 参数")
		}
		robotgo.TypeStr(text)
		return fmt.Sprintf("Typed: %s", text), nil

	case "press":
		keys := getStringSlice(params, "keys")
		if len(keys) == 0 {
			return "", fmt.Errorf("缺少 keys 参数")
		}
		if len(keys) == 1 {
			robotgo.KeyTap(keys[0])
		} else {
			args := make([]interface{}, len(keys)-1)
			for i, k := range keys[1:] {
				args[i] = k
			}
			robotgo.KeyTap(keys[0], args...)
		}
		return fmt.Sprintf("Pressed: %v", keys), nil

	case "scroll":
		amount := int(getFloat(params, "amount", 3))
		robotgo.Scroll(0, amount)
		return fmt.Sprintf("Scrolled: %d", amount), nil

	case "drag":
		var sx, sy, ex, ey int
		if strategy == "gemini" || strategy == "qwen3-vl" {
			sx, sy = getCoord(map[string]interface{}{"bbox": params["start_bbox"]}, strategy)
			ex, ey = getCoord(map[string]interface{}{"bbox": params["end_bbox"]}, strategy)
		} else {
			sx, sy = getCoord(map[string]interface{}{"x": getFloat(params, "start_x", 500), "y": getFloat(params, "start_y", 500)}, strategy)
			ex, ey = getCoord(map[string]interface{}{"x": getFloat(params, "end_x", 500), "y": getFloat(params, "end_y", 500)}, strategy)
		}
		dragSmooth(sx, sy, ex, ey)
		return fmt.Sprintf("Dragged (%d,%d)->(%d,%d)", sx, sy, ex, ey), nil

	case "move":
		x, y := getCoord(params, strategy)
		setCursorPos(x, y)
		return fmt.Sprintf("Moved to (%d, %d)", x, y), nil

	case "wait":
		seconds := getFloat(params, "seconds", 1.0)
		time.Sleep(time.Duration(seconds*1000) * time.Millisecond)
		return fmt.Sprintf("Waited %.1fs", seconds), nil

	case "screenshot":
		return "Screenshot captured", nil

	case "task_complete":
		result, _ := params["result"].(string)
		return fmt.Sprintf("Task completed: %s", result), nil

	default:
		return "", fmt.Errorf("未知动作: %s", action)
	}
}

func (e *Executor) executeAIAction(taskID string, payload map[string]interface{}, startTime time.Time) {
	var ap aiActionPayload
	raw, _ := json.Marshal(payload)
	if err := json.Unmarshal(raw, &ap); err != nil {
		taskErr := newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_PARAM_ERROR, fmt.Sprintf("解析失败: %v", err))
		e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
		return
	}
	if ap.ScreenshotQuality <= 0 {
		ap.ScreenshotQuality = 90
	}
	if ap.Parameters == nil {
		ap.Parameters = map[string]interface{}{}
	}

	strategy := ap.CoordinateStrategy
	if strategy == "" {
		strategy = "normalized-1000"
	}

	msg, err := executeAIOperation(ap.Action, ap.Parameters, strategy)
	if err != nil {
		taskErr := newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_SYSTEM_ERROR, fmt.Sprintf("操作失败: %v", err))
		e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
		return
	}

	time.Sleep(500 * time.Millisecond)

	screenshot, sw, sh, screenshotErr := captureScreenBase64(ap.ScreenshotQuality)
	if screenshotErr != nil {
		log("WARN", fmt.Sprintf("[Task:%s] 截屏失败: %v", taskID, screenshotErr))
	}

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
