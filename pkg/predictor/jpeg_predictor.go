package predictor

import (
	"time"

	"go.uber.org/zap"
)

// JPEGPredictor JPEG专用预测器
// 核心策略：JPEG永远用JXL lossless_jpeg=1（完美可逆）
// 就像PNG永远用distance=0一样简单且有效！
type JPEGPredictor struct {
	logger *zap.Logger
}

// NewJPEGPredictor 创建JPEG预测器
func NewJPEGPredictor(logger *zap.Logger) *JPEGPredictor {
	return &JPEGPredictor{
		logger: logger,
	}
}

// Predict 预测JPEG的最优转换参数
// JPEG黄金规则：永远使用JXL lossless_jpeg=1
// 原因：完全无损、可逆、格式最优
func (jp *JPEGPredictor) Predict(features *FileFeatures) *Prediction {
	startTime := time.Now()

	jp.logger.Debug("JPEG预测",
		zap.String("file", features.FilePath),
		zap.String("pix_fmt", features.PixFmt),
		zap.Int("estimated_quality", features.EstimatedQuality))

	// JPEG黄金规则：永远是JXL lossless_jpeg=1
	// 就像PNG永远是distance=0一样
	params := &ConversionParams{
		TargetFormat: "jxl",
		Lossless:     true,
		LosslessJPEG: true, // 关键参数：完美保留JPEG数据
		Distance:     0,
		Effort:       jp.calculateOptimalEffort(features),
		Threads:      8,
	}

	// 预测空间节省（保守估计）
	// JPEG→JXL lossless_jpeg=1 通常节省10-30%
	expectedSaving := jp.estimateSaving(features)

	return &Prediction{
		Params:                params,
		Confidence:            0.95, // 95%置信度（lossless_jpeg=1非常稳定）
		Method:                "rule_based",
		RuleName:              "JPEG_ALWAYS_JXL_LOSSLESS",
		ExpectedSaving:        expectedSaving,
		ExpectedSizeBytes:     int64(float64(features.FileSize) * (1 - expectedSaving)),
		ShouldExplore:         false, // JPEG不需要探索，直接lossless_jpeg=1
		ExplorationCandidates: nil,
		PredictionTime:        time.Since(startTime),
	}
}

// calculateOptimalEffort 计算最优effort
// 与PNG策略一致：根据文件大小智能调整
func (jp *JPEGPredictor) calculateOptimalEffort(features *FileFeatures) int {
	fileSizeMB := float64(features.FileSize) / (1024 * 1024)

	if fileSizeMB > 10 {
		return 5 // 大文件快速处理
	} else if fileSizeMB < 0.1 {
		return 9 // 小文件极致压缩
	} else {
		return 7 // 中等文件平衡
	}
}

// estimateSaving 估算空间节省率
// JPEG→JXL lossless_jpeg=1 的保守预测
func (jp *JPEGPredictor) estimateSaving(features *FileFeatures) float64 {
	// 基于实测数据的保守估计
	// JPEG→JXL lossless_jpeg=1 通常节省10-30%

	// 根据pix_fmt调整
	switch features.PixFmt {
	case "yuv444p", "yuvj444p":
		// 4:4:4采样，已接近无损，节省较少
		return 0.15
	case "yuv422p", "yuvj422p":
		// 4:2:2采样，中等节省
		return 0.20
	case "yuv420p", "yuvj420p":
		// 4:2:0采样，标准节省
		return 0.25
	default:
		// 未知格式，保守估计
		return 0.15
	}
}

// GetConfidenceThreshold JPEG预测器的置信度阈值
func (jp *JPEGPredictor) GetConfidenceThreshold() float64 {
	return 0.80 // JPEG的lossless_jpeg=1非常稳定
}

// NeedsExploration 判断是否需要探索
// JPEG不需要探索，直接lossless_jpeg=1即可
func (jp *JPEGPredictor) NeedsExploration(prediction *Prediction) bool {
	return false
}
