package auto

import (
	"os/exec"
	"strings"

	"github.com/zoeyai/zoeyworker/pkg/cmdutil"
)

// PythonInfo Python 环境信息
type PythonInfo struct {
	Available bool   // Python 是否可用
	Version   string // 版本号，如 "3.11.5"
	Path      string // 可执行文件路径
}

// DetectPython 检测 Python 环境
// 按优先级检测 python3 / python，返回环境信息
func DetectPython() *PythonInfo {
	info := &PythonInfo{}

	// 按优先级尝试检测
	candidates := []string{"python3", "python"}

	for _, name := range candidates {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}

		// 获取版本号
		version, err := getPythonVersion(path)
		if err != nil {
			continue
		}

		// 确保不是 Python 2.x
		if strings.HasPrefix(version, "2.") {
			continue
		}

		info.Available = true
		info.Version = version
		info.Path = path
		return info
	}

	return info
}

// getPythonVersion 执行 python --version 获取版本号
func getPythonVersion(pythonPath string) (string, error) {
	cmd := exec.Command(pythonPath, "--version")
	cmdutil.HideWindow(cmd) // Windows 上隐藏 cmd 黑色窗口
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	// 输出格式: "Python 3.11.5\n"
	line := strings.TrimSpace(string(output))
	parts := strings.SplitN(line, " ", 2)
	if len(parts) == 2 {
		return parts[1], nil
	}

	return line, nil
}
