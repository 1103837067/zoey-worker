//go:build windows

package auto

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32                     = syscall.NewLazyDLL("user32.dll")
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	psapi                      = syscall.NewLazyDLL("psapi.dll")
	procEnumWindows            = user32.NewProc("EnumWindows")
	procGetWindowTextW         = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW   = user32.NewProc("GetWindowTextLengthW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetWindowRect          = user32.NewProc("GetWindowRect")
	procIsWindowVisible        = user32.NewProc("IsWindowVisible")
	procGetWindowLongW         = user32.NewProc("GetWindowLongW")
	procOpenProcess            = kernel32.NewProc("OpenProcess")
	procCloseHandle            = kernel32.NewProc("CloseHandle")
	procGetModuleBaseNameW     = psapi.NewProc("GetModuleBaseNameW")
	procSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	procShowWindow             = user32.NewProc("ShowWindow")
	procBringWindowToTop       = user32.NewProc("BringWindowToTop")
	procGetForegroundWindow    = user32.NewProc("GetForegroundWindow")
	procAttachThreadInput      = user32.NewProc("AttachThreadInput")
	procGetCurrentThreadId     = kernel32.NewProc("GetCurrentThreadId")
)

const (
	GWL_STYLE           = -16
	GWL_EXSTYLE         = -20
	WS_VISIBLE          = 0x10000000
	WS_EX_TOOLWINDOW    = 0x00000080
	WS_EX_APPWINDOW     = 0x00040000
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_VM_READ     = 0x0010
	SW_RESTORE          = 9
	SW_SHOW             = 5
)

// RECT Windows 矩形结构
type RECT struct {
	Left, Top, Right, Bottom int32
}

// windowEnumData 用于 EnumWindows 回调的数据
type windowEnumData struct {
	windows []WindowInfo
	filter  string
}

// getWindowsWindows 使用 Windows 原生 API 获取窗口列表
// 正确处理 UTF-16 编码，解决中文乱码问题
func getWindowsWindows(filter ...string) ([]WindowInfo, error) {
	data := &windowEnumData{
		windows: make([]WindowInfo, 0, 64),
	}
	if len(filter) > 0 {
		data.filter = strings.ToLower(filter[0])
	}

	// EnumWindows 枚举所有顶级窗口
	callback := syscall.NewCallback(func(hwnd syscall.Handle, lParam uintptr) uintptr {
		enumWindowsCallback(hwnd, lParam)
		return 1 // 继续枚举
	})

	procEnumWindows.Call(callback, uintptr(unsafe.Pointer(data)))

	return data.windows, nil
}

// enumWindowsCallback EnumWindows 回调函数
func enumWindowsCallback(hwnd syscall.Handle, lParam uintptr) {
	data := (*windowEnumData)(unsafe.Pointer(lParam))

	// 检查窗口是否可见
	ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	if ret == 0 {
		return
	}

	// 获取窗口样式，过滤工具窗口等
	style, _, _ := procGetWindowLongW.Call(uintptr(hwnd), uintptr(uint32(GWL_STYLE)))
	exStyle, _, _ := procGetWindowLongW.Call(uintptr(hwnd), uintptr(uint32(GWL_EXSTYLE)))

	// 跳过不可见窗口
	if style&WS_VISIBLE == 0 {
		return
	}

	// 跳过工具窗口（除非它有 APPWINDOW 样式）
	if exStyle&WS_EX_TOOLWINDOW != 0 && exStyle&WS_EX_APPWINDOW == 0 {
		return
	}

	// 获取窗口标题长度
	length, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if length == 0 {
		return
	}

	// 获取窗口标题 (UTF-16)
	buf := make([]uint16, length+1)
	procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(length+1))
	
	// UTF-16 转 UTF-8
	title := syscall.UTF16ToString(buf)
	if title == "" {
		return
	}

	// 获取进程 ID
	var pid uint32
	procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return
	}

	// 获取进程名称
	ownerName := getProcessName(pid)

	// 获取窗口边界
	var rect RECT
	procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))

	// 计算宽高
	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)

	// 跳过太小的窗口
	if width < 50 || height < 50 {
		return
	}

	// 过滤检查
	if data.filter != "" {
		titleLower := strings.ToLower(title)
		ownerLower := strings.ToLower(ownerName)
		if !strings.Contains(titleLower, data.filter) && !strings.Contains(ownerLower, data.filter) {
			return
		}
	}

	data.windows = append(data.windows, WindowInfo{
		PID:       int(pid),
		Title:     title,
		OwnerName: ownerName,
		Bounds: Region{
			X:      int(rect.Left),
			Y:      int(rect.Top),
			Width:  width,
			Height: height,
		},
	})
}

// getProcessName 通过 PID 获取进程名称
func getProcessName(pid uint32) string {
	// 打开进程
	handle, _, _ := procOpenProcess.Call(
		uintptr(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return ""
	}
	defer procCloseHandle.Call(handle)

	// 获取模块名称 (UTF-16)
	buf := make([]uint16, 260) // MAX_PATH
	ret, _, _ := procGetModuleBaseNameW.Call(
		handle,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if ret == 0 {
		return ""
	}

	// UTF-16 转 UTF-8
	name := syscall.UTF16ToString(buf)
	
	// 去掉 .exe 后缀
	if strings.HasSuffix(strings.ToLower(name), ".exe") {
		name = name[:len(name)-4]
	}
	
	return name
}

// activateWindowPlatform 激活窗口（Windows 实现）
func activateWindowPlatform(name string) error {
	windows, err := getWindowsWindows(name)
	if err != nil {
		return err
	}
	if len(windows) == 0 {
		return fmt.Errorf("未找到窗口: %s", name)
	}

	// 通过标题找到窗口句柄并激活
	return activateWindowByTitleInternal(windows[0].Title)
}

// activateWindowByPIDPlatform 通过 PID 激活窗口（Windows 实现）
func activateWindowByPIDPlatform(pid int) error {
	// 遍历所有窗口找到匹配 PID 的
	var targetHwnd syscall.Handle

	callback := syscall.NewCallback(func(hwnd syscall.Handle, lParam uintptr) uintptr {
		var windowPid uint32
		procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&windowPid)))
		if int(windowPid) == pid {
			ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
			if ret != 0 {
				targetHwnd = hwnd
				return 0 // 停止枚举
			}
		}
		return 1 // 继续枚举
	})

	procEnumWindows.Call(callback, 0)

	if targetHwnd == 0 {
		return fmt.Errorf("未找到 PID %d 的窗口", pid)
	}

	return activateWindowByHandle(targetHwnd)
}

// activateWindowByTitlePlatform 通过标题激活窗口（Windows 实现）
func activateWindowByTitlePlatform(appName, windowTitle string) error {
	windows, err := getWindowsWindows()
	if err != nil {
		return err
	}

	appNameLower := strings.ToLower(appName)
	windowTitleLower := strings.ToLower(windowTitle)

	// 优先级 1: 进程名 + 窗口标题都匹配
	for _, w := range windows {
		ownerMatch := strings.Contains(strings.ToLower(w.OwnerName), appNameLower)
		titleMatch := strings.Contains(strings.ToLower(w.Title), windowTitleLower)
		if ownerMatch && titleMatch {
			return activateWindowByTitleInternal(w.Title)
		}
	}

	// 优先级 2: 只匹配进程名
	for _, w := range windows {
		if strings.Contains(strings.ToLower(w.OwnerName), appNameLower) {
			return activateWindowByTitleInternal(w.Title)
		}
	}

	// 优先级 3: 只匹配窗口标题
	for _, w := range windows {
		if strings.Contains(strings.ToLower(w.Title), windowTitleLower) {
			return activateWindowByTitleInternal(w.Title)
		}
	}

	return fmt.Errorf("未找到匹配的窗口: appName=%s, windowTitle=%s", appName, windowTitle)
}

// activateWindowByTitleInternal 通过窗口标题激活窗口
func activateWindowByTitleInternal(title string) error {
	var targetHwnd syscall.Handle
	titleLower := strings.ToLower(title)

	callback := syscall.NewCallback(func(hwnd syscall.Handle, lParam uintptr) uintptr {
		length, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
		if length == 0 {
			return 1
		}

		buf := make([]uint16, length+1)
		procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(length+1))
		windowTitle := syscall.UTF16ToString(buf)

		if strings.ToLower(windowTitle) == titleLower || strings.Contains(strings.ToLower(windowTitle), titleLower) {
			targetHwnd = hwnd
			return 0 // 停止枚举
		}
		return 1 // 继续枚举
	})

	procEnumWindows.Call(callback, 0)

	if targetHwnd == 0 {
		return fmt.Errorf("未找到窗口: %s", title)
	}

	return activateWindowByHandle(targetHwnd)
}

// activateWindowByHandle 通过窗口句柄激活窗口
func activateWindowByHandle(hwnd syscall.Handle) error {
	// 获取当前前台窗口的线程 ID
	foregroundHwnd, _, _ := procGetForegroundWindow.Call()
	var foregroundThreadId uint32
	if foregroundHwnd != 0 {
		foregroundThreadId, _, _ = procGetWindowThreadProcessId.Call(foregroundHwnd, 0)
	}

	// 获取当前线程 ID
	currentThreadId, _, _ := procGetCurrentThreadId.Call()

	// 获取目标窗口的线程 ID
	targetThreadId, _, _ := procGetWindowThreadProcessId.Call(uintptr(hwnd), 0)

	// 附加输入线程以允许 SetForegroundWindow
	if foregroundThreadId != 0 && foregroundThreadId != uint32(currentThreadId) {
		procAttachThreadInput.Call(uintptr(currentThreadId), uintptr(foregroundThreadId), 1)
		defer procAttachThreadInput.Call(uintptr(currentThreadId), uintptr(foregroundThreadId), 0)
	}

	if targetThreadId != 0 && uint32(targetThreadId) != uint32(currentThreadId) {
		procAttachThreadInput.Call(uintptr(currentThreadId), targetThreadId, 1)
		defer procAttachThreadInput.Call(uintptr(currentThreadId), targetThreadId, 0)
	}

	// 恢复窗口（如果最小化）
	procShowWindow.Call(uintptr(hwnd), SW_RESTORE)

	// 将窗口置顶
	procBringWindowToTop.Call(uintptr(hwnd))

	// 设置为前台窗口
	ret, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	if ret == 0 {
		return fmt.Errorf("SetForegroundWindow 失败")
	}

	return nil
}
