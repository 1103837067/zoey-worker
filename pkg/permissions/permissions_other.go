//go:build !darwin

// Package permissions 提供系统权限检查功能
package permissions

// PermissionStatus 权限状态
type PermissionStatus struct {
	Accessibility   bool `json:"accessibility"`
	ScreenRecording bool `json:"screen_recording"`
	AllGranted      bool `json:"all_granted"`
}

// CheckPermissions 检查所需权限
// 非 macOS 系统通常不需要特殊权限
func CheckPermissions() *PermissionStatus {
	return &PermissionStatus{
		Accessibility:   true,
		ScreenRecording: true,
		AllGranted:      true,
	}
}

// RequestAccessibilityPermission 请求辅助功能权限
func RequestAccessibilityPermission() bool {
	return true
}

// OpenAccessibilitySettings 打开辅助功能设置页面
func OpenAccessibilitySettings() {}

// OpenScreenRecordingSettings 打开屏幕录制设置页面
func OpenScreenRecordingSettings() {}

// GetPermissionInstructions 获取权限说明
func GetPermissionInstructions(status *PermissionStatus) string {
	return ""
}

// EnsurePermissions 确保权限已授予
func EnsurePermissions() (bool, string) {
	return true, ""
}

// PrintPermissionStatus 打印权限状态
func PrintPermissionStatus() {}

// ResetPermissions 重置权限状态
func ResetPermissions() error {
	return nil
}
