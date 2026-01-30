package auto

import (
	"fmt"
	"image"
	"time"

	"github.com/go-vgo/robotgo"
	"gocv.io/x/gocv"

	"github.com/zoeyai/zoeyworker/pkg/vision/cv"
	"github.com/zoeyai/zoeyworker/pkg/vision/ocr"
)

// ==================== 截图操作 ====================

// CaptureScreen 截取屏幕
// displayID: 显示器 ID，-1 表示当前显示器
func CaptureScreen(displayID ...int) (image.Image, error) {
	if len(displayID) > 0 && displayID[0] >= 0 {
		robotgo.DisplayID = displayID[0]
	}
	img, err := robotgo.CaptureImg()
	if err != nil {
		return nil, fmt.Errorf("截屏失败: %w", err)
	}
	return img, nil
}

// CaptureRegion 截取屏幕区域
func CaptureRegion(x, y, width, height int) (image.Image, error) {
	img, err := robotgo.CaptureImg(x, y, width, height)
	if err != nil {
		return nil, fmt.Errorf("截取区域失败: %w", err)
	}
	return img, nil
}

// GetScreenSize 获取屏幕尺寸
func GetScreenSize() (width, height int) {
	return robotgo.GetScreenSize()
}

// GetDisplayCount 获取显示器数量
func GetDisplayCount() int {
	return robotgo.DisplaysNum()
}

// ==================== 图像操作 ====================

// ClickImage 点击图像位置
func ClickImage(templatePath string, opts ...Option) error {
	o := applyOptions(opts...)

	pos, err := waitForImageInternal(templatePath, o)
	if err != nil {
		return err
	}

	return clickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
}

// ClickImageData 点击图像位置（使用图像数据）
func ClickImageData(template image.Image, opts ...Option) error {
	o := applyOptions(opts...)

	pos, err := waitForImageDataInternal(template, o)
	if err != nil {
		return err
	}

	return clickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
}

// WaitForImage 等待图像出现
func WaitForImage(templatePath string, opts ...Option) (*Point, error) {
	o := applyOptions(opts...)
	return waitForImageInternal(templatePath, o)
}

// WaitForImageData 等待图像出现（使用图像数据）
func WaitForImageData(template image.Image, opts ...Option) (*Point, error) {
	o := applyOptions(opts...)
	return waitForImageDataInternal(template, o)
}

// ImageExists 检查图像是否存在
func ImageExists(templatePath string, opts ...Option) bool {
	o := applyOptions(opts...)
	o.Timeout = 0 // 不等待
	pos, _ := waitForImageInternal(templatePath, o)
	return pos != nil
}

// ImageExistsData 检查图像是否存在（使用图像数据）
func ImageExistsData(template image.Image, opts ...Option) bool {
	o := applyOptions(opts...)
	o.Timeout = 0
	pos, _ := waitForImageDataInternal(template, o)
	return pos != nil
}

// waitForImageInternal 内部等待图像函数
func waitForImageInternal(templatePath string, o *Options) (*Point, error) {
	tmpl := cv.NewTemplate(templatePath,
		cv.WithTemplateThreshold(o.Threshold),
		cv.WithTemplateMethods(o.Methods...),
	)

	startTime := time.Now()
	for {
		screen, err := captureForMatch(o)
		if err != nil {
			return nil, err
		}

		pos, err := tmpl.MatchIn(screen)
		screen.Close()

		if err != nil {
			return nil, fmt.Errorf("匹配失败: %w", err)
		}
		if pos != nil {
			// 如果有区域限制，加上区域偏移
			if o.Region != nil {
				return &Point{X: pos.X + o.Region.X, Y: pos.Y + o.Region.Y}, nil
			}
			return &Point{X: pos.X, Y: pos.Y}, nil
		}

		if o.Timeout == 0 || time.Since(startTime) > o.Timeout {
			return nil, fmt.Errorf("等待图像超时: %s", templatePath)
		}

		time.Sleep(o.Interval)
	}
}

// waitForImageDataInternal 内部等待图像函数（使用图像数据）
func waitForImageDataInternal(template image.Image, o *Options) (*Point, error) {
	// 转换 image.Image 为 gocv.Mat
	templateMat, err := gocv.ImageToMatRGB(template)
	if err != nil {
		return nil, fmt.Errorf("转换模板图像失败: %w", err)
	}
	defer templateMat.Close()

	startTime := time.Now()
	for {
		screen, err := captureForMatch(o)
		if err != nil {
			return nil, err
		}

		// 使用模板匹配
		matcher := cv.NewTemplateMatching(templateMat, screen, o.Threshold, false)
		result, err := matcher.FindBestResult()
		screen.Close()

		if err != nil {
			return nil, fmt.Errorf("匹配失败: %w", err)
		}
		if result != nil {
			pos := result.Result
			if o.Region != nil {
				return &Point{X: pos.X + o.Region.X, Y: pos.Y + o.Region.Y}, nil
			}
			return &Point{X: pos.X, Y: pos.Y}, nil
		}

		if o.Timeout == 0 || time.Since(startTime) > o.Timeout {
			return nil, fmt.Errorf("等待图像超时")
		}

		time.Sleep(o.Interval)
	}
}

// ==================== 文字操作 ====================

// 全局 OCR 识别器
var globalTextRecognizer *ocr.TextRecognizer

// InitOCR 初始化 OCR（可选，不调用会自动初始化）
func InitOCR(config ocr.Config) error {
	var err error
	globalTextRecognizer, err = ocr.NewTextRecognizer(config)
	return err
}

// getTextRecognizer 获取或创建 OCR 识别器
func getTextRecognizer() (*ocr.TextRecognizer, error) {
	if globalTextRecognizer == nil {
		// 尝试使用插件提供的配置
		ocrPlugin := getOCRPlugin()
		if ocrPlugin != nil && ocrPlugin.IsInstalled() {
			onnxPath, detPath, recPath, dictPath, err := ocrPlugin.GetConfig()
			if err == nil {
				config := ocr.Config{
					OnnxRuntimeLibPath: onnxPath,
					DetModelPath:       detPath,
					RecModelPath:       recPath,
					DictPath:           dictPath,
				}
				recognizer, err := ocr.NewTextRecognizer(config)
				if err == nil {
					globalTextRecognizer = recognizer
					return globalTextRecognizer, nil
				}
			}
		}

		// 回退到默认配置
		recognizer, err := ocr.GetGlobalRecognizer()
		if err != nil {
			return nil, fmt.Errorf("初始化 OCR 失败: %w", err)
		}
		globalTextRecognizer = recognizer
	}
	return globalTextRecognizer, nil
}

// OCRPluginInterface OCR 插件接口
type OCRPluginInterface interface {
	IsInstalled() bool
	GetConfig() (onnxPath, detPath, recPath, dictPath string, err error)
}

var ocrPluginInstance OCRPluginInterface

// SetOCRPlugin 设置 OCR 插件实例（避免循环导入）
func SetOCRPlugin(p OCRPluginInterface) {
	ocrPluginInstance = p
}

func getOCRPlugin() OCRPluginInterface {
	return ocrPluginInstance
}

// ClickText 点击文字位置
func ClickText(text string, opts ...Option) error {
	o := applyOptions(opts...)

	pos, err := waitForTextInternal(text, o)
	if err != nil {
		return err
	}

	return clickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
}

// WaitForText 等待文字出现
func WaitForText(text string, opts ...Option) (*Point, error) {
	o := applyOptions(opts...)
	return waitForTextInternal(text, o)
}

// TextExists 检查文字是否存在
func TextExists(text string, opts ...Option) bool {
	o := applyOptions(opts...)
	o.Timeout = 0
	pos, _ := waitForTextInternal(text, o)
	return pos != nil
}

// waitForTextInternal 内部等待文字函数
func waitForTextInternal(text string, o *Options) (*Point, error) {
	recognizer, err := getTextRecognizer()
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	for {
		// 截图
		var img image.Image
		var captureErr error
		if o.Region != nil {
			img, captureErr = CaptureRegion(o.Region.X, o.Region.Y, o.Region.Width, o.Region.Height)
		} else {
			img, captureErr = CaptureScreen(o.DisplayID)
		}
		if captureErr != nil {
			return nil, captureErr
		}

		// OCR 查找文字
		result, err := recognizer.FindText(img, text)
		if err != nil {
			return nil, fmt.Errorf("OCR 识别失败: %w", err)
		}

		if result != nil {
			if o.Region != nil {
				return &Point{X: result.X + o.Region.X, Y: result.Y + o.Region.Y}, nil
			}
			return &Point{X: result.X, Y: result.Y}, nil
		}

		if o.Timeout == 0 || time.Since(startTime) > o.Timeout {
			return nil, fmt.Errorf("等待文字超时: %s", text)
		}

		time.Sleep(o.Interval)
	}
}

// ==================== 窗口操作 ====================

// ActivateWindow 激活窗口
func ActivateWindow(name string) error {
	return activateWindowPlatform(name)
}

// ActivateWindowByPID 通过 PID 激活窗口
func ActivateWindowByPID(pid int) error {
	return activateWindowByPIDPlatform(pid)
}

// GetActiveWindowTitle 获取当前活动窗口标题
func GetActiveWindowTitle() string {
	return robotgo.GetTitle()
}

// FindWindowPIDs 查找窗口 PID
func FindWindowPIDs(name string) ([]int, error) {
	pids, err := robotgo.FindIds(name)
	if err != nil {
		return nil, fmt.Errorf("查找窗口失败: %w", err)
	}

	// 转换 int32 -> int
	result := make([]int, len(pids))
	for i, pid := range pids {
		result[i] = int(pid)
	}
	return result, nil
}

// ==================== 鼠标操作 ====================

// MoveTo 移动鼠标到指定位置
func MoveTo(x, y int) {
	robotgo.Move(x, y)
}

// MoveSmooth 平滑移动鼠标
func MoveSmooth(x, y int) {
	robotgo.MoveSmooth(x, y)
}

// Click 点击
func Click(button ...string) {
	btn := "left"
	if len(button) > 0 {
		btn = button[0]
	}
	robotgo.Click(btn, false)
}

// DoubleClick 双击
func DoubleClick(button ...string) {
	btn := "left"
	if len(button) > 0 {
		btn = button[0]
	}
	robotgo.Click(btn, true)
}

// RightClick 右键点击
func RightClick() {
	robotgo.Click("right", false)
}

// Scroll 滚动
func Scroll(x, y int) {
	robotgo.Scroll(x, y)
}

// ScrollUp 向上滚动
func ScrollUp(lines int) {
	robotgo.ScrollDir(lines, "up")
}

// ScrollDown 向下滚动
func ScrollDown(lines int) {
	robotgo.ScrollDir(lines, "down")
}

// Drag 拖拽
func Drag(x, y int) {
	robotgo.DragSmooth(x, y)
}

// GetMousePosition 获取鼠标位置
func GetMousePosition() (x, y int) {
	return robotgo.Location()
}

// ==================== 键盘操作 ====================

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

// ==================== 剪贴板操作 ====================

// CopyToClipboard 复制到剪贴板
func CopyToClipboard(text string) error {
	return robotgo.WriteAll(text)
}

// ReadClipboard 读取剪贴板
func ReadClipboard() (string, error) {
	return robotgo.ReadAll()
}

// ==================== 内部辅助函数 ====================

// captureForMatch 截图用于匹配
func captureForMatch(o *Options) (gocv.Mat, error) {
	var img image.Image
	var err error

	if o.DisplayID >= 0 {
		robotgo.DisplayID = o.DisplayID
	}

	if o.Region != nil {
		img, err = robotgo.CaptureImg(o.Region.X, o.Region.Y, o.Region.Width, o.Region.Height)
	} else {
		img, err = robotgo.CaptureImg()
	}

	if err != nil {
		return gocv.Mat{}, fmt.Errorf("截屏失败: %w", err)
	}

	mat, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("转换图像失败: %w", err)
	}

	return mat, nil
}

// clickAt 在指定位置点击
func clickAt(x, y int, o *Options) error {
	robotgo.Move(x, y)
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

// Sleep 休眠
func Sleep(d time.Duration) {
	time.Sleep(d)
}

// MilliSleep 毫秒休眠
func MilliSleep(ms int) {
	robotgo.MilliSleep(ms)
}
