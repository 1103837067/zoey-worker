package input

import "github.com/go-vgo/robotgo"

// TypeText 输入文字
func TypeText(text string) {
	robotgo.TypeStr(text)
}

// KeyTap 按键
func KeyTap(key string, modifiers ...string) {
	if len(modifiers) > 0 {
		robotgo.KeyTap(key, modifiers)
	} else {
		robotgo.KeyTap(key)
	}
}

// KeyDown 按下键
func KeyDown(key string) {
	robotgo.KeyToggle(key, "down")
}

// KeyUp 释放键
func KeyUp(key string) {
	robotgo.KeyToggle(key, "up")
}

// HotKey 组合键
func HotKey(keys ...string) {
	if len(keys) == 0 {
		return
	}
	if len(keys) == 1 {
		robotgo.KeyTap(keys[0])
		return
	}
	robotgo.KeyTap(keys[len(keys)-1], keys[:len(keys)-1])
}
