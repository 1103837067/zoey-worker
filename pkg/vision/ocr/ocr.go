// Package ocr 提供基于 PaddleOCR 的文字识别功能
//
// 基本用法:
//
//	// 识别图片中的所有文字
//	results, err := ocr.RecognizeText("image.png")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, r := range results {
//	    fmt.Printf("文字: %s, 置信度: %.2f\n", r.Text, r.Confidence)
//	}
//
//	// 查找特定文字的位置
//	pos, err := ocr.FindTextPosition("image.png", "登录")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("位置: (%d, %d)\n", pos.X, pos.Y)
package ocr

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/zoeyai/zoeyworker/internal/logger"
)

// RecognizeText 识别图像中的所有文字
// 支持文件路径或 image.Image
func RecognizeText(input interface{}) ([]OcrResult, error) {
	img, err := loadImage(input)
	if err != nil {
		return nil, err
	}

	recognizer, err := GetGlobalRecognizer()
	if err != nil {
		return nil, err
	}

	return recognizer.Recognize(img)
}

// FindTextPosition 查找特定文字的位置
func FindTextPosition(input interface{}, targetText string) (*Point, error) {
	if targetText == "" {
		return nil, nil
	}

	img, err := loadImage(input)
	if err != nil {
		return nil, err
	}

	recognizer, err := GetGlobalRecognizer()
	if err != nil {
		return nil, err
	}

	return recognizer.FindText(img, targetText)
}

// GetAllText 获取图像中的所有文字
func GetAllText(input interface{}) (string, error) {
	img, err := loadImage(input)
	if err != nil {
		return "", err
	}

	recognizer, err := GetGlobalRecognizer()
	if err != nil {
		return "", err
	}

	return recognizer.GetAllText(img)
}

// loadImage 加载图像
func loadImage(input interface{}) (image.Image, error) {
	switch v := input.(type) {
	case string:
		return loadImageFromFile(v)
	case image.Image:
		return v, nil
	default:
		return nil, fmt.Errorf("不支持的图像输入类型: %T", input)
	}
}

// loadImageFromFile 从文件加载图像
func loadImageFromFile(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		logger.Error("打开图像文件失败: %s, %v", filename, err)
		return nil, fmt.Errorf("打开图像文件失败: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		logger.Error("解码图像失败: %s, %v", filename, err)
		return nil, fmt.Errorf("解码图像失败: %w", err)
	}

	return img, nil
}
