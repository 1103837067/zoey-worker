package cv

import (
	"image"
	"path/filepath"
	"testing"
	"time"

	"gocv.io/x/gocv"
)

// TestAlgorithmComparison 全面对比各算法的效果
func TestAlgorithmComparison(t *testing.T) {
	testdataDir := "testdata"
	templates := []string{"template1.png", "template2.png", "template3.png"}
	targetPath := filepath.Join(testdataDir, "target.png")

	target, err := ReadImage(targetPath)
	if err != nil {
		t.Fatalf("读取目标图像失败: %v", err)
	}
	defer target.Close()

	t.Log("=" + string(make([]byte, 79, 79)))
	t.Log(" 算法对比测试 - 原始尺寸匹配")
	t.Log("=" + string(make([]byte, 79, 79)))

	for _, tmplName := range templates {
		tmplPath := filepath.Join(testdataDir, tmplName)
		template, err := ReadImage(tmplPath)
		if err != nil {
			t.Logf("跳过 %s: %v", tmplName, err)
			continue
		}

		t.Logf("\n模板: %s (%dx%d)", tmplName, template.Cols(), template.Rows())
		t.Log("-" + string(make([]byte, 60, 60)))

		// 测试各算法
		algorithms := []struct {
			name    string
			matcher func() (*MatchResult, error)
		}{
			{"模板匹配 (tpl)", func() (*MatchResult, error) {
				return NewTemplateMatching(template, target, 0.8, false).FindBestResult()
			}},
			{"多尺度匹配 (mstpl)", func() (*MatchResult, error) {
				return NewMultiScaleTemplateMatching(template, target, 0.8, false).FindBestResult()
			}},
			{"AKAZE 特征点", func() (*MatchResult, error) {
				m := NewAKAZEMatching(template, target, 0.8, false)
				defer m.Close()
				return m.FindBestResult()
			}},
			{"BRISK 特征点", func() (*MatchResult, error) {
				m := NewBRISKMatching(template, target, 0.8, false)
				defer m.Close()
				return m.FindBestResult()
			}},
		}

		for _, alg := range algorithms {
			start := time.Now()
			result, err := alg.matcher()
			elapsed := time.Since(start)

			if err != nil {
				t.Logf("  %-20s: 错误 - %v", alg.name, err)
			} else if result != nil {
				t.Logf("  %-20s: ✓ 置信度=%.4f 位置=(%d,%d) 耗时=%v",
					alg.name, result.Confidence, result.Result.X, result.Result.Y, elapsed.Round(time.Millisecond))
			} else {
				t.Logf("  %-20s: ✗ 未找到 耗时=%v", alg.name, elapsed.Round(time.Millisecond))
			}
		}

		template.Close()
	}
}

// TestAlgorithmComparison_ScaledImages 测试不同缩放比例下的表现
func TestAlgorithmComparison_ScaledImages(t *testing.T) {
	testdataDir := "testdata"
	templatePath := filepath.Join(testdataDir, "template1.png")
	targetPath := filepath.Join(testdataDir, "target.png")

	originalTemplate, err := ReadImage(templatePath)
	if err != nil {
		t.Fatalf("读取模板失败: %v", err)
	}
	defer originalTemplate.Close()

	target, err := ReadImage(targetPath)
	if err != nil {
		t.Fatalf("读取目标失败: %v", err)
	}
	defer target.Close()

	t.Log("=" + string(make([]byte, 79, 79)))
	t.Log(" 算法对比测试 - 不同缩放比例 (模拟 DPI/分辨率差异)")
	t.Log("=" + string(make([]byte, 79, 79)))
	t.Logf("原始模板大小: %dx%d", originalTemplate.Cols(), originalTemplate.Rows())
	t.Log("")

	scales := []float64{0.5, 0.75, 1.0, 1.25, 1.5, 2.0}

	// 结果统计
	type algResult struct {
		name     string
		found    int
		avgConf  float64
		avgTime  time.Duration
		totalCnt int
	}
	results := map[string]*algResult{
		"tpl":   {name: "模板匹配"},
		"mstpl": {name: "多尺度匹配"},
		"akaze": {name: "AKAZE"},
		"brisk": {name: "BRISK"},
	}

	for _, scale := range scales {
		newW := int(float64(originalTemplate.Cols()) * scale)
		newH := int(float64(originalTemplate.Rows()) * scale)

		scaledTemplate := gocv.NewMat()
		gocv.Resize(originalTemplate, &scaledTemplate, image.Point{X: newW, Y: newH}, 0, 0, gocv.InterpolationLinear)

		t.Logf("缩放 %.0f%% (模板: %dx%d)", scale*100, newW, newH)

		// 模板匹配
		start := time.Now()
		r1, _ := NewTemplateMatching(scaledTemplate, target, 0.8, false).FindBestResult()
		t1 := time.Since(start)
		results["tpl"].totalCnt++
		if r1 != nil {
			results["tpl"].found++
			results["tpl"].avgConf += r1.Confidence
			results["tpl"].avgTime += t1
			t.Logf("  tpl:   ✓ %.4f  %v", r1.Confidence, t1.Round(time.Millisecond))
		} else {
			t.Logf("  tpl:   ✗")
		}

		// 多尺度匹配
		start = time.Now()
		r2, _ := NewMultiScaleTemplateMatching(scaledTemplate, target, 0.8, false).FindBestResult()
		t2 := time.Since(start)
		results["mstpl"].totalCnt++
		if r2 != nil {
			results["mstpl"].found++
			results["mstpl"].avgConf += r2.Confidence
			results["mstpl"].avgTime += t2
			t.Logf("  mstpl: ✓ %.4f  %v", r2.Confidence, t2.Round(time.Millisecond))
		} else {
			t.Logf("  mstpl: ✗")
		}

		// AKAZE
		start = time.Now()
		m3 := NewAKAZEMatching(scaledTemplate, target, 0.8, false)
		r3, _ := m3.FindBestResult()
		m3.Close()
		t3 := time.Since(start)
		results["akaze"].totalCnt++
		if r3 != nil {
			results["akaze"].found++
			results["akaze"].avgConf += r3.Confidence
			results["akaze"].avgTime += t3
			t.Logf("  akaze: ✓ %.4f  %v", r3.Confidence, t3.Round(time.Millisecond))
		} else {
			t.Logf("  akaze: ✗")
		}

		// BRISK
		start = time.Now()
		m4 := NewBRISKMatching(scaledTemplate, target, 0.8, false)
		r4, _ := m4.FindBestResult()
		m4.Close()
		t4 := time.Since(start)
		results["brisk"].totalCnt++
		if r4 != nil {
			results["brisk"].found++
			results["brisk"].avgConf += r4.Confidence
			results["brisk"].avgTime += t4
			t.Logf("  brisk: ✓ %.4f  %v", r4.Confidence, t4.Round(time.Millisecond))
		} else {
			t.Logf("  brisk: ✗")
		}

		t.Log("")
		scaledTemplate.Close()
	}

	// 输出统计
	t.Log("=" + string(make([]byte, 79, 79)))
	t.Log(" 统计结果")
	t.Log("=" + string(make([]byte, 79, 79)))
	t.Logf("%-15s  成功率    平均置信度  平均耗时", "算法")
	t.Log("-" + string(make([]byte, 60, 60)))

	for _, key := range []string{"tpl", "mstpl", "akaze", "brisk"} {
		r := results[key]
		successRate := float64(r.found) / float64(r.totalCnt) * 100
		avgConf := float64(0)
		avgTime := time.Duration(0)
		if r.found > 0 {
			avgConf = r.avgConf / float64(r.found)
			avgTime = r.avgTime / time.Duration(r.found)
		}
		t.Logf("%-15s  %5.1f%%    %.4f      %v", r.name, successRate, avgConf, avgTime.Round(time.Millisecond))
	}
}

// TestShouldUseMultiScaleOnly 分析是否应该只用多尺度
func TestShouldUseMultiScaleOnly(t *testing.T) {
	t.Log("=" + string(make([]byte, 79, 79)))
	t.Log(" 是否应该只使用多尺度匹配？分析报告")
	t.Log("=" + string(make([]byte, 79, 79)))
	t.Log("")

	t.Log("【优点】")
	t.Log("  ✓ 自动适应不同分辨率/DPI，无需调整模板")
	t.Log("  ✓ 单一算法，代码简单，易于维护")
	t.Log("  ✓ 对缩放变化鲁棒性最好（50%-200%）")
	t.Log("")

	t.Log("【缺点】")
	t.Log("  ✗ 速度较慢：~800ms vs 模板匹配 ~10ms")
	t.Log("  ✗ 精确匹配时置信度略低于普通模板匹配")
	t.Log("  ✗ 对旋转不鲁棒（特征点匹配更好）")
	t.Log("")

	t.Log("【建议策略】")
	t.Log("")
	t.Log("  场景 A: 已知分辨率一致 → 使用普通模板匹配 (最快)")
	t.Log("    auto.ClickImage('btn.png', auto.WithMethods(cv.MatchMethodTemplate))")
	t.Log("")
	t.Log("  场景 B: 跨分辨率/DPI → 只用多尺度 (推荐)")
	t.Log("    auto.ClickImage('btn.png', auto.WithMultiScale())")
	t.Log("")
	t.Log("  场景 C: 通用场景 → 先快后慢的降级策略")
	t.Log("    auto.ClickImage('btn.png', auto.WithMultiScaleFallback())")
	t.Log("    // 先尝试快速模板匹配，失败则用多尺度")
	t.Log("")
	t.Log("  场景 D: 复杂 UI（旋转/透视变换）→ 特征点匹配")
	t.Log("    auto.ClickImage('btn.png', auto.WithMethods(cv.MatchMethodAKAZE))")
}

// TestRecommendedStrategy 测试推荐策略的效果
func TestRecommendedStrategy(t *testing.T) {
	testdataDir := "testdata"
	templatePath := filepath.Join(testdataDir, "template1.png")
	targetPath := filepath.Join(testdataDir, "target.png")

	template, _ := ReadImage(templatePath)
	defer template.Close()
	target, _ := ReadImage(targetPath)
	defer target.Close()

	t.Log("=" + string(make([]byte, 79, 79)))
	t.Log(" 推荐策略测试：先快后慢")
	t.Log("=" + string(make([]byte, 79, 79)))
	t.Log("")

	// 场景 1: 尺寸完全一致
	t.Log("场景 1: 尺寸完全一致")
	start := time.Now()
	result := tryMatchWithFallback(template, target, 0.8)
	elapsed := time.Since(start)
	if result != nil {
		t.Logf("  结果: ✓ 置信度=%.4f 总耗时=%v", result.Confidence, elapsed.Round(time.Millisecond))
		t.Log("  说明: 普通模板匹配直接成功，无需多尺度")
	}

	// 场景 2: 模板缩小 75%
	t.Log("")
	t.Log("场景 2: 模板缩小到 75%")
	scaled75 := gocv.NewMat()
	gocv.Resize(template, &scaled75, image.Point{
		X: int(float64(template.Cols()) * 0.75),
		Y: int(float64(template.Rows()) * 0.75),
	}, 0, 0, gocv.InterpolationLinear)
	defer scaled75.Close()

	start = time.Now()
	result = tryMatchWithFallback(scaled75, target, 0.8)
	elapsed = time.Since(start)
	if result != nil {
		t.Logf("  结果: ✓ 置信度=%.4f 总耗时=%v", result.Confidence, elapsed.Round(time.Millisecond))
		t.Log("  说明: 普通模板匹配失败，降级到多尺度成功")
	}

	// 场景 3: 模板放大 150%
	t.Log("")
	t.Log("场景 3: 模板放大到 150%")
	scaled150 := gocv.NewMat()
	gocv.Resize(template, &scaled150, image.Point{
		X: int(float64(template.Cols()) * 1.5),
		Y: int(float64(template.Rows()) * 1.5),
	}, 0, 0, gocv.InterpolationLinear)
	defer scaled150.Close()

	start = time.Now()
	result = tryMatchWithFallback(scaled150, target, 0.8)
	elapsed = time.Since(start)
	if result != nil {
		t.Logf("  结果: ✓ 置信度=%.4f 总耗时=%v", result.Confidence, elapsed.Round(time.Millisecond))
		t.Log("  说明: 普通模板匹配失败，降级到多尺度成功")
	}
}

// tryMatchWithFallback 先快后慢的降级策略
func tryMatchWithFallback(template, target gocv.Mat, threshold float64) *MatchResult {
	// 先尝试普通模板匹配
	result, _ := NewTemplateMatching(template, target, threshold, false).FindBestResult()
	if result != nil {
		return result
	}

	// 降级到多尺度匹配
	result, _ = NewMultiScaleTemplateMatching(template, target, threshold, false).FindBestResult()
	return result
}

// BenchmarkAlgorithmComparison 性能基准测试
func BenchmarkAlgorithmComparison(b *testing.B) {
	testdataDir := "testdata"
	templatePath := filepath.Join(testdataDir, "template1.png")
	targetPath := filepath.Join(testdataDir, "target.png")

	template, _ := ReadImage(templatePath)
	defer template.Close()
	target, _ := ReadImage(targetPath)
	defer target.Close()

	b.Run("TemplateMatching", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewTemplateMatching(template, target, 0.8, false).FindBestResult()
		}
	})

	b.Run("MultiScaleTemplate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewMultiScaleTemplateMatching(template, target, 0.8, false).FindBestResult()
		}
	})

	b.Run("AKAZE", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := NewAKAZEMatching(template, target, 0.8, false)
			m.FindBestResult()
			m.Close()
		}
	})

	b.Run("BRISK", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := NewBRISKMatching(template, target, 0.8, false)
			m.FindBestResult()
			m.Close()
		}
	})
}
