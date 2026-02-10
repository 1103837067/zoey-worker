package screen

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
)

// ImageToBase64 将图像转换为 Base64 字符串
// format: "png" 或 "jpeg"，默认 "jpeg"（更小的体积）
// quality: JPEG 质量 1-100，默认 80
func ImageToBase64(img image.Image, format string, quality int) (string, error) {
	if img == nil {
		return "", fmt.Errorf("图像为空")
	}

	var buf bytes.Buffer
	var mimeType string

	if format == "" {
		format = "jpeg"
	}
	if quality <= 0 || quality > 100 {
		quality = 80
	}

	switch format {
	case "png":
		err := png.Encode(&buf, img)
		if err != nil {
			return "", fmt.Errorf("PNG 编码失败: %w", err)
		}
		mimeType = "image/png"
	case "jpeg", "jpg":
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		if err != nil {
			return "", fmt.Errorf("JPEG 编码失败: %w", err)
		}
		mimeType = "image/jpeg"
	default:
		return "", fmt.Errorf("不支持的图像格式: %s", format)
	}

	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str), nil
}

// CaptureScreenToBase64 截取屏幕并转换为 Base64
func CaptureScreenToBase64(quality int) (string, error) {
	img, err := CaptureScreen()
	if err != nil {
		return "", err
	}
	return ImageToBase64(img, "jpeg", quality)
}
