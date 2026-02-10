// Package process 提供进程管理功能
package process

import (
	"fmt"
	"strings"

	"github.com/go-vgo/robotgo"
	"github.com/shirou/gopsutil/v4/process"
)

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID  int    `json:"pid"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// GetProcesses 获取所有进程
func GetProcesses() ([]ProcessInfo, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, fmt.Errorf("获取进程列表失败: %w", err)
	}

	var processes []ProcessInfo
	for _, pid := range pids {
		proc, err := process.NewProcess(pid)
		if err != nil {
			continue
		}

		name, _ := proc.Name()
		exe, _ := proc.Exe()

		processes = append(processes, ProcessInfo{
			PID:  int(pid),
			Name: name,
			Path: exe,
		})
	}

	return processes, nil
}

// FindProcess 按名称查找进程 (不区分大小写，支持部分匹配)
func FindProcess(name string) ([]ProcessInfo, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, fmt.Errorf("获取进程列表失败: %w", err)
	}

	name = strings.ToLower(name)
	var matches []ProcessInfo

	for _, pid := range pids {
		proc, err := process.NewProcess(pid)
		if err != nil {
			continue
		}

		procName, err := proc.Name()
		if err != nil {
			continue
		}

		if strings.Contains(strings.ToLower(procName), name) {
			exe, _ := proc.Exe()
			matches = append(matches, ProcessInfo{
				PID:  int(pid),
				Name: procName,
				Path: exe,
			})
		}
	}

	return matches, nil
}

// GetProcessByPID 按 PID 获取进程信息
func GetProcessByPID(pid int) (*ProcessInfo, error) {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, fmt.Errorf("进程不存在: PID=%d", pid)
	}

	name, _ := proc.Name()
	exe, _ := proc.Exe()

	return &ProcessInfo{
		PID:  pid,
		Name: name,
		Path: exe,
	}, nil
}

// IsProcessRunning 检查进程是否正在运行
func IsProcessRunning(pid int) bool {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return false
	}
	running, err := proc.IsRunning()
	if err != nil {
		return false
	}
	return running
}

// KillProcess 终止进程
func KillProcess(pid int) error {
	return robotgo.Kill(pid)
}

// FindPIDsByName 按名称查找进程 PID
func FindPIDsByName(name string) ([]int, error) {
	pids, err := robotgo.FindIds(name)
	if err != nil {
		return nil, fmt.Errorf("查找进程失败: %w", err)
	}
	return pids, nil
}
