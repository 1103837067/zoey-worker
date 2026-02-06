package grpc

import (
	"encoding/json"
	"fmt"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	"github.com/zoeyai/zoeyworker/pkg/uia"
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

// LogFunc 日志函数类型
type LogFunc func(level, message string)

// 全局日志函数
var globalLogFunc LogFunc

// SetLogFunc 设置日志函数
func SetLogFunc(fn LogFunc) {
	globalLogFunc = fn
}

// log 内部日志函数
func log(level, message string) {
	if globalLogFunc != nil {
		globalLogFunc(level, message)
	} else {
		fmt.Printf("[%s] %s\n", level, message)
	}
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
	log("DEBUG", "handleGetApplications called")

	processes, err := auto.GetProcesses()
	if err != nil {
		log("ERROR", fmt.Sprintf("GetProcesses failed: %v", err))
		return &DataResponseResult{
			RequestType: RequestTypeGetApplications,
			Success:     false,
			Message:     fmt.Sprintf("获取进程列表失败: %v", err),
			PayloadJSON: `{"applications":[]}`,
		}
	}

	log("DEBUG", fmt.Sprintf("Got %d processes", len(processes)))

	// 转换为应用程序信息格式
	type ApplicationInfo struct {
		PID   int    `json:"pid"`
		Name  string `json:"name"`
		Title string `json:"title"`
		Path  string `json:"path"`
	}

	// 获取所有窗口（只调用一次，避免在循环中重复调用）
	windows, windowsErr := auto.GetWindows()
	if windowsErr != nil {
		log("WARN", fmt.Sprintf("GetWindows failed: %v", windowsErr))
	} else {
		log("DEBUG", fmt.Sprintf("Got %d windows", len(windows)))
	}

	// 创建 PID -> 窗口标题的映射
	windowTitles := make(map[int]string)
	for _, win := range windows {
		if win.Title != "" && windowTitles[win.PID] == "" {
			windowTitles[win.PID] = win.Title
		}
	}

	apps := make([]ApplicationInfo, 0, len(processes))
	for _, proc := range processes {
		// 只返回有名称的进程
		if proc.Name != "" {
			apps = append(apps, ApplicationInfo{
				PID:   proc.PID,
				Name:  proc.Name,
				Title: windowTitles[proc.PID],
				Path:  proc.Path,
			})
		}
	}

	log("DEBUG", fmt.Sprintf("Returning %d applications", len(apps)))

	data, err := json.Marshal(map[string]interface{}{
		"applications": apps,
	})
	if err != nil {
		log("ERROR", fmt.Sprintf("JSON marshal failed: %v", err))
		return &DataResponseResult{
			RequestType: RequestTypeGetApplications,
			Success:     false,
			Message:     fmt.Sprintf("JSON序列化失败: %v", err),
			PayloadJSON: `{"applications":[]}`,
		}
	}

	log("DEBUG", fmt.Sprintf("Response size: %d bytes", len(data)))

	return &DataResponseResult{
		RequestType: RequestTypeGetApplications,
		Success:     true,
		Message:     "",
		PayloadJSON: string(data),
	}
}

// handleGetWindows 处理获取窗口列表请求
func handleGetWindows(payload map[string]interface{}) *DataResponseResult {
	// 检查权限
	permStatus := auto.CheckPermissions()
	log("DEBUG", fmt.Sprintf("Permissions: Accessibility=%v, ScreenRecording=%v", 
		permStatus.Accessibility, permStatus.ScreenRecording))
	
	log("DEBUG", fmt.Sprintf("handleGetWindows payload: %+v", payload))

	// 解析筛选参数
	var filter string
	if processName, ok := payload["process_name"].(string); ok && processName != "" {
		filter = processName
		log("DEBUG", fmt.Sprintf("Filter by process_name: %s", filter))
	}

	var windows []auto.WindowInfo
	var err error

	if filter != "" {
		windows, err = auto.GetWindows(filter)
	} else {
		windows, err = auto.GetWindows()
	}

	log("DEBUG", fmt.Sprintf("GetWindows returned %d windows, err=%v", len(windows), err))
	
	// 打印每个窗口的标题用于调试
	for i, win := range windows {
		log("DEBUG", fmt.Sprintf("Window[%d]: PID=%d, Title=%q", i, win.PID, win.Title))
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
// 使用 Python 桥接支持 Windows UI Automation
func handleGetElements(payload map[string]interface{}) *DataResponseResult {
	// 检查是否支持 UIA
	if !uia.IsSupported() {
		return &DataResponseResult{
			RequestType: RequestTypeGetElements,
			Success:     false,
			Message:     "UI 元素检查需要 Windows + Python + pywinauto 环境",
			PayloadJSON: `{"elements":[]}`,
		}
	}

	// 解析窗口句柄
	windowHandle := 0
	if handle, ok := payload["window_handle"].(float64); ok {
		windowHandle = int(handle)
	}

	if windowHandle == 0 {
		return &DataResponseResult{
			RequestType: RequestTypeGetElements,
			Success:     false,
			Message:     "缺少有效的 window_handle 参数",
			PayloadJSON: `{"elements":[]}`,
		}
	}

	// 解析筛选参数
	opts := &uia.GetElementsOptions{
		MaxDepth: 3,
	}
	if automationID, ok := payload["automation_id"].(string); ok {
		opts.AutomationID = automationID
	}
	if controlType, ok := payload["control_type"].(string); ok {
		opts.ControlType = controlType
	}
	if maxDepth, ok := payload["max_depth"].(float64); ok {
		opts.MaxDepth = int(maxDepth)
	}

	// 获取元素
	elements, err := uia.GetElements(windowHandle, opts)
	if err != nil {
		return &DataResponseResult{
			RequestType: RequestTypeGetElements,
			Success:     false,
			Message:     fmt.Sprintf("获取 UI 元素失败: %v", err),
			PayloadJSON: `{"elements":[]}`,
		}
	}

	// 转换为输出格式
	type ElementOutput struct {
		AutomationID string `json:"automation_id"`
		Name         string `json:"name"`
		ClassName    string `json:"class_name"`
		ControlType  string `json:"control_type"`
		Rect         struct {
			X      int `json:"x"`
			Y      int `json:"y"`
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"rect"`
		IsEnabled bool   `json:"is_enabled"`
		IsVisible bool   `json:"is_visible"`
		Value     string `json:"value"`
	}

	output := make([]ElementOutput, len(elements))
	for i, elem := range elements {
		output[i] = ElementOutput{
			AutomationID: elem.AutomationID,
			Name:         elem.Name,
			ClassName:    elem.ClassName,
			ControlType:  elem.ControlType,
			IsEnabled:    elem.IsEnabled,
			IsVisible:    elem.IsVisible,
			Value:        elem.Value,
		}
		output[i].Rect.X = elem.Rect.X
		output[i].Rect.Y = elem.Rect.Y
		output[i].Rect.Width = elem.Rect.Width
		output[i].Rect.Height = elem.Rect.Height
	}

	data, _ := json.Marshal(map[string]interface{}{
		"elements": output,
	})

	return &DataResponseResult{
		RequestType: RequestTypeGetElements,
		Success:     true,
		Message:     "",
		PayloadJSON: string(data),
	}
}
