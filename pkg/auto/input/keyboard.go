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

// TypeText 输入文字
func TypeText(text string) {
	robotgo.TypeStr(text)
}

// KeyTap 按键
func KeyTap(key string, modifiers ...string) {
	key = normalizeKeyName(key)
	if len(modifiers) == 0 {
		robotgo.KeyTap(key)
		return
	}

	// 转换为 []interface{}（robotgo 要求）
	mods := make([]interface{}, len(modifiers))
	for i, m := range modifiers {
		mods[i] = normalizeKeyName(m)
	}
	robotgo.KeyTap(key, mods...)
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

	// 规范化所有键名
	normalizedKeys := make([]string, len(keys))
	for i, k := range keys {
		normalizedKeys[i] = normalizeKeyName(k)
	}

	if len(normalizedKeys) == 1 {
		robotgo.KeyTap(normalizedKeys[0])
		return
	}

	// 最后一个键是主键，前面的都是修饰键
	mainKey := normalizedKeys[len(normalizedKeys)-1]
	modifiers := normalizedKeys[:len(normalizedKeys)-1]

	// 转换为 []interface{}（robotgo 要求）
	mods := make([]interface{}, len(modifiers))
	for i, m := range modifiers {
		mods[i] = m
	}
	robotgo.KeyTap(mainKey, mods...)
}
