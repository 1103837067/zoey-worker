// Package uia provides Windows UI Automation support via Python subprocess
package uia

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/zoeyai/zoeyworker/pkg/cmdutil"
)

// ElementInfo represents a UI element
type ElementInfo struct {
	AutomationID string   `json:"automation_id"`
	Name         string   `json:"name"`
	ClassName    string   `json:"class_name"`
	ControlType  string   `json:"control_type"`
	Rect         RectInfo `json:"rect"`
	IsEnabled    bool     `json:"is_enabled"`
	IsVisible    bool     `json:"is_visible"`
	Value        string   `json:"value"`
}

// RectInfo represents element bounds
type RectInfo struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// GetElementsOptions specifies filters for element retrieval
type GetElementsOptions struct {
	AutomationID string
	ControlType  string
	MaxDepth     int
}

// IsSupported returns true if UIA is available on the current platform
func IsSupported() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	// Check if Python is available
	_, err := exec.LookPath("python")
	if err != nil {
		_, err = exec.LookPath("python3")
		if err != nil {
			return false
		}
	}
	return true
}

// GetElements retrieves UI elements from a window
func GetElements(windowHandle int, opts *GetElementsOptions) ([]ElementInfo, error) {
	if !IsSupported() {
		return nil, fmt.Errorf("UI Automation is not supported on this platform (requires Windows + Python)")
	}

	if opts == nil {
		opts = &GetElementsOptions{MaxDepth: 3}
	}

	// Build Python command
	script := buildGetElementsScript(windowHandle, opts)
	
	output, err := runPythonScript(script)
	if err != nil {
		return nil, fmt.Errorf("failed to get elements: %w", err)
	}

	var elements []ElementInfo
	if err := json.Unmarshal([]byte(output), &elements); err != nil {
		return nil, fmt.Errorf("failed to parse elements: %w", err)
	}

	return elements, nil
}

// FindElement finds an element by AutomationID
func FindElement(windowHandle int, automationID string) (*ElementInfo, error) {
	elements, err := GetElements(windowHandle, &GetElementsOptions{
		AutomationID: automationID,
		MaxDepth:     5,
	})
	if err != nil {
		return nil, err
	}

	for _, elem := range elements {
		if elem.AutomationID == automationID {
			return &elem, nil
		}
	}

	return nil, fmt.Errorf("element not found: %s", automationID)
}

// ClickElement clicks an element by AutomationID
func ClickElement(windowHandle int, automationID string) error {
	if !IsSupported() {
		return fmt.Errorf("UI Automation is not supported on this platform")
	}

	script := buildClickElementScript(windowHandle, automationID)
	
	_, err := runPythonScript(script)
	if err != nil {
		return fmt.Errorf("failed to click element: %w", err)
	}

	return nil
}

// SetElementValue sets the value of an input element
func SetElementValue(windowHandle int, automationID, value string) error {
	if !IsSupported() {
		return fmt.Errorf("UI Automation is not supported on this platform")
	}

	script := buildSetValueScript(windowHandle, automationID, value)
	
	_, err := runPythonScript(script)
	if err != nil {
		return fmt.Errorf("failed to set element value: %w", err)
	}

	return nil
}

// runPythonScript executes a Python script and returns the output
func runPythonScript(script string) (string, error) {
	// Try python first, then python3
	pythonPath := "python"
	if _, err := exec.LookPath(pythonPath); err != nil {
		pythonPath = "python3"
	}

	cmd := exec.Command(pythonPath, "-c", script)
	cmdutil.HideWindow(cmd) // Windows 上隐藏 cmd 黑色窗口
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("python error: %s\noutput: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// buildGetElementsScript generates Python code to get elements
func buildGetElementsScript(windowHandle int, opts *GetElementsOptions) string {
	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 3
	}

	script := fmt.Sprintf(`
import json
import sys

try:
    from pywinauto.application import Application
    from pywinauto import Desktop
except ImportError:
    print(json.dumps([]))
    sys.exit(0)

def get_rect(elem):
    try:
        rect = elem.rectangle()
        return {"x": rect.left, "y": rect.top, "width": rect.width(), "height": rect.height()}
    except:
        return {"x": 0, "y": 0, "width": 0, "height": 0}

def collect_elements(elem, results, max_depth, current_depth=0):
    if current_depth > max_depth:
        return
    
    try:
        automation_id = elem.automation_id() or ""
        name = elem.window_text() or ""
        class_name = elem.class_name() or ""
        control_type = elem.control_type() or ""
        is_enabled = elem.is_enabled()
        is_visible = elem.is_visible()
        
        value = ""
        try:
            if hasattr(elem, "get_value"):
                value = elem.get_value() or ""
        except:
            pass
        
        filter_automation_id = %q
        filter_control_type = %q
        
        if filter_automation_id and automation_id != filter_automation_id:
            pass
        elif filter_control_type and control_type != filter_control_type:
            pass
        elif automation_id or name or is_visible:
            results.append({
                "automation_id": automation_id,
                "name": name,
                "class_name": class_name,
                "control_type": control_type,
                "rect": get_rect(elem),
                "is_enabled": is_enabled,
                "is_visible": is_visible,
                "value": value
            })
        
        for child in elem.children():
            collect_elements(child, results, max_depth, current_depth + 1)
    except Exception as e:
        pass

try:
    app = Application(backend="uia").connect(handle=%d)
    window = app.window(handle=%d)
    results = []
    collect_elements(window.wrapper_object(), results, %d)
    print(json.dumps(results))
except Exception as e:
    print(json.dumps([]))
`, opts.AutomationID, opts.ControlType, windowHandle, windowHandle, maxDepth)

	return script
}

// buildClickElementScript generates Python code to click an element
func buildClickElementScript(windowHandle int, automationID string) string {
	return fmt.Sprintf(`
import sys

try:
    from pywinauto.application import Application
except ImportError:
    print("pywinauto not installed")
    sys.exit(1)

try:
    app = Application(backend="uia").connect(handle=%d)
    window = app.window(handle=%d)
    elem = window.child_window(auto_id=%q)
    elem.click_input()
    print("ok")
except Exception as e:
    print(f"Error: {e}")
    sys.exit(1)
`, windowHandle, windowHandle, automationID)
}

// buildSetValueScript generates Python code to set element value
func buildSetValueScript(windowHandle int, automationID, value string) string {
	return fmt.Sprintf(`
import sys

try:
    from pywinauto.application import Application
except ImportError:
    print("pywinauto not installed")
    sys.exit(1)

try:
    app = Application(backend="uia").connect(handle=%d)
    window = app.window(handle=%d)
    elem = window.child_window(auto_id=%q)
    elem.set_text(%q)
    print("ok")
except Exception as e:
    print(f"Error: {e}")
    sys.exit(1)
`, windowHandle, windowHandle, automationID, value)
}
