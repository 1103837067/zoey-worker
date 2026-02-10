//go:build darwin

// Package permissions 提供系统权限检查功能（macOS 专用）
package permissions

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework CoreGraphics
#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>
#import <CoreGraphics/CoreGraphics.h>

int checkAccessibilityPermission() {
    NSDictionary *options = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @NO};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options) ? 1 : 0;
}

int requestAccessibilityPermission() {
    NSDictionary *options = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @YES};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options) ? 1 : 0;
}

int checkScreenRecordingPermission() {
    if (@available(macOS 10.15, *)) {
        CFArrayRef windowList = CGWindowListCopyWindowInfo(
            kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
            kCGNullWindowID
        );

        if (windowList == NULL) {
            return 0;
        }

        CFIndex count = CFArrayGetCount(windowList);
        int hasNames = 0;

        for (CFIndex i = 0; i < count; i++) {
            CFDictionaryRef window = (CFDictionaryRef)CFArrayGetValueAtIndex(windowList, i);
            CFStringRef name = (CFStringRef)CFDictionaryGetValue(window, kCGWindowName);
            if (name != NULL && CFStringGetLength(name) > 0) {
                hasNames = 1;
                break;
            }
        }

        CFRelease(windowList);
        return (count == 0 || hasNames) ? 1 : 0;
    }
    return 1;
}

void openAccessibilityPreferences() {
    NSString *urlString = @"x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility";
    [[NSWorkspace sharedWorkspace] openURL:[NSURL URLWithString:urlString]];
}

void openScreenRecordingPreferences() {
    NSString *urlString = @"x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture";
    [[NSWorkspace sharedWorkspace] openURL:[NSURL URLWithString:urlString]];
}
*/
import "C"
import (
	"fmt"
	"os/exec"
)

// PermissionStatus 权限状态
type PermissionStatus struct {
	Accessibility   bool `json:"accessibility"`
	ScreenRecording bool `json:"screen_recording"`
	AllGranted      bool `json:"all_granted"`
}

// CheckPermissions 检查所需权限（不触发弹窗）
func CheckPermissions() *PermissionStatus {
	accessibility := C.checkAccessibilityPermission() == 1
	screenRecording := C.checkScreenRecordingPermission() == 1

	return &PermissionStatus{
		Accessibility:   accessibility,
		ScreenRecording: screenRecording,
		AllGranted:      accessibility && screenRecording,
	}
}

// RequestAccessibilityPermission 请求辅助功能权限（触发系统弹窗）
func RequestAccessibilityPermission() bool {
	return C.requestAccessibilityPermission() == 1
}

// OpenAccessibilitySettings 打开辅助功能设置页面
func OpenAccessibilitySettings() {
	C.openAccessibilityPreferences()
}

// OpenScreenRecordingSettings 打开屏幕录制设置页面
func OpenScreenRecordingSettings() {
	C.openScreenRecordingPreferences()
}

// GetPermissionInstructions 获取权限说明
func GetPermissionInstructions(status *PermissionStatus) string {
	if status.AllGranted {
		return ""
	}

	msg := "需要授权以下权限才能正常工作:\n\n"

	if !status.Accessibility {
		msg += "1. 辅助功能权限 (用于控制鼠标/键盘)\n"
		msg += "   系统偏好设置 > 安全性与隐私 > 隐私 > 辅助功能\n\n"
	}

	if !status.ScreenRecording {
		msg += "2. 屏幕录制权限 (用于截屏和图像识别)\n"
		msg += "   系统偏好设置 > 安全性与隐私 > 隐私 > 屏幕录制\n\n"
	}

	msg += "授权后需要重启应用才能生效。"

	return msg
}

// EnsurePermissions 确保权限已授予
func EnsurePermissions() (bool, string) {
	status := CheckPermissions()
	if status.AllGranted {
		return true, ""
	}

	return false, GetPermissionInstructions(status)
}

// PrintPermissionStatus 打印权限状态
func PrintPermissionStatus() {
	status := CheckPermissions()
	fmt.Printf("权限状态:\n")
	fmt.Printf("  辅助功能: %v\n", status.Accessibility)
	fmt.Printf("  屏幕录制: %v\n", status.ScreenRecording)

	if !status.AllGranted {
		fmt.Println(GetPermissionInstructions(status))
	}
}

// ResetPermissions 重置权限状态
func ResetPermissions() error {
	bundleID := "com.zoey.worker"

	cmd1 := fmt.Sprintf("tccutil reset Accessibility %s", bundleID)
	if err := exec.Command("sh", "-c", cmd1).Run(); err != nil {
		return fmt.Errorf("重置辅助功能权限失败: %v", err)
	}

	cmd2 := fmt.Sprintf("tccutil reset ScreenCapture %s", bundleID)
	if err := exec.Command("sh", "-c", cmd2).Run(); err != nil {
		return fmt.Errorf("重置屏幕录制权限失败: %v", err)
	}

	return nil
}
