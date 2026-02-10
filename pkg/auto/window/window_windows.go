//go:build windows

package window

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"github.com/zoeyai/zoeyworker/pkg/auto"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	psapi                        = syscall.NewLazyDLL("psapi.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetWindowLongW           = user32.NewProc("GetWindowLongW")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procGetModuleBaseNameW       = psapi.NewProc("GetModuleBaseNameW")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procShowWindow               = user32.NewProc("ShowWindow")
	procBringWindowToTop         = user32.NewProc("BringWindowToTop")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procGetCurrentThreadId       = kernel32.NewProc("GetCurrentThreadId")
)

const (
	gwlStyle   = ^uintptr(15) // -16
	gwlExStyle = ^uintptr(19) // -20

	wsVisible       uintptr = 0x10000000
	wsExToolWindow  uintptr = 0x00000080
	wsExAppWindow   uintptr = 0x00040000

	processQueryInformation = 0x0400
	processVMRead           = 0x0010
	swRestore               = 9
	swShow                  = 5
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

// getWindowsPlatform Windows 平台实现
func getWindowsPlatform(filter ...string) ([]WindowInfo, error) {
	return getWindowsWindows(filter...)
}

// getWindowsWindows 使用 Windows 原生 API 获取窗口列表
func getWindowsWindows(filter ...string) ([]WindowInfo, error) {
	data := &windowEnumData{
		windows: make([]WindowInfo, 0, 64),
	}
	if len(filter) > 0 {
		data.filter = strings.ToLower(filter[0])
	}

	callback := syscall.NewCallback(func(hwnd syscall.Handle, _ uintptr) uintptr {
		// 直接通过闭包捕获 data，避免 unsafe.Pointer(uintptr) 转换
		ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		if ret == 0 {
			return 1
		}

		style, _, _ := procGetWindowLongW.Call(uintptr(hwnd), gwlStyle)
		exStyle, _, _ := procGetWindowLongW.Call(uintptr(hwnd), gwlExStyle)

		if style&wsVisible == 0 {
			return 1
		}

		if exStyle&wsExToolWindow != 0 && exStyle&wsExAppWindow == 0 {
			return 1
		}

		length, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
		if length == 0 {
			return 1
		}

		buf := make([]uint16, length+1)
		procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(length+1))
		title := syscall.UTF16ToString(buf)
		if title == "" {
			return 1
		}

		var pid uint32
		procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
		if pid == 0 {
			return 1
		}

		ownerName := getProcessName(pid)

		var rect RECT
		procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))

		width := int(rect.Right - rect.Left)
		height := int(rect.Bottom - rect.Top)

		if width < 50 || height < 50 {
			return 1
		}

		if data.filter != "" {
			titleLower := strings.ToLower(title)
			ownerLower := strings.ToLower(ownerName)
			if !strings.Contains(titleLower, data.filter) && !strings.Contains(ownerLower, data.filter) {
				return 1
			}
		}

		data.windows = append(data.windows, WindowInfo{
			PID:       int(pid),
			Title:     title,
			OwnerName: ownerName,
			Bounds: auto.Region{
				X:      int(rect.Left),
				Y:      int(rect.Top),
				Width:  width,
				Height: height,
			},
		})
		return 1
	})

	procEnumWindows.Call(callback, 0)

	return data.windows, nil
}

// getProcessName 通过 PID 获取进程名称
func getProcessName(pid uint32) string {
	handle, _, _ := procOpenProcess.Call(
		uintptr(processQueryInformation|processVMRead),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return ""
	}
	defer procCloseHandle.Call(handle)

	buf := make([]uint16, 260)
	ret, _, _ := procGetModuleBaseNameW.Call(
		handle,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if ret == 0 {
		return ""
	}

	name := syscall.UTF16ToString(buf)
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

	return activateWindowByTitleInternal(windows[0].Title)
}

// activateWindowByPIDPlatform 通过 PID 激活窗口（Windows 实现）
func activateWindowByPIDPlatform(pid int) error {
	var targetHwnd syscall.Handle

	callback := syscall.NewCallback(func(hwnd syscall.Handle, lParam uintptr) uintptr {
		var windowPid uint32
		procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&windowPid)))
		if int(windowPid) == pid {
			ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
			if ret != 0 {
				targetHwnd = hwnd
				return 0
			}
		}
		return 1
	})

	procEnumWindows.Call(callback, 0)

	if targetHwnd == 0 {
		return fmt.Errorf("未找到 PID %d 的窗口", pid)
	}

	return activateWindowByHandle(targetHwnd)
}

// activateWindowByTitlePlatform 通过应用名和窗口标题激活特定窗口
func activateWindowByTitlePlatform(appName, windowTitle string) error {
	windows, err := getWindowsWindows()
	if err != nil {
		return err
	}

	appNameLower := strings.ToLower(appName)
	windowTitleLower := strings.ToLower(windowTitle)

	for _, w := range windows {
		ownerMatch := strings.Contains(strings.ToLower(w.OwnerName), appNameLower)
		titleMatch := strings.Contains(strings.ToLower(w.Title), windowTitleLower)
		if ownerMatch && titleMatch {
			return activateWindowByTitleInternal(w.Title)
		}
	}

	for _, w := range windows {
		if strings.Contains(strings.ToLower(w.OwnerName), appNameLower) {
			return activateWindowByTitleInternal(w.Title)
		}
	}

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
			return 0
		}
		return 1
	})

	procEnumWindows.Call(callback, 0)

	if targetHwnd == 0 {
		return fmt.Errorf("未找到窗口: %s", title)
	}

	return activateWindowByHandle(targetHwnd)
}

// activateWindowByHandle 通过窗口句柄激活窗口
func activateWindowByHandle(hwnd syscall.Handle) error {
	foregroundHwnd, _, _ := procGetForegroundWindow.Call()
	var foregroundThreadId uintptr
	if foregroundHwnd != 0 {
		foregroundThreadId, _, _ = procGetWindowThreadProcessId.Call(foregroundHwnd, 0)
	}

	currentThreadId, _, _ := procGetCurrentThreadId.Call()
	targetThreadId, _, _ := procGetWindowThreadProcessId.Call(uintptr(hwnd), 0)

	if foregroundThreadId != 0 && foregroundThreadId != currentThreadId {
		procAttachThreadInput.Call(currentThreadId, foregroundThreadId, 1)
		defer procAttachThreadInput.Call(currentThreadId, foregroundThreadId, 0)
	}

	if targetThreadId != 0 && targetThreadId != currentThreadId {
		procAttachThreadInput.Call(currentThreadId, targetThreadId, 1)
		defer procAttachThreadInput.Call(currentThreadId, targetThreadId, 0)
	}

	procShowWindow.Call(uintptr(hwnd), swRestore)
	procBringWindowToTop.Call(uintptr(hwnd))

	ret, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	if ret == 0 {
		return fmt.Errorf("SetForegroundWindow 失败")
	}

	return nil
}
