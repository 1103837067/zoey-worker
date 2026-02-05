//go:build !darwin

package auto

// PermissionStatus 权限状态
type PermissionStatus struct {
	Accessibility    bool `json:"accessibility"`
	ScreenRecording  bool `json:"screen_recording"`
	AllGranted       bool `json:"all_granted"`
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
func OpenAccessibilitySettings() {
	// 非 macOS 不需要
}

// OpenScreenRecordingSettings 打开屏幕录制设置页面
func OpenScreenRecordingSettings() {
	// 非 macOS 不需要
}

// GetPermissionInstructions 获取权限说明
func GetPermissionInstructions(status *PermissionStatus) string {
	return ""
}

// EnsurePermissions 确保权限已授予
func EnsurePermissions() (bool, string) {
	return true, ""
}

// PrintPermissionStatus 打印权限状态
func PrintPermissionStatus() {
	// 非 macOS 不需要打印
}

// ResetPermissions 重置权限状态
func ResetPermissions() error {
	// 非 macOS 不需要重置权限
	return nil
}
