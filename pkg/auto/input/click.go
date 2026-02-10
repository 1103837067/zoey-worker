package input

import (
	"time"

	"github.com/go-vgo/robotgo"

	"github.com/zoeyai/zoeyworker/pkg/auto"
)

// ClickAt 在指定位置点击（根据 Options 决定点击方式）
func ClickAt(x, y int, o *auto.Options) error {
	MoveTo(x, y)
	time.Sleep(50 * time.Millisecond) // 短暂延迟确保鼠标到位

	if o.RightClick {
		robotgo.Click("right", false)
	} else if o.DoubleClick {
		robotgo.Click("left", true)
	} else {
		robotgo.Click("left", false)
	}

	return nil
}
