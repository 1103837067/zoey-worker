//go:build darwin

package auto

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework Cocoa
#import <CoreGraphics/CoreGraphics.h>
#import <Cocoa/Cocoa.h>

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

// 获取窗口列表 (不触发权限弹窗)
// 注意：macOS 的标签页（如浏览器标签）不是独立窗口，需要辅助功能API获取
// 这里返回的是实际的窗口，每个浏览器窗口（可能包含多个标签）是一个条目
int getWindowList(WindowInfoC* windows, int maxCount) {
    // 使用 CGWindowListCopyWindowInfo 获取窗口列表
    // 这个 API 不会触发辅助功能权限弹窗
    CFArrayRef windowList = CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
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
			PID:   int(w.pid),
			Title: title,
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
