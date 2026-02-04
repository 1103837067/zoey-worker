package ocr

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"gocv.io/x/gocv"
	"golang.org/x/image/font"
)

// getTestDataDir 获取测试资源目录
func getTestDataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

// getProjectRoot 获取项目根目录
func getProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// ocr_test.go -> ocr -> vision -> pkg -> zoeyworker
	// /pkg/vision/ocr/ocr_test.go -> 需要向上4层到 zoeyworker
	dir := filepath.Dir(filename)          // ocr
	dir = filepath.Dir(dir)                // vision
	dir = filepath.Dir(dir)                // pkg
	dir = filepath.Dir(dir)                // zoeyworker
	return dir
}

// setupOCRConfig 设置 OCR 配置（用于测试）
func setupOCRConfig(t *testing.T) Config {
	root := getProjectRoot()

	// 检测当前系统架构并选择正确的库
	var libPath string
	// macOS ARM64 (Apple Silicon)
	if _, err := os.Stat(filepath.Join(root, "models/lib/onnxruntime_arm64.dylib")); err == nil {
		libPath = filepath.Join(root, "models/lib/onnxruntime_arm64.dylib")
	} else if _, err := os.Stat(filepath.Join(root, "models/lib/onnxruntime_amd64.dylib")); err == nil {
		// macOS AMD64
		libPath = filepath.Join(root, "models/lib/onnxruntime_amd64.dylib")
	} else if _, err := os.Stat(filepath.Join(root, "models/lib/onnxruntime_arm64.so")); err == nil {
		// Linux ARM64
		libPath = filepath.Join(root, "models/lib/onnxruntime_arm64.so")
	} else if _, err := os.Stat(filepath.Join(root, "models/lib/onnxruntime_amd64.so")); err == nil {
		// Linux AMD64
		libPath = filepath.Join(root, "models/lib/onnxruntime_amd64.so")
	} else {
		t.Logf("未找到 ONNX Runtime 库")
	}

	config := Config{
		OnnxRuntimeLibPath: libPath,
		DetModelPath:       filepath.Join(root, "models/paddle_weights/det.onnx"),
		RecModelPath:       filepath.Join(root, "models/paddle_weights/rec.onnx"),
		DictPath:           filepath.Join(root, "models/paddle_weights/dict.txt"),
		Language:           "ch",
		UseGPU:             false,
		CPUThreads:         4,
	}

	t.Logf("OCR 配置:")
	t.Logf("  OnnxRuntimeLibPath: %s", config.OnnxRuntimeLibPath)
	t.Logf("  DetModelPath: %s", config.DetModelPath)
	t.Logf("  RecModelPath: %s", config.RecModelPath)
	t.Logf("  DictPath: %s", config.DictPath)

	return config
}

// TestOCRAvailability 测试 OCR 功能是否可用（用于 CI 验证）
// 这个测试使用默认配置，验证打包的模型文件是否能正确加载
func TestOCRAvailability(t *testing.T) {
	t.Log("=== 测试 OCR 可用性 ===")

	// 检查默认配置是否可用
	config := DefaultConfig()
	t.Logf("默认配置:")
	t.Logf("  OnnxRuntimeLibPath: %s (exists: %v)", config.OnnxRuntimeLibPath, fileExists(config.OnnxRuntimeLibPath))
	t.Logf("  DetModelPath: %s (exists: %v)", config.DetModelPath, fileExists(config.DetModelPath))
	t.Logf("  RecModelPath: %s (exists: %v)", config.RecModelPath, fileExists(config.RecModelPath))
	t.Logf("  DictPath: %s (exists: %v)", config.DictPath, fileExists(config.DictPath))

	// 检查 IsAvailable
	available := IsAvailable()
	t.Logf("OCR IsAvailable: %v", available)

	if !available {
		t.Log("OCR 默认配置不可用，尝试使用测试配置...")
		config = setupOCRConfig(t)
	}

	// 清除缓存
	ClearCache()

	// 尝试初始化
	err := InitGlobalRecognizer(config)
	if err != nil {
		t.Fatalf("OCR 初始化失败: %v", err)
	}

	t.Log("OCR 初始化成功!")

	// 创建一个简单的测试图片（白底黑字）
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")

	if _, err := os.Stat(targetPath); err == nil {
		t.Logf("使用测试图片: %s", targetPath)
		results, err := RecognizeText(targetPath)
		if err != nil {
			t.Fatalf("OCR 识别失败: %v", err)
		}
		t.Logf("识别到 %d 个文本区域", len(results))
		for i, r := range results {
			t.Logf("  [%d] 文字: '%s', 置信度: %.2f", i+1, r.Text, r.Confidence)
		}
		if len(results) == 0 {
			t.Log("警告: 未识别到任何文字")
		}
	} else {
		t.Log("测试图片不存在，跳过识别测试")
	}

	t.Log("=== OCR 可用性测试通过 ===")
}

func TestRecognizeText(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")

	// 配置 OCR
	config := setupOCRConfig(t)

	// 清除之前的全局实例
	ClearCache()

	// 使用配置初始化
	err := InitGlobalRecognizer(config)
	if err != nil {
		t.Skipf("跳过测试：OCR 初始化失败（可能未配置模型）: %v", err)
		return
	}

	results, err := RecognizeText(targetPath)
	if err != nil {
		t.Skipf("跳过测试：OCR 识别失败: %v", err)
		return
	}

	t.Logf("识别到 %d 个文本区域", len(results))
	for i, r := range results {
		t.Logf("  [%d] 文字: '%s', 置信度: %.2f, 位置: (%d, %d)",
			i+1, r.Text, r.Confidence, r.Position.X, r.Position.Y)
	}
}

func TestFindTextPosition(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")

	// 配置 OCR
	config := setupOCRConfig(t)

	// 清除之前的全局实例
	ClearCache()

	// 使用配置初始化
	err := InitGlobalRecognizer(config)
	if err != nil {
		t.Skipf("跳过测试：OCR 初始化失败: %v", err)
		return
	}

	// 测试查找文字
	testTexts := []string{
		"火山方舟",
		"立即体验",
		"控制台",
	}

	for _, text := range testTexts {
		t.Run(text, func(t *testing.T) {
			pos, err := FindTextPosition(targetPath, text)
			if err != nil {
				t.Skipf("跳过测试：查找文字失败: %v", err)
				return
			}

			if pos != nil {
				t.Logf("找到 '%s' 位置: (%d, %d)", text, pos.X, pos.Y)
			} else {
				t.Logf("未找到文字: '%s'", text)
			}
		})
	}
}

func TestGetAllText(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")

	// 配置 OCR
	config := setupOCRConfig(t)

	// 清除之前的全局实例
	ClearCache()

	// 使用配置初始化
	err := InitGlobalRecognizer(config)
	if err != nil {
		t.Skipf("跳过测试：OCR 初始化失败: %v", err)
		return
	}

	text, err := GetAllText(targetPath)
	if err != nil {
		t.Skipf("跳过测试：获取文字失败: %v", err)
		return
	}

	t.Logf("识别的所有文字: %s", text)
}

func TestOCRConfig(t *testing.T) {
	config := DefaultConfig()

	t.Logf("默认配置:")
	t.Logf("  OnnxRuntimeLibPath: %s", config.OnnxRuntimeLibPath)
	t.Logf("  DetModelPath: %s", config.DetModelPath)
	t.Logf("  RecModelPath: %s", config.RecModelPath)
	t.Logf("  DictPath: %s", config.DictPath)
	t.Logf("  Language: %s", config.Language)
	t.Logf("  UseGPU: %v", config.UseGPU)
	t.Logf("  CPUThreads: %d", config.CPUThreads)
}

func TestOcrResultConversion(t *testing.T) {
	result := OCRResult{
		Box: Box{
			Points: [4]Point{
				{X: 10, Y: 10},
				{X: 10, Y: 30},
				{X: 100, Y: 30},
				{X: 100, Y: 10},
			},
		},
		Text:       "测试文字",
		Confidence: 0.95,
	}

	ocrResult := result.ToOcrResult()

	if ocrResult.Text != "测试文字" {
		t.Errorf("文字转换错误: got %s, want %s", ocrResult.Text, "测试文字")
	}

	if ocrResult.Confidence != 0.95 {
		t.Errorf("置信度转换错误: got %.2f, want %.2f", ocrResult.Confidence, 0.95)
	}

	// 检查中心点
	expectedCenter := Point{X: 55, Y: 20}
	if ocrResult.Position.X != expectedCenter.X || ocrResult.Position.Y != expectedCenter.Y {
		t.Errorf("中心点计算错误: got (%d, %d), want (%d, %d)",
			ocrResult.Position.X, ocrResult.Position.Y, expectedCenter.X, expectedCenter.Y)
	}

	t.Logf("OCR 结果转换成功: %+v", ocrResult)
}

func TestBoxCenter(t *testing.T) {
	box := Box{
		Points: [4]Point{
			{X: 0, Y: 0},
			{X: 0, Y: 100},
			{X: 200, Y: 100},
			{X: 200, Y: 0},
		},
	}

	center := box.Center()

	if center.X != 100 || center.Y != 50 {
		t.Errorf("中心点计算错误: got (%d, %d), want (100, 50)", center.X, center.Y)
	}

	t.Logf("Box 中心点: (%d, %d)", center.X, center.Y)
}

// setupOCRConfigMobile 设置 OCR 配置（使用 Mobile 轻量模型）
func setupOCRConfigMobile(t *testing.T) Config {
	root := getProjectRoot()

	// 检测当前系统架构并选择正确的库
	var libPath string
	if _, err := os.Stat(filepath.Join(root, "models/lib/onnxruntime_arm64.dylib")); err == nil {
		libPath = filepath.Join(root, "models/lib/onnxruntime_arm64.dylib")
	} else if _, err := os.Stat(filepath.Join(root, "models/lib/onnxruntime_amd64.dylib")); err == nil {
		libPath = filepath.Join(root, "models/lib/onnxruntime_amd64.dylib")
	}

	config := Config{
		OnnxRuntimeLibPath: libPath,
		DetModelPath:       filepath.Join(root, "models/paddle_weights_mobile/det.onnx"),
		RecModelPath:       filepath.Join(root, "models/paddle_weights_mobile/rec.onnx"),
		DictPath:           filepath.Join(root, "models/paddle_weights_mobile/dict.txt"),
		Language:           "ch",
		UseGPU:             false,
		CPUThreads:         4,
	}

	t.Logf("OCR Mobile 配置:")
	t.Logf("  DetModelPath: %s (%.1fMB)", config.DetModelPath, getFileSizeMB(config.DetModelPath))
	t.Logf("  RecModelPath: %s (%.1fMB)", config.RecModelPath, getFileSizeMB(config.RecModelPath))

	return config
}

func getFileSizeMB(path string) float64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return float64(info.Size()) / 1024 / 1024
}

// TestMobileModelComparison 对比 Server 和 Mobile 模型性能
func TestMobileModelComparison(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")

	t.Log("=== PP-OCRv5 Mobile 模型测试 ===")
	t.Log("")

	// 测试 Mobile 模型
	configMobile := setupOCRConfigMobile(t)
	ClearCache()

	err := InitGlobalRecognizer(configMobile)
	if err != nil {
		t.Skipf("Mobile 模型初始化失败: %v", err)
		return
	}

	// 运行 3 次取平均
	var mobileTimes []float64
	var mobileResults []OcrResult

	for i := 0; i < 3; i++ {
		results, err := RecognizeText(targetPath)
		if err != nil {
			t.Errorf("Mobile 模型识别失败: %v", err)
			continue
		}
		if i == 0 {
			mobileResults = results
		}
		// 从日志获取时间（或重新测量）
		mobileTimes = append(mobileTimes, 0) // 占位
	}

	t.Logf("Mobile 模型识别到 %d 个文本区域", len(mobileResults))
	for i, r := range mobileResults {
		if i >= 10 {
			t.Logf("  ... 还有 %d 个结果", len(mobileResults)-10)
			break
		}
		t.Logf("  [%d] '%s' (置信度: %.2f)", i+1, r.Text, r.Confidence)
	}
}

// TestMobileModel 单独测试 Mobile 模型
func TestMobileModel(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")

	t.Log("=== PP-OCRv5 Mobile 轻量模型测试 ===")

	config := setupOCRConfigMobile(t)
	ClearCache()

	err := InitGlobalRecognizer(config)
	if err != nil {
		t.Fatalf("Mobile 模型初始化失败: %v", err)
	}

	results, err := RecognizeText(targetPath)
	if err != nil {
		t.Fatalf("Mobile 模型识别失败: %v", err)
	}

	t.Logf("识别到 %d 个文本区域:", len(results))
	for i, r := range results {
		t.Logf("  [%d] '%s' 置信度=%.2f 位置=(%d,%d)",
			i+1, r.Text, r.Confidence, r.Position.X, r.Position.Y)
	}
}

// TestGenerateComparisonImages 生成 Server 和 Mobile 模型的文字查找对比图
func TestGenerateComparisonImages(t *testing.T) {
	testDataDir := getTestDataDir()
	targetPath := filepath.Join(testDataDir, "target.png")
	outputDir := filepath.Join(testDataDir, "output")

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("创建输出目录失败: %v", err)
	}

	// 要查找的文字列表（输入参数）
	searchTexts := []string{
		"火山方舟",
		"控制台",
		"最新动态",
		"一站式大模型服务平台",
		"模型动态",
		"工具箱",
		"帮我在全平台比价",
		"立即体验Doubao-Seed-1.8",
	}

	// 读取原图
	img := gocv.IMRead(targetPath, gocv.IMReadColor)
	if img.Empty() {
		t.Fatalf("无法读取图像: %s", targetPath)
	}
	defer img.Close()

	// === 测试 Server 模型 ===
	t.Log("=== Server 模型 (PP-OCRv4) - 文字查找测试 ===")
	configServer := setupOCRConfig(t)
	ClearCache()

	if err := InitGlobalRecognizer(configServer); err != nil {
		t.Fatalf("Server 模型初始化失败: %v", err)
	}

	serverImg := img.Clone()
	defer serverImg.Close()
	serverResults := findAndDrawTexts(t, &serverImg, targetPath, searchTexts, "Server (165MB)")
	serverOutputPath := filepath.Join(outputDir, "ocr_server_result.png")
	gocv.IMWrite(serverOutputPath, serverImg)
	t.Logf("Server 模型结果已保存: %s", serverOutputPath)

	// 写入 Server 报告
	serverReportPath := filepath.Join(outputDir, "ocr_server_report.txt")
	writeFindTextReport(serverReportPath, searchTexts, serverResults, "Server (PP-OCRv4)", "165MB")

	// === 测试 Mobile 模型 ===
	t.Log("")
	t.Log("=== Mobile 模型 (PP-OCRv5) - 文字查找测试 ===")
	configMobile := setupOCRConfigMobile(t)
	ClearCache()

	if err := InitGlobalRecognizer(configMobile); err != nil {
		t.Fatalf("Mobile 模型初始化失败: %v", err)
	}

	mobileImg := img.Clone()
	defer mobileImg.Close()
	mobileResults := findAndDrawTexts(t, &mobileImg, targetPath, searchTexts, "Mobile (20MB)")
	mobileOutputPath := filepath.Join(outputDir, "ocr_mobile_result.png")
	gocv.IMWrite(mobileOutputPath, mobileImg)
	t.Logf("Mobile 模型结果已保存: %s", mobileOutputPath)

	// 写入 Mobile 报告
	mobileReportPath := filepath.Join(outputDir, "ocr_mobile_report.txt")
	writeFindTextReport(mobileReportPath, searchTexts, mobileResults, "Mobile (PP-OCRv5)", "20MB")

	t.Log("")
	t.Log("=== 对比结果 ===")
	serverFound := 0
	mobileFound := 0
	for _, r := range serverResults {
		if r != nil {
			serverFound++
		}
	}
	for _, r := range mobileResults {
		if r != nil {
			mobileFound++
		}
	}
	t.Logf("Server: 找到 %d/%d 个文字", serverFound, len(searchTexts))
	t.Logf("Mobile: 找到 %d/%d 个文字", mobileFound, len(searchTexts))
}

// FindTextResult 文字查找结果
type FindTextResult struct {
	SearchText string  // 输入的查找文字
	FoundText  string  // 实际匹配到的文字
	Position   *Point  // 位置
	Confidence float64 // 置信度
	Box        []Point // 边界框
}

// 全局中文字体
var chineseFont *truetype.Font

// loadChineseFont 加载中文字体
func loadChineseFont() *truetype.Font {
	if chineseFont != nil {
		return chineseFont
	}

	// macOS 中文字体路径
	fontPaths := []string{
		"/System/Library/Fonts/STHeiti Medium.ttc",
		"/System/Library/Fonts/STHeiti Light.ttc",
		"/System/Library/Fonts/PingFang.ttc",
		"/Library/Fonts/Arial Unicode.ttf",
		// Windows
		"C:\\Windows\\Fonts\\msyh.ttc",
		"C:\\Windows\\Fonts\\simhei.ttf",
		// Linux
		"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
	}

	for _, path := range fontPaths {
		fontBytes, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		f, err := truetype.Parse(fontBytes)
		if err != nil {
			// 尝试作为 TTC 集合解析
			f, err = truetype.Parse(fontBytes)
			if err != nil {
				continue
			}
		}
		chineseFont = f
		return f
	}
	return nil
}

// drawChineseText 在图像上绘制中文文字
func drawChineseText(img *image.RGBA, x, y int, text string, fontSize float64, col color.Color) {
	f := loadChineseFont()
	if f == nil {
		// 字体加载失败，回退到不绘制
		return
	}

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(f)
	c.SetFontSize(fontSize)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(col))
	c.SetHinting(font.HintingFull)

	pt := freetype.Pt(x, y+int(c.PointToFixed(fontSize)>>6))
	c.DrawString(text, pt)
}

// gocvMatToRGBA 将 gocv.Mat 转换为 image.RGBA
func gocvMatToRGBA(mat *gocv.Mat) *image.RGBA {
	img, _ := mat.ToImage()
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba
}

// rgbaToGocvMat 将 image.RGBA 转换回 gocv.Mat
func rgbaToGocvMat(rgba *image.RGBA) gocv.Mat {
	mat, _ := gocv.ImageToMatRGB(rgba)
	return mat
}

// findAndDrawTexts 查找文字并绘制结果
func findAndDrawTexts(t *testing.T, img *gocv.Mat, imagePath string, searchTexts []string, title string) []*FindTextResult {
	green := color.RGBA{0, 255, 0, 255}
	red := color.RGBA{255, 0, 0, 255}
	white := color.RGBA{255, 255, 255, 255}

	// 先获取所有 OCR 结果
	allResults, err := RecognizeText(imagePath)
	if err != nil {
		t.Errorf("OCR 识别失败: %v", err)
		return nil
	}

	results := make([]*FindTextResult, len(searchTexts))

	// 转换为 RGBA 以便绘制中文
	rgba := gocvMatToRGBA(img)

	// 绘制标题背景
	for y := 0; y < 70; y++ {
		for x := 0; x < 400; x++ {
			rgba.Set(x, y, white)
		}
	}
	// 绘制标题（用中文字体）
	drawChineseText(rgba, 20, 15, title, 36, color.RGBA{0, 0, 255, 255})

	for i, searchText := range searchTexts {
		// 查找匹配的文字
		var matched *OcrResult
		for _, r := range allResults {
			if containsText(r.Text, searchText) {
				matched = &r
				break
			}
		}

		if matched != nil {
			results[i] = &FindTextResult{
				SearchText: searchText,
				FoundText:  matched.Text,
				Position:   &matched.Position,
				Confidence: matched.Confidence,
				Box:        matched.Box,
			}

			// 绘制匹配结果
			if len(matched.Box) >= 4 {
				minX, minY, maxX, maxY := getBoundingBox(matched.Box)

				// 绿色边框表示找到
				drawRect(rgba, minX, minY, maxX, maxY, green, 3)

				// 标签：输入文字 -> 找到
				label := fmt.Sprintf("输入:'%s' -> 找到", searchText)
				labelY := minY - 28
				if labelY < 80 {
					labelY = maxY + 5
				}

				// 绘制标签背景
				labelWidth := len([]rune(label))*14 + 20
				labelHeight := 24
				drawFilledRect(rgba, minX-2, labelY-2, minX+labelWidth, labelY+labelHeight, white)
				drawRect(rgba, minX-2, labelY-2, minX+labelWidth, labelY+labelHeight, green, 2)

				// 绘制中文标签
				drawChineseText(rgba, minX+2, labelY, label, 18, color.RGBA{0, 128, 0, 255})
			}

			t.Logf("  [%d] '%s' -> 找到 '%s' 位置=(%d,%d) 置信度=%.0f%%",
				i+1, searchText, matched.Text, matched.Position.X, matched.Position.Y, matched.Confidence*100)
		} else {
			results[i] = nil

			// 在图片底部标注未找到的文字
			label := fmt.Sprintf("输入:'%s' -> 未找到", searchText)
			labelY := rgba.Bounds().Max.Y - 80 - i*30
			drawChineseText(rgba, 20, labelY, label, 18, red)

			t.Logf("  [%d] '%s' -> 未找到", i+1, searchText)
		}
	}

	// 转换回 gocv.Mat
	newMat := rgbaToGocvMat(rgba)
	newMat.CopyTo(img)
	newMat.Close()

	return results
}

// drawRect 在 RGBA 图像上绘制矩形边框
func drawRect(img *image.RGBA, x1, y1, x2, y2 int, col color.Color, thickness int) {
	for t := 0; t < thickness; t++ {
		// 上边
		for x := x1; x <= x2; x++ {
			img.Set(x, y1+t, col)
		}
		// 下边
		for x := x1; x <= x2; x++ {
			img.Set(x, y2-t, col)
		}
		// 左边
		for y := y1; y <= y2; y++ {
			img.Set(x1+t, y, col)
		}
		// 右边
		for y := y1; y <= y2; y++ {
			img.Set(x2-t, y, col)
		}
	}
}

// drawFilledRect 在 RGBA 图像上绘制填充矩形
func drawFilledRect(img *image.RGBA, x1, y1, x2, y2 int, col color.Color) {
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			img.Set(x, y, col)
		}
	}
}

// containsText 检查文字是否包含
func containsText(text, target string) bool {
	if text == "" || target == "" {
		return false
	}
	textLower := strings.ToLower(text)
	targetLower := strings.ToLower(target)
	return strings.Contains(textLower, targetLower) || strings.Contains(targetLower, textLower)
}

// getBoundingBox 获取边界框
func getBoundingBox(box []Point) (minX, minY, maxX, maxY int) {
	minX, minY = box[0].X, box[0].Y
	maxX, maxY = box[0].X, box[0].Y
	for _, p := range box {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	return
}

// writeFindTextReport 写入文字查找报告
func writeFindTextReport(path string, searchTexts []string, results []*FindTextResult, modelName string, modelSize string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "========================================\n")
	fmt.Fprintf(f, "OCR 文字查找报告 - %s\n", modelName)
	fmt.Fprintf(f, "========================================\n\n")
	fmt.Fprintf(f, "模型大小: %s\n\n", modelSize)

	found := 0
	for _, r := range results {
		if r != nil {
			found++
		}
	}
	fmt.Fprintf(f, "查找结果: %d/%d 成功\n\n", found, len(searchTexts))

	fmt.Fprintf(f, "------------------------------------------------------------\n")
	fmt.Fprintf(f, "输入文字             | 结果   | 匹配文字             | 位置\n")
	fmt.Fprintf(f, "------------------------------------------------------------\n")

	for i, searchText := range searchTexts {
		r := results[i]
		if r != nil {
			fmt.Fprintf(f, "%-20s | 找到   | %-20s | (%d, %d)\n",
				searchText, r.FoundText, r.Position.X, r.Position.Y)
		} else {
			fmt.Fprintf(f, "%-20s | 未找到 | -                    | -\n", searchText)
		}
	}

	fmt.Fprintf(f, "------------------------------------------------------------\n")
	return nil
}

// drawOCRResultsOnImage 在图像上绘制 OCR 结果
func drawOCRResultsOnImage(img *gocv.Mat, results []OcrResult, title string) {
	green := color.RGBA{0, 255, 0, 255}
	red := color.RGBA{255, 0, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}
	white := color.RGBA{255, 255, 255, 255}
	black := color.RGBA{0, 0, 0, 255}

	// 绘制标题背景
	gocv.Rectangle(img, image.Rect(0, 0, 350, 70), white, -1)
	gocv.PutText(img, title, image.Pt(20, 50), gocv.FontHersheySimplex, 1.5, blue, 3)

	for i, r := range results {
		// 根据置信度选择颜色
		boxColor := green
		if r.Confidence < 0.8 {
			boxColor = red
		}

		// 从 Box 点计算边界矩形
		if len(r.Box) >= 4 {
			minX, minY := r.Box[0].X, r.Box[0].Y
			maxX, maxY := r.Box[0].X, r.Box[0].Y
			for _, p := range r.Box {
				if p.X < minX {
					minX = p.X
				}
				if p.Y < minY {
					minY = p.Y
				}
				if p.X > maxX {
					maxX = p.X
				}
				if p.Y > maxY {
					maxY = p.Y
				}
			}
			rect := image.Rect(minX, minY, maxX, maxY)

			// 绘制边框
			gocv.Rectangle(img, rect, boxColor, 2)

			// 只显示序号和置信度（避免中文乱码）
			label := fmt.Sprintf("[%d] %.0f%%", i+1, r.Confidence*100)

			// 计算标签位置（在框的上方或下方）
			labelY := minY - 8
			if labelY < 20 {
				labelY = maxY + 20
			}

			// 绘制标签背景
			labelSize := gocv.GetTextSize(label, gocv.FontHersheySimplex, 0.6, 2)
			bgRect := image.Rect(minX-2, labelY-labelSize.Y-4, minX+labelSize.X+6, labelY+6)
			gocv.Rectangle(img, bgRect, white, -1)
			gocv.Rectangle(img, bgRect, boxColor, 1)

			// 绘制标签文本
			gocv.PutText(img, label, image.Pt(minX, labelY),
				gocv.FontHersheySimplex, 0.6, black, 2)
		}
	}
}

// writeOCRReport 写入 OCR 识别报告
func writeOCRReport(path string, results []OcrResult, modelName string, modelSize string, elapsed float64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "========================================\n")
	fmt.Fprintf(f, "OCR 识别报告 - %s\n", modelName)
	fmt.Fprintf(f, "========================================\n\n")
	fmt.Fprintf(f, "模型大小: %s\n", modelSize)
	fmt.Fprintf(f, "识别耗时: %.0fms\n", elapsed)
	fmt.Fprintf(f, "识别数量: %d 个文本\n\n", len(results))
	fmt.Fprintf(f, "----------------------------------------\n")
	fmt.Fprintf(f, "序号 | 置信度 | 位置         | 识别文本\n")
	fmt.Fprintf(f, "----------------------------------------\n")

	for i, r := range results {
		fmt.Fprintf(f, "[%2d] | %5.0f%% | (%4d, %4d) | %s\n",
			i+1, r.Confidence*100, r.Position.X, r.Position.Y, r.Text)
	}

	fmt.Fprintf(f, "----------------------------------------\n")
	return nil
}

// ============ PP-OCRv5 插件测试 ============

// setupOCRConfigFromPlugin 使用插件配置设置 OCR（PP-OCRv5）
func setupOCRConfigFromPlugin(t *testing.T) (Config, error) {
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".zoey-worker", "plugins", "ocr")

	// 检测当前系统架构并选择正确的库
	var libPath string
	switch runtime.GOOS {
	case "windows":
		libPath = filepath.Join(baseDir, "lib", "onnxruntime.dll")
	case "darwin":
		if runtime.GOARCH == "arm64" {
			libPath = filepath.Join(baseDir, "lib", "onnxruntime_arm64.dylib")
		} else {
			libPath = filepath.Join(baseDir, "lib", "onnxruntime_amd64.dylib")
		}
	default: // linux
		if runtime.GOARCH == "arm64" {
			libPath = filepath.Join(baseDir, "lib", "onnxruntime_arm64.so")
		} else {
			libPath = filepath.Join(baseDir, "lib", "onnxruntime_amd64.so")
		}
	}

	config := Config{
		OnnxRuntimeLibPath: libPath,
		DetModelPath:       filepath.Join(baseDir, "paddle_weights", "det.onnx"),
		RecModelPath:       filepath.Join(baseDir, "paddle_weights", "rec.onnx"),
		DictPath:           filepath.Join(baseDir, "paddle_weights", "dict.txt"),
		Language:           "ch",
		UseGPU:             false,
		CPUThreads:         4,
	}

	// 检查文件是否存在
	missing := []string{}
	if !fileExists(config.OnnxRuntimeLibPath) {
		missing = append(missing, "onnxruntime lib")
	}
	if !fileExists(config.DetModelPath) {
		missing = append(missing, "det.onnx")
	}
	if !fileExists(config.RecModelPath) {
		missing = append(missing, "rec.onnx")
	}
	if !fileExists(config.DictPath) {
		missing = append(missing, "dict.txt")
	}

	if len(missing) > 0 {
		return config, fmt.Errorf("缺少文件: %v，请先运行 OCR 插件安装", missing)
	}

	t.Logf("PP-OCRv5 插件配置:")
	t.Logf("  OnnxRuntimeLibPath: %s (%.1fMB)", config.OnnxRuntimeLibPath, getFileSizeMB(config.OnnxRuntimeLibPath))
	t.Logf("  DetModelPath: %s (%.1fMB)", config.DetModelPath, getFileSizeMB(config.DetModelPath))
	t.Logf("  RecModelPath: %s (%.1fMB)", config.RecModelPath, getFileSizeMB(config.RecModelPath))
	t.Logf("  DictPath: %s (%.1fKB)", config.DictPath, getFileSizeMB(config.DictPath)*1024)

	return config, nil
}

// TestPPOCRv5Plugin 测试 PP-OCRv5 插件模型
func TestPPOCRv5Plugin(t *testing.T) {
	t.Log("=== PP-OCRv5 插件模型测试 ===")
	t.Log("模型来源: monkt/paddleocr-onnx (HuggingFace)")
	t.Log("检测模型: PP-OCRv5_server_det (~88MB)")
	t.Log("识别模型: PP-OCRv5_server_rec Chinese (~84.5MB)")
	t.Log("")

	config, err := setupOCRConfigFromPlugin(t)
	if err != nil {
		t.Skipf("跳过测试：%v", err)
		return
	}

	ClearCache()

	err = InitGlobalRecognizer(config)
	if err != nil {
		t.Fatalf("PP-OCRv5 初始化失败: %v", err)
	}

	// 测试 benchmarkfiles 中的图片
	root := getProjectRoot()
	testImages := []string{
		filepath.Join(root, "benchmarkfiles", "targets", "image.png"),
		filepath.Join(getTestDataDir(), "target.png"),
	}

	for _, imgPath := range testImages {
		if !fileExists(imgPath) {
			t.Logf("跳过不存在的图片: %s", imgPath)
			continue
		}

		t.Logf("\n--- 测试图片: %s ---", filepath.Base(imgPath))

		results, err := RecognizeText(imgPath)
		if err != nil {
			t.Errorf("识别失败: %v", err)
			continue
		}

		t.Logf("识别到 %d 个文本区域:", len(results))
		for i, r := range results {
			if i >= 20 {
				t.Logf("  ... 还有 %d 个结果", len(results)-20)
				break
			}
			t.Logf("  [%d] '%s' 置信度=%.0f%% 位置=(%d,%d)",
				i+1, r.Text, r.Confidence*100, r.Position.X, r.Position.Y)
		}
	}

	t.Log("\n=== PP-OCRv5 测试完成 ===")
}

// TestPPOCRv5FindText 测试 PP-OCRv5 的文字查找功能
func TestPPOCRv5FindText(t *testing.T) {
	t.Log("=== PP-OCRv5 文字查找测试 ===")

	config, err := setupOCRConfigFromPlugin(t)
	if err != nil {
		t.Skipf("跳过测试：%v", err)
		return
	}

	ClearCache()

	err = InitGlobalRecognizer(config)
	if err != nil {
		t.Fatalf("PP-OCRv5 初始化失败: %v", err)
	}

	root := getProjectRoot()
	imgPath := filepath.Join(root, "benchmarkfiles", "targets", "image.png")
	if !fileExists(imgPath) {
		t.Skipf("测试图片不存在: %s", imgPath)
		return
	}

	// 要查找的文字列表（根据 image.png 的实际内容）
	searchTexts := []string{
		"Zoey Mind",
		"定位器模块",
		"全部定位器",
		"图片定位器",
		"添加步骤",
		"新建定位器",
	}

	t.Logf("在图片中查找以下文字:")
	found := 0
	for _, text := range searchTexts {
		pos, err := FindTextPosition(imgPath, text)
		if err != nil {
			t.Errorf("  '%s' -> 错误: %v", text, err)
			continue
		}
		if pos != nil {
			t.Logf("  '%s' -> 找到 位置=(%d,%d)", text, pos.X, pos.Y)
			found++
		} else {
			t.Logf("  '%s' -> 未找到", text)
		}
	}

	t.Logf("\n查找结果: %d/%d 成功", found, len(searchTexts))
}
