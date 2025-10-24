package predictor

import (
	"fmt"

	"pixly/pkg/knowledge"

	"go.uber.org/zap"
)

// PredictorV31 v3.1增强预测器
// 增加自定义格式支持和知识库微调
type PredictorV31 struct {
	*Predictor // 继承v3.0预测器

	tuner           *knowledge.PredictionTuner
	customPredictor *CustomPredictor
	enableTuning    bool
}

// NewPredictorV31 创建v3.1增强预测器
func NewPredictorV31(
	logger *zap.Logger,
	ffprobePath string,
	knowledgeDB *knowledge.Database,
) *PredictorV31 {
	// 创建基础v3.0预测器
	basePredictor := NewPredictor(logger, ffprobePath)

	// 创建微调器
	var tuner *knowledge.PredictionTuner
	var customPred *CustomPredictor
	enableTuning := false

	if knowledgeDB != nil {
		tuner = knowledge.NewPredictionTuner(knowledgeDB, logger)
		customPred = NewCustomPredictor(logger, tuner)
		enableTuning = true

		logger.Info("v3.1增强预测器初始化成功（知识库微调已启用）")
	} else {
		logger.Warn("知识库未启用，v3.1功能受限（仅使用v3.0黄金规则）")
	}

	return &PredictorV31{
		Predictor:       basePredictor,
		tuner:           tuner,
		customPredictor: customPred,
		enableTuning:    enableTuning,
	}
}

// PredictWithCustomTarget 自定义目标格式预测
func (pv31 *PredictorV31) PredictWithCustomTarget(
	filePath string,
	customReq *CustomFormatRequest,
) (*Prediction, error) {
	if !pv31.enableTuning {
		return nil, fmt.Errorf("知识库未启用，无法使用自定义格式预测")
	}

	// 提取特征
	features, err := pv31.Predictor.featureExtractor.ExtractFeatures(filePath)
	if err != nil {
		return nil, fmt.Errorf("特征提取失败: %w", err)
	}

	// 使用自定义预测器
	prediction := pv31.customPredictor.PredictCustomFormat(features, customReq)

	pv31.logger.Info("自定义格式预测完成",
		zap.String("file", filePath),
		zap.String("target", customReq.TargetFormat),
		zap.Float64("confidence", prediction.Confidence),
		zap.Bool("should_explore", prediction.ShouldExplore))

	return prediction, nil
}

// PredictOptimalParamsWithTuning 预测最优参数（带微调）
// v3.1增强版本：如果知识库有数据，使用微调参数提高准确性
func (pv31 *PredictorV31) PredictOptimalParamsWithTuning(filePath string) (*Prediction, error) {
	// 先使用v3.0黄金规则预测
	prediction, err := pv31.Predictor.PredictOptimalParams(filePath)
	if err != nil {
		return nil, err
	}

	// 如果启用微调，尝试优化预测
	if pv31.enableTuning {
		// 提取特征
		features, err := pv31.Predictor.featureExtractor.ExtractFeatures(filePath)
		if err == nil {
			// 尝试获取微调参数
			tunedParams, err := pv31.tuner.GetTunedParams(
				features.Format,
				prediction.Params.TargetFormat,
				"default",
			)

			if err == nil && tunedParams != nil && tunedParams.SampleCount >= 10 {
				// 有足够的历史数据，微调预测
				pv31.logger.Debug("使用微调参数优化预测",
					zap.String("format", features.Format),
					zap.Int("samples", tunedParams.SampleCount),
					zap.Float64("tuned_saving", tunedParams.OptimalSaving),
					zap.Float64("original_saving", prediction.ExpectedSaving))

				// 更新预期节省（使用历史实际数据）
				prediction.ExpectedSaving = tunedParams.OptimalSaving
				prediction.ExpectedSizeBytes = int64(float64(features.FileSize) * (1 - tunedParams.OptimalSaving))

				// 如果微调的effort/CRF更优，也更新
				if tunedParams.OptimalEffort > 0 {
					prediction.Params.Effort = tunedParams.OptimalEffort
				}
				if tunedParams.OptimalCRF > 0 {
					prediction.Params.CRF = tunedParams.OptimalCRF
				}

				// 更新置信度和方法
				prediction.Confidence = tunedParams.Confidence
				prediction.Method = "rule_based_tuned"
				prediction.RuleName = prediction.RuleName + "_TUNED"
			}
		}
	}

	return prediction, nil
}

// SuggestBestFormat 建议最佳目标格式
func (pv31 *PredictorV31) SuggestBestFormat(filePath string) (*FormatSuggestion, error) {
	if !pv31.enableTuning {
		return nil, fmt.Errorf("知识库未启用")
	}

	// 提取特征
	features, err := pv31.Predictor.featureExtractor.ExtractFeatures(filePath)
	if err != nil {
		return nil, err
	}

	// 基于知识库建议最佳格式
	return pv31.customPredictor.SuggestBestTargetFormat(features.Format)
}

// GetTuningStats 获取微调统计
func (pv31 *PredictorV31) GetTuningStats() map[string]interface{} {
	if !pv31.enableTuning {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	stats := pv31.tuner.GetCacheStats()
	stats["enabled"] = true

	return stats
}

// ClearTuningCache 清除微调缓存
func (pv31 *PredictorV31) ClearTuningCache() {
	if pv31.enableTuning {
		pv31.tuner.ClearCache()
	}
}
