package predictor

import (
	"fmt"
	"time"

	"pixly/pkg/knowledge"

	"go.uber.org/zap"
)

// CustomFormatRequest 自定义格式请求
type CustomFormatRequest struct {
	SourceFormat string // "png", "jpg", "gif" etc.
	TargetFormat string // "jxl", "avif", "webp" etc.
	QualityGoal  string // "lossless", "high", "balanced", "small"
}

// CustomPredictor 自定义格式预测器
// 支持用户指定目标格式并基于知识库预测最优参数
type CustomPredictor struct {
	logger *zap.Logger
	tuner  *knowledge.PredictionTuner
}

// NewCustomPredictor 创建自定义格式预测器
func NewCustomPredictor(logger *zap.Logger, tuner *knowledge.PredictionTuner) *CustomPredictor {
	return &CustomPredictor{
		logger: logger,
		tuner:  tuner,
	}
}

// PredictCustomFormat 预测自定义格式转换
func (cp *CustomPredictor) PredictCustomFormat(
	features *FileFeatures,
	req *CustomFormatRequest,
) *Prediction {
	startTime := time.Now()

	cp.logger.Info("自定义格式预测",
		zap.String("source", req.SourceFormat),
		zap.String("target", req.TargetFormat),
		zap.String("quality", req.QualityGoal))

	// 步骤1: 尝试从知识库获取微调参数
	tunedParams, err := cp.tuner.GetTunedParams(
		req.SourceFormat,
		req.TargetFormat,
		req.QualityGoal,
	)

	if err == nil && tunedParams != nil && tunedParams.SampleCount >= 10 {
		// 有足够的历史数据，使用微调参数
		return cp.buildPredictionFromTuned(features, tunedParams, startTime)
	}

	// 步骤2: 知识库数据不足，使用保守默认值 + 触发探索
	cp.logger.Warn("知识库数据不足，使用保守策略",
		zap.String("combination", fmt.Sprintf("%s→%s", req.SourceFormat, req.TargetFormat)),
		zap.Error(err))

	return cp.buildConservativePrediction(features, req, startTime)
}

// buildPredictionFromTuned 基于微调参数构建预测
func (cp *CustomPredictor) buildPredictionFromTuned(
	features *FileFeatures,
	tuned *knowledge.TunedParams,
	startTime time.Time,
) *Prediction {
	params := &ConversionParams{
		TargetFormat: tuned.TargetFormat,
		Threads:      8,
	}

	// 根据目标格式设置参数
	switch tuned.TargetFormat {
	case "jxl":
		params.Distance = 0
		params.Effort = tuned.OptimalEffort
		if tuned.OptimalEffort == 0 {
			params.Effort = 7 // 默认
		}
		params.Lossless = true

	case "avif":
		params.CRF = tuned.OptimalCRF
		if tuned.OptimalCRF == 0 {
			params.CRF = 30 // 默认高质量
		}
		params.Speed = tuned.OptimalSpeed
		if tuned.OptimalSpeed == 0 {
			params.Speed = 6
		}

	case "webp":
		// WebP参数（可扩展）
		params.Quality = 90 // 默认高质量
	}

	return &Prediction{
		Params:                params,
		Confidence:            tuned.Confidence,
		Method:                "knowledge_tuned",
		RuleName:              fmt.Sprintf("CUSTOM_%s_TO_%s_TUNED", tuned.SourceFormat, tuned.TargetFormat),
		ExpectedSaving:        tuned.OptimalSaving,
		ExpectedSizeBytes:     int64(float64(features.FileSize) * (1 - tuned.OptimalSaving)),
		ShouldExplore:         false, // 已有充足数据，无需探索
		ExplorationCandidates: nil,
		PredictionTime:        time.Since(startTime),
	}
}

// buildConservativePrediction 构建保守预测（数据不足时）
func (cp *CustomPredictor) buildConservativePrediction(
	features *FileFeatures,
	req *CustomFormatRequest,
	startTime time.Time,
) *Prediction {
	params := &ConversionParams{
		TargetFormat: req.TargetFormat,
		Threads:      8,
	}

	// 保守的默认参数
	var expectedSaving float64
	var shouldExplore bool
	var candidates []*ConversionParams

	switch req.TargetFormat {
	case "jxl":
		params.Distance = 0
		params.Effort = 7
		params.Lossless = true
		expectedSaving = 0.50 // 保守预测50%
		shouldExplore = true

		// 生成探索候选
		candidates = []*ConversionParams{
			&ConversionParams{TargetFormat: "jxl", Distance: 0, Effort: 5, Lossless: true, Threads: 8},
			&ConversionParams{TargetFormat: "jxl", Distance: 0, Effort: 7, Lossless: true, Threads: 8},
			&ConversionParams{TargetFormat: "jxl", Distance: 0, Effort: 9, Lossless: true, Threads: 8},
		}

	case "avif":
		params.CRF = 30
		params.Speed = 6
		expectedSaving = 0.40 // 保守预测40%
		shouldExplore = true

		// 生成探索候选
		candidates = []*ConversionParams{
			&ConversionParams{TargetFormat: "avif", CRF: 25, Speed: 6, Threads: 8},
			&ConversionParams{TargetFormat: "avif", CRF: 30, Speed: 6, Threads: 8},
			&ConversionParams{TargetFormat: "avif", CRF: 35, Speed: 6, Threads: 8},
		}

	case "webp":
		params.Quality = 90
		expectedSaving = 0.30 // 保守预测30%
		shouldExplore = true

		candidates = []*ConversionParams{
			&ConversionParams{TargetFormat: "webp", Quality: 85, Threads: 8},
			&ConversionParams{TargetFormat: "webp", Quality: 90, Threads: 8},
			&ConversionParams{TargetFormat: "webp", Quality: 95, Threads: 8},
		}

	default:
		// 未知格式，使用极保守策略
		expectedSaving = 0.20
		shouldExplore = true
	}

	pred := &Prediction{
		Params:            params,
		Confidence:        0.50, // 低置信度（数据不足）
		Method:            "conservative_default",
		RuleName:          fmt.Sprintf("CUSTOM_%s_TO_%s_DEFAULT", req.SourceFormat, req.TargetFormat),
		ExpectedSaving:    expectedSaving,
		ExpectedSizeBytes: int64(float64(features.FileSize) * (1 - expectedSaving)),
		ShouldExplore:     shouldExplore,
		PredictionTime:    time.Since(startTime),
	}

	// 如果有候选，添加到ExplorationCandidates
	if len(candidates) > 0 {
		pred.ExplorationCandidates = candidates
	}

	return pred
}

// SuggestBestTargetFormat 建议最佳目标格式
// 基于知识库数据推荐空间节省最大的格式
func (cp *CustomPredictor) SuggestBestTargetFormat(
	sourceFormat string,
) (*FormatSuggestion, error) {
	// 获取所有格式组合
	combinations, err := cp.tuner.GetFormatCombinations()
	if err != nil {
		return nil, err
	}

	// 筛选该源格式的所有目标格式
	var bestCombo *knowledge.FormatCombination
	bestSaving := 0.0

	for i := range combinations {
		combo := &combinations[i]
		if combo.SourceFormat == sourceFormat {
			if combo.AvgSaving > bestSaving && combo.SuccessRate > 0.80 {
				bestSaving = combo.AvgSaving
				bestCombo = combo
			}
		}
	}

	if bestCombo == nil {
		return nil, fmt.Errorf("没有足够的历史数据")
	}

	return &FormatSuggestion{
		RecommendedFormat: bestCombo.TargetFormat,
		ExpectedSaving:    bestCombo.AvgSaving,
		SuccessRate:       bestCombo.SuccessRate,
		SampleCount:       bestCombo.SampleCount,
		Reason:            fmt.Sprintf("基于%d个样本，平均节省%.1f%%", bestCombo.SampleCount, bestCombo.AvgSaving*100),
	}, nil
}

// FormatSuggestion 格式建议
type FormatSuggestion struct {
	RecommendedFormat string
	ExpectedSaving    float64
	SuccessRate       float64
	SampleCount       int
	Reason            string
}
