package cv

import (
	"fmt"
	"image"
	"path/filepath"
	"testing"

	"gocv.io/x/gocv"
)

func TestMultiScaleTemplateMatching_Basic(t *testing.T) {
	testdataDir := "testdata"
	templatePath := filepath.Join(testdataDir, "template1.png")
	targetPath := filepath.Join(testdataDir, "target.png")

	template, err := ReadImage(templatePath)
	if err != nil {
		t.Fatalf("读取模板失败: %v", err)
	}
	defer template.Close()

	target, err := ReadImage(targetPath)
	if err != nil {
		t.Fatalf("读取目标失败: %v", err)
	}
	defer target.Close()

	matcher := NewMultiScaleTemplateMatching(template, target, 0.8, false)
	result, err := matcher.FindBestResult()

	if err != nil {
		t.Fatalf("匹配失败: %v", err)
	}

	if result != nil {
		t.Logf("多尺度匹配结果: 位置=(%d,%d), 置信度=%.4f, 耗时=%.2fms",
			result.Result.X, result.Result.Y, result.Confidence, result.Time)
	} else {
		t.Log("未找到匹配结果（可能是阈值过高或图像不匹配）")
	}
}

func TestMultiScaleTemplateMatching_ScaledTemplate(t *testing.T) {
	testdataDir := "testdata"
	templatePath := filepath.Join(testdataDir, "template1.png")
	targetPath := filepath.Join(testdataDir, "target.png")

	template, err := ReadImage(templatePath)
	if err != nil {
		t.Fatalf("读取模板失败: %v", err)
	}
	defer template.Close()

	target, err := ReadImage(targetPath)
	if err != nil {
		t.Fatalf("读取目标失败: %v", err)
	}
	defer target.Close()

	// 创建缩放后的模板（模拟不同分辨率）
	scales := []float64{0.5, 0.75, 1.0, 1.25, 1.5}

	for _, scale := range scales {
		newW := int(float64(template.Cols()) * scale)
		newH := int(float64(template.Rows()) * scale)

		scaledTemplate := gocv.NewMat()
		gocv.Resize(template, &scaledTemplate, image.Point{X: newW, Y: newH}, 0, 0, gocv.InterpolationLinear)

		t.Run(fmt.Sprintf("scale_%.2f", scale), func(t *testing.T) {
			// 普通模板匹配
			tplMatcher := NewTemplateMatching(scaledTemplate, target, 0.8, false)
			tplResult, _ := tplMatcher.FindBestResult()

			// 多尺度模板匹配
			msMatcher := NewMultiScaleTemplateMatching(scaledTemplate, target, 0.8, false)
			msResult, _ := msMatcher.FindBestResult()

			t.Logf("缩放比例 %.2fx (模板大小: %dx%d):", scale, newW, newH)
			if tplResult != nil {
				t.Logf("  普通模板匹配: 置信度=%.4f, 位置=(%d,%d)", tplResult.Confidence, tplResult.Result.X, tplResult.Result.Y)
			} else {
				t.Logf("  普通模板匹配: 未找到")
			}
			if msResult != nil {
				t.Logf("  多尺度匹配:   置信度=%.4f, 位置=(%d,%d), 耗时=%.2fms", msResult.Confidence, msResult.Result.X, msResult.Result.Y, msResult.Time)
			} else {
				t.Logf("  多尺度匹配:   未找到")
			}
		})

		scaledTemplate.Close()
	}
}

func TestMultiScaleTemplateMatching_Performance(t *testing.T) {
	testdataDir := "testdata"
	templatePath := filepath.Join(testdataDir, "template1.png")
	targetPath := filepath.Join(testdataDir, "target.png")

	template, err := ReadImage(templatePath)
	if err != nil {
		t.Fatalf("读取模板失败: %v", err)
	}
	defer template.Close()

	target, err := ReadImage(targetPath)
	if err != nil {
		t.Fatalf("读取目标失败: %v", err)
	}
	defer target.Close()

	t.Log("性能测试 - 不同步长设置:")

	steps := []float64{0.01, 0.005, 0.002}
	for _, step := range steps {
		matcher := NewMultiScaleTemplateMatchingWithParams(template, target, 0.8, false, 800, step)
		result, _ := matcher.FindBestResult()

		if result != nil {
			t.Logf("  步长=%.3f: 置信度=%.4f, 耗时=%.2fms", step, result.Confidence, result.Time)
		} else {
			t.Logf("  步长=%.3f: 未找到", step)
		}
	}
}

func TestMultiScaleTemplateMatching_UseCase(t *testing.T) {
	t.Log("多尺度模板匹配适用场景:")
	t.Log("  1. 不同分辨率显示器 (1080p vs 4K)")
	t.Log("  2. DPI 缩放 (125%/150%/200%)")
	t.Log("  3. 响应式 UI 元素大小变化")
	t.Log("  4. 录制和回放时分辨率不同")
	t.Log("")
	t.Log("使用方法:")
	t.Log("  tmpl := cv.NewTemplate('button.png', cv.WithTemplateMethods(cv.MatchMethodMultiScaleTemplate))")
	t.Log("  pos, err := tmpl.MatchIn(screen)")
}

// BenchmarkMultiScaleTemplateMatching 基准测试
func BenchmarkMultiScaleTemplateMatching(b *testing.B) {
	testdataDir := "testdata"
	templatePath := filepath.Join(testdataDir, "template1.png")
	targetPath := filepath.Join(testdataDir, "target.png")

	template, err := ReadImage(templatePath)
	if err != nil {
		b.Fatalf("读取模板失败: %v", err)
	}
	defer template.Close()

	target, err := ReadImage(targetPath)
	if err != nil {
		b.Fatalf("读取目标失败: %v", err)
	}
	defer target.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher := NewMultiScaleTemplateMatching(template, target, 0.8, false)
		matcher.FindBestResult()
	}
}
