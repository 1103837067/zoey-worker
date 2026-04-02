package input

import (
	"strings"

	"github.com/go-vgo/robotgo"
)

// normalizeKeyName 规范化键名为 robotgo 期望的格式（robotgo 对大小写敏感）
func normalizeKeyName(key string) string {
	// 统一转换为小写
	key = strings.ToLower(key)

	// 常见别名映射
	switch key {
	case "control":
		return "ctrl"
	case "cmd", "command", "win", "meta":
		return "command"
	case "esc":
		return "escape"
	}

	return key
}

// normalizeKeys 规范化所有键名
func normalizeKeys(keys []string) []string {
	result := make([]string, len(keys))
	for i, k := range keys {
		result[i] = normalizeKeyName(k)
	}
	return result
}

// TypeText 输入文字
func TypeText(text string) {
	robotgo.TypeStr(text)
}

// KeyTap 按键
func KeyTap(key string, modifiers ...string) {
	key = normalizeKeyName(key)
	normalizedMods := normalizeKeys(modifiers)
	if len(normalizedMods) > 0 {
		robotgo.KeyTap(key, normalizedMods)
	} else {
		robotgo.KeyTap(key)
	}
}

// KeyDown 按下键
func KeyDown(key string) {
	robotgo.KeyToggle(normalizeKeyName(key), "down")
}

// KeyUp 释放键
func KeyUp(key string) {
	robotgo.KeyToggle(normalizeKeyName(key), "up")
}

// HotKey 组合键
func HotKey(keys ...string) {
	if len(keys) == 0 {
		return
	}
	normalizedKeys := normalizeKeys(keys)
	if len(normalizedKeys) == 1 {
		robotgo.KeyTap(normalizedKeys[0])
		return
	}
	// 最后一个键是主键，前面的都是修饰键
	robotgo.KeyTap(normalizedKeys[len(normalizedKeys)-1], normalizedKeys[:len(normalizedKeys)-1]...)
}
