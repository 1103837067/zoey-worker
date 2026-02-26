// Package cv 提供图像匹配功能
//
// 仅保留 SIFT 特征点匹配
//
// 基本用法:
//
//	// 在屏幕截图中查找模板
//	pos, err := cv.FindLocation("screen.png", "template.png")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("找到位置: (%d, %d)\n", pos.X, pos.Y)
//
//	// 使用自定义选项
//	pos, err := cv.FindLocation("screen.png", "template.png",
//	    cv.WithTemplateThreshold(0.9),
//	)
package cv
