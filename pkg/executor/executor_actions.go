package executor

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	"github.com/zoeyai/zoeyworker/pkg/cmdutil"
	"github.com/zoeyai/zoeyworker/pkg/plugin"
	"github.com/zoeyai/zoeyworker/pkg/uia"
	"github.com/zoeyai/zoeyworker/pkg/vision/ocr"
)

// ==================== 单步操作实现 ====================

// executeClickImage 执行点击图像
func (e *Executor) executeClickImage(payload map[string]interface{}) (interface{}, error) {
	imagePath, ok := payload["image"].(string)
	if !ok || imagePath == "" {
		return nil, fmt.Errorf("缺少 image 参数")
	}

	// 检查是否有网格参数
	gridStr, _ := payload["grid"].(string)

	opts := e.parseAutoOptions(payload)

	// 获取任务 ID（用于调试）
	taskID, _ := payload["task_id"].(string)
	startTime := time.Now()

	// 发送调试数据的辅助函数
	sendDebugData := func(status string, matched bool, confidence float64, x, y int, errMsg string) {
		// 截取当前屏幕
		screenBase64 := ""
		if screen, err := auto.CaptureScreen(); err == nil {
			var buf bytes.Buffer
			if png.Encode(&buf, screen) == nil {
				screenBase64 = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
			}
		}

		emitDebugMatch(DebugMatchData{
			TaskID:         taskID,
			ActionType:     "click_image",
			Status:         status,
			TemplateBase64: imagePath, // 模板图片（已经是 base64 或 URL）
			ScreenBase64:   screenBase64,
			Matched:        matched,
			Confidence:     confidence,
			X:              x,
			Y:              y,
			Duration:       time.Since(startTime).Milliseconds(),
			Error:          errMsg,
		})
	}

	// 🔴 立即发送调试数据：开始搜索
	sendDebugData("searching", false, 0, 0, 0, "")

	if gridStr != "" {
		// 使用网格点击
		err := auto.ClickImageWithGrid(imagePath, gridStr, opts...)
		if err != nil {
			sendDebugData("not_found", false, 0, 0, 0, err.Error())
			return nil, err
		}
		x, y := auto.GetMousePosition()
		sendDebugData("found", true, 1.0, x, y, "")
		return map[string]interface{}{"clicked": true, "grid": gridStr}, nil
	}

	// 普通点击
	err := auto.ClickImage(imagePath, opts...)
	if err != nil {
		sendDebugData("not_found", false, 0, 0, 0, err.Error())
		return nil, err
	}

	x, y := auto.GetMousePosition()
	sendDebugData("found", true, 1.0, x, y, "")
	return map[string]bool{"clicked": true}, nil
}

// isOCRAvailable 检查 OCR 功能是否可用（插件安装或默认配置可用）
func isOCRAvailable() bool {
	// 先检查插件是否已安装
	if plugin.GetOCRPlugin().IsInstalled() {
		return true
	}
	// 再检查默认配置（打包的模型文件）是否可用
	return ocr.IsAvailable()
}

// executeClickText 执行点击文字
func (e *Executor) executeClickText(payload map[string]interface{}) (interface{}, error) {
	// 检查 OCR 是否可用（插件或默认配置）
	if !isOCRAvailable() {
		return nil, fmt.Errorf("OCR 功能未安装，请在客户端设置中下载安装 OCR 支持")
	}

	text, ok := payload["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	opts := e.parseAutoOptions(payload)
	err := auto.ClickText(text, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"clicked": true}, nil
}

// executeTypeText 执行输入文字
func (e *Executor) executeTypeText(payload map[string]interface{}) (interface{}, error) {
	text, ok := payload["text"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	auto.TypeText(text)
	return map[string]bool{"typed": true}, nil
}

// executeKeyPress 执行按键
func (e *Executor) executeKeyPress(payload map[string]interface{}) (interface{}, error) {
	// 新格式：keys 数组 (如 ["Ctrl", "C"] 或 ["Enter"])
	if keysRaw, ok := payload["keys"].([]interface{}); ok && len(keysRaw) > 0 {
		var keys []string
		for _, k := range keysRaw {
			if s, ok := k.(string); ok {
				keys = append(keys, s)
			}
		}

		if len(keys) == 0 {
			return nil, fmt.Errorf("keys 数组为空")
		}

		// 最后一个是主键，前面的是修饰键
		if len(keys) == 1 {
			// 单个按键
			auto.KeyTap(keys[0])
		} else {
			// 组合键：前面的是修饰键，最后一个是主键
			mainKey := keys[len(keys)-1]
			modifiers := keys[:len(keys)-1]
			auto.KeyTap(mainKey, modifiers...)
		}

		return map[string]interface{}{"pressed": true, "keys": keys}, nil
	}

	// 旧格式兼容：key + modifiers
	key, ok := payload["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("缺少 key 参数")
	}

	// 解析修饰键
	var modifiers []string
	if mods, ok := payload["modifiers"].([]interface{}); ok {
		for _, m := range mods {
			if s, ok := m.(string); ok {
				modifiers = append(modifiers, s)
			}
		}
	}

	auto.KeyTap(key, modifiers...)
	return map[string]bool{"pressed": true}, nil
}

// executeScreenshot 执行截屏
func (e *Executor) executeScreenshot(payload map[string]interface{}) (interface{}, error) {
	savePath, _ := payload["save_path"].(string)

	img, err := auto.CaptureScreen()
	if err != nil {
		return nil, err
	}

	if savePath != "" {
		// 保存截图
		file, err := os.Create(savePath)
		if err != nil {
			return nil, fmt.Errorf("创建文件失败: %w", err)
		}
		defer file.Close()

		if err := png.Encode(file, img); err != nil {
			return nil, fmt.Errorf("编码图片失败: %w", err)
		}
		return map[string]string{"path": savePath}, nil
	}

	// 不保存时返回截图信息
	bounds := img.Bounds()
	return map[string]interface{}{
		"width":  bounds.Dx(),
		"height": bounds.Dy(),
	}, nil
}

// executeWaitImage 执行等待图像
func (e *Executor) executeWaitImage(payload map[string]interface{}) (interface{}, error) {
	imagePath, ok := payload["image"].(string)
	if !ok || imagePath == "" {
		return nil, fmt.Errorf("缺少 image 参数")
	}

	opts := e.parseAutoOptions(payload)
	pos, err := auto.WaitForImage(imagePath, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"found": true,
		"x":     pos.X,
		"y":     pos.Y,
	}, nil
}

// executeWaitText 执行等待文字
func (e *Executor) executeWaitText(payload map[string]interface{}) (interface{}, error) {
	// 检查 OCR 是否可用（插件或默认配置）
	if !isOCRAvailable() {
		return nil, fmt.Errorf("OCR 功能未安装，请在客户端设置中下载安装 OCR 支持")
	}

	text, ok := payload["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	opts := e.parseAutoOptions(payload)
	pos, err := auto.WaitForText(text, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"found": true,
		"x":     pos.X,
		"y":     pos.Y,
	}, nil
}

// executeMouseMove 执行鼠标移动
func (e *Executor) executeMouseMove(payload map[string]interface{}) (interface{}, error) {
	x, xOk := payload["x"].(float64)
	y, yOk := payload["y"].(float64)

	if !xOk || !yOk {
		return nil, fmt.Errorf("缺少 x 或 y 参数")
	}

	auto.MoveTo(int(x), int(y))
	return map[string]bool{"moved": true}, nil
}

// executeMouseClick 执行鼠标点击
func (e *Executor) executeMouseClick(payload map[string]interface{}) (interface{}, error) {
	x, xOk := payload["x"].(float64)
	y, yOk := payload["y"].(float64)

	if !xOk || !yOk {
		return nil, fmt.Errorf("缺少 x 或 y 参数")
	}

	double, _ := payload["double"].(bool)
	right, _ := payload["right"].(bool)

	auto.MoveTo(int(x), int(y))

	if double {
		auto.DoubleClick()
	} else if right {
		auto.RightClick()
	} else {
		auto.Click()
	}

	return map[string]bool{"clicked": true}, nil
}

// executeActivateApp 执行激活应用
func (e *Executor) executeActivateApp(payload map[string]interface{}) (interface{}, error) {
	appName, _ := payload["app_name"].(string)
	windowTitle, _ := payload["window_title"].(string)

	log("DEBUG", fmt.Sprintf("executeActivateApp: app_name='%s', window_title='%s'", appName, windowTitle))

	// 如果同时有应用名和窗口标题，使用精确匹配
	if appName != "" && windowTitle != "" {
		log("DEBUG", fmt.Sprintf("Using ActivateWindowByTitle('%s', '%s')", appName, windowTitle))
		err := auto.ActivateWindowByTitle(appName, windowTitle)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"activated": true}, nil
	}

	// 只有应用名，直接激活应用
	if appName != "" {
		log("DEBUG", fmt.Sprintf("Using ActivateWindow('%s')", appName))
		err := auto.ActivateWindow(appName)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"activated": true}, nil
	}

	// 只有窗口标题，尝试通过标题查找并激活
	if windowTitle != "" {
		log("DEBUG", fmt.Sprintf("Using ActivateWindow by title: '%s'", windowTitle))
		err := auto.ActivateWindow(windowTitle)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"activated": true}, nil
	}

	return nil, fmt.Errorf("缺少 app_name 或 window_title 参数")
}

// executeGridClick 执行网格点击
func (e *Executor) executeGridClick(payload map[string]interface{}) (interface{}, error) {
	grid, ok := payload["grid"].(string)
	if !ok || grid == "" {
		return nil, fmt.Errorf("缺少 grid 参数")
	}

	// 获取区域
	var region auto.Region
	if r, ok := payload["region"].(map[string]interface{}); ok {
		region.X = int(r["x"].(float64))
		region.Y = int(r["y"].(float64))
		region.Width = int(r["width"].(float64))
		region.Height = int(r["height"].(float64))
	} else {
		// 默认使用全屏
		w, h := auto.GetScreenSize()
		region = auto.Region{X: 0, Y: 0, Width: w, Height: h}
	}

	opts := e.parseAutoOptions(payload)
	err := auto.ClickGrid(region, grid, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"clicked": true}, nil
}

// executeImageExists 执行检查图像存在
func (e *Executor) executeImageExists(payload map[string]interface{}) (interface{}, error) {
	imagePath, ok := payload["image"].(string)
	if !ok || imagePath == "" {
		return nil, fmt.Errorf("缺少 image 参数")
	}

	opts := e.parseAutoOptions(payload)
	exists := auto.ImageExists(imagePath, opts...)

	return map[string]bool{"exists": exists}, nil
}

// executeTextExists 执行检查文字存在
func (e *Executor) executeTextExists(payload map[string]interface{}) (interface{}, error) {
	text, ok := payload["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	opts := e.parseAutoOptions(payload)
	exists := auto.TextExists(text, opts...)

	return map[string]bool{"exists": exists}, nil
}

// executeGetClipboard 执行获取剪贴板
func (e *Executor) executeGetClipboard(payload map[string]interface{}) (interface{}, error) {
	text, err := auto.ReadClipboard()
	if err != nil {
		return nil, err
	}

	return map[string]string{"text": text}, nil
}

// executeSetClipboard 执行设置剪贴板
func (e *Executor) executeSetClipboard(payload map[string]interface{}) (interface{}, error) {
	text, ok := payload["text"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	err := auto.CopyToClipboard(text)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"copied": true}, nil
}

// executeClickNative 执行原生控件点击
func (e *Executor) executeClickNative(payload map[string]interface{}) (interface{}, error) {
	// 检查是否支持 UIA
	if !uia.IsSupported() {
		return nil, fmt.Errorf("原生控件点击需要 Windows + Python + pywinauto 环境")
	}

	automationID, _ := payload["automation_id"].(string)
	windowTitle, _ := payload["window_title"].(string)

	if automationID == "" {
		return nil, fmt.Errorf("缺少 automation_id 参数")
	}

	// 获取窗口句柄
	var windowHandle int
	if windowTitle != "" {
		// 通过标题查找窗口
		windows, err := auto.GetWindows(windowTitle)
		if err != nil || len(windows) == 0 {
			return nil, fmt.Errorf("未找到窗口: %s", windowTitle)
		}
		windowHandle = windows[0].PID
	} else {
		// 获取活动窗口
		windows, err := auto.GetWindows()
		if err != nil || len(windows) == 0 {
			return nil, fmt.Errorf("未找到活动窗口")
		}
		windowHandle = windows[0].PID
	}

	// 尝试使用 UIA 点击
	err := uia.ClickElement(windowHandle, automationID)
	if err != nil {
		return nil, fmt.Errorf("点击控件失败: %w", err)
	}

	return map[string]bool{"clicked": true}, nil
}

// executeWaitTime 执行等待时间
func (e *Executor) executeWaitTime(payload map[string]interface{}) (interface{}, error) {
	duration, ok := payload["duration"].(float64)
	if !ok {
		duration = 1000 // 默认 1 秒
	}

	time.Sleep(time.Duration(duration) * time.Millisecond)
	return map[string]interface{}{"waited": true, "duration_ms": duration}, nil
}

// executeCloseApp 执行关闭应用
func (e *Executor) executeCloseApp(payload map[string]interface{}) (interface{}, error) {
	appName, ok := payload["app_name"].(string)
	if !ok || appName == "" {
		return nil, fmt.Errorf("缺少 app_name 参数")
	}

	// 查找进程并终止
	processes, err := auto.GetProcesses()
	if err != nil {
		return nil, fmt.Errorf("获取进程列表失败: %w", err)
	}

	for _, proc := range processes {
		if proc.Name == appName {
			if err := auto.KillProcess(proc.PID); err != nil {
				return nil, fmt.Errorf("终止进程失败: %w", err)
			}
			return map[string]interface{}{"closed": true, "pid": proc.PID}, nil
		}
	}

	return nil, fmt.Errorf("未找到进程: %s", appName)
}

// executeAssertImage 执行图像断言
func (e *Executor) executeAssertImage(payload map[string]interface{}) (interface{}, error) {
	imagePath, ok := payload["image"].(string)
	if !ok || imagePath == "" {
		return nil, fmt.Errorf("缺少 image 参数")
	}

	opts := e.parseAutoOptions(payload)
	exists := auto.ImageExists(imagePath, opts...)

	if !exists {
		return nil, fmt.Errorf("断言失败: 未找到指定图像")
	}

	return map[string]bool{"asserted": true, "exists": true}, nil
}

// executeAssertText 执行文字断言
func (e *Executor) executeAssertText(payload map[string]interface{}) (interface{}, error) {
	text, ok := payload["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	opts := e.parseAutoOptions(payload)
	exists := auto.TextExists(text, opts...)

	if !exists {
		return nil, fmt.Errorf("断言失败: 未找到指定文字 '%s'", text)
	}

	return map[string]bool{"asserted": true, "exists": true}, nil
}

// executeRunPython 执行 Python 代码
func (e *Executor) executeRunPython(payload map[string]interface{}) (interface{}, error) {
	code, ok := payload["code"].(string)
	if !ok || code == "" {
		return nil, fmt.Errorf("缺少 code 参数")
	}

	// 超时时间（秒），默认 30 秒
	timeoutSec := 30.0
	if t, ok := payload["timeout"].(float64); ok && t > 0 {
		timeoutSec = t
	}

	// 检测 Python 环境
	pythonInfo := auto.DetectPython()
	if !pythonInfo.Available {
		return nil, fmt.Errorf("Python 环境未安装，请在 Agent 所在机器安装 Python 3")
	}

	// 创建临时文件写入代码
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("zoey_python_%d.py", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tmpFile)

	// 使用 context 超时控制
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, pythonInfo.Path, tmpFile)
	cmdutil.HideWindow(cmd) // Windows 上隐藏 cmd 黑色窗口

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	durationMs := time.Since(startTime).Milliseconds()

	exitCode := 0
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("Python 脚本执行超时（超过 %.0f 秒）", timeoutSec)
		}
		// 获取退出码
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("执行 Python 脚本失败: %w", err)
		}
	}

	result := map[string]interface{}{
		"stdout":      stdout.String(),
		"stderr":      stderr.String(),
		"exit_code":   exitCode,
		"duration_ms": durationMs,
	}

	// 非零退出码视为失败
	if exitCode != 0 {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = fmt.Sprintf("Python 脚本退出码: %d", exitCode)
		}
		return result, fmt.Errorf("Python 脚本执行失败: %s", errMsg)
	}

	return result, nil
}

// ==================== 步骤分发 ====================

// executeSingleStep 执行单个步骤（内部方法，不发送确认）
func (e *Executor) executeSingleStep(taskType string, payload map[string]interface{}) (interface{}, error) {
	switch taskType {
	case TaskTypeClickImage:
		return e.executeClickImage(payload)
	case TaskTypeClickText:
		return e.executeClickText(payload)
	case TaskTypeClickNative:
		return e.executeClickNative(payload)
	case TaskTypeTypeText:
		return e.executeTypeText(payload)
	case TaskTypeKeyPress:
		return e.executeKeyPress(payload)
	case TaskTypeScreenshot:
		return e.executeScreenshot(payload)
	case TaskTypeWaitImage:
		return e.executeWaitImage(payload)
	case TaskTypeWaitText:
		return e.executeWaitText(payload)
	case TaskTypeWaitTime:
		return e.executeWaitTime(payload)
	case TaskTypeMouseMove:
		return e.executeMouseMove(payload)
	case TaskTypeMouseClick:
		return e.executeMouseClick(payload)
	case TaskTypeActivateApp:
		return e.executeActivateApp(payload)
	case TaskTypeCloseApp:
		return e.executeCloseApp(payload)
	case TaskTypeGridClick:
		return e.executeGridClick(payload)
	case TaskTypeImageExists:
		return e.executeImageExists(payload)
	case TaskTypeTextExists:
		return e.executeTextExists(payload)
	case TaskTypeAssertImage:
		return e.executeAssertImage(payload)
	case TaskTypeAssertText:
		return e.executeAssertText(payload)
	case TaskTypeGetClipboard:
		return e.executeGetClipboard(payload)
	case TaskTypeSetClipboard:
		return e.executeSetClipboard(payload)
	case TaskTypeRunPython:
		return e.executeRunPython(payload)
	default:
		return nil, fmt.Errorf("未知的任务类型: %s", taskType)
	}
}

// executeSingleStepV2 执行单个步骤（增强版，返回更多信息用于回放）
func (e *Executor) executeSingleStepV2(taskType string, payload map[string]interface{}) *ActionResult {
	result := &ActionResult{Success: true}

	// 记录输入文本（用于 type_text 等操作）
	if text, ok := payload["text"].(string); ok && taskType == TaskTypeTypeText {
		result.InputText = text
	}

	// 获取鼠标当前位置（执行前），用于某些操作的位置记录
	mouseX, mouseY := auto.GetMousePosition()

	// 执行操作
	var data interface{}
	var err error

	switch taskType {
	case TaskTypeClickImage:
		data, err = e.executeClickImageV2(payload, result)
	case TaskTypeClickText:
		data, err = e.executeClickTextV2(payload, result)
	case TaskTypeMouseClick:
		data, err = e.executeMouseClickV2(payload, result)
	case TaskTypeGridClick:
		data, err = e.executeGridClickV2(payload, result)
	default:
		// 对于其他操作，使用原始方法
		data, err = e.executeSingleStep(taskType, payload)
	}

	if err != nil {
		result.Success = false
		result.Error = err
		// 记录失败时的鼠标位置（可能有助于调试）
		if result.ClickPosition == nil {
			result.ClickPosition = &PositionInfo{X: mouseX, Y: mouseY}
		}
	}

	result.Data = data
	return result
}

// ==================== V2 增强版操作 ====================

// executeClickImageV2 执行点击图像（增强版，记录位置信息）
// 复用 executeClickImage 的逻辑，额外记录点击位置
func (e *Executor) executeClickImageV2(payload map[string]interface{}, result *ActionResult) (interface{}, error) {
	// 调用基础版本（包含调试数据发送）
	data, err := e.executeClickImage(payload)

	// 记录点击位置
	if err == nil {
		x, y := auto.GetMousePosition()
		result.ClickPosition = &PositionInfo{X: x, Y: y}
	}

	return data, err
}

// executeClickTextV2 执行点击文字（增强版，记录位置信息）
func (e *Executor) executeClickTextV2(payload map[string]interface{}, result *ActionResult) (interface{}, error) {
	// 检查 OCR 是否可用（插件或默认配置）
	if !isOCRAvailable() {
		return nil, fmt.Errorf("OCR 功能未安装，请在客户端设置中下载安装 OCR 支持")
	}

	text, ok := payload["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	opts := e.parseAutoOptions(payload)

	// 先获取文字位置
	pos, err := auto.WaitForText(text, opts...)
	if err != nil {
		return nil, err
	}

	// 记录点击位置
	result.ClickPosition = &PositionInfo{X: pos.X, Y: pos.Y}

	// 执行点击
	err = auto.ClickText(text, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"clicked": true}, nil
}

// executeMouseClickV2 执行鼠标点击（增强版，记录位置信息）
func (e *Executor) executeMouseClickV2(payload map[string]interface{}, result *ActionResult) (interface{}, error) {
	x, xOk := payload["x"].(float64)
	y, yOk := payload["y"].(float64)
	if !xOk || !yOk {
		return nil, fmt.Errorf("缺少 x 或 y 参数")
	}

	// 记录点击位置
	result.ClickPosition = &PositionInfo{X: int(x), Y: int(y)}

	auto.MoveTo(int(x), int(y))

	button, _ := payload["button"].(string)
	if button == "" {
		button = "left"
	}

	double, _ := payload["double"].(bool)
	if double {
		auto.DoubleClick(button)
	} else {
		auto.Click(button)
	}

	return map[string]bool{"clicked": true}, nil
}

// executeGridClickV2 执行网格点击（增强版，记录位置信息）
func (e *Executor) executeGridClickV2(payload map[string]interface{}, result *ActionResult) (interface{}, error) {
	gridStr, ok := payload["grid"].(string)
	if !ok || gridStr == "" {
		return nil, fmt.Errorf("缺少 grid 参数")
	}

	// 计算网格位置
	screenWidth, screenHeight := auto.GetScreenSize()
	region := auto.Region{X: 0, Y: 0, Width: screenWidth, Height: screenHeight}

	pos, err := auto.CalculateGridCenterFromString(region, gridStr)
	if err != nil {
		return nil, err
	}

	// 记录点击位置
	result.ClickPosition = &PositionInfo{X: pos.X, Y: pos.Y}

	// 执行点击
	auto.MoveTo(pos.X, pos.Y)
	auto.Click()

	return map[string]interface{}{"clicked": true, "grid": gridStr, "x": pos.X, "y": pos.Y}, nil
}

// ==================== 选项解析 ====================

// parseAutoOptions 解析自动化选项
func (e *Executor) parseAutoOptions(payload map[string]interface{}) []auto.Option {
	var opts []auto.Option

	if timeout, ok := payload["timeout"].(float64); ok {
		opts = append(opts, auto.WithTimeout(time.Duration(timeout)*time.Second))
	}

	if threshold, ok := payload["threshold"].(float64); ok {
		opts = append(opts, auto.WithThreshold(threshold))
	}

	if double, ok := payload["double"].(bool); ok && double {
		opts = append(opts, auto.WithDoubleClick())
	}

	if right, ok := payload["right"].(bool); ok && right {
		opts = append(opts, auto.WithRightClick())
	}

	return opts
}
