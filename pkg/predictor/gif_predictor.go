package predictor

import (
	"time"

	"go.uber.org/zap"
)

// GIFPredictor GIF专用预测器
// 核心策略：检测动静图，应用黄金规则
type GIFPredictor struct {
	logger *zap.Logger
}

// NewGIFPredictor 创建GIF预测器
func NewGIFPredictor(logger *zap.Logger) *GIFPredictor {
	return &GIFPredictor{
		logger: logger,
	}
}

// Predict 预测GIF的最优转换参数
// GIF黄金规则：
//
//	静态GIF → JXL distance=0（无损）
//	动态GIF → AVIF（动画格式）
func (gp *GIFPredictor) Predict(features *FileFeatures) *Prediction {
	startTime := time.Now()

	gp.logger.Debug("GIF预测",
		zap.String("file", features.FilePath),
		zap.Bool("is_animated", features.IsAnimated),
		zap.Int("frame_count", features.FrameCount))

	// 动静图检测（关键）
	if features.IsAnimated && features.FrameCount > 1 {
		// 动态GIF → AVIF
		return gp.predictAnimatedGIF(features, startTime)
	} else {
		// 静态GIF → JXL
		return gp.predictStaticGIF(features, startTime)
	}
}

// predictStaticGIF 预测静态GIF
// 策略：JXL distance=0（无损，与PNG策略一致）
func (gp *GIFPredictor) predictStaticGIF(features *FileFeatures, startTime time.Time) *Prediction {
	gp.logger.Debug("静态GIF：使用JXL无损",
		zap.String("file", features.FilePath))

	params := &ConversionParams{
		TargetFormat: "jxl",
		Lossless:     true,
		Distance:     0, // 无损
		Effort:       7,
		Threads:      8,
	}

	// GIF→JXL无损通常节省50-90%
	// GIF色彩限制(256色)，JXL压缩效率高
	expectedSaving := 0.60 // 保守预测60%

	return &Prediction{
		Params:                params,
		Confidence:            0.90, // 90%置信度
		Method:                "rule_based",
		RuleName:              "GIF_STATIC_JXL_LOSSLESS",
		ExpectedSaving:        expectedSaving,
		ExpectedSizeBytes:     int64(float64(features.FileSize) * (1 - expectedSaving)),
		ShouldExplore:         false,
		ExplorationCandidates: nil,
		PredictionTime:        time.Since(startTime),
	}
}

// predictAnimatedGIF 预测动态GIF
// 策略：AVIF（动画专用格式）
func (gp *GIFPredictor) predictAnimatedGIF(features *FileFeatures, startTime time.Time) *Prediction {
	gp.logger.Debug("动态GIF：使用AVIF",
		zap.String("file", features.FilePath),
		zap.Int("frames", features.FrameCount))

	// GIF动画通常质量较低，可以使用适中的CRF
	crf := 35 // 默认CRF

	// 根据帧数调整（帧数多→文件大→可以更激进）
	if features.FrameCount > 50 {
		crf = 38
	}

	params := &ConversionParams{
		TargetFormat: "avif",
		Lossless:     false,
		CRF:          crf,
		Speed:        6,
		Threads:      8,
	}

	// GIF→AVIF通常节省70-95%
	// GIF压缩效率低，AVIF可大幅压缩
	expectedSaving := 0.75 // 保守预测75%

	return &Prediction{
		Params:                params,
		Confidence:            0.90, // 90%置信度
		Method:                "rule_based",
		RuleName:              "GIF_ANIMATED_AVIF",
		ExpectedSaving:        expectedSaving,
		ExpectedSizeBytes:     int64(float64(features.FileSize) * (1 - expectedSaving)),
		ShouldExplore:         false,
		ExplorationCandidates: nil,
		PredictionTime:        time.Since(startTime),
	}
}

// GetConfidenceThreshold GIF预测器的置信度阈值
func (gp *GIFPredictor) GetConfidenceThreshold() float64 {
	return 0.80
}
