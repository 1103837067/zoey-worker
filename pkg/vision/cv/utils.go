package cv

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"gocv.io/x/gocv"
	_ "image/gif"
	_ "image/jpeg"
)

// ReadImage 读取图像文件或 base64 数据
// 支持:
//   - 文件路径: "path/to/image.png"
//   - Base64 Data URL: "data:image/png;base64,iVBORw0KGgo..."
//   - 纯 Base64 字符串 (自动检测)
func ReadImage(filename string) (gocv.Mat, error) {
	// 检查是否是 base64 data URL
	if strings.HasPrefix(filename, "data:image/") {
		return readBase64Image(filename)
	}
	
	// 检查是否是纯 base64 字符串 (长度较长且不含路径分隔符)
	if len(filename) > 100 && !strings.ContainsAny(filename, "/\\") {
		// 尝试作为 base64 解码
		mat, err := readBase64Image("data:image/png;base64," + filename)
		if err == nil {
			return mat, nil
		}
	}

	// 作为文件路径读取
	mat := gocv.IMRead(filename, gocv.IMReadColor)
	if mat.Empty() {
		return mat, fmt.Errorf("无法读取图像: %s", filename)
	}
	return mat, nil
}

// readBase64Image 从 base64 data URL 读取图像
func readBase64Image(dataURL string) (gocv.Mat, error) {
	// 解析 data URL: data:image/png;base64,xxxxx
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return gocv.Mat{}, fmt.Errorf("无效的 base64 data URL 格式")
	}

	// 解码 base64
	imgData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("base64 解码失败: %w", err)
	}

	// 解码图像
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("图像解码失败: %w", err)
	}

	// 转换为 gocv.Mat
	return ImageToMat(img)
}

// ReadImageGray 读取灰度图像
func ReadImageGray(filename string) (gocv.Mat, error) {
	mat := gocv.IMRead(filename, gocv.IMReadGrayScale)
	if mat.Empty() {
		return mat, fmt.Errorf("无法读取图像: %s", filename)
	}
	return mat, nil
}

// WriteImage 保存图像文件
func WriteImage(filename string, img gocv.Mat) error {
	// 确保目录存在
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	if ok := gocv.IMWrite(filename, img); !ok {
		return fmt.Errorf("保存图像失败: %s", filename)
	}
	return nil
}

// ToGray 转换为灰度图
func ToGray(src gocv.Mat) gocv.Mat {
	if src.Channels() == 1 {
		return src.Clone()
	}
	dst := gocv.NewMat()
	gocv.CvtColor(src, &dst, gocv.ColorBGRToGray)
	return dst
}

// GetResolution 获取图像分辨率 (width, height)
func GetResolution(img gocv.Mat) (int, int) {
	return img.Cols(), img.Rows()
}

// CropImage 裁剪图像
// rect: [xMin, yMin, xMax, yMax]
func CropImage(img gocv.Mat, rect [4]int) gocv.Mat {
	xMin, yMin, xMax, yMax := rect[0], rect[1], rect[2], rect[3]

	// 边界检查
	if xMin < 0 {
		xMin = 0
	}
	if yMin < 0 {
		yMin = 0
	}
	if xMax > img.Cols() {
		xMax = img.Cols()
	}
	if yMax > img.Rows() {
		yMax = img.Rows()
	}

	region := img.Region(image.Rect(xMin, yMin, xMax, yMax))
	return region.Clone()
}

// ResizeImage 调整图像大小
func ResizeImage(img gocv.Mat, width, height int) gocv.Mat {
	dst := gocv.NewMat()
	gocv.Resize(img, &dst, image.Point{X: width, Y: height}, 0, 0, gocv.InterpolationLinear)
	return dst
}

// RotateImage 旋转图像
func RotateImage(img gocv.Mat, angle float64) gocv.Mat {
	center := image.Point{X: img.Cols() / 2, Y: img.Rows() / 2}
	rotMat := gocv.GetRotationMatrix2D(center, angle, 1.0)
	defer rotMat.Close()

	dst := gocv.NewMat()
	gocv.WarpAffine(img, &dst, rotMat, image.Point{X: img.Cols(), Y: img.Rows()})
	return dst
}

// ImageToMat 将 image.Image 转换为 gocv.Mat
func ImageToMat(img image.Image) (gocv.Mat, error) {
	mat, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("图像转换失败: %w", err)
	}
	// 转换为 BGR（OpenCV 默认格式）
	dst := gocv.NewMat()
	gocv.CvtColor(mat, &dst, gocv.ColorRGBToBGR)
	mat.Close()
	return dst, nil
}

// MatToImage 将 gocv.Mat 转换为 image.Image
func MatToImage(mat gocv.Mat) (image.Image, error) {
	img, err := mat.ToImage()
	if err != nil {
		return nil, fmt.Errorf("Mat 转换失败: %w", err)
	}
	return img, nil
}

// LoadImageInput 加载图像输入
// 支持 string (文件路径)、image.Image、gocv.Mat
func LoadImageInput(input interface{}) (gocv.Mat, error) {
	switch v := input.(type) {
	case string:
		return ReadImage(v)
	case image.Image:
		return ImageToMat(v)
	case gocv.Mat:
		return v.Clone(), nil
	case *gocv.Mat:
		return v.Clone(), nil
	default:
		return gocv.Mat{}, fmt.Errorf("不支持的图像输入类型: %T", input)
	}
}

// min 返回较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max 返回较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// abs 返回绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
