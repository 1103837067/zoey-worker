//go:build darwin

package auto

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework Cocoa -framework AppKit
#import <CoreGraphics/CoreGraphics.h>
#import <Cocoa/Cocoa.h>
#import <AppKit/AppKit.h>

// 窗口信息结构
typedef struct {
    int pid;
    int windowId;
    int x;
    int y;
    int width;
    int height;
    char title[512];
    char ownerName[256];
} WindowInfoC;

// 通过 PID 激活应用窗口
int activateAppByPID(int pid) {
    NSRunningApplication* app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
    if (app == nil) {
        return 0;
    }
    // 使用 NSApplicationActivateAllWindows 替代废弃的 NSApplicationActivateIgnoringOtherApps
    // macOS 14+ 中 ignoringOtherApps 已被废弃且无效
    [app activateWithOptions:NSApplicationActivateAllWindows];
    return 1;
}

// 通过 PID 获取应用的 Bundle ID
const char* getBundleIDByPID(int pid) {
    NSRunningApplication* app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
    if (app == nil) {
        return "";
    }
    NSString* bundleID = [app bundleIdentifier];
    if (bundleID == nil) {
        return "";
    }
    return [bundleID UTF8String];
}

// 通过应用名称激活窗口
int activateAppByName(const char* name) {
    NSString* appName = [NSString stringWithUTF8String:name];
    NSArray* apps = [NSRunningApplication runningApplicationsWithBundleIdentifier:appName];
    
    // 如果 bundle ID 找不到，尝试用名称匹配
    if ([apps count] == 0) {
        NSArray* allApps = [[NSWorkspace sharedWorkspace] runningApplications];
        for (NSRunningApplication* app in allApps) {
            NSString* localizedName = [app localizedName];
            if (localizedName && [localizedName localizedCaseInsensitiveContainsString:appName]) {
                [app activateWithOptions:NSApplicationActivateAllWindows];
                return 1;
            }
        }
        return 0;
    }
    
    NSRunningApplication* app = [apps firstObject];
    [app activateWithOptions:NSApplicationActivateAllWindows];
    return 1;
}

// 获取窗口列表 (不触发权限弹窗)
// 注意：macOS 的标签页（如浏览器标签）不是独立窗口，需要辅助功能API获取
// 这里返回的是实际的窗口，每个浏览器窗口（可能包含多个标签）是一个条目
int getWindowList(WindowInfoC* windows, int maxCount) {
    // 使用 CGWindowListCopyWindowInfo 获取窗口列表
    // 使用 kCGWindowListOptionAll 获取所有窗口（包括最小化的）
    // kCGWindowListExcludeDesktopElements 排除桌面元素
    CFArrayRef windowList = CGWindowListCopyWindowInfo(
        kCGWindowListOptionAll | kCGWindowListExcludeDesktopElements,
        kCGNullWindowID
    );

    if (windowList == NULL) {
        return 0;
    }

    CFIndex count = CFArrayGetCount(windowList);
    int resultCount = 0;

    for (CFIndex i = 0; i < count && resultCount < maxCount; i++) {
        CFDictionaryRef window = (CFDictionaryRef)CFArrayGetValueAtIndex(windowList, i);

        // 获取窗口层级，只保留普通窗口
        CFNumberRef layerRef = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowLayer);
        int layer = 0;
        if (layerRef) {
            CFNumberGetValue(layerRef, kCFNumberIntType, &layer);
        }
        // 跳过非普通窗口层级 (layer 0 是普通窗口)
        if (layer != 0) {
            continue;
        }

        // 获取 PID
        CFNumberRef pidRef = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowOwnerPID);
        int pid = 0;
        if (pidRef) {
            CFNumberGetValue(pidRef, kCFNumberIntType, &pid);
        }
        if (pid == 0) {
            continue;
        }

        // 获取窗口 ID
        CFNumberRef windowIdRef = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowNumber);
        int windowId = 0;
        if (windowIdRef) {
            CFNumberGetValue(windowIdRef, kCFNumberIntType, &windowId);
        }

        // 获取应用名称
        char ownerName[256] = {0};
        CFStringRef ownerRef = (CFStringRef)CFDictionaryGetValue(window, kCGWindowOwnerName);
        if (ownerRef) {
            CFStringGetCString(ownerRef, ownerName, sizeof(ownerName), kCFStringEncodingUTF8);
        }

        // 获取窗口标题
        CFStringRef nameRef = (CFStringRef)CFDictionaryGetValue(window, kCGWindowName);
        char title[512] = {0};
        if (nameRef) {
            CFStringGetCString(nameRef, title, sizeof(title), kCFStringEncodingUTF8);
        }

        // 如果窗口没有标题，使用应用名称
        if (strlen(title) == 0) {
            strncpy(title, ownerName, sizeof(title) - 1);
        }

        if (strlen(title) == 0) {
            continue;
        }

        // 获取窗口边界
        CFDictionaryRef boundsRef = (CFDictionaryRef)CFDictionaryGetValue(window, kCGWindowBounds);
        CGRect bounds;
        if (boundsRef) {
            CGRectMakeWithDictionaryRepresentation(boundsRef, &bounds);
        } else {
            bounds = CGRectZero;
        }

        // 跳过太小的窗口
        if (bounds.size.width < 50 || bounds.size.height < 50) {
            continue;
        }

        // 填充结果
        windows[resultCount].pid = pid;
        windows[resultCount].windowId = windowId;
        windows[resultCount].x = (int)bounds.origin.x;
        windows[resultCount].y = (int)bounds.origin.y;
        windows[resultCount].width = (int)bounds.size.width;
        windows[resultCount].height = (int)bounds.size.height;
        strncpy(windows[resultCount].title, title, sizeof(windows[resultCount].title) - 1);
        strncpy(windows[resultCount].ownerName, ownerName, sizeof(windows[resultCount].ownerName) - 1);

        resultCount++;
    }

    CFRelease(windowList);
    return resultCount;
}
*/
import "C"
import (
	"fmt"
	"os/exec"
	"strings"
	"unsafe"
)

// getWindowsDarwin 使用 macOS 原生 API 获取窗口列表
// 不会触发辅助功能权限弹窗
// 注意：浏览器的标签页不是独立窗口，每个浏览器窗口只会显示一次
func getWindowsDarwin(filter ...string) ([]WindowInfo, error) {
	const maxWindows = 256
	windows := make([]C.WindowInfoC, maxWindows)

	count := C.getWindowList(&windows[0], C.int(maxWindows))

	filterStr := ""
	if len(filter) > 0 {
		filterStr = strings.ToLower(filter[0])
	}

	result := make([]WindowInfo, 0, int(count))

	for i := 0; i < int(count); i++ {
		w := windows[i]
		title := C.GoString(&w.title[0])
		ownerName := C.GoString(&w.ownerName[0])

		// 过滤 - 检查窗口标题或应用名称
		if filterStr != "" {
			titleMatch := strings.Contains(strings.ToLower(title), filterStr)
			ownerMatch := strings.Contains(strings.ToLower(ownerName), filterStr)
			if !titleMatch && !ownerMatch {
				continue
			}
		}

		result = append(result, WindowInfo{
			PID:       int(w.pid),
			Title:     title,
			OwnerName: ownerName,
			Bounds: Region{
				X:      int(w.x),
				Y:      int(w.y),
				Width:  int(w.width),
				Height: int(w.height),
			},
		})
	}

	return result, nil
}

// 确保 unsafe 被使用（虽然这里不需要，但保留以备后用）
var _ = unsafe.Pointer(nil)

// activateWindowPlatform 使用 AppleScript 激活窗口
// 支持两种方式：
// 1. 应用名称（如 "Cursor", "Microsoft Edge"）- 激活应用的最前面窗口
// 2. 窗口标题（如 "build.yml — zoeymind"）- 激活包含该标题的特定窗口
func activateWindowPlatform(name string) error {
	// 先尝试作为应用名称激活
	script := fmt.Sprintf(`tell application "%s" to activate`, name)
	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err == nil {
		return nil
	}
	
	// 如果失败，尝试通过窗口标题查找并激活
	// 先查找包含该标题的窗口
	windows, _ := getWindowsDarwin(name)
	if len(windows) > 0 {
		// 找到了匹配的窗口，通过 PID 激活该应用，然后用 AppleScript 切换到特定窗口
		w := windows[0]
		
		// 先激活应用
		result := C.activateAppByPID(C.int(w.PID))
		if result == 0 {
			return fmt.Errorf("无法激活窗口: %s", name)
		}
		
		// 尝试通过 System Events 激活特定窗口（按标题匹配）
		// 这需要辅助功能权限
		activateScript := fmt.Sprintf(`
			tell application "System Events"
				set targetWindow to first window of (first process whose frontmost is true) whose name contains "%s"
				perform action "AXRaise" of targetWindow
			end tell
		`, name)
		exec.Command("osascript", "-e", activateScript).Run()
		
		return nil
	}
	
	return fmt.Errorf("无法激活窗口 %s: 未找到匹配的应用或窗口", name)
}

// activateWindowByPIDPlatform 使用 macOS 原生 API 通过 PID 激活窗口
func activateWindowByPIDPlatform(pid int) error {
	result := C.activateAppByPID(C.int(pid))
	if result == 0 {
		return fmt.Errorf("无法激活 PID %d 的窗口", pid)
	}
	return nil
}

// activateWindowByTitlePlatform 通过进程名和窗口标题激活特定窗口
// appName: 进程名（如 "Feishu", "WeChat", "Microsoft Edge"）
// windowTitle: 窗口标题的部分内容（如 "飞书", "微信"）
//
// 统一逻辑：获取所有窗口 → 按进程名/窗口标题匹配 → 用 PID 激活
// 不依赖 AppleScript 的应用名，避免本地化名称问题
func activateWindowByTitlePlatform(appName, windowTitle string) error {
	// 1. 获取所有窗口（不过滤）
	allWindows, err := getWindowsDarwin()
	if err != nil {
		return fmt.Errorf("获取窗口列表失败: %w", err)
	}

	// 2. 查找匹配的窗口
	var targetWindow *WindowInfo
	appNameLower := strings.ToLower(appName)
	windowTitleLower := strings.ToLower(windowTitle)

	// 优先级 1: 进程名 + 窗口标题都匹配
	for i := range allWindows {
		w := &allWindows[i]
		ownerMatch := strings.Contains(strings.ToLower(w.OwnerName), appNameLower)
		titleMatch := strings.Contains(strings.ToLower(w.Title), windowTitleLower)
		if ownerMatch && titleMatch {
			targetWindow = w
			break
		}
	}

	// 优先级 2: 只匹配进程名
	if targetWindow == nil {
		for i := range allWindows {
			w := &allWindows[i]
			if strings.Contains(strings.ToLower(w.OwnerName), appNameLower) {
				targetWindow = w
				break
			}
		}
	}

	// 优先级 3: 只匹配窗口标题
	if targetWindow == nil {
		for i := range allWindows {
			w := &allWindows[i]
			if strings.Contains(strings.ToLower(w.Title), windowTitleLower) {
				targetWindow = w
				break
			}
		}
	}

	if targetWindow == nil {
		return fmt.Errorf("未找到匹配的窗口: appName=%s, windowTitle=%s", appName, windowTitle)
	}

	// 3. 获取 Bundle ID，用于激活隐藏/最小化的应用
	bundleID := C.GoString(C.getBundleIDByPID(C.int(targetWindow.PID)))

	// 4. 使用 open -b 命令打开应用
	// 这是最可靠的方式，可以：
	// - 恢复隐藏到 Dock 的窗口
	// - 重新打开已关闭但后台运行的应用窗口
	if bundleID != "" {
		exec.Command("open", "-b", bundleID).Run()
	} else {
		// 回退到原生 API
		result := C.activateAppByPID(C.int(targetWindow.PID))
		if result == 0 {
			return fmt.Errorf("无法激活 PID %d 的应用", targetWindow.PID)
		}
	}

	// 5. 如果有多个窗口，尝试激活特定窗口
	processName := targetWindow.OwnerName
	if processName != "" && windowTitle != "" {
		windowScript := fmt.Sprintf(`
			tell application "System Events"
				tell process "%s"
					set frontmost to true
					repeat with w in windows
						if name of w contains "%s" then
							perform action "AXRaise" of w
							exit repeat
						end if
					end repeat
				end tell
			end tell
		`, processName, windowTitle)

		exec.Command("osascript", "-e", windowScript).Run()
	}

	return nil
}
