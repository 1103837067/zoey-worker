package vision

import (
	"testing"
	"time"
)

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version 不应为空")
	}
	t.Logf("Version: %s", Version)
}

func TestPoint(t *testing.T) {
	p := NewPoint(10, 20)

	if p.X != 10 || p.Y != 20 {
		t.Errorf("Point 创建错误: got (%d, %d), want (10, 20)", p.X, p.Y)
	}
}

func TestRectangle(t *testing.T) {
	r := NewRectangle(10, 20, 100, 50)

	// 检查四个角点
	if r.TopLeft.X != 10 || r.TopLeft.Y != 20 {
		t.Errorf("TopLeft 错误: got (%d, %d)", r.TopLeft.X, r.TopLeft.Y)
	}
	if r.BottomLeft.X != 10 || r.BottomLeft.Y != 70 {
		t.Errorf("BottomLeft 错误: got (%d, %d)", r.BottomLeft.X, r.BottomLeft.Y)
	}
	if r.BottomRight.X != 110 || r.BottomRight.Y != 70 {
		t.Errorf("BottomRight 错误: got (%d, %d)", r.BottomRight.X, r.BottomRight.Y)
	}
	if r.TopRight.X != 110 || r.TopRight.Y != 20 {
		t.Errorf("TopRight 错误: got (%d, %d)", r.TopRight.X, r.TopRight.Y)
	}

	// 检查中心点
	center := r.Center()
	if center.X != 60 || center.Y != 45 {
		t.Errorf("Center 错误: got (%d, %d), want (60, 45)", center.X, center.Y)
	}

	// 检查宽高
	if r.Width() != 100 {
		t.Errorf("Width 错误: got %d, want 100", r.Width())
	}
	if r.Height() != 50 {
		t.Errorf("Height 错误: got %d, want 50", r.Height())
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions

	if opts.CVThreshold != 0.8 {
		t.Errorf("CVThreshold 错误: got %.2f, want 0.8", opts.CVThreshold)
	}

	if opts.OCRLanguage != "ch" {
		t.Errorf("OCRLanguage 错误: got %s, want ch", opts.OCRLanguage)
	}

	if !opts.LogEnabled {
		t.Error("LogEnabled 应为 true")
	}

	t.Logf("DefaultOptions: %+v", opts)
}

func TestOptions(t *testing.T) {
	// 保存原始配置
	original := *GetOptions()
	defer SetOptions(original)

	// 修改配置
	newOpts := Options{
		CVThreshold: 0.9,
		OCRLanguage: "en",
	}
	SetOptions(newOpts)

	current := GetOptions()
	if current.CVThreshold != 0.9 {
		t.Errorf("SetOptions 失败: CVThreshold got %.2f, want 0.9", current.CVThreshold)
	}

	// 重置配置
	ResetOptions()
	current = GetOptions()
	if current.CVThreshold != 0.8 {
		t.Errorf("ResetOptions 失败: CVThreshold got %.2f, want 0.8", current.CVThreshold)
	}
}

func TestMatchConfig(t *testing.T) {
	cfg := defaultMatchConfig()

	if cfg.threshold != 0.8 {
		t.Errorf("默认阈值错误: got %.2f", cfg.threshold)
	}

	if cfg.timeout != 10*time.Second {
		t.Errorf("默认超时错误: got %s", cfg.timeout)
	}

	// 测试 Option 函数
	WithThreshold(0.9)(cfg)
	if cfg.threshold != 0.9 {
		t.Errorf("WithThreshold 失败: got %.2f", cfg.threshold)
	}

	WithTimeout(2 * time.Second)(cfg)
	if cfg.timeout != 2*time.Second {
		t.Errorf("WithTimeout 失败: got %s", cfg.timeout)
	}
}

func TestMatchMethod(t *testing.T) {
	if MatchMethodSIFT == "" {
		t.Error("MatchMethod 不应为空")
	}
	t.Logf("MatchMethod: %s", MatchMethodSIFT)
}

func TestDefaultMatchMethods(t *testing.T) {
	if len(DefaultMatchMethods) == 0 {
		t.Error("DefaultMatchMethods 不应为空")
	}

	t.Logf("DefaultMatchMethods: %v", DefaultMatchMethods)
}
