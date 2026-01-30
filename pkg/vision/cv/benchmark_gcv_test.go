package cv

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vcaesar/gcv"
	"gocv.io/x/gocv"
)

// BenchmarkResult 基准测试结果
type BenchmarkResult struct {
	Library    string
	Method     string
	Template   string
	Found      bool
	Position   image.Point
	Confidence float64
	Duration   time.Duration
}

// TestGcvVsOurCV 对比测试 gcv 和我们的 cv 模块
func TestGcvVsOurCV(t *testing.T) {
	// 获取测试数据路径
	testdataDir := "testdata"
	targetPath := filepath.Join(testdataDir, "target.png")
	templates := []string{
		filepath.Join(testdataDir, "template1.png"),
		filepath.Join(testdataDir, "template2.png"),
		filepath.Join(testdataDir, "template3.png"),
	}

	// 检查文件是否存在
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Skipf("测试文件不存在: %s", targetPath)
	}

	// 加载目标图像
	targetMat := gocv.IMRead(targetPath, gocv.IMReadColor)
	if targetMat.Empty() {
		t.Fatalf("无法加载目标图像: %s", targetPath)
	}
	defer targetMat.Close()

	targetImg, err := targetMat.ToImage()
	if err != nil {
		t.Fatalf("转换目标图像失败: %v", err)
	}

	var results []BenchmarkResult

	for _, templatePath := range templates {
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			t.Logf("跳过不存在的模板: %s", templatePath)
			continue
		}

		templateName := filepath.Base(templatePath)
		t.Logf("\n===== 测试模板: %s =====", templateName)

		// 加载模板图像
		templateMat := gocv.IMRead(templatePath, gocv.IMReadColor)
		if templateMat.Empty() {
			t.Logf("无法加载模板图像: %s", templatePath)
			continue
		}
		templateImg, _ := templateMat.ToImage()

		// ========== 测试 gcv ==========
		t.Log("\n--- gcv 测试 ---")

		// gcv: FindAllImg (模板匹配)
		start := time.Now()
		gcvResults := gcv.FindAllImg(templateImg, targetImg, 0.8)
		gcvDuration := time.Since(start)
		gcvFound := len(gcvResults) > 0
		var gcvPos image.Point
		var gcvConf float64
		if gcvFound {
			gcvPos = image.Point{X: gcvResults[0].Middle.X, Y: gcvResults[0].Middle.Y}
			if len(gcvResults[0].MaxVal) > 0 {
				gcvConf = float64(gcvResults[0].MaxVal[0])
			}
		}
		results = append(results, BenchmarkResult{
			Library:    "gcv",
			Method:     "FindAllImg (tpl)",
			Template:   templateName,
			Found:      gcvFound,
			Position:   gcvPos,
			Confidence: gcvConf,
			Duration:   gcvDuration,
		})
		t.Logf("gcv FindAllImg: found=%v, pos=%v, conf=%.3f, time=%v", gcvFound, gcvPos, gcvConf, gcvDuration)

		// gcv: Find (模板+SIFT)
		start = time.Now()
		gcvSiftResult := gcv.Find(templateImg, targetImg, 0.8)
		gcvSiftDuration := time.Since(start)
		gcvSiftFound := gcvSiftResult.Middle.X != 0 || gcvSiftResult.Middle.Y != 0
		gcvSiftPos := image.Point{X: gcvSiftResult.Middle.X, Y: gcvSiftResult.Middle.Y}
		var gcvSiftConf float64
		if len(gcvSiftResult.MaxVal) > 0 {
			gcvSiftConf = float64(gcvSiftResult.MaxVal[0])
		}
		results = append(results, BenchmarkResult{
			Library:    "gcv",
			Method:     "Find (tpl+sift)",
			Template:   templateName,
			Found:      gcvSiftFound,
			Position:   gcvSiftPos,
			Confidence: gcvSiftConf,
			Duration:   gcvSiftDuration,
		})
		t.Logf("gcv Find (sift): found=%v, pos=%v, conf=%.3f, time=%v", gcvSiftFound, gcvSiftPos, gcvSiftConf, gcvSiftDuration)

		// ========== 测试我们的 cv ==========
		t.Log("\n--- 我们的 cv 模块测试 ---")

		// 我们的: Template Matching
		start = time.Now()
		tplMatcher := NewTemplateMatching(templateMat, targetMat, 0.8, false)
		ourTplResult, _ := tplMatcher.FindBestResult()
		ourTplDuration := time.Since(start)
		ourTplFound := ourTplResult != nil
		var ourTplPos image.Point
		var ourTplConf float64
		if ourTplFound {
			ourTplPos = image.Point{X: ourTplResult.Result.X, Y: ourTplResult.Result.Y}
			ourTplConf = ourTplResult.Confidence
		}
		results = append(results, BenchmarkResult{
			Library:    "our-cv",
			Method:     "TemplateMatching",
			Template:   templateName,
			Found:      ourTplFound,
			Position:   ourTplPos,
			Confidence: ourTplConf,
			Duration:   ourTplDuration,
		})
		t.Logf("our TemplateMatching: found=%v, pos=%v, conf=%.3f, time=%v", ourTplFound, ourTplPos, ourTplConf, ourTplDuration)

		// 我们的: AKAZE
		start = time.Now()
		akazeMatcher := NewAKAZEMatching(templateMat, targetMat, 0.8, false)
		ourAkazeResult, _ := akazeMatcher.FindBestResult()
		ourAkazeDuration := time.Since(start)
		akazeMatcher.Close()
		ourAkazeFound := ourAkazeResult != nil
		var ourAkazePos image.Point
		var ourAkazeConf float64
		if ourAkazeFound {
			ourAkazePos = image.Point{X: ourAkazeResult.Result.X, Y: ourAkazeResult.Result.Y}
			ourAkazeConf = ourAkazeResult.Confidence
		}
		results = append(results, BenchmarkResult{
			Library:    "our-cv",
			Method:     "AKAZE",
			Template:   templateName,
			Found:      ourAkazeFound,
			Position:   ourAkazePos,
			Confidence: ourAkazeConf,
			Duration:   ourAkazeDuration,
		})
		t.Logf("our AKAZE: found=%v, pos=%v, conf=%.3f, time=%v", ourAkazeFound, ourAkazePos, ourAkazeConf, ourAkazeDuration)

		// 我们的: BRISK
		start = time.Now()
		briskMatcher := NewBRISKMatching(templateMat, targetMat, 0.8, false)
		ourBriskResult, _ := briskMatcher.FindBestResult()
		ourBriskDuration := time.Since(start)
		briskMatcher.Close()
		ourBriskFound := ourBriskResult != nil
		var ourBriskPos image.Point
		var ourBriskConf float64
		if ourBriskFound {
			ourBriskPos = image.Point{X: ourBriskResult.Result.X, Y: ourBriskResult.Result.Y}
			ourBriskConf = ourBriskResult.Confidence
		}
		results = append(results, BenchmarkResult{
			Library:    "our-cv",
			Method:     "BRISK",
			Template:   templateName,
			Found:      ourBriskFound,
			Position:   ourBriskPos,
			Confidence: ourBriskConf,
			Duration:   ourBriskDuration,
		})
		t.Logf("our BRISK: found=%v, pos=%v, conf=%.3f, time=%v", ourBriskFound, ourBriskPos, ourBriskConf, ourBriskDuration)

		// 我们的: 完整流程 (多方法回退)
		start = time.Now()
		tmpl := NewTemplate(templatePath)
		ourFullResult, _ := tmpl.MatchIn(targetMat)
		ourFullDuration := time.Since(start)
		ourFullFound := ourFullResult != nil
		var ourFullPos image.Point
		if ourFullFound {
			ourFullPos = image.Point{X: ourFullResult.X, Y: ourFullResult.Y}
		}
		results = append(results, BenchmarkResult{
			Library:    "our-cv",
			Method:     "Full (tpl→akaze→brisk)",
			Template:   templateName,
			Found:      ourFullFound,
			Position:   ourFullPos,
			Confidence: 0, // 完整流程不返回置信度
			Duration:   ourFullDuration,
		})
		t.Logf("our Full: found=%v, pos=%v, time=%v", ourFullFound, ourFullPos, ourFullDuration)

		templateMat.Close()
	}

	// 输出汇总报告
	t.Log("\n\n========== 性能对比汇总 ==========")
	printReport(t, results)
}

func printReport(t *testing.T, results []BenchmarkResult) {
	// 按模板分组
	byTemplate := make(map[string][]BenchmarkResult)
	for _, r := range results {
		byTemplate[r.Template] = append(byTemplate[r.Template], r)
	}

	var report strings.Builder
	report.WriteString("\n")
	report.WriteString(fmt.Sprintf("%-15s %-25s %-8s %-15s %-10s %-12s\n",
		"Template", "Method", "Found", "Position", "Conf", "Time"))
	report.WriteString(strings.Repeat("-", 90) + "\n")

	for template, rs := range byTemplate {
		for i, r := range rs {
			foundStr := "❌"
			if r.Found {
				foundStr = "✅"
			}
			posStr := "-"
			if r.Found {
				posStr = fmt.Sprintf("(%d,%d)", r.Position.X, r.Position.Y)
			}
			confStr := "-"
			if r.Confidence > 0 {
				confStr = fmt.Sprintf("%.3f", r.Confidence)
			}

			templateCol := ""
			if i == 0 {
				templateCol = template
			}

			report.WriteString(fmt.Sprintf("%-15s %-25s %-8s %-15s %-10s %-12v\n",
				templateCol, r.Method, foundStr, posStr, confStr, r.Duration.Round(time.Microsecond)))
		}
		report.WriteString("\n")
	}

	t.Log(report.String())

	// 写入文件
	outputPath := "testdata/output/benchmark_gcv_vs_ours.txt"
	os.MkdirAll(filepath.Dir(outputPath), 0755)
	os.WriteFile(outputPath, []byte(report.String()), 0644)
	t.Logf("报告已保存到: %s", outputPath)
}

// BenchmarkGcvFindAllImg gcv FindAllImg 基准测试
func BenchmarkGcvFindAllImg(b *testing.B) {
	targetPath := "testdata/target.png"
	templatePath := "testdata/template1.png"

	targetMat := gocv.IMRead(targetPath, gocv.IMReadColor)
	if targetMat.Empty() {
		b.Skip("测试文件不存在")
	}
	defer targetMat.Close()
	targetImg, _ := targetMat.ToImage()

	templateMat := gocv.IMRead(templatePath, gocv.IMReadColor)
	if templateMat.Empty() {
		b.Skip("测试文件不存在")
	}
	defer templateMat.Close()
	templateImg, _ := templateMat.ToImage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gcv.FindAllImg(templateImg, targetImg, 0.8)
	}
}

// BenchmarkOurTemplateMatching 我们的 TemplateMatching 基准测试
func BenchmarkOurTemplateMatching(b *testing.B) {
	targetPath := "testdata/target.png"
	templatePath := "testdata/template1.png"

	targetMat := gocv.IMRead(targetPath, gocv.IMReadColor)
	if targetMat.Empty() {
		b.Skip("测试文件不存在")
	}
	defer targetMat.Close()

	templateMat := gocv.IMRead(templatePath, gocv.IMReadColor)
	if templateMat.Empty() {
		b.Skip("测试文件不存在")
	}
	defer templateMat.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher := NewTemplateMatching(templateMat, targetMat, 0.8, false)
		matcher.FindBestResult()
	}
}

// BenchmarkOurAKAZE 我们的 AKAZE 基准测试
func BenchmarkOurAKAZE(b *testing.B) {
	targetPath := "testdata/target.png"
	templatePath := "testdata/template1.png"

	targetMat := gocv.IMRead(targetPath, gocv.IMReadColor)
	if targetMat.Empty() {
		b.Skip("测试文件不存在")
	}
	defer targetMat.Close()

	templateMat := gocv.IMRead(templatePath, gocv.IMReadColor)
	if templateMat.Empty() {
		b.Skip("测试文件不存在")
	}
	defer templateMat.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher := NewAKAZEMatching(templateMat, targetMat, 0.8, false)
		matcher.FindBestResult()
		matcher.Close()
	}
}

// BenchmarkGcvSift gcv SIFT 基准测试
func BenchmarkGcvSift(b *testing.B) {
	targetPath := "testdata/target.png"
	templatePath := "testdata/template1.png"

	targetMat := gocv.IMRead(targetPath, gocv.IMReadColor)
	if targetMat.Empty() {
		b.Skip("测试文件不存在")
	}
	defer targetMat.Close()
	targetImg, _ := targetMat.ToImage()

	templateMat := gocv.IMRead(templatePath, gocv.IMReadColor)
	if templateMat.Empty() {
		b.Skip("测试文件不存在")
	}
	defer templateMat.Close()
	templateImg, _ := templateMat.ToImage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gcv.Find(templateImg, targetImg, 0.8)
	}
}
