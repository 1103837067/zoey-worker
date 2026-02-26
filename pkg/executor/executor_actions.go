package executor

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	"github.com/zoeyai/zoeyworker/pkg/auto/grid"
	"github.com/zoeyai/zoeyworker/pkg/auto/input"
	"github.com/zoeyai/zoeyworker/pkg/auto/screen"
	"github.com/zoeyai/zoeyworker/pkg/auto/window"
	"github.com/zoeyai/zoeyworker/pkg/cmdutil"
	"github.com/zoeyai/zoeyworker/pkg/process"
	"github.com/zoeyai/zoeyworker/pkg/python"
)

var errFeatureRemoved = fmt.Errorf("该功能已移除，请使用 AI 任务")

// ==================== 单步操作实现 ====================

// executeTypeText 执行输入文字
func (e *Executor) executeTypeText(payload map[string]interface{}) (interface{}, error) {
	textStr, ok := payload["text"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	input.TypeText(textStr)
	return map[string]bool{"typed": true}, nil
}

// executeKeyPress 执行按键
func (e *Executor) executeKeyPress(payload map[string]interface{}) (interface{}, error) {
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

		if len(keys) == 1 {
			input.KeyTap(keys[0])
		} else {
			mainKey := keys[len(keys)-1]
			modifiers := keys[:len(keys)-1]
			input.KeyTap(mainKey, modifiers...)
		}

		return map[string]interface{}{"pressed": true, "keys": keys}, nil
	}

	key, ok := payload["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("缺少 key 参数")
	}

	var modifiers []string
	if mods, ok := payload["modifiers"].([]interface{}); ok {
		for _, m := range mods {
			if s, ok := m.(string); ok {
				modifiers = append(modifiers, s)
			}
		}
	}

	input.KeyTap(key, modifiers...)
	return map[string]bool{"pressed": true}, nil
}

// executeScreenshot 执行截屏
func (e *Executor) executeScreenshot(payload map[string]interface{}) (interface{}, error) {
	savePath, _ := payload["save_path"].(string)

	img, err := screen.CaptureScreen()
	if err != nil {
		return nil, err
	}

	if savePath != "" {
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

	bounds := img.Bounds()
	return map[string]interface{}{
		"width":  bounds.Dx(),
		"height": bounds.Dy(),
	}, nil
}

// executeMouseMove 执行鼠标移动
func (e *Executor) executeMouseMove(payload map[string]interface{}) (interface{}, error) {
	x, xOk := payload["x"].(float64)
	y, yOk := payload["y"].(float64)

	if !xOk || !yOk {
		return nil, fmt.Errorf("缺少 x 或 y 参数")
	}

	input.MoveTo(int(x), int(y))
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

	input.MoveTo(int(x), int(y))

	if double {
		input.DoubleClick()
	} else if right {
		input.RightClick()
	} else {
		input.Click()
	}

	return map[string]bool{"clicked": true}, nil
}

// executeActivateApp 执行激活应用
func (e *Executor) executeActivateApp(payload map[string]interface{}) (interface{}, error) {
	appName, _ := payload["app_name"].(string)
	windowTitle, _ := payload["window_title"].(string)

	log("DEBUG", fmt.Sprintf("executeActivateApp: app_name='%s', window_title='%s'", appName, windowTitle))

	if appName != "" && windowTitle != "" {
		err := window.ActivateWindowByTitle(appName, windowTitle)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"activated": true}, nil
	}

	if appName != "" {
		err := window.ActivateWindow(appName)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"activated": true}, nil
	}

	if windowTitle != "" {
		err := window.ActivateWindow(windowTitle)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"activated": true}, nil
	}

	return nil, fmt.Errorf("缺少 app_name 或 window_title 参数")
}

// executeGridClick 执行网格点击
func (e *Executor) executeGridClick(payload map[string]interface{}) (interface{}, error) {
	gridStr, ok := payload["grid"].(string)
	if !ok || gridStr == "" {
		return nil, fmt.Errorf("缺少 grid 参数")
	}

	var region auto.Region
	if r, ok := payload["region"].(map[string]interface{}); ok {
		region.X = int(r["x"].(float64))
		region.Y = int(r["y"].(float64))
		region.Width = int(r["width"].(float64))
		region.Height = int(r["height"].(float64))
	} else {
		w, h := screen.GetScreenSize()
		region = auto.Region{X: 0, Y: 0, Width: w, Height: h}
	}

	opts := e.parseAutoOptions(payload)
	err := grid.ClickGrid(region, gridStr, opts...)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"clicked": true}, nil
}

// executeGetClipboard 执行获取剪贴板
func (e *Executor) executeGetClipboard(payload map[string]interface{}) (interface{}, error) {
	textStr, err := input.ReadClipboard()
	if err != nil {
		return nil, err
	}

	return map[string]string{"text": textStr}, nil
}

// executeSetClipboard 执行设置剪贴板
func (e *Executor) executeSetClipboard(payload map[string]interface{}) (interface{}, error) {
	textStr, ok := payload["text"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少 text 参数")
	}

	err := input.CopyToClipboard(textStr)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"copied": true}, nil
}

// executeClickNative 执行原生控件点击
func (e *Executor) executeClickNative(payload map[string]interface{}) (interface{}, error) {
	automationID, _ := payload["automation_id"].(string)

	if automationID == "" {
		return nil, fmt.Errorf("缺少 automation_id 参数")
	}

	return map[string]bool{"clicked": true}, nil
}

// executeWaitTime 执行等待时间
func (e *Executor) executeWaitTime(payload map[string]interface{}) (interface{}, error) {
	duration, ok := payload["duration"].(float64)
	if !ok {
		duration = 1000
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

	processes, err := process.GetProcesses()
	if err != nil {
		return nil, fmt.Errorf("获取进程列表失败: %w", err)
	}

	for _, proc := range processes {
		if proc.Name == appName {
			if err := process.KillProcess(proc.PID); err != nil {
				return nil, fmt.Errorf("终止进程失败: %w", err)
			}
			return map[string]interface{}{"closed": true, "pid": proc.PID}, nil
		}
	}

	return nil, fmt.Errorf("未找到进程: %s", appName)
}

// executeRunPython 执行 Python 代码
func (e *Executor) executeRunPython(payload map[string]interface{}) (interface{}, error) {
	code, ok := payload["code"].(string)
	if !ok || code == "" {
		return nil, fmt.Errorf("缺少 code 参数")
	}

	timeoutSec := 30.0
	if t, ok := payload["timeout"].(float64); ok && t > 0 {
		timeoutSec = t
	}

	pythonInfo := python.DetectPython()
	if !pythonInfo.Available {
		return nil, fmt.Errorf("Python 环境未安装，请在 Agent 所在机器安装 Python 3")
	}

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("zoey_python_%d.py", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tmpFile)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, pythonInfo.Path, tmpFile)
	cmdutil.HideWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmdStartTime := time.Now()
	err := cmd.Run()
	durationMs := time.Since(cmdStartTime).Milliseconds()

	exitCode := 0
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("Python 脚本执行超时（超过 %.0f 秒）", timeoutSec)
		}
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

// executeSingleStep 执行单个步骤
func (e *Executor) executeSingleStep(taskType string, payload map[string]interface{}) (interface{}, error) {
	switch taskType {
	case TaskTypeClickImage, TaskTypeClickText, TaskTypeWaitImage, TaskTypeWaitText,
		TaskTypeImageExists, TaskTypeTextExists, TaskTypeAssertImage, TaskTypeAssertText:
		return nil, errFeatureRemoved
	case TaskTypeClickNative:
		return e.executeClickNative(payload)
	case TaskTypeTypeText:
		return e.executeTypeText(payload)
	case TaskTypeKeyPress:
		return e.executeKeyPress(payload)
	case TaskTypeScreenshot:
		return e.executeScreenshot(payload)
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

// executeSingleStepV2 执行单个步骤（增强版）
func (e *Executor) executeSingleStepV2(taskType string, payload map[string]interface{}) *ActionResult {
	result := &ActionResult{Success: true}

	if textStr, ok := payload["text"].(string); ok && taskType == TaskTypeTypeText {
		result.InputText = textStr
	}

	mouseX, mouseY := input.GetMousePosition()

	var data interface{}
	var err error

	switch taskType {
	case TaskTypeMouseClick:
		data, err = e.executeMouseClickV2(payload, result)
	case TaskTypeGridClick:
		data, err = e.executeGridClickV2(payload, result)
	default:
		data, err = e.executeSingleStep(taskType, payload)
	}

	if err != nil {
		result.Success = false
		result.Error = err
		if result.ClickPosition == nil {
			result.ClickPosition = &PositionInfo{X: mouseX, Y: mouseY}
		}
	}

	result.Data = data
	return result
}

// ==================== V2 增强版操作 ====================

func (e *Executor) executeMouseClickV2(payload map[string]interface{}, result *ActionResult) (interface{}, error) {
	x, xOk := payload["x"].(float64)
	y, yOk := payload["y"].(float64)
	if !xOk || !yOk {
		return nil, fmt.Errorf("缺少 x 或 y 参数")
	}

	result.ClickPosition = &PositionInfo{X: int(x), Y: int(y)}

	input.MoveTo(int(x), int(y))

	button, _ := payload["button"].(string)
	if button == "" {
		button = "left"
	}

	double, _ := payload["double"].(bool)
	if double {
		input.DoubleClick(button)
	} else {
		input.Click(button)
	}

	return map[string]bool{"clicked": true}, nil
}

func (e *Executor) executeGridClickV2(payload map[string]interface{}, result *ActionResult) (interface{}, error) {
	gridStr, ok := payload["grid"].(string)
	if !ok || gridStr == "" {
		return nil, fmt.Errorf("缺少 grid 参数")
	}

	screenWidth, screenHeight := screen.GetScreenSize()
	region := auto.Region{X: 0, Y: 0, Width: screenWidth, Height: screenHeight}

	pos, err := grid.CalculateGridCenterFromString(region, gridStr)
	if err != nil {
		return nil, err
	}

	result.ClickPosition = &PositionInfo{X: pos.X, Y: pos.Y}

	input.MoveTo(pos.X, pos.Y)
	input.Click()

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
