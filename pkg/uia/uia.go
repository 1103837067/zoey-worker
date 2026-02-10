package uia

// GetElementsOptions UI 元素获取选项
type GetElementsOptions struct {
	AutomationID string
	ControlType  string
	MaxDepth     int
}

// ElementInfo UI 元素信息
type ElementInfo struct {
	AutomationID string
	Name         string
	ClassName    string
	ControlType  string
	Rect         Rect
	IsEnabled    bool
	IsVisible    bool
	Value        string
}

// Rect 矩形区域
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// IsSupported 检查是否支持 UI Automation
func IsSupported() bool {
	return false
}

// GetElements 获取 UI 元素列表
func GetElements(windowHandle int, opts *GetElementsOptions) ([]ElementInfo, error) {
	return nil, nil
}
