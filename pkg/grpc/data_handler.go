package grpc

import (
	"encoding/json"
	"fmt"

	"github.com/zoeyai/zoeyworker/pkg/auto"
)

// DataRequestType 数据请求类型
const (
	RequestTypeGetApplications = "GET_APPLICATIONS"
	RequestTypeGetWindows      = "GET_WINDOWS"
	RequestTypeGetElements     = "GET_ELEMENTS"
)

// DataResponseResult 数据响应结果
type DataResponseResult struct {
	RequestType string
	Success     bool
	Message     string
	PayloadJSON string
}

// HandleDataRequest 处理数据请求
func HandleDataRequest(requestType string, payloadJSON string) *DataResponseResult {
	var payload map[string]interface{}
	if payloadJSON != "" && payloadJSON != "{}" {
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			payload = make(map[string]interface{})
		}
	} else {
		payload = make(map[string]interface{})
	}

	switch requestType {
	case RequestTypeGetApplications:
		return handleGetApplications()
	case RequestTypeGetWindows:
		return handleGetWindows(payload)
	case RequestTypeGetElements:
		return handleGetElements(payload)
	default:
		return &DataResponseResult{
			RequestType: requestType,
			Success:     false,
			Message:     fmt.Sprintf("未知的请求类型: %s", requestType),
			PayloadJSON: "{}",
		}
	}
}

// handleGetApplications 处理获取应用程序列表请求
func handleGetApplications() *DataResponseResult {
	processes, err := auto.GetProcesses()
	if err != nil {
		return &DataResponseResult{
			RequestType: RequestTypeGetApplications,
			Success:     false,
			Message:     fmt.Sprintf("获取进程列表失败: %v", err),
			PayloadJSON: `{"applications":[]}`,
		}
	}

	// 转换为应用程序信息格式
	type ApplicationInfo struct {
		PID   int    `json:"pid"`
		Name  string `json:"name"`
		Title string `json:"title"`
		Path  string `json:"path"`
	}

	apps := make([]ApplicationInfo, 0, len(processes))
	for _, proc := range processes {
		// 获取窗口标题（如果有）
		title := ""
		windows, _ := auto.GetWindows()
		for _, win := range windows {
			if win.PID == proc.PID {
				title = win.Title
				break
			}
		}

		// 只返回有名称的进程
		if proc.Name != "" {
			apps = append(apps, ApplicationInfo{
				PID:   proc.PID,
				Name:  proc.Name,
				Title: title,
				Path:  proc.Path,
			})
		}
	}

	data, _ := json.Marshal(map[string]interface{}{
		"applications": apps,
	})

	return &DataResponseResult{
		RequestType: RequestTypeGetApplications,
		Success:     true,
		Message:     "",
		PayloadJSON: string(data),
	}
}

// handleGetWindows 处理获取窗口列表请求
func handleGetWindows(payload map[string]interface{}) *DataResponseResult {
	// 解析筛选参数
	var filter string
	if processName, ok := payload["process_name"].(string); ok && processName != "" {
		filter = processName
	}

	var windows []auto.WindowInfo
	var err error

	if filter != "" {
		windows, err = auto.GetWindows(filter)
	} else {
		windows, err = auto.GetWindows()
	}

	if err != nil {
		return &DataResponseResult{
			RequestType: RequestTypeGetWindows,
			Success:     false,
			Message:     fmt.Sprintf("获取窗口列表失败: %v", err),
			PayloadJSON: `{"windows":[]}`,
		}
	}

	// 转换为 proto 格式
	type WindowInfoOutput struct {
		Handle    int    `json:"handle"`
		Title     string `json:"title"`
		ClassName string `json:"class_name"`
		PID       int    `json:"pid"`
		IsVisible bool   `json:"is_visible"`
		Rect      struct {
			X      int `json:"x"`
			Y      int `json:"y"`
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"rect"`
	}

	output := make([]WindowInfoOutput, 0, len(windows))
	for _, win := range windows {
		w := WindowInfoOutput{
			Handle:    win.PID, // Go 版本使用 PID 作为句柄
			Title:     win.Title,
			ClassName: "",           // robotgo 不提供 class_name
			PID:       win.PID,
			IsVisible: win.Title != "", // 有标题视为可见
		}
		w.Rect.X = win.Bounds.X
		w.Rect.Y = win.Bounds.Y
		w.Rect.Width = win.Bounds.Width
		w.Rect.Height = win.Bounds.Height
		output = append(output, w)
	}

	data, _ := json.Marshal(map[string]interface{}{
		"windows": output,
	})

	return &DataResponseResult{
		RequestType: RequestTypeGetWindows,
		Success:     true,
		Message:     "",
		PayloadJSON: string(data),
	}
}

// handleGetElements 处理获取 UI 元素请求
// 注意: UI 元素检查依赖平台特定 API (Windows: pywinauto)
// Go 版本暂不支持，返回空列表
func handleGetElements(payload map[string]interface{}) *DataResponseResult {
	return &DataResponseResult{
		RequestType: RequestTypeGetElements,
		Success:     false,
		Message:     "UI 元素检查在 Go 版本中暂不支持（需要平台特定 API）",
		PayloadJSON: `{"elements":[]}`,
	}
}
