package cv

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"gocv.io/x/gocv"
)

// ReadImage 读取图像文件
func ReadImage(filename string) (gocv.Mat, error) {
	mat := gocv.IMRead(filename, gocv.IMReadColor)
	if mat.Empty() {
		return mat, fmt.Errorf("无法读取图像: %s", filename)
	}
	return mat, nil
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
