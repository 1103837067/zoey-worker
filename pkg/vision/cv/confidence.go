package cv

import (
	"gocv.io/x/gocv"
)

// CalRGBConfidence 计算 RGB 三通道置信度
// 对两张同大小彩图计算相似度，返回最小通道的置信度
func CalRGBConfidence(imgSrc, imgSearch gocv.Mat) float64 {
	// 确保图像大小一致
	if imgSrc.Rows() != imgSearch.Rows() || imgSrc.Cols() != imgSearch.Cols() {
		return 0
	}

	// 裁剪到有效像素范围 [10, 245]
	srcCropped := cropToValidRange(imgSrc)
	searchCropped := cropToValidRange(imgSearch)
	defer srcCropped.Close()
	defer searchCropped.Close()

	// 分离三个通道
	srcChannels := gocv.Split(srcCropped)
	searchChannels := gocv.Split(searchCropped)
	defer func() {
		for _, ch := range srcChannels {
			ch.Close()
		}
		for _, ch := range searchChannels {
			ch.Close()
		}
	}()

	// 计算每个通道的匹配度
	minConfidence := 1.0
	for i := 0; i < len(srcChannels) && i < len(searchChannels); i++ {
		confidence := calChannelConfidence(srcChannels[i], searchChannels[i])
		if confidence < minConfidence {
			minConfidence = confidence
		}
	}

	return minConfidence
}

// cropToValidRange 裁剪像素值到有效范围
func cropToValidRange(img gocv.Mat) gocv.Mat {
	dst := gocv.NewMat()
	// 将像素值限制在 [10, 245] 范围内
	gocv.Threshold(img, &dst, 245, 245, gocv.ThresholdTrunc)
	gocv.Threshold(dst, &dst, 10, 0, gocv.ThresholdToZero)
	return dst
}

// calChannelConfidence 计算单通道置信度
func calChannelConfidence(src, search gocv.Mat) float64 {
	result := gocv.NewMat()
	defer result.Close()

	gocv.MatchTemplate(src, search, &result, gocv.TmCcoeffNormed, gocv.NewMat())

	_, maxVal, _, _ := gocv.MinMaxLoc(result)
	return float64(maxVal)
}

// CalCcoeffConfidence 使用 TM_CCOEFF_NORMED 计算置信度
func CalCcoeffConfidence(imgSource, imgSearch gocv.Mat) float64 {
	// 转为灰度图
	srcGray := ToGray(imgSource)
	searchGray := ToGray(imgSearch)
	defer srcGray.Close()
	defer searchGray.Close()

	result := gocv.NewMat()
	defer result.Close()

	gocv.MatchTemplate(srcGray, searchGray, &result, gocv.TmCcoeffNormed, gocv.NewMat())

	_, maxVal, _, _ := gocv.MinMaxLoc(result)
	return float64(maxVal)
}
