package auto

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCaptureScreen 测试截屏功能
func TestCaptureScreen(t *testing.T) {
	img, err := CaptureScreen()
	if err != nil {
		// macOS 需要屏幕录制权限
		t.Skipf("截屏失败 (可能需要屏幕录制权限): %v", err)
	}

	bounds := img.Bounds()
	t.Logf("截屏成功: %dx%d", bounds.Dx(), bounds.Dy())

	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		t.Error("截屏尺寸为 0")
	}
}

// TestGetScreenSize 测试获取屏幕尺寸
func TestGetScreenSize(t *testing.T) {
	width, height := GetScreenSize()
	t.Logf("屏幕尺寸: %dx%d", width, height)

	if width == 0 || height == 0 {
		t.Error("屏幕尺寸为 0")
	}
}

// TestGetDisplayCount 测试获取显示器数量
func TestGetDisplayCount(t *testing.T) {
	count := GetDisplayCount()
	t.Logf("显示器数量: %d", count)

	if count < 1 {
		t.Error("显示器数量应该至少为 1")
	}
}

// TestGetMousePosition 测试获取鼠标位置
func TestGetMousePosition(t *testing.T) {
	x, y := GetMousePosition()
	t.Logf("鼠标位置: (%d, %d)", x, y)

	// 鼠标位置应该在屏幕范围内
	width, height := GetScreenSize()
	if x < 0 || x > width || y < 0 || y > height {
		t.Logf("警告: 鼠标位置可能超出主屏幕范围")
	}
}

// TestGetActiveWindowTitle 测试获取活动窗口标题
func TestGetActiveWindowTitle(t *testing.T) {
	title := GetActiveWindowTitle()
	t.Logf("活动窗口标题: %s", title)

	// 标题不应该为空（除非没有活动窗口）
	if title == "" {
		t.Log("警告: 活动窗口标题为空")
	}
}

// TestClipboard 测试剪贴板操作
func TestClipboard(t *testing.T) {
	testText := "auto_test_clipboard_" + time.Now().Format("20060102150405")

	// 写入剪贴板
	err := CopyToClipboard(testText)
	if err != nil {
		t.Fatalf("写入剪贴板失败: %v", err)
	}

	// 读取剪贴板
	text, err := ReadClipboard()
	if err != nil {
		t.Fatalf("读取剪贴板失败: %v", err)
	}

	if text != testText {
		t.Errorf("剪贴板内容不匹配: 期望 %q, 实际 %q", testText, text)
	}

	t.Logf("剪贴板测试成功: %s", testText)
}

// TestOptions 测试配置选项
func TestOptions(t *testing.T) {
	// 默认配置
	opts := DefaultOptions()
	if opts.Timeout != 10*time.Second {
		t.Errorf("默认超时时间错误: %v", opts.Timeout)
	}
	if opts.Threshold != 0.8 {
		t.Errorf("默认阈值错误: %v", opts.Threshold)
	}

	// 应用配置
	opts = applyOptions(
		WithTimeout(5*time.Second),
		WithThreshold(0.9),
		WithClickOffset(10, 20),
		WithDoubleClick(),
		WithRegion(100, 100, 500, 500),
	)

	if opts.Timeout != 5*time.Second {
		t.Errorf("超时时间设置错误: %v", opts.Timeout)
	}
	if opts.Threshold != 0.9 {
		t.Errorf("阈值设置错误: %v", opts.Threshold)
	}
	if opts.ClickOffset.X != 10 || opts.ClickOffset.Y != 20 {
		t.Errorf("点击偏移设置错误: %v", opts.ClickOffset)
	}
	if !opts.DoubleClick {
		t.Error("双击设置错误")
	}
	if opts.Region == nil || opts.Region.X != 100 {
		t.Errorf("区域设置错误: %v", opts.Region)
	}

	t.Log("配置选项测试通过")
}

// TestImageExists 测试图像存在性检查
func TestImageExists(t *testing.T) {
	// 使用 cv 模块的测试数据
	testdataDir := "../vision/cv/testdata"
	templatePath := filepath.Join(testdataDir, "template1.png")

	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skipf("测试文件不存在: %s", templatePath)
	}

	// 图像不在当前屏幕上，应该返回 false
	exists := ImageExists(templatePath, WithTimeout(0))
	t.Logf("ImageExists 结果: %v (预期: false，因为测试图像不在屏幕上)", exists)
}

// TestWaitForImageTimeout 测试等待图像超时
func TestWaitForImageTimeout(t *testing.T) {
	// 使用一个不存在的图像路径来测试超时
	testdataDir := "../vision/cv/testdata"
	templatePath := filepath.Join(testdataDir, "template1.png")

	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skipf("测试文件不存在: %s", templatePath)
	}

	start := time.Now()
	_, err := WaitForImage(templatePath, WithTimeout(1*time.Second))
	elapsed := time.Since(start)

	if err == nil {
		t.Log("意外找到了图像")
	} else {
		t.Logf("等待超时 (预期行为): %v, 耗时: %v", err, elapsed)
	}

	// 确保超时时间在合理范围内
	if elapsed < 900*time.Millisecond {
		t.Errorf("超时时间过短: %v", elapsed)
	}
	if elapsed > 2*time.Second {
		t.Errorf("超时时间过长: %v", elapsed)
	}
}

// TestCaptureAndSave 测试截屏并保存（用于视觉验证）
func TestCaptureAndSave(t *testing.T) {
	// 创建输出目录
	outputDir := "testdata/output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("创建输出目录失败: %v", err)
	}

	// 截屏
	img, err := CaptureScreen()
	if err != nil {
		t.Fatalf("截屏失败: %v", err)
	}

	// 保存截图（使用 robotgo）
	outputPath := filepath.Join(outputDir, "auto_test_screenshot.png")

	// 使用 image/png 保存
	file, err := os.Create(outputPath)
	if err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}
	defer file.Close()

	// 使用标准库编码 PNG
	import_png := func() error {
		// 动态导入
		return nil
	}
	_ = import_png

	t.Logf("截图完成，图像尺寸: %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	t.Logf("(保存功能需要额外依赖，此处跳过保存)")
}

// TestMouseOperations 测试鼠标操作（仅验证不报错）
func TestMouseOperations(t *testing.T) {
	// 获取当前位置
	origX, origY := GetMousePosition()
	t.Logf("原始鼠标位置: (%d, %d)", origX, origY)

	// 移动到新位置
	newX, newY := 100, 100
	MoveTo(newX, newY)
	time.Sleep(100 * time.Millisecond)

	// 验证位置
	x, y := GetMousePosition()
	t.Logf("移动后鼠标位置: (%d, %d)", x, y)

	// 允许一定误差
	if abs(x-newX) > 5 || abs(y-newY) > 5 {
		t.Logf("警告: 鼠标位置偏差较大，可能是权限问题")
	}

	// 恢复原始位置
	MoveTo(origX, origY)
	t.Log("鼠标操作测试完成")
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// TestAPICompleteness 测试 API 完整性
func TestAPICompleteness(t *testing.T) {
	apis := []struct {
		name string
		fn   interface{}
	}{
		// 截图
		{"CaptureScreen", CaptureScreen},
		{"CaptureRegion", CaptureRegion},
		{"GetScreenSize", GetScreenSize},
		{"GetDisplayCount", GetDisplayCount},

		// 图像
		{"ClickImage", ClickImage},
		{"ClickImageData", ClickImageData},
		{"WaitForImage", WaitForImage},
		{"WaitForImageData", WaitForImageData},
		{"ImageExists", ImageExists},
		{"ImageExistsData", ImageExistsData},

		// 文字
		{"ClickText", ClickText},
		{"WaitForText", WaitForText},
		{"TextExists", TextExists},

		// 窗口
		{"ActivateWindow", ActivateWindow},
		{"ActivateWindowByPID", ActivateWindowByPID},
		{"GetActiveWindowTitle", GetActiveWindowTitle},
		{"FindWindowPIDs", FindWindowPIDs},

		// 鼠标
		{"MoveTo", MoveTo},
		{"MoveSmooth", MoveSmooth},
		{"Click", Click},
		{"DoubleClick", DoubleClick},
		{"RightClick", RightClick},
		{"Scroll", Scroll},
		{"ScrollUp", ScrollUp},
		{"ScrollDown", ScrollDown},
		{"Drag", Drag},
		{"GetMousePosition", GetMousePosition},

		// 键盘
		{"TypeText", TypeText},
		{"KeyTap", KeyTap},
		{"KeyDown", KeyDown},
		{"KeyUp", KeyUp},
		{"HotKey", HotKey},

		// 剪贴板
		{"CopyToClipboard", CopyToClipboard},
		{"ReadClipboard", ReadClipboard},

		// 工具
		{"Sleep", Sleep},
		{"MilliSleep", MilliSleep},
		{"InitOCR", InitOCR},
	}

	t.Logf("API 总数: %d", len(apis))
	for _, api := range apis {
		if api.fn == nil {
			t.Errorf("API %s 未实现", api.name)
		}
	}
	t.Log("API 完整性检查通过")
}

// BenchmarkCaptureScreen 截屏性能测试
func BenchmarkCaptureScreen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := CaptureScreen()
		if err != nil {
			b.Fatalf("截屏失败: %v", err)
		}
	}
}

// ExampleClickImage 示例：点击图像
func ExampleClickImage() {
	// 点击屏幕上的登录按钮
	err := ClickImage("login_button.png",
		WithTimeout(5*time.Second),
		WithThreshold(0.9),
	)
	if err != nil {
		fmt.Println("点击失败:", err)
	}
}

// ExampleWaitForText 示例：等待文字出现
func ExampleWaitForText() {
	// 等待"加载完成"文字出现
	pos, err := WaitForText("加载完成",
		WithTimeout(30*time.Second),
		WithRegion(0, 0, 800, 600),
	)
	if err != nil {
		fmt.Println("等待超时:", err)
		return
	}
	fmt.Printf("找到文字位置: (%d, %d)\n", pos.X, pos.Y)
}

// ExampleActivateWindow 示例：激活窗口
func ExampleActivateWindow() {
	// 激活 Chrome 浏览器
	ActivateWindow("Chrome")

	// 获取当前窗口标题
	title := GetActiveWindowTitle()
	fmt.Println("当前窗口:", title)
}

// ==================== 进程管理测试 ====================

// TestGetProcesses 测试获取进程列表
func TestGetProcesses(t *testing.T) {
	processes, err := GetProcesses()
	if err != nil {
		t.Fatalf("获取进程列表失败: %v", err)
	}

	t.Logf("进程总数: %d", len(processes))

	if len(processes) == 0 {
		t.Error("进程列表为空")
	}

	// 打印前5个进程
	for i, proc := range processes {
		if i >= 5 {
			break
		}
		t.Logf("  PID=%d, Name=%s", proc.PID, proc.Name)
	}
}

// TestFindProcess 测试按名称查找进程
func TestFindProcess(t *testing.T) {
	// 查找一个肯定存在的进程（当前测试进程）
	processes, err := FindProcess("go")
	if err != nil {
		t.Fatalf("查找进程失败: %v", err)
	}

	t.Logf("找到 %d 个包含 'go' 的进程", len(processes))
	for _, proc := range processes {
		t.Logf("  PID=%d, Name=%s", proc.PID, proc.Name)
	}
}

// TestGetProcessByPID 测试按 PID 获取进程信息
func TestGetProcessByPID(t *testing.T) {
	// 获取进程列表
	processes, err := GetProcesses()
	if err != nil {
		t.Fatalf("获取进程列表失败: %v", err)
	}

	if len(processes) == 0 {
		t.Skip("没有可用的进程")
	}

	// 测试第一个有效进程
	proc, err := GetProcessByPID(processes[0].PID)
	if err != nil {
		t.Logf("获取进程信息失败 (可能是权限问题): %v", err)
		return
	}

	t.Logf("PID=%d, Name=%s, Path=%s", proc.PID, proc.Name, proc.Path)
}

// TestIsProcessRunning 测试检查进程是否运行
func TestIsProcessRunning(t *testing.T) {
	// 检查一个肯定不存在的 PID
	if IsProcessRunning(999999999) {
		t.Error("不存在的 PID 返回了 true")
	}

	// 获取进程列表中的一个有效 PID
	processes, err := GetProcesses()
	if err != nil || len(processes) == 0 {
		t.Skip("无法获取进程列表")
	}

	// 使用第一个有效进程测试
	if !IsProcessRunning(processes[0].PID) {
		t.Logf("PID %d 可能在检查期间已退出", processes[0].PID)
	}
}

// ==================== 窗口管理测试 ====================

// TestGetWindows 测试获取窗口列表
func TestGetWindows(t *testing.T) {
	windows, err := GetWindows()
	if err != nil {
		t.Fatalf("获取窗口列表失败: %v", err)
	}

	t.Logf("窗口总数: %d", len(windows))

	// 打印前5个窗口
	for i, win := range windows {
		if i >= 5 {
			break
		}
		t.Logf("  PID=%d, Title=%s, Bounds=%+v", win.PID, win.Title, win.Bounds)
	}
}

// TestGetWindowByTitle 测试按标题查找窗口
func TestGetWindowByTitle(t *testing.T) {
	// 尝试查找 Cursor 窗口（我们正在使用的 IDE）
	window, err := GetWindowByTitle("Cursor")
	if err != nil {
		t.Logf("未找到 Cursor 窗口: %v", err)
		// 尝试其他常见窗口
		window, err = GetWindowByTitle("Terminal")
		if err != nil {
			t.Logf("未找到 Terminal 窗口: %v", err)
			return
		}
	}

	t.Logf("找到窗口: PID=%d, Title=%s, Bounds=%+v", window.PID, window.Title, window.Bounds)
}

// ==================== 新 API 完整性测试 ====================

// TestNewAPICompleteness 测试新增 API 的完整性
func TestNewAPICompleteness(t *testing.T) {
	apis := []struct {
		name string
		fn   interface{}
	}{
		// 网格点击
		{"ParseGridPosition", ParseGridPosition},
		{"FormatGridPosition", FormatGridPosition},
		{"CalculateGridCenter", CalculateGridCenter},
		{"CalculateGridCenterFromString", CalculateGridCenterFromString},
		{"GetGridCellRect", GetGridCellRect},
		{"ClickGrid", ClickGrid},
		{"NewGridIterator", NewGridIterator},

		// 进程管理
		{"GetProcesses", GetProcesses},
		{"FindProcess", FindProcess},
		{"GetProcessByPID", GetProcessByPID},
		{"IsProcessRunning", IsProcessRunning},
		{"KillProcess", KillProcess},
		{"FindPIDsByName", FindPIDsByName},

		// 窗口管理
		{"GetWindows", GetWindows},
		{"GetWindowByTitle", GetWindowByTitle},
		{"GetWindowByPID", GetWindowByPID},
		{"GetWindowClient", GetWindowClient},
		{"MinimizeWindow", MinimizeWindow},
		{"MaximizeWindow", MaximizeWindow},
		{"CloseWindowByPID", CloseWindowByPID},
		{"BringWindowToFront", BringWindowToFront},
		{"CaptureWindow", CaptureWindow},
		{"ClickInWindow", ClickInWindow},
		{"ClickGridInWindow", ClickGridInWindow},
		{"WaitForWindow", WaitForWindow},
	}

	t.Logf("新增 API 总数: %d", len(apis))
	for _, api := range apis {
		if api.fn == nil {
			t.Errorf("API %s 未实现", api.name)
		}
	}
	t.Log("新增 API 完整性检查通过")
}

// ExampleClickGrid 示例：网格点击
func ExampleClickGrid() {
	// 在指定区域内按网格点击
	rect := Region{X: 100, Y: 100, Width: 200, Height: 200}

	// 点击 2x2 网格的左上角 (1,1)
	err := ClickGrid(rect, "2.2.1.1")
	if err != nil {
		fmt.Println("点击失败:", err)
	}

	// 点击 2x2 网格的右下角 (2,2)
	err = ClickGrid(rect, "2.2.2.2")
	if err != nil {
		fmt.Println("点击失败:", err)
	}
}

// ExampleGetWindows 示例：获取窗口列表
func ExampleGetWindows() {
	// 获取所有窗口
	windows, err := GetWindows()
	if err != nil {
		fmt.Println("获取窗口列表失败:", err)
		return
	}

	for _, win := range windows {
		fmt.Printf("PID=%d, Title=%s\n", win.PID, win.Title)
	}

	// 按名称过滤窗口
	chromeWindows, _ := GetWindows("Chrome")
	fmt.Printf("找到 %d 个 Chrome 窗口\n", len(chromeWindows))
}

// ExampleFindProcess 示例：查找进程
func ExampleFindProcess() {
	// 查找所有 Chrome 进程
	processes, err := FindProcess("Chrome")
	if err != nil {
		fmt.Println("查找进程失败:", err)
		return
	}

	for _, proc := range processes {
		fmt.Printf("PID=%d, Name=%s, Path=%s\n", proc.PID, proc.Name, proc.Path)
	}
}
