package predictor

import (
	"pixly/pkg/quality"
)

// QualityAdjuster adjusts prediction parameters based on quality analysis
type QualityAdjuster struct {
	analyzer *quality.Analyzer
}

// NewQualityAdjuster creates a new quality adjuster
func NewQualityAdjuster() *QualityAdjuster {
	return &QualityAdjuster{
		analyzer: quality.NewAnalyzer(),
	}
}

// AdjustParams adjusts prediction parameters based on quality analysis
func (qa *QualityAdjuster) AdjustParams(
	prediction *Prediction,
	qualityMetrics *quality.QualityMetrics,
) *Prediction {
	if prediction == nil || qualityMetrics == nil {
		return prediction
	}
	
	// 根据格式调整
	switch qualityMetrics.Format {
	case "png":
		qa.adjustPNGParams(prediction, qualityMetrics)
	case "jpg", "jpeg":
		qa.adjustJPEGParams(prediction, qualityMetrics)
	case "gif":
		qa.adjustGIFParams(prediction, qualityMetrics)
	case "webp":
		qa.adjustWebPParams(prediction, qualityMetrics)
	}
	
	return prediction
}

// adjustPNGParams adjusts parameters for PNG files
func (qa *QualityAdjuster) adjustPNGParams(
	prediction *Prediction,
	qualityMetrics *quality.QualityMetrics,
) {
	// 高质量PNG使用更高effort
	if qualityMetrics.QualityClass == "极高" || qualityMetrics.QualityClass == "高" {
		if prediction.Params.Effort < 9 {
			prediction.Params.Effort = 9
		}
	}
	
	// 已经高度压缩的PNG可能不值得转换
	if qualityMetrics.BytesPerPixel < 0.5 {
		prediction.ShouldExplore = false
		if prediction.Confidence > 0.3 {
			prediction.Confidence = 0.3
		}
	}
	
	// 大文件降低effort以提高速度
	if qualityMetrics.SizeClass == "极大" || qualityMetrics.SizeClass == "大" {
		if prediction.Params.Effort > 5 {
			prediction.Params.Effort = 5
		}
	}
	
	// 小文件使用最高effort
	if qualityMetrics.SizeClass == "小" || qualityMetrics.SizeClass == "极小" {
		prediction.Params.Effort = 9
	}
}

// adjustJPEGParams adjusts parameters for JPEG files
func (qa *QualityAdjuster) adjustJPEGParams(
	prediction *Prediction,
	qualityMetrics *quality.QualityMetrics,
) {
	// JPEG 4:4:4采样有更大压缩潜力
	if strings.Contains(qualityMetrics.PixelFormat, "444") || 
	   strings.Contains(qualityMetrics.PixelFormat, "yuvj444p") {
		prediction.ExpectedSaving = 0.35  // 预期节省35%
		prediction.Confidence = 0.9
	}
	
	// JPEG 4:2:0采样压缩潜力较小
	if strings.Contains(qualityMetrics.PixelFormat, "420") || 
	   strings.Contains(qualityMetrics.PixelFormat, "yuvj420p") {
		prediction.ExpectedSaving = 0.18  // 预期节省18%
		prediction.Confidence = 0.75
	}
	
	// 照片类型使用无损JPEG转换
	if qualityMetrics.ContentType == "photo" {
		prediction.Params.LosslessJPEG = 1
	}
}

// adjustGIFParams adjusts parameters for GIF files
func (qa *QualityAdjuster) adjustGIFParams(
	prediction *Prediction,
	qualityMetrics *quality.QualityMetrics,
) {
	// GIF通常压缩潜力很大
	prediction.ExpectedSaving = 0.75  // 预期节省75%
	prediction.Confidence = 0.95
	
	// 根据大小调整CRF
	if qualityMetrics.SizeClass == "极大" || qualityMetrics.SizeClass == "大" {
		// 大GIF使用稍高CRF以加快速度
		if prediction.Params.CRF == 0 {
			prediction.Params.CRF = 32
		}
	}
}

// adjustWebPParams adjusts parameters for WebP files
func (qa *QualityAdjuster) adjustWebPParams(
	prediction *Prediction,
	qualityMetrics *quality.QualityMetrics,
) {
	// WebP处理类似PNG/JPEG
	if qualityMetrics.BytesPerPixel > 2.0 {
		// 高质量WebP
		prediction.Params.Effort = 8
	} else if qualityMetrics.BytesPerPixel < 0.5 {
		// 已压缩的WebP
		prediction.Confidence = 0.5
	}
}

// AnalyzeAndAdjust analyzes a file and adjusts prediction parameters
func (qa *QualityAdjuster) AnalyzeAndAdjust(
	filePath string,
	prediction *Prediction,
) (*Prediction, *quality.QualityMetrics, error) {
	// 分析文件质量
	metrics, err := qa.analyzer.Analyze(filePath)
	if err != nil {
		return prediction, nil, err
	}
	
	// 调整预测参数
	adjusted := qa.AdjustParams(prediction, metrics)
	
	return adjusted, metrics, nil
}
