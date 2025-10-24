package knowledge

import (
	"fmt"
	"time"
)

// QueryAPI 知识库查询API
type QueryAPI struct {
	db *Database
}

// NewQueryAPI 创建查询API
func NewQueryAPI(db *Database) *QueryAPI {
	return &QueryAPI{db: db}
}

// QueryBuilder 查询构建器
type QueryBuilder struct {
	api        *QueryAPI
	conditions []string
	params     []interface{}
	orderBy    string
	limit      int
}

// NewQuery 创建新查询
func (api *QueryAPI) NewQuery() *QueryBuilder {
	return &QueryBuilder{
		api:        api,
		conditions: []string{},
		params:     []interface{}{},
		limit:      100,
	}
}

// WhereFormat 按格式筛选
func (qb *QueryBuilder) WhereFormat(format string) *QueryBuilder {
	qb.conditions = append(qb.conditions, "original_format = ?")
	qb.params = append(qb.params, format)
	return qb
}

// WherePredictor 按预测器筛选
func (qb *QueryBuilder) WherePredictor(predictorName string) *QueryBuilder {
	qb.conditions = append(qb.conditions, "predictor_name = ?")
	qb.params = append(qb.params, predictorName)
	return qb
}

// WhereRule 按规则筛选
func (qb *QueryBuilder) WhereRule(rule string) *QueryBuilder {
	qb.conditions = append(qb.conditions, "prediction_rule = ?")
	qb.params = append(qb.params, rule)
	return qb
}

// WhereValidationPassed 按验证结果筛选
func (qb *QueryBuilder) WhereValidationPassed(passed bool) *QueryBuilder {
	qb.conditions = append(qb.conditions, "validation_passed = ?")
	qb.params = append(qb.params, passed)
	return qb
}

// WhereDateRange 按时间范围筛选
func (qb *QueryBuilder) WhereDateRange(from, to time.Time) *QueryBuilder {
	qb.conditions = append(qb.conditions, "created_at BETWEEN ? AND ?")
	qb.params = append(qb.params, from, to)
	return qb
}

// WhereSavingGreaterThan 按空间节省筛选
func (qb *QueryBuilder) WhereSavingGreaterThan(percent float64) *QueryBuilder {
	qb.conditions = append(qb.conditions, "actual_saving_percent > ?")
	qb.params = append(qb.params, percent)
	return qb
}

// OrderByCreatedAt 按创建时间排序
func (qb *QueryBuilder) OrderByCreatedAt(desc bool) *QueryBuilder {
	if desc {
		qb.orderBy = "created_at DESC"
	} else {
		qb.orderBy = "created_at ASC"
	}
	return qb
}

// OrderBySaving 按空间节省排序
func (qb *QueryBuilder) OrderBySaving(desc bool) *QueryBuilder {
	if desc {
		qb.orderBy = "actual_saving_percent DESC"
	} else {
		qb.orderBy = "actual_saving_percent ASC"
	}
	return qb
}

// Limit 限制结果数量
func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
	qb.limit = n
	return qb
}

// Execute 执行查询
func (qb *QueryBuilder) Execute() ([]*ConversionRecord, error) {
	query := "SELECT * FROM conversion_records"

	if len(qb.conditions) > 0 {
		query += " WHERE "
		for i, cond := range qb.conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	if qb.orderBy != "" {
		query += " ORDER BY " + qb.orderBy
	}

	if qb.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.limit)
	}

	rows, err := qb.api.db.db.Query(query, qb.params...)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	var records []*ConversionRecord
	for rows.Next() {
		var r ConversionRecord
		err := rows.Scan(
			&r.ID, &r.CreatedAt,
			&r.FilePath, &r.FileName, &r.OriginalFormat, &r.OriginalSize,
			&r.Width, &r.Height, &r.HasAlpha, &r.PixFmt, &r.IsAnimated, &r.FrameCount, &r.EstimatedQuality,
			&r.PredictorName, &r.PredictionRule, &r.PredictionConfidence, &r.PredictionTimeMs,
			&r.PredictedFormat, &r.PredictedLossless, &r.PredictedDistance, &r.PredictedEffort,
			&r.PredictedLosslessJPEG, &r.PredictedCRF, &r.PredictedSpeed,
			&r.PredictedSavingPercent, &r.PredictedOutputSize,
			&r.ActualFormat, &r.ActualOutputSize, &r.ActualConversionTimeMs,
			&r.ActualSavingPercent, &r.ActualSavingBytes,
			&r.ValidationMethod, &r.ValidationPassed, &r.PixelDiffPercent, &r.PSNRValue, &r.SSIMValue,
			&r.PredictionErrorPercent, &r.WasExplored,
			&r.UserRating, &r.UserComment,
			&r.PixlyVersion, &r.HostOS,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描记录失败: %w", err)
		}
		records = append(records, &r)
	}

	return records, nil
}

// GetBestConversions 获取效果最好的转换
func (api *QueryAPI) GetBestConversions(format string, limit int) ([]*ConversionRecord, error) {
	return api.NewQuery().
		WhereFormat(format).
		WhereValidationPassed(true).
		OrderBySaving(true).
		Limit(limit).
		Execute()
}

// GetWorstConversions 获取效果最差的转换
func (api *QueryAPI) GetWorstConversions(format string, limit int) ([]*ConversionRecord, error) {
	return api.NewQuery().
		WhereFormat(format).
		OrderBySaving(false).
		Limit(limit).
		Execute()
}

// GetRecentByPredictor 获取特定预测器的最近转换
func (api *QueryAPI) GetRecentByPredictor(predictorName string, limit int) ([]*ConversionRecord, error) {
	return api.NewQuery().
		WherePredictor(predictorName).
		OrderByCreatedAt(true).
		Limit(limit).
		Execute()
}

// GetFailedConversions 获取失败的转换
func (api *QueryAPI) GetFailedConversions(limit int) ([]*ConversionRecord, error) {
	return api.NewQuery().
		WhereValidationPassed(false).
		OrderByCreatedAt(true).
		Limit(limit).
		Execute()
}

// AggregateStats 聚合统计
type AggregateStats struct {
	TotalRecords       int
	AvgSavingPercent   float64
	TotalBytesSaved    int64
	AvgPredictionError float64
	QualityPassRate    float64
}

// GetAggregateStats 获取聚合统计
func (api *QueryAPI) GetAggregateStats(format string) (*AggregateStats, error) {
	query := `
		SELECT 
			COUNT(*),
			AVG(actual_saving_percent),
			SUM(actual_saving_bytes),
			AVG(prediction_error_percent),
			100.0 * SUM(CASE WHEN validation_passed = 1 THEN 1 ELSE 0 END) / COUNT(*)
		FROM conversion_records
		WHERE original_format = ?
	`

	var stats AggregateStats
	err := api.db.db.QueryRow(query, format).Scan(
		&stats.TotalRecords,
		&stats.AvgSavingPercent,
		&stats.TotalBytesSaved,
		&stats.AvgPredictionError,
		&stats.QualityPassRate,
	)

	if err != nil {
		return nil, fmt.Errorf("查询聚合统计失败: %w", err)
	}

	return &stats, nil
}
