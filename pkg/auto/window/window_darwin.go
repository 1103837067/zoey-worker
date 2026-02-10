//go:build darwin

package window

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

// 获取窗口列表
int getWindowList(WindowInfoC* windows, int maxCount) {
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

        CFNumberRef layerRef = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowLayer);
        int layer = 0;
        if (layerRef) {
            CFNumberGetValue(layerRef, kCFNumberIntType, &layer);
        }
        if (layer != 0) {
            continue;
        }

        CFNumberRef pidRef = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowOwnerPID);
        int pid = 0;
        if (pidRef) {
            CFNumberGetValue(pidRef, kCFNumberIntType, &pid);
        }
        if (pid == 0) {
            continue;
        }

        CFNumberRef windowIdRef = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowNumber);
        int windowId = 0;
        if (windowIdRef) {
            CFNumberGetValue(windowIdRef, kCFNumberIntType, &windowId);
        }

        char ownerName[256] = {0};
        CFStringRef ownerRef = (CFStringRef)CFDictionaryGetValue(window, kCGWindowOwnerName);
        if (ownerRef) {
            CFStringGetCString(ownerRef, ownerName, sizeof(ownerName), kCFStringEncodingUTF8);
        }

        CFStringRef nameRef = (CFStringRef)CFDictionaryGetValue(window, kCGWindowName);
        char title[512] = {0};
        if (nameRef) {
            CFStringGetCString(nameRef, title, sizeof(title), kCFStringEncodingUTF8);
        }

        if (strlen(title) == 0) {
            strncpy(title, ownerName, sizeof(title) - 1);
        }

        if (strlen(title) == 0) {
            continue;
        }

        CFDictionaryRef boundsRef = (CFDictionaryRef)CFDictionaryGetValue(window, kCGWindowBounds);
        CGRect bounds;
        if (boundsRef) {
            CGRectMakeWithDictionaryRepresentation(boundsRef, &bounds);
        } else {
            bounds = CGRectZero;
        }

        if (bounds.size.width < 50 || bounds.size.height < 50) {
            continue;
        }

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

	"github.com/zoeyai/zoeyworker/pkg/auto"
)

var _ = unsafe.Pointer(nil)

// getWindowsPlatform macOS 平台实现
func getWindowsPlatform(filter ...string) ([]WindowInfo, error) {
	return getWindowsDarwin(filter...)
}

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
			Bounds: auto.Region{
				X:      int(w.x),
				Y:      int(w.y),
				Width:  int(w.width),
				Height: int(w.height),
			},
		})
	}

	return result, nil
}

func activateWindowPlatform(name string) error {
	script := fmt.Sprintf(`tell application "%s" to activate`, name)
	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err == nil {
		return nil
	}

	windows, _ := getWindowsDarwin(name)
	if len(windows) > 0 {
		w := windows[0]
		result := C.activateAppByPID(C.int(w.PID))
		if result == 0 {
			return fmt.Errorf("无法激活窗口: %s", name)
		}

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

func activateWindowByPIDPlatform(pid int) error {
	result := C.activateAppByPID(C.int(pid))
	if result == 0 {
		return fmt.Errorf("无法激活 PID %d 的窗口", pid)
	}
	return nil
}

func activateWindowByTitlePlatform(appName, windowTitle string) error {
	allWindows, err := getWindowsDarwin()
	if err != nil {
		return fmt.Errorf("获取窗口列表失败: %w", err)
	}

	var targetWindow *WindowInfo
	appNameLower := strings.ToLower(appName)
	windowTitleLower := strings.ToLower(windowTitle)

	for i := range allWindows {
		w := &allWindows[i]
		ownerMatch := strings.Contains(strings.ToLower(w.OwnerName), appNameLower)
		titleMatch := strings.Contains(strings.ToLower(w.Title), windowTitleLower)
		if ownerMatch && titleMatch {
			targetWindow = w
			break
		}
	}

	if targetWindow == nil {
		for i := range allWindows {
			w := &allWindows[i]
			if strings.Contains(strings.ToLower(w.OwnerName), appNameLower) {
				targetWindow = w
				break
			}
		}
	}

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

	bundleID := C.GoString(C.getBundleIDByPID(C.int(targetWindow.PID)))

	if bundleID != "" {
		exec.Command("open", "-b", bundleID).Run()
	} else {
		result := C.activateAppByPID(C.int(targetWindow.PID))
		if result == 0 {
			return fmt.Errorf("无法激活 PID %d 的应用", targetWindow.PID)
		}
	}

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
