package predictor

import (
	"fmt"

	"go.uber.org/zap"
)

// Predictor 主预测器
// 协调特征提取和参数预测
type Predictor struct {
	logger           *zap.Logger
	featureExtractor *FeatureExtractor
	pngPredictor     *PNGPredictor
	jpegPredictor    *JPEGPredictor  // Week 3-4
	gifPredictor     *GIFPredictor   // Week 5-6
	webpPredictor    *WebPPredictor  // Week 5-6
	videoPredictor   *VideoPredictor // Week 5-6
}

// NewPredictor 创建主预测器
func NewPredictor(logger *zap.Logger, ffprobePath string) *Predictor {
	return &Predictor{
		logger:           logger,
		featureExtractor: NewFeatureExtractor(logger, ffprobePath),
		pngPredictor:     NewPNGPredictor(logger),
		jpegPredictor:    NewJPEGPredictor(logger),
		gifPredictor:     NewGIFPredictor(logger),   // Week 5-6新增
		webpPredictor:    NewWebPPredictor(logger),  // Week 5-6新增
		videoPredictor:   NewVideoPredictor(logger), // Week 5-6新增
	}
}

// PredictOptimalParams 预测最优转换参数
// 这是主入口函数
func (p *Predictor) PredictOptimalParams(filePath string) (*Prediction, error) {
	// 步骤1: 提取特征
	features, err := p.featureExtractor.ExtractFeatures(filePath)
	if err != nil {
		return nil, fmt.Errorf("特征提取失败: %w", err)
	}

	// 步骤2: 根据格式选择预测器
	prediction := p.selectAndPredict(features)

	// 步骤3: 日志记录
	p.logger.Info("预测完成",
		zap.String("file", filePath),
		zap.String("format", features.Format),
		zap.String("target", prediction.Params.TargetFormat),
		zap.Float64("confidence", prediction.Confidence),
		zap.String("method", prediction.Method),
		zap.Float64("expected_saving", prediction.ExpectedSaving*100),
		zap.Bool("should_explore", prediction.ShouldExplore))

	return prediction, nil
}

// selectAndPredict 选择合适的预测器并执行预测
func (p *Predictor) selectAndPredict(features *FileFeatures) *Prediction {
	switch features.Format {
	case "png":
		return p.pngPredictor.Predict(features)

	case "jpg", "jpeg":
		return p.jpegPredictor.Predict(features)

	case "gif":
		// Week 5-6新增：GIF预测器
		return p.gifPredictor.Predict(features)

	case "webp":
		// Week 5-6新增：WebP预测器
		return p.webpPredictor.Predict(features)

	case "mp4", "avi", "mkv", "mov", "wmv", "flv", "m4v":
		// Week 5-6新增：视频预测器
		return p.videoPredictor.Predict(features)

	default:
		// 对于未支持的格式，返回保守的默认预测
		return p.getDefaultPrediction(features)
	}
}

// getDefaultPrediction 获取默认预测（fallback）
// 用于MVP阶段未实现的格式
func (p *Predictor) getDefaultPrediction(features *FileFeatures) *Prediction {
	p.logger.Warn("使用默认预测（格式未完全支持）",
		zap.String("format", features.Format))

	// 保守的默认策略：标记为需要探索
	params := &ConversionParams{
		TargetFormat: "jxl", // 默认JXL
		Lossless:     true,
		Distance:     0,
		Effort:       7,
		Threads:      8,
	}

	return &Prediction{
		Params:            params,
		Confidence:        0.50, // 低置信度
		Method:            "default_fallback",
		RuleName:          "UNSUPPORTED_FORMAT_DEFAULT",
		ExpectedSaving:    0.20, // 保守估计20%
		ExpectedSizeBytes: int64(float64(features.FileSize) * 0.8),
		ShouldExplore:     true, // 标记为需要探索
	}
}

// GetFeatures 获取文件特征（辅助方法）
// 用于调试和测试
func (p *Predictor) GetFeatures(filePath string) (*FileFeatures, error) {
	return p.featureExtractor.ExtractFeatures(filePath)
}
