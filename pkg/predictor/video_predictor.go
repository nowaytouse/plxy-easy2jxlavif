package predictor

import (
	"time"

	"go.uber.org/zap"
)

// VideoPredictor 视频专用预测器
// 核心策略：极简——永远是MOV重封装
type VideoPredictor struct {
	logger *zap.Logger
}

// NewVideoPredictor 创建视频预测器
func NewVideoPredictor(logger *zap.Logger) *VideoPredictor {
	return &VideoPredictor{
		logger: logger,
	}
}

// Predict 预测视频的最优转换参数
// 视频黄金规则：
//   所有视频 → MOV重封装（不重新编码）
func (vp *VideoPredictor) Predict(features *FileFeatures) *Prediction {
	startTime := time.Now()
	
	vp.logger.Debug("视频预测",
		zap.String("file", features.FilePath),
		zap.String("format", features.Format))
	
	// 视频黄金规则：永远是MOV重封装
	// 不重新编码，仅改变容器格式
	params := &ConversionParams{
		TargetFormat: "mov",
		Repackage:    true,  // 仅重封装
		CopyCodec:    true,  // 复制编码流
		Threads:      8,
	}
	
	// 视频重封装节省空间有限（0-5%）
	// 主要是容器格式优化
	expectedSaving := 0.02 // 保守预测2%
	
	return &Prediction{
		Params:              params,
		Confidence:          0.95, // 95%置信度（重封装非常稳定）
		Method:              "rule_based",
		RuleName:            "VIDEO_MOV_REPACKAGE",
		ExpectedSaving:      expectedSaving,
		ExpectedSizeBytes:   int64(float64(features.FileSize) * (1 - expectedSaving)),
		ShouldExplore:       false,
		ExplorationCandidates: nil,
		PredictionTime:      time.Since(startTime),
	}
}

// GetConfidenceThreshold 视频预测器的置信度阈值
func (vp *VideoPredictor) GetConfidenceThreshold() float64 {
	return 0.80
}

