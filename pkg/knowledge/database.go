package knowledge

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

//go:embed schema.sql
var schemaSQLRaw string

// Database 知识库数据库
type Database struct {
	db     *sql.DB
	logger *zap.Logger
	path   string
}

// NewDatabase 创建知识库数据库
func NewDatabase(dbPath string, logger *zap.Logger) (*Database, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// 打开数据库
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 初始化Schema
	if _, err := db.Exec(schemaSQLRaw); err != nil {
		return nil, fmt.Errorf("初始化数据库Schema失败: %w", err)
	}

	logger.Info("知识库数据库初始化成功",
		zap.String("path", dbPath))

	return &Database{
		db:     db,
		logger: logger,
		path:   dbPath,
	}, nil
}

// Close 关闭数据库
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// SaveRecord 保存转换记录
func (d *Database) SaveRecord(record *ConversionRecord) error {
	query := `
		INSERT INTO conversion_records (
			created_at, file_path, file_name, original_format, original_size,
			width, height, has_alpha, pix_fmt, is_animated, frame_count, estimated_quality,
			predictor_name, prediction_rule, prediction_confidence, prediction_time_ms,
			predicted_format, predicted_lossless, predicted_distance, predicted_effort,
			predicted_lossless_jpeg, predicted_crf, predicted_speed,
			predicted_saving_percent, predicted_output_size,
			actual_format, actual_output_size, actual_conversion_time_ms,
			actual_saving_percent, actual_saving_bytes,
			validation_method, validation_passed, pixel_diff_percent, psnr_value, ssim_value,
			prediction_error_percent, was_explored,
			user_rating, user_comment,
			pixly_version, host_os
		) VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?,
			?, ?,
			?, ?, ?,
			?, ?,
			?, ?, ?, ?, ?,
			?, ?,
			?, ?,
			?, ?
		)
	`

	result, err := d.db.Exec(query,
		record.CreatedAt, record.FilePath, record.FileName, record.OriginalFormat, record.OriginalSize,
		record.Width, record.Height, record.HasAlpha, record.PixFmt, record.IsAnimated, record.FrameCount, record.EstimatedQuality,
		record.PredictorName, record.PredictionRule, record.PredictionConfidence, record.PredictionTimeMs,
		record.PredictedFormat, record.PredictedLossless, record.PredictedDistance, record.PredictedEffort,
		record.PredictedLosslessJPEG, record.PredictedCRF, record.PredictedSpeed,
		record.PredictedSavingPercent, record.PredictedOutputSize,
		record.ActualFormat, record.ActualOutputSize, record.ActualConversionTimeMs,
		record.ActualSavingPercent, record.ActualSavingBytes,
		record.ValidationMethod, record.ValidationPassed, record.PixelDiffPercent, record.PSNRValue, record.SSIMValue,
		record.PredictionErrorPercent, record.WasExplored,
		record.UserRating, record.UserComment,
		record.PixlyVersion, record.HostOS,
	)

	if err != nil {
		return fmt.Errorf("保存转换记录失败: %w", err)
	}

	id, _ := result.LastInsertId()
	record.ID = id

	d.logger.Debug("转换记录已保存",
		zap.Int64("id", id),
		zap.String("file", record.FileName),
		zap.String("rule", record.PredictionRule))

	return nil
}

// GetPredictionStats 获取预测统计
func (d *Database) GetPredictionStats(predictorName, rule, format string) (*PredictionStats, error) {
	query := `
		SELECT 
			id, predictor_name, prediction_rule, original_format,
			stats_from, stats_to,
			total_conversions, successful_conversions,
			avg_prediction_error_percent, median_prediction_error_percent, std_prediction_error_percent,
			avg_predicted_saving, avg_actual_saving,
			perfect_quality_count, good_quality_count,
			avg_conversion_time_ms,
			updated_at
		FROM prediction_stats
		WHERE predictor_name = ? AND prediction_rule = ? AND original_format = ?
	`

	var stats PredictionStats
	err := d.db.QueryRow(query, predictorName, rule, format).Scan(
		&stats.ID, &stats.PredictorName, &stats.PredictionRule, &stats.OriginalFormat,
		&stats.StatsFrom, &stats.StatsTo,
		&stats.TotalConversions, &stats.SuccessfulConversions,
		&stats.AvgPredictionErrorPercent, &stats.MedianPredictionErrorPercent, &stats.StdPredictionErrorPercent,
		&stats.AvgPredictedSaving, &stats.AvgActualSaving,
		&stats.PerfectQualityCount, &stats.GoodQualityCount,
		&stats.AvgConversionTimeMs,
		&stats.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // 没有统计数据
	}

	if err != nil {
		return nil, fmt.Errorf("查询预测统计失败: %w", err)
	}

	return &stats, nil
}

// UpdateStats 更新预测统计（自动聚合）
func (d *Database) UpdateStats(predictorName, rule, format string) error {
	query := `
		INSERT OR REPLACE INTO prediction_stats (
			predictor_name, prediction_rule, original_format,
			stats_from, stats_to,
			total_conversions, successful_conversions,
			avg_prediction_error_percent,
			avg_predicted_saving, avg_actual_saving,
			perfect_quality_count, good_quality_count,
			avg_conversion_time_ms,
			updated_at
		)
		SELECT 
			?, ?, ?,
			MIN(created_at), MAX(created_at),
			COUNT(*),
			SUM(CASE WHEN validation_passed = 1 THEN 1 ELSE 0 END),
			AVG(prediction_error_percent),
			AVG(predicted_saving_percent), AVG(actual_saving_percent),
			SUM(CASE WHEN pixel_diff_percent = 0 THEN 1 ELSE 0 END),
			SUM(CASE WHEN psnr_value > 40 OR ssim_value > 0.95 THEN 1 ELSE 0 END),
			AVG(actual_conversion_time_ms),
			CURRENT_TIMESTAMP
		FROM conversion_records
		WHERE predictor_name = ? AND prediction_rule = ? AND original_format = ?
	`

	_, err := d.db.Exec(query, predictorName, rule, format, predictorName, rule, format)
	if err != nil {
		return fmt.Errorf("更新预测统计失败: %w", err)
	}

	d.logger.Debug("预测统计已更新",
		zap.String("predictor", predictorName),
		zap.String("rule", rule),
		zap.String("format", format))

	return nil
}

// GetRecentConversions 获取最近的转换记录
func (d *Database) GetRecentConversions(limit int) ([]*ConversionRecord, error) {
	query := `
		SELECT 
			id, created_at, file_path, file_name, original_format, original_size,
			predicted_format, actual_format, actual_output_size,
			actual_saving_percent, validation_passed, prediction_error_percent
		FROM conversion_records
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("查询最近转换记录失败: %w", err)
	}
	defer rows.Close()

	var records []*ConversionRecord
	for rows.Next() {
		var r ConversionRecord
		err := rows.Scan(
			&r.ID, &r.CreatedAt, &r.FilePath, &r.FileName, &r.OriginalFormat, &r.OriginalSize,
			&r.PredictedFormat, &r.ActualFormat, &r.ActualOutputSize,
			&r.ActualSavingPercent, &r.ValidationPassed, &r.PredictionErrorPercent,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描转换记录失败: %w", err)
		}
		records = append(records, &r)
	}

	return records, nil
}

// DetectAnomalies 检测异常案例
func (d *Database) DetectAnomalies() ([]*AnomalyCase, error) {
	// 异常检测规则：
	// 1. 预测误差 > 30%
	// 2. 质量验证失败
	// 3. 实际节省为负（文件变大）

	query := `
		SELECT id, file_name, original_format, prediction_rule,
		       prediction_error_percent, validation_passed, actual_saving_percent
		FROM conversion_records
		WHERE 
			(prediction_error_percent > 0.3) OR
			(validation_passed = 0) OR
			(actual_saving_percent < 0)
		ORDER BY created_at DESC
		LIMIT 50
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询异常案例失败: %w", err)
	}
	defer rows.Close()

	var anomalies []*AnomalyCase
	for rows.Next() {
		var recordID int64
		var fileName, format, rule string
		var predError, saving float64
		var validPassed bool

		err := rows.Scan(&recordID, &fileName, &format, &rule, &predError, &validPassed, &saving)
		if err != nil {
			continue
		}

		anomaly := &AnomalyCase{
			ConversionRecordID: recordID,
		}

		// 判断异常类型
		if predError > 0.3 {
			anomaly.AnomalyType = "large_prediction_error"
			anomaly.AnomalySeverity = "medium"
			anomaly.Description = fmt.Sprintf("预测误差%.1f%%", predError*100)
		} else if !validPassed {
			anomaly.AnomalyType = "quality_validation_failed"
			anomaly.AnomalySeverity = "high"
			anomaly.Description = "质量验证失败"
		} else if saving < 0 {
			anomaly.AnomalyType = "file_size_increased"
			anomaly.AnomalySeverity = "medium"
			anomaly.Description = fmt.Sprintf("文件变大%.1f%%", -saving*100)
		}

		anomalies = append(anomalies, anomaly)
	}

	return anomalies, nil
}

// GetStatsSummary 获取统计摘要
func (d *Database) GetStatsSummary() (map[string]interface{}, error) {
	summary := make(map[string]interface{})

	// 总转换次数
	var totalConversions int
	err := d.db.QueryRow("SELECT COUNT(*) FROM conversion_records").Scan(&totalConversions)
	if err != nil {
		return nil, err
	}
	summary["total_conversions"] = totalConversions

	// 平均空间节省
	var avgSaving float64
	err = d.db.QueryRow("SELECT AVG(actual_saving_percent) FROM conversion_records").Scan(&avgSaving)
	if err == nil {
		summary["avg_saving_percent"] = avgSaving * 100
	}

	// 质量验证通过率
	var passRate float64
	err = d.db.QueryRow(`
		SELECT 100.0 * SUM(CASE WHEN validation_passed = 1 THEN 1 ELSE 0 END) / COUNT(*)
		FROM conversion_records
	`).Scan(&passRate)
	if err == nil {
		summary["quality_pass_rate"] = passRate
	}

	// 平均预测误差
	var avgError float64
	err = d.db.QueryRow("SELECT AVG(prediction_error_percent) FROM conversion_records").Scan(&avgError)
	if err == nil {
		summary["avg_prediction_error"] = avgError * 100
	}

	return summary, nil
}
