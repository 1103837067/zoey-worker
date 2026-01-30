package cv

import (
	"path/filepath"
	"runtime"
	"testing"
)

// getTestDataDir 获取测试资源目录
func getTestDataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func TestTemplateMatching(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")
	templatePath := filepath.Join(testDataDir, "template1.png")

	// 读取图像
	target, err := ReadImage(targetPath)
	if err != nil {
		t.Skipf("跳过测试：无法读取目标图像 %s: %v", targetPath, err)
		return
	}
	defer target.Close()

	template, err := ReadImage(templatePath)
	if err != nil {
		t.Skipf("跳过测试：无法读取模板图像 %s: %v", templatePath, err)
		return
	}
	defer template.Close()

	// 创建模板匹配器
	matcher := NewTemplateMatching(template, target, 0.8, false)

	// 查找最佳匹配
	result, err := matcher.FindBestResult()
	if err != nil {
		t.Errorf("模板匹配失败: %v", err)
		return
	}

	if result != nil {
		t.Logf("匹配成功: 位置=(%d, %d), 置信度=%.2f",
			result.Result.X, result.Result.Y, result.Confidence)
	} else {
		t.Log("未找到匹配")
	}
}

func TestTemplateMatchingFindAll(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")
	templatePath := filepath.Join(testDataDir, "template1.png")

	target, err := ReadImage(targetPath)
	if err != nil {
		t.Skipf("跳过测试：无法读取目标图像: %v", err)
		return
	}
	defer target.Close()

	template, err := ReadImage(templatePath)
	if err != nil {
		t.Skipf("跳过测试：无法读取模板图像: %v", err)
		return
	}
	defer template.Close()

	matcher := NewTemplateMatching(template, target, 0.7, false)

	results, err := matcher.FindAllResults()
	if err != nil {
		t.Errorf("查找所有匹配失败: %v", err)
		return
	}

	t.Logf("找到 %d 个匹配", len(results))
	for i, r := range results {
		t.Logf("  [%d] 位置=(%d, %d), 置信度=%.2f",
			i+1, r.Result.X, r.Result.Y, r.Confidence)
	}
}

func TestKeypointMatching(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")
	templatePath := filepath.Join(testDataDir, "template1.png")

	target, err := ReadImage(targetPath)
	if err != nil {
		t.Skipf("跳过测试：无法读取目标图像: %v", err)
		return
	}
	defer target.Close()

	template, err := ReadImage(templatePath)
	if err != nil {
		t.Skipf("跳过测试：无法读取模板图像: %v", err)
		return
	}
	defer template.Close()

	testCases := []struct {
		name    string
		matcher func() (interface{ FindBestResult() (*MatchResult, error) }, func())
	}{
		{
			name: "KAZE",
			matcher: func() (interface{ FindBestResult() (*MatchResult, error) }, func()) {
				m := NewKAZEMatching(template, target, 0.6, false)
				return m, m.Close
			},
		},
		{
			name: "BRISK",
			matcher: func() (interface{ FindBestResult() (*MatchResult, error) }, func()) {
				m := NewBRISKMatching(template, target, 0.6, false)
				return m, m.Close
			},
		},
		{
			name: "AKAZE",
			matcher: func() (interface{ FindBestResult() (*MatchResult, error) }, func()) {
				m := NewAKAZEMatching(template, target, 0.6, false)
				return m, m.Close
			},
		},
		{
			name: "ORB",
			matcher: func() (interface{ FindBestResult() (*MatchResult, error) }, func()) {
				m := NewORBMatching(template, target, 0.6, false)
				return m, m.Close
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matcher, cleanup := tc.matcher()
			defer cleanup()

			result, err := matcher.FindBestResult()
			if err != nil {
				t.Logf("%s 匹配失败: %v", tc.name, err)
				return
			}

			if result != nil {
				t.Logf("%s 匹配成功: 位置=(%d, %d), 置信度=%.2f",
					tc.name, result.Result.X, result.Result.Y, result.Confidence)
			} else {
				t.Logf("%s 未找到匹配", tc.name)
			}
		})
	}
}

func TestFindLocation(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")
	templatePath := filepath.Join(testDataDir, "template1.png")

	pos, err := FindLocation(targetPath, templatePath)
	if err != nil {
		t.Logf("FindLocation 失败: %v", err)
		return
	}

	if pos != nil {
		t.Logf("找到位置: (%d, %d)", pos.X, pos.Y)
	} else {
		t.Log("未找到位置")
	}
}

func TestFindLocationWithOptions(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")
	templatePath := filepath.Join(testDataDir, "template1.png")

	pos, err := FindLocation(targetPath, templatePath,
		WithTemplateThreshold(0.9),
		WithTemplateRGB(true),
	)
	if err != nil {
		t.Logf("FindLocation 失败: %v", err)
		return
	}

	if pos != nil {
		t.Logf("找到位置 (threshold=0.9, rgb=true): (%d, %d)", pos.X, pos.Y)
	} else {
		t.Log("未找到位置")
	}
}

func TestTemplate(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")
	templatePath := filepath.Join(testDataDir, "template1.png")

	target, err := ReadImage(targetPath)
	if err != nil {
		t.Skipf("跳过测试：无法读取目标图像: %v", err)
		return
	}
	defer target.Close()

	tmpl := NewTemplate(templatePath,
		WithTemplateThreshold(0.8),
		WithTemplateTargetPos(TargetPosMid),
	)
	defer tmpl.Close()

	pos, err := tmpl.MatchIn(target)
	if err != nil {
		t.Errorf("Template.MatchIn 失败: %v", err)
		return
	}

	if pos != nil {
		t.Logf("Template 匹配成功: (%d, %d)", pos.X, pos.Y)
	} else {
		t.Log("Template 未找到匹配")
	}
}

func TestImageUtils(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")

	// 测试读取图像
	img, err := ReadImage(targetPath)
	if err != nil {
		t.Skipf("跳过测试：无法读取图像: %v", err)
		return
	}
	defer img.Close()

	// 测试获取分辨率
	w, h := GetResolution(img)
	t.Logf("图像分辨率: %dx%d", w, h)

	// 测试灰度转换
	gray := ToGray(img)
	defer gray.Close()
	t.Logf("灰度图通道数: %d", gray.Channels())

	// 测试裁剪
	cropped := CropImage(img, [4]int{0, 0, 100, 100})
	defer cropped.Close()
	cropW, cropH := GetResolution(cropped)
	t.Logf("裁剪后分辨率: %dx%d", cropW, cropH)

	// 测试缩放
	resized := ResizeImage(img, w/2, h/2)
	defer resized.Close()
	resizedW, resizedH := GetResolution(resized)
	t.Logf("缩放后分辨率: %dx%d", resizedW, resizedH)
}
