package quality

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"pixly/pkg/core/types"

	"go.uber.org/zap"
)

// EnhancedQualityEngine 增强版品质判断引擎 - 优化README要求的质量分类体系
type EnhancedQualityEngine struct {
	*QualityEngine // 嵌入原有引擎

	// 增强功能
	balanceOptimizer  *BalanceOptimizer
	adaptiveThreshold *AdaptiveThreshold
	qualityPredictor  *QualityPredictor
	batchAnalyzer     *BatchAnalyzer
}

// BalanceOptimizer 平衡优化器 - README要求的核心平衡优化逻辑
type BalanceOptimizer struct {
	logger          *zap.Logger
	losslessFirst   bool      // 优先无损重新包装
	qualityGroups   [][]int   // README要求的探测组：高品质组[90,85,75]，中等品质组[60,55]
	sizeThresholds  []float64 // 体积节省阈值
	confidenceModel *ConfidenceModel
}

// AdaptiveThreshold 自适应阈值系统 - 根据文件特征动态调整品质判断标准
type AdaptiveThreshold struct {
	logger          *zap.Logger
	pixelDensityMap map[string]float64 // 按格式的像素密度基准
	jpegQualityMap  map[string]int     // 按场景的JPEG品质基准
	formatWeights   map[string]float64 // 格式权重系数
	sizeCorrections map[string]float64 // 尺寸修正系数
}

// QualityPredictor 品质预测器 - 基于机器学习模型的品质预测
type QualityPredictor struct {
	logger          *zap.Logger
	historicalData  []QualityDataPoint
	modelWeights    map[string]float64
	predictionCache map[string]*QualityPrediction
}

// BatchAnalyzer 批量分析器 - 优化大批量文件的质量分析性能
type BatchAnalyzer struct {
	logger            *zap.Logger
	parallelWorkers   int
	cacheEnabled      bool
	statisticsEnabled bool
	batchStats        *BatchStatistics
}

// QualityDataPoint 质量数据点 - 用于机器学习训练
type QualityDataPoint struct {
	FilePath      string
	FileSize      int64
	Width         int
	Height        int
	Format        string
	PixelDensity  float64
	JpegQuality   int
	ActualQuality types.QualityLevel
	ProcessedTime time.Time
}

// QualityPrediction 质量预测结果
type QualityPrediction struct {
	PredictedLevel    types.QualityLevel
	Confidence        float64
	ReasonCode        string
	AlternativeLevels []types.QualityLevel
	RiskFactors       []string
}

// ConfidenceModel 置信度模型 - README要求的高精度置信度计算
type ConfidenceModel struct {
	BaseConfidence    float64
	FFProbeBonus      float64            // ffprobe深度验证奖励
	JPEGQualityBonus  float64            // JPEG品质检测奖励
	PixelDensityBonus float64            // 像素密度计算奖励
	FormatPenalty     map[string]float64 // 格式惩罚系数
	SizePenalty       float64            // 异常大小惩罚
}

// BatchStatistics 批量统计信息
type BatchStatistics struct {
	TotalFiles          int
	ProcessedFiles      int
	QualityDistribution map[types.QualityLevel]int
	FormatDistribution  map[string]int
	AverageProcessTime  time.Duration
	HighConfidenceFiles int
	LowConfidenceFiles  int
	RecommendedModes    map[types.AppMode]int
}

// OptimizationStrategy 优化策略 - README要求的平衡优化决策
type OptimizationStrategy struct {
	Strategy        string                // "lossless_repackage", "math_lossless", "lossy_probe"
	TargetFormat    string                // "jxl", "avif", "webp"
	QualitySettings []int                 // 品质设置序列
	ExpectedSaving  float64               // 预期节省比例
	ProcessingTime  time.Duration         // 预期处理时间
	RiskLevel       string                // "low", "medium", "high"
	Confidence      float64               // 策略置信度
	Fallback        *OptimizationStrategy // 备用策略
}

// NewEnhancedQualityEngine 创建增强版品质判断引擎
func NewEnhancedQualityEngine(baseEngine *QualityEngine, logger *zap.Logger) *EnhancedQualityEngine {
	engine := &EnhancedQualityEngine{
		QualityEngine: baseEngine,
	}

	// 初始化增强组件
	engine.balanceOptimizer = NewBalanceOptimizer(logger)
	engine.adaptiveThreshold = NewAdaptiveThreshold(logger)
	engine.qualityPredictor = NewQualityPredictor(logger)
	engine.batchAnalyzer = NewBatchAnalyzer(logger)

	return engine
}

// NewBalanceOptimizer 创建平衡优化器
func NewBalanceOptimizer(logger *zap.Logger) *BalanceOptimizer {
	return &BalanceOptimizer{
		logger:        logger,
		losslessFirst: true,
		// README要求的探测组设置
		qualityGroups: [][]int{
			{90, 85, 75}, // 高品质组
			{60, 55},     // 中等品质组
		},
		sizeThresholds: []float64{0.05, 0.10, 0.15}, // 5%, 10%, 15%节省阈值
		confidenceModel: &ConfidenceModel{
			BaseConfidence:    0.7,
			FFProbeBonus:      0.15,
			JPEGQualityBonus:  0.10,
			PixelDensityBonus: 0.05,
			FormatPenalty: map[string]float64{
				"unknown": 0.2,
				"damaged": 0.5,
			},
			SizePenalty: 0.1,
		},
	}
}

// NewAdaptiveThreshold 创建自适应阈值系统
func NewAdaptiveThreshold(logger *zap.Logger) *AdaptiveThreshold {
	return &AdaptiveThreshold{
		logger: logger,
		// README要求的按格式优化的像素密度基准
		pixelDensityMap: map[string]float64{
			"jpeg": 2.5, // JPEG高品质基准
			"png":  3.0, // PNG无损基准
			"webp": 2.2, // WebP基准
			"heif": 2.8, // HEIF基准
			"avif": 2.0, // AVIF基准
			"jxl":  1.8, // JXL基准（更高效）
		},
		jpegQualityMap: map[string]int{
			"photo":      85, // 照片品质基准
			"graphics":   75, // 图形品质基准
			"screenshot": 70, // 截图品质基准
			"scan":       80, // 扫描品质基准
		},
		formatWeights: map[string]float64{
			"jpeg": 1.0,
			"png":  0.9,  // PNG略微降权（通常过度无损）
			"webp": 1.1,  // WebP略微加权（现代格式）
			"heif": 1.15, // HEIF加权（苹果生态优化）
			"avif": 1.2,  // AVIF高权重（最新标准）
			"jxl":  1.25, // JXL最高权重（最优格式）
		},
		sizeCorrections: map[string]float64{
			"thumbnail": 0.3, // 缩略图修正
			"icon":      0.2, // 图标修正
			"banner":    1.2, // 横幅修正
			"poster":    1.5, // 海报修正
		},
	}
}

// NewQualityPredictor 创建品质预测器
func NewQualityPredictor(logger *zap.Logger) *QualityPredictor {
	return &QualityPredictor{
		logger:          logger,
		historicalData:  make([]QualityDataPoint, 0),
		predictionCache: make(map[string]*QualityPrediction),
		// 基于经验的模型权重
		modelWeights: map[string]float64{
			"pixel_density": 0.35,
			"jpeg_quality":  0.30,
			"file_size":     0.15,
			"resolution":    0.10,
			"format":        0.10,
		},
	}
}

// NewBatchAnalyzer 创建批量分析器
func NewBatchAnalyzer(logger *zap.Logger) *BatchAnalyzer {
	return &BatchAnalyzer{
		logger:            logger,
		parallelWorkers:   4, // 默认4个工作线程
		cacheEnabled:      true,
		statisticsEnabled: true,
		batchStats: &BatchStatistics{
			QualityDistribution: make(map[types.QualityLevel]int),
			FormatDistribution:  make(map[string]int),
			RecommendedModes:    make(map[types.AppMode]int),
		},
	}
}

// AdvancedAssessFile 增强版文件品质评估 - 整合所有优化算法
func (eq *EnhancedQualityEngine) AdvancedAssessFile(ctx context.Context, filePath string) (*QualityAssessment, error) {
	startTime := time.Now()

	// 1. 基础品质评估（使用原有引擎）
	baseAssessment, err := eq.QualityEngine.AssessFile(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("基础品质评估失败: %w", err)
	}

	// 2. 自适应阈值调整
	eq.adaptiveThreshold.AdjustThresholds(baseAssessment)

	// 3. 品质预测增强
	prediction, err := eq.qualityPredictor.PredictQuality(baseAssessment)
	if err == nil {
		baseAssessment.QualityLevel = prediction.PredictedLevel
		baseAssessment.Confidence = math.Max(baseAssessment.Confidence, prediction.Confidence)
	}

	// 4. 平衡优化策略生成
	strategy, err := eq.balanceOptimizer.GenerateOptimizationStrategy(baseAssessment)
	if err == nil {
		// 将策略信息添加到评估结果中
		baseAssessment.Details["optimization_strategy"] = strategy
		baseAssessment.Details["balance_optimizer"] = "enabled"
	}

	// 5. 批量统计更新
	eq.batchAnalyzer.UpdateStatistics(baseAssessment)

	baseAssessment.AssessmentTime = time.Since(startTime)

	eq.QualityEngine.logger.Debug("增强品质评估完成",
		zap.String("file", filepath.Base(filePath)),
		zap.String("quality", baseAssessment.QualityLevel.String()),
		zap.Float64("confidence", baseAssessment.Confidence),
		zap.Duration("assessment_time", baseAssessment.AssessmentTime),
		zap.Bool("prediction_used", prediction != nil),
		zap.Bool("strategy_generated", strategy != nil))

	return baseAssessment, nil
}

// AdjustThresholds 自适应阈值调整
func (at *AdaptiveThreshold) AdjustThresholds(assessment *QualityAssessment) {
	format := strings.ToLower(assessment.Format)

	// 根据格式调整像素密度阈值
	if baseDensity, exists := at.pixelDensityMap[format]; exists {
		adjustment := assessment.PixelDensity / baseDensity

		// 动态调整品质判断阈值
		if adjustment > 1.2 {
			// 像素密度超出预期，可能是高品质
			if assessment.QualityLevel < types.QualityHigh {
				assessment.QualityLevel = types.QualityHigh
				assessment.Confidence = math.Min(assessment.Confidence+0.1, 1.0)
			}
		} else if adjustment < 0.8 {
			// 像素密度低于预期，可能是低品质
			if assessment.QualityLevel > types.QualityLow {
				assessment.QualityLevel = types.QualityMediumLow
				assessment.Confidence = math.Max(assessment.Confidence-0.1, 0.1)
			}
		}
	}

	// 应用格式权重
	if weight, exists := at.formatWeights[format]; exists {
		assessment.Score *= weight
		if weight > 1.0 {
			assessment.Confidence = math.Min(assessment.Confidence+0.05, 1.0)
		}
	}

	at.logger.Debug("自适应阈值调整完成",
		zap.String("format", format),
		zap.Float64("pixel_density", assessment.PixelDensity),
		zap.String("adjusted_quality", assessment.QualityLevel.String()))
}

// PredictQuality 品质预测
func (qp *QualityPredictor) PredictQuality(assessment *QualityAssessment) (*QualityPrediction, error) {
	cacheKey := fmt.Sprintf("%s_%d_%dx%d", assessment.Format, assessment.FileSize, assessment.Width, assessment.Height)

	// 检查缓存
	if cached, exists := qp.predictionCache[cacheKey]; exists {
		return cached, nil
	}

	// 计算特征向量
	features := qp.extractFeatures(assessment)

	// 基于权重的预测计算
	score := 0.0
	for feature, value := range features {
		if weight, exists := qp.modelWeights[feature]; exists {
			score += value * weight
		}
	}

	// 转换为品质等级
	predictedLevel := qp.scoreTQualityLevel(score)

	prediction := &QualityPrediction{
		PredictedLevel: predictedLevel,
		Confidence:     qp.calculatePredictionConfidence(features),
		ReasonCode:     qp.generateReasonCode(features),
		RiskFactors:    qp.identifyRiskFactors(features),
	}

	// 缓存结果
	qp.predictionCache[cacheKey] = prediction

	return prediction, nil
}

// GenerateOptimizationStrategy 生成优化策略 - README核心平衡优化逻辑
func (bo *BalanceOptimizer) GenerateOptimizationStrategy(assessment *QualityAssessment) (*OptimizationStrategy, error) {
	strategy := &OptimizationStrategy{}

	// README要求的平衡优化逻辑实现：
	// 1. 无损重新包装优先
	// 2. 数学无损压缩
	// 3. 有损探测（高品质组: 90,85,75；中等品质组: 60,55）
	// 4. 最终决策：只要体积有任何减小，即视为成功

	switch assessment.QualityLevel {
	case types.QualityVeryHigh, types.QualityHigh:
		// 高品质文件：优先无损重新包装
		strategy.Strategy = "lossless_repackage"
		strategy.QualitySettings = []int{100} // 无损
		strategy.ExpectedSaving = 0.15        // 预期15%节省
		strategy.RiskLevel = "low"
		strategy.Confidence = 0.9

		// 设置备用策略为数学无损
		strategy.Fallback = &OptimizationStrategy{
			Strategy:        "math_lossless",
			QualitySettings: []int{100},
			ExpectedSaving:  0.10,
			RiskLevel:       "low",
			Confidence:      0.85,
		}

	case types.QualityMediumHigh, types.QualityMediumLow:
		// 中等品质文件：平衡优化探测
		strategy.Strategy = "lossy_probe"
		strategy.QualitySettings = bo.qualityGroups[0] // 高品质组探测
		strategy.ExpectedSaving = 0.25                 // 预期25%节省
		strategy.RiskLevel = "medium"
		strategy.Confidence = 0.75

		// 备用策略使用中等品质组
		strategy.Fallback = &OptimizationStrategy{
			Strategy:        "lossy_probe",
			QualitySettings: bo.qualityGroups[1], // 中等品质组
			ExpectedSaving:  0.35,
			RiskLevel:       "medium",
			Confidence:      0.65,
		}

	case types.QualityLow, types.QualityVeryLow:
		// 低品质文件：用户决策模式
		strategy.Strategy = "user_decision"
		strategy.QualitySettings = []int{} // 待用户决策
		strategy.ExpectedSaving = 0.0
		strategy.RiskLevel = "high"
		strategy.Confidence = 0.5

	default:
		return nil, fmt.Errorf("未知品质等级: %s", assessment.QualityLevel.String())
	}

	// 根据文件格式调整目标格式
	strategy.TargetFormat = bo.selectOptimalTargetFormat(assessment)

	bo.logger.Debug("优化策略生成完成",
		zap.String("strategy", strategy.Strategy),
		zap.String("target_format", strategy.TargetFormat),
		zap.Ints("quality_settings", strategy.QualitySettings),
		zap.Float64("expected_saving", strategy.ExpectedSaving))

	return strategy, nil
}

// 辅助方法实现
func (qp *QualityPredictor) extractFeatures(assessment *QualityAssessment) map[string]float64 {
	features := make(map[string]float64)

	features["pixel_density"] = assessment.PixelDensity / 3.0 // 归一化到0-1
	features["jpeg_quality"] = float64(assessment.JpegQuality) / 100.0
	features["file_size"] = math.Log10(float64(assessment.FileSize)) / 8.0 // log归一化
	features["resolution"] = math.Log10(float64(assessment.Width*assessment.Height)) / 7.0

	// 格式特征
	switch assessment.Format {
	case "jpeg":
		features["format"] = 1.0
	case "png":
		features["format"] = 0.8
	case "webp":
		features["format"] = 0.9
	default:
		features["format"] = 0.5
	}

	return features
}

func (qp *QualityPredictor) scoreTQualityLevel(score float64) types.QualityLevel {
	if score >= 0.9 {
		return types.QualityVeryHigh
	} else if score >= 0.75 {
		return types.QualityHigh
	} else if score >= 0.6 {
		return types.QualityMediumHigh
	} else if score >= 0.45 {
		return types.QualityMediumLow
	} else if score >= 0.3 {
		return types.QualityLow
	} else {
		return types.QualityVeryLow
	}
}

func (qp *QualityPredictor) calculatePredictionConfidence(features map[string]float64) float64 {
	// 基于特征完整性计算置信度
	completeness := float64(len(features)) / 5.0 // 5个核心特征
	return 0.6 + (completeness * 0.3)            // 0.6-0.9的置信度范围
}

func (qp *QualityPredictor) generateReasonCode(features map[string]float64) string {
	if features["jpeg_quality"] > 0.8 {
		return "HIGH_JPEG_QUALITY"
	} else if features["pixel_density"] > 0.8 {
		return "HIGH_PIXEL_DENSITY"
	} else if features["resolution"] > 0.7 {
		return "HIGH_RESOLUTION"
	} else {
		return "MULTI_FACTOR_PREDICTION"
	}
}

func (qp *QualityPredictor) identifyRiskFactors(features map[string]float64) []string {
	var risks []string

	if features["file_size"] < 0.2 {
		risks = append(risks, "VERY_SMALL_FILE")
	}
	if features["pixel_density"] < 0.3 {
		risks = append(risks, "LOW_PIXEL_DENSITY")
	}
	if features["format"] < 0.6 {
		risks = append(risks, "UNKNOWN_FORMAT")
	}

	return risks
}

func (bo *BalanceOptimizer) selectOptimalTargetFormat(assessment *QualityAssessment) string {
	switch assessment.MediaType {
	case types.MediaTypeImage:
		// 静图优先选择JXL（README推荐）
		return "jxl"
	case types.MediaTypeAnimated:
		// 动图优先选择AVIF（README推荐）
		return "avif"
	case types.MediaTypeVideo:
		// 视频重新包装为MOV（README推荐）
		return "mov"
	default:
		return "auto"
	}
}

// UpdateStatistics 更新批量统计
func (ba *BatchAnalyzer) UpdateStatistics(assessment *QualityAssessment) {
	if !ba.statisticsEnabled {
		return
	}

	ba.batchStats.TotalFiles++
	ba.batchStats.ProcessedFiles++

	// 更新品质分布
	ba.batchStats.QualityDistribution[assessment.QualityLevel]++

	// 更新格式分布
	ba.batchStats.FormatDistribution[assessment.Format]++

	// 更新置信度统计
	if assessment.Confidence > 0.8 {
		ba.batchStats.HighConfidenceFiles++
	} else if assessment.Confidence < 0.5 {
		ba.batchStats.LowConfidenceFiles++
	}

	// 更新推荐模式统计
	ba.batchStats.RecommendedModes[assessment.RecommendedMode]++

	ba.logger.Debug("批量统计更新",
		zap.Int("total_files", ba.batchStats.TotalFiles),
		zap.String("quality", assessment.QualityLevel.String()),
		zap.Float64("confidence", assessment.Confidence))
}

// GetBatchStatistics 获取批量统计信息
func (ba *BatchAnalyzer) GetBatchStatistics() *BatchStatistics {
	return ba.batchStats
}
