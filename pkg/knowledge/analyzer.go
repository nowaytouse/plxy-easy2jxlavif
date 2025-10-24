package knowledge

import (
	"fmt"
	"math"

	"go.uber.org/zap"
)

// Analyzer 预测准确性分析器
type Analyzer struct {
	db     *Database
	logger *zap.Logger
}

// NewAnalyzer 创建分析器
func NewAnalyzer(db *Database, logger *zap.Logger) *Analyzer {
	return &Analyzer{
		db:     db,
		logger: logger,
	}
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	PredictorName  string
	PredictionRule string
	OriginalFormat string

	// 样本统计
	TotalSamples      int
	SuccessfulSamples int
	SuccessRate       float64

	// 预测准确性
	AvgPredictionError    float64 // 平均预测误差（百分比）
	MedianPredictionError float64
	MaxPredictionError    float64
	MinPredictionError    float64

	// 空间节省
	AvgPredictedSaving float64
	AvgActualSaving    float64
	SavingDifference   float64 // predicted - actual

	// 质量
	PerfectQualityCount int // 100%完美
	PerfectQualityRate  float64
	GoodQualityCount    int // PSNR > 40 或 SSIM > 0.95
	GoodQualityRate     float64

	// 性能
	AvgConversionTimeMs int64

	// 建议
	Recommendations []string
}

// AnalyzePredictor 分析特定预测器的准确性
func (a *Analyzer) AnalyzePredictor(predictorName, rule, format string) (*AnalysisResult, error) {
	// 查询所有相关记录
	query := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN validation_passed = 1 THEN 1 ELSE 0 END) as successful,
			AVG(prediction_error_percent) as avg_error,
			MAX(prediction_error_percent) as max_error,
			MIN(prediction_error_percent) as min_error,
			AVG(predicted_saving_percent) as avg_pred_saving,
			AVG(actual_saving_percent) as avg_actual_saving,
			SUM(CASE WHEN pixel_diff_percent = 0 THEN 1 ELSE 0 END) as perfect_count,
			SUM(CASE WHEN psnr_value > 40 OR ssim_value > 0.95 THEN 1 ELSE 0 END) as good_count,
			AVG(actual_conversion_time_ms) as avg_time
		FROM conversion_records
		WHERE predictor_name = ? AND prediction_rule = ? AND original_format = ?
	`

	var total, successful, perfectCount, goodCount int
	var avgError, maxError, minError, avgPredSaving, avgActualSaving float64
	var avgTime int64

	err := a.db.db.QueryRow(query, predictorName, rule, format).Scan(
		&total, &successful,
		&avgError, &maxError, &minError,
		&avgPredSaving, &avgActualSaving,
		&perfectCount, &goodCount,
		&avgTime,
	)

	if err != nil {
		return nil, fmt.Errorf("查询预测器数据失败: %w", err)
	}

	if total == 0 {
		return nil, fmt.Errorf("没有足够的样本数据")
	}

	result := &AnalysisResult{
		PredictorName:       predictorName,
		PredictionRule:      rule,
		OriginalFormat:      format,
		TotalSamples:        total,
		SuccessfulSamples:   successful,
		SuccessRate:         float64(successful) / float64(total),
		AvgPredictionError:  avgError,
		MaxPredictionError:  maxError,
		MinPredictionError:  minError,
		AvgPredictedSaving:  avgPredSaving,
		AvgActualSaving:     avgActualSaving,
		SavingDifference:    avgPredSaving - avgActualSaving,
		PerfectQualityCount: perfectCount,
		PerfectQualityRate:  float64(perfectCount) / float64(total),
		GoodQualityCount:    goodCount,
		GoodQualityRate:     float64(goodCount) / float64(total),
		AvgConversionTimeMs: avgTime,
		Recommendations:     []string{},
	}

	// 计算中位数（需要单独查询）
	medianQuery := `
		WITH numbered AS (
			SELECT prediction_error_percent,
			       ROW_NUMBER() OVER (ORDER BY prediction_error_percent) as row_num,
			       COUNT(*) OVER () as total_count
			FROM conversion_records
			WHERE predictor_name = ? AND prediction_rule = ? AND original_format = ?
		)
		SELECT AVG(prediction_error_percent)
		FROM numbered
		WHERE row_num IN ((total_count + 1) / 2, (total_count + 2) / 2)
	`

	var median float64
	err = a.db.db.QueryRow(medianQuery, predictorName, rule, format).Scan(&median)
	if err == nil {
		result.MedianPredictionError = median
	}

	// 生成建议
	result.Recommendations = a.generateRecommendations(result)

	a.logger.Info("预测器分析完成",
		zap.String("predictor", predictorName),
		zap.String("rule", rule),
		zap.Int("samples", total),
		zap.Float64("avg_error", avgError*100),
		zap.Float64("success_rate", result.SuccessRate*100))

	return result, nil
}

// generateRecommendations 生成优化建议
func (a *Analyzer) generateRecommendations(result *AnalysisResult) []string {
	var recommendations []string

	// 预测准确性建议
	if result.AvgPredictionError > 0.20 {
		recommendations = append(recommendations,
			fmt.Sprintf("⚠️ 预测误差较大(%.1f%%)，建议调整预测参数", result.AvgPredictionError*100))
	} else if result.AvgPredictionError < 0.05 {
		recommendations = append(recommendations,
			fmt.Sprintf("✅ 预测准确性优秀(误差仅%.1f%%)", result.AvgPredictionError*100))
	}

	// 空间节省建议
	if result.SavingDifference > 0.10 {
		recommendations = append(recommendations,
			fmt.Sprintf("📊 预测过于乐观，实际节省比预测少%.1f%%", result.SavingDifference*100))
	} else if result.SavingDifference < -0.10 {
		recommendations = append(recommendations,
			fmt.Sprintf("🎯 预测过于保守，实际节省比预测多%.1f%%", -result.SavingDifference*100))
	}

	// 质量建议
	if result.PerfectQualityRate >= 0.95 {
		recommendations = append(recommendations,
			fmt.Sprintf("🏆 质量完美率%.1f%%，无损转换非常稳定", result.PerfectQualityRate*100))
	} else if result.PerfectQualityRate < 0.80 {
		recommendations = append(recommendations,
			fmt.Sprintf("⚠️ 完美质量率仅%.1f%%，建议检查转换参数", result.PerfectQualityRate*100))
	}

	// 成功率建议
	if result.SuccessRate < 0.90 {
		recommendations = append(recommendations,
			fmt.Sprintf("❌ 成功率偏低(%.1f%%)，需要优化转换流程", result.SuccessRate*100))
	}

	// 性能建议
	if result.AvgConversionTimeMs > 10000 {
		recommendations = append(recommendations,
			fmt.Sprintf("⏱️ 平均转换时间较长(%.1fs)，考虑优化参数", float64(result.AvgConversionTimeMs)/1000))
	}

	return recommendations
}

// CompareFormats 对比不同格式的转换效果
func (a *Analyzer) CompareFormats() (map[string]*AnalysisResult, error) {
	formats := []string{"png", "jpg", "jpeg", "gif", "webp"}
	results := make(map[string]*AnalysisResult)

	for _, format := range formats {
		// 查询该格式的主要预测规则
		query := `
			SELECT predictor_name, prediction_rule, COUNT(*) as count
			FROM conversion_records
			WHERE original_format = ?
			GROUP BY predictor_name, prediction_rule
			ORDER BY count DESC
			LIMIT 1
		`

		var predictorName, rule string
		var count int
		err := a.db.db.QueryRow(query, format).Scan(&predictorName, &rule, &count)
		if err != nil {
			continue // 跳过没有数据的格式
		}

		// 分析该格式
		result, err := a.AnalyzePredictor(predictorName, rule, format)
		if err != nil {
			continue
		}

		results[format] = result
	}

	return results, nil
}

// GetTopAnomalies 获取最严重的异常案例
func (a *Analyzer) GetTopAnomalies(limit int) ([]*AnomalyCase, error) {
	return a.db.DetectAnomalies()
}

// GenerateReport 生成完整的分析报告
func (a *Analyzer) GenerateReport() (string, error) {
	// 获取总体统计
	summary, err := a.db.GetStatsSummary()
	if err != nil {
		return "", err
	}

	// 对比不同格式
	formatResults, err := a.CompareFormats()
	if err != nil {
		return "", err
	}

	// 生成报告
	report := "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	report += "📊 Pixly v3.0 知识库分析报告\n"
	report += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"

	// 总体统计
	report += "🎯 总体统计:\n"
	report += fmt.Sprintf("  总转换次数: %v\n", summary["total_conversions"])
	report += fmt.Sprintf("  平均空间节省: %.1f%%\n", summary["avg_saving_percent"])
	report += fmt.Sprintf("  质量通过率: %.1f%%\n", summary["quality_pass_rate"])
	report += fmt.Sprintf("  平均预测误差: %.1f%%\n\n", summary["avg_prediction_error"])

	// 各格式详情
	report += "📈 各格式转换效果:\n\n"
	for format, result := range formatResults {
		report += fmt.Sprintf("  [%s]\n", format)
		report += fmt.Sprintf("    样本数: %d\n", result.TotalSamples)
		report += fmt.Sprintf("    预测误差: %.1f%% (中位数: %.1f%%)\n",
			result.AvgPredictionError*100, result.MedianPredictionError*100)
		report += fmt.Sprintf("    空间节省: %.1f%% (预测: %.1f%%)\n",
			result.AvgActualSaving*100, result.AvgPredictedSaving*100)
		report += fmt.Sprintf("    完美质量率: %.1f%%\n", result.PerfectQualityRate*100)

		if len(result.Recommendations) > 0 {
			report += "    建议:\n"
			for _, rec := range result.Recommendations {
				report += fmt.Sprintf("      %s\n", rec)
			}
		}
		report += "\n"
	}

	report += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"

	return report, nil
}

// OptimizePrediction 根据历史数据优化预测
func (a *Analyzer) OptimizePrediction(format string) (map[string]interface{}, error) {
	// 分析历史数据，返回优化建议
	query := `
		SELECT 
			AVG(actual_saving_percent) as optimal_saving,
			AVG(predicted_effort) as optimal_effort,
			AVG(predicted_distance) as optimal_distance,
			AVG(predicted_crf) as optimal_crf
		FROM conversion_records
		WHERE original_format = ? 
		  AND validation_passed = 1
		  AND actual_saving_percent > 0
	`

	var optimalSaving, optimalEffort, optimalDistance, optimalCRF float64
	err := a.db.db.QueryRow(query, format).Scan(
		&optimalSaving, &optimalEffort, &optimalDistance, &optimalCRF,
	)

	if err != nil {
		return nil, fmt.Errorf("查询优化参数失败: %w", err)
	}

	optimization := map[string]interface{}{
		"format":           format,
		"optimal_saving":   optimalSaving,
		"optimal_effort":   int(math.Round(optimalEffort)),
		"optimal_distance": optimalDistance,
		"optimal_crf":      int(math.Round(optimalCRF)),
	}

	return optimization, nil
}
