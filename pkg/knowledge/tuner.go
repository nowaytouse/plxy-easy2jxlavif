package knowledge

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// PredictionTuner 预测微调器
// 基于历史数据动态调整预测参数
type PredictionTuner struct {
	db       *Database
	logger   *zap.Logger
	cache    map[string]*CachedTuning
	mutex    sync.RWMutex
	cacheTTL time.Duration
}

// TunedParams 微调后的参数
type TunedParams struct {
	SourceFormat string
	TargetFormat string
	QualityGoal  string

	// 优化后的参数
	OptimalSaving float64 // 基于历史的最优节省率
	OptimalEffort int     // 基于文件大小的最优effort
	OptimalCRF    int     // 基于质量的最优CRF
	OptimalSpeed  int     // AVIF编码速度

	// 元数据
	Confidence  float64 // 微调置信度（基于样本数）
	SampleCount int     // 样本数量
	AvgError    float64 // 平均预测误差
	LastUpdated time.Time
}

// CachedTuning 缓存的微调结果
type CachedTuning struct {
	Params   *TunedParams
	CachedAt time.Time
	HitCount int
}

// NewPredictionTuner 创建预测微调器
func NewPredictionTuner(db *Database, logger *zap.Logger) *PredictionTuner {
	return &PredictionTuner{
		db:       db,
		logger:   logger,
		cache:    make(map[string]*CachedTuning),
		cacheTTL: 1 * time.Hour, // 缓存1小时
	}
}

// GetTunedParams 获取微调后的参数
func (pt *PredictionTuner) GetTunedParams(
	sourceFormat, targetFormat, qualityGoal string,
) (*TunedParams, error) {
	key := fmt.Sprintf("%s->%s:%s", sourceFormat, targetFormat, qualityGoal)

	// 检查缓存
	if cached := pt.getCached(key); cached != nil {
		pt.logger.Debug("使用缓存的微调参数",
			zap.String("key", key),
			zap.Int("hit_count", cached.HitCount))
		return cached.Params, nil
	}

	// 查询数据库计算最优参数
	params, err := pt.calculateOptimalParams(sourceFormat, targetFormat, qualityGoal)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	pt.setCached(key, params)

	return params, nil
}

// calculateOptimalParams 计算最优参数
func (pt *PredictionTuner) calculateOptimalParams(
	sourceFormat, targetFormat, qualityGoal string,
) (*TunedParams, error) {
	// 查询历史记录
	query := `
		SELECT 
			COUNT(*) as sample_count,
			AVG(actual_saving_percent) as avg_saving,
			AVG(prediction_error_percent) as avg_error,
			AVG(predicted_effort) as avg_effort,
			AVG(predicted_crf) as avg_crf,
			AVG(predicted_speed) as avg_speed
		FROM conversion_records
		WHERE original_format = ?
		  AND actual_format = ?
		  AND validation_passed = 1
	`

	var sampleCount int
	var avgSaving, avgError, avgEffort, avgCRF, avgSpeed float64

	err := pt.db.db.QueryRow(query, sourceFormat, targetFormat).Scan(
		&sampleCount,
		&avgSaving,
		&avgError,
		&avgEffort,
		&avgCRF,
		&avgSpeed,
	)

	if err != nil {
		return nil, fmt.Errorf("查询历史记录失败: %w", err)
	}

	if sampleCount == 0 {
		return nil, fmt.Errorf("没有足够的历史数据")
	}

	// 计算置信度（基于样本数）
	confidence := pt.calculateConfidence(sampleCount)

	params := &TunedParams{
		SourceFormat:  sourceFormat,
		TargetFormat:  targetFormat,
		QualityGoal:   qualityGoal,
		OptimalSaving: avgSaving,
		OptimalEffort: int(avgEffort),
		OptimalCRF:    int(avgCRF),
		OptimalSpeed:  int(avgSpeed),
		Confidence:    confidence,
		SampleCount:   sampleCount,
		AvgError:      avgError,
		LastUpdated:   time.Now(),
	}

	pt.logger.Info("计算最优参数完成",
		zap.String("source", sourceFormat),
		zap.String("target", targetFormat),
		zap.Int("samples", sampleCount),
		zap.Float64("saving", avgSaving*100),
		zap.Float64("confidence", confidence))

	return params, nil
}

// calculateConfidence 计算置信度
func (pt *PredictionTuner) calculateConfidence(sampleCount int) float64 {
	switch {
	case sampleCount < 5:
		return 0.50 // 样本太少，低置信度
	case sampleCount < 10:
		return 0.60
	case sampleCount < 20:
		return 0.70
	case sampleCount < 50:
		return 0.80
	case sampleCount < 100:
		return 0.85
	case sampleCount < 200:
		return 0.90
	default:
		return 0.95 // 大量样本，高置信度
	}
}

// GetConfidenceThreshold 获取置信度阈值
// 用于决定是否直接使用微调参数还是触发探索
func (pt *PredictionTuner) GetConfidenceThreshold(sampleCount int) float64 {
	switch {
	case sampleCount < 10:
		return 0.60 // 样本少，低阈值，容易触发探索
	case sampleCount < 50:
		return 0.75
	case sampleCount < 200:
		return 0.85
	default:
		return 0.90 // 大量样本，高阈值
	}
}

// getCached 获取缓存
func (pt *PredictionTuner) getCached(key string) *CachedTuning {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	cached, exists := pt.cache[key]
	if !exists {
		return nil
	}

	// 检查是否过期
	if time.Since(cached.CachedAt) > pt.cacheTTL {
		return nil
	}

	// 增加命中计数
	cached.HitCount++
	return cached
}

// setCached 设置缓存
func (pt *PredictionTuner) setCached(key string, params *TunedParams) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	pt.cache[key] = &CachedTuning{
		Params:   params,
		CachedAt: time.Now(),
		HitCount: 0,
	}
}

// ClearCache 清除缓存
func (pt *PredictionTuner) ClearCache() {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	pt.cache = make(map[string]*CachedTuning)
	pt.logger.Info("微调缓存已清除")
}

// GetCacheStats 获取缓存统计
func (pt *PredictionTuner) GetCacheStats() map[string]interface{} {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	totalHits := 0
	for _, cached := range pt.cache {
		totalHits += cached.HitCount
	}

	return map[string]interface{}{
		"cache_size": len(pt.cache),
		"total_hits": totalHits,
		"cache_ttl":  pt.cacheTTL.String(),
	}
}

// SuggestExploration 建议是否需要探索
func (pt *PredictionTuner) SuggestExploration(
	sourceFormat, targetFormat string,
	confidence float64,
) bool {
	// 查询样本数
	query := `
		SELECT COUNT(*)
		FROM conversion_records
		WHERE original_format = ?
		  AND actual_format = ?
	`

	var sampleCount int
	err := pt.db.db.QueryRow(query, sourceFormat, targetFormat).Scan(&sampleCount)
	if err != nil {
		return true // 查询失败，建议探索
	}

	threshold := pt.GetConfidenceThreshold(sampleCount)

	// 如果置信度低于阈值，建议探索
	return confidence < threshold
}

// GetFormatCombinations 获取所有已知的格式组合
func (pt *PredictionTuner) GetFormatCombinations() ([]FormatCombination, error) {
	query := `
		SELECT 
			original_format,
			actual_format,
			COUNT(*) as count,
			AVG(actual_saving_percent) as avg_saving,
			SUM(CASE WHEN validation_passed = 1 THEN 1 ELSE 0 END) as success_count
		FROM conversion_records
		GROUP BY original_format, actual_format
		HAVING count > 5
		ORDER BY count DESC
	`

	rows, err := pt.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询格式组合失败: %w", err)
	}
	defer rows.Close()

	var combinations []FormatCombination
	for rows.Next() {
		var combo FormatCombination
		err := rows.Scan(
			&combo.SourceFormat,
			&combo.TargetFormat,
			&combo.SampleCount,
			&combo.AvgSaving,
			&combo.SuccessCount,
		)
		if err != nil {
			continue
		}

		combo.SuccessRate = float64(combo.SuccessCount) / float64(combo.SampleCount)
		combinations = append(combinations, combo)
	}

	return combinations, nil
}

// FormatCombination 格式组合
type FormatCombination struct {
	SourceFormat string
	TargetFormat string
	SampleCount  int
	AvgSaving    float64
	SuccessCount int
	SuccessRate  float64
}
