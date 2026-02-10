package input

import "github.com/go-vgo/robotgo"

// CopyToClipboard 复制到剪贴板
func CopyToClipboard(text string) error {
	return robotgo.WriteAll(text)
}

// ReadClipboard 读取剪贴板
func ReadClipboard() (string, error) {
	return robotgo.ReadAll()
}
