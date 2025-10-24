package predictor

import (
	"time"

	"go.uber.org/zap"
)

// WebPPredictor WebP专用预测器
// 核心策略：与GIF完全相同（检测动静图）
type WebPPredictor struct {
	logger *zap.Logger
}

// NewWebPPredictor 创建WebP预测器
func NewWebPPredictor(logger *zap.Logger) *WebPPredictor {
	return &WebPPredictor{
		logger: logger,
	}
}

// Predict 预测WebP的最优转换参数
// WebP黄金规则：
//
//	静态WebP → JXL distance=0（无损）
//	动态WebP → AVIF（动画格式）
func (wp *WebPPredictor) Predict(features *FileFeatures) *Prediction {
	startTime := time.Now()

	wp.logger.Debug("WebP预测",
		zap.String("file", features.FilePath),
		zap.Bool("is_animated", features.IsAnimated),
		zap.Int("frame_count", features.FrameCount))

	// 动静图检测（与GIF策略一致）
	if features.IsAnimated && features.FrameCount > 1 {
		// 动态WebP → AVIF
		return wp.predictAnimatedWebP(features, startTime)
	} else {
		// 静态WebP → JXL
		return wp.predictStaticWebP(features, startTime)
	}
}

// predictStaticWebP 预测静态WebP
// 策略：JXL distance=0（与PNG/静态GIF一致）
func (wp *WebPPredictor) predictStaticWebP(features *FileFeatures, startTime time.Time) *Prediction {
	wp.logger.Debug("静态WebP：使用JXL无损",
		zap.String("file", features.FilePath))

	params := &ConversionParams{
		TargetFormat: "jxl",
		Lossless:     true,
		Distance:     0,
		Effort:       7,
		Threads:      8,
	}

	// WebP→JXL节省率取决于WebP是有损还是无损
	expectedSaving := 0.40 // 保守预测40%

	return &Prediction{
		Params:                params,
		Confidence:            0.85, // 85%置信度（WebP情况多样）
		Method:                "rule_based",
		RuleName:              "WEBP_STATIC_JXL_LOSSLESS",
		ExpectedSaving:        expectedSaving,
		ExpectedSizeBytes:     int64(float64(features.FileSize) * (1 - expectedSaving)),
		ShouldExplore:         false,
		ExplorationCandidates: nil,
		PredictionTime:        time.Since(startTime),
	}
}

// predictAnimatedWebP 预测动态WebP
// 策略：AVIF（与动态GIF一致）
func (wp *WebPPredictor) predictAnimatedWebP(features *FileFeatures, startTime time.Time) *Prediction {
	wp.logger.Debug("动态WebP：使用AVIF",
		zap.String("file", features.FilePath),
		zap.Int("frames", features.FrameCount))

	params := &ConversionParams{
		TargetFormat: "avif",
		Lossless:     false,
		CRF:          35,
		Speed:        6,
		Threads:      8,
	}

	expectedSaving := 0.50 // 保守预测50%

	return &Prediction{
		Params:                params,
		Confidence:            0.85,
		Method:                "rule_based",
		RuleName:              "WEBP_ANIMATED_AVIF",
		ExpectedSaving:        expectedSaving,
		ExpectedSizeBytes:     int64(float64(features.FileSize) * (1 - expectedSaving)),
		ShouldExplore:         false,
		ExplorationCandidates: nil,
		PredictionTime:        time.Since(startTime),
	}
}

// GetConfidenceThreshold WebP预测器的置信度阈值
func (wp *WebPPredictor) GetConfidenceThreshold() float64 {
	return 0.80
}
