package quality

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Reporter generates quality analysis reports
type Reporter struct {
	report *QualityReport
}

// NewReporter creates a new quality reporter
func NewReporter(sessionID string) *Reporter {
	return &Reporter{
		report: &QualityReport{
			SessionID:  sessionID,
			StartTime:  time.Now(),
			QualityDistribution: QualityDistribution{},
			CompressionEffectiveness: make(map[string]CompressionStats),
			FormatDistribution: struct {
				Source map[string]int
				Target map[string]int
			}{
				Source: make(map[string]int),
				Target: make(map[string]int),
			},
			ContentTypeDistribution:  make(map[string]int),
			QualityClassDistribution: make(map[string]int),
		},
	}
}

// AddMetrics adds quality metrics to the report
func (r *Reporter) AddMetrics(metrics *QualityMetrics) {
	if metrics == nil {
		return
	}
	
	r.report.TotalFiles++
	
	// 更新质量分布
	switch metrics.QualityClass {
	case "极高":
		r.report.QualityDistribution.ExtremelyHigh++
	case "高":
		r.report.QualityDistribution.High++
	case "中":
		r.report.QualityDistribution.Medium++
	case "低":
		r.report.QualityDistribution.Low++
	case "极低":
		r.report.QualityDistribution.ExtremelyLow++
	}
	r.report.QualityDistribution.Total++
	
	// 更新格式分布
	r.report.FormatDistribution.Source[metrics.Format]++
	
	// 更新内容类型分布
	r.report.ContentTypeDistribution[metrics.ContentType]++
	
	// 更新质量类别分布
	r.report.QualityClassDistribution[metrics.QualityClass]++
}

// AddConversionResult adds conversion result to compression stats
func (r *Reporter) AddConversionResult(
	format string,
	sizeBefore, sizeAfter int64,
	bppBefore, bppAfter float64,
) {
	stats, exists := r.report.CompressionEffectiveness[format]
	if !exists {
		stats = CompressionStats{
			Format: format,
		}
	}
	
	stats.FileCount++
	stats.TotalBefore += sizeBefore
	stats.TotalAfter += sizeAfter
	
	// 计算节省率
	saving := 1.0 - (float64(sizeAfter) / float64(sizeBefore))
	
	// 更新统计
	if stats.FileCount == 1 {
		stats.AvgSaving = saving
		stats.BestSaving = saving
		stats.WorstSaving = saving
		stats.AvgBPP = bppBefore
	} else {
		stats.AvgSaving = (stats.AvgSaving*float64(stats.FileCount-1) + saving) / float64(stats.FileCount)
		if saving > stats.BestSaving {
			stats.BestSaving = saving
		}
		if saving < stats.WorstSaving {
			stats.WorstSaving = saving
		}
		stats.AvgBPP = (stats.AvgBPP*float64(stats.FileCount-1) + bppBefore) / float64(stats.FileCount)
	}
	
	r.report.CompressionEffectiveness[format] = stats
}

// Finalize finalizes the report
func (r *Reporter) Finalize() {
	r.report.EndTime = time.Now()
	r.report.Duration = r.report.EndTime.Sub(r.report.StartTime)
	
	// 计算平均BytesPerPixel
	totalBPP := 0.0
	count := 0
	for _, stats := range r.report.CompressionEffectiveness {
		totalBPP += stats.AvgBPP
		count++
	}
	if count > 0 {
		r.report.AvgBytesPerPixel.Before = totalBPP / float64(count)
	}
}

// SaveJSON saves report as JSON
func (r *Reporter) SaveJSON(outputPath string) error {
	r.Finalize()
	
	// 创建目录
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	
	// 序列化
	data, err := json.MarshalIndent(r.report, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化报告失败: %w", err)
	}
	
	// 写入文件
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("写入报告失败: %w", err)
	}
	
	return nil
}

// SaveText saves report as text
func (r *Reporter) SaveText(outputPath string) error {
	r.Finalize()
	
	// 创建目录
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	
	// 生成文本报告
	text := r.generateTextReport()
	
	// 写入文件
	if err := os.WriteFile(outputPath, []byte(text), 0644); err != nil {
		return fmt.Errorf("写入报告失败: %w", err)
	}
	
	return nil
}

// generateTextReport generates text format report
func (r *Reporter) generateTextReport() string {
	var text string
	
	text += "╔═══════════════════════════════════════════════════════════════╗\n"
	text += "║                                                               ║\n"
	text += "║   📊 Pixly质量分析报告                                       ║\n"
	text += "║                                                               ║\n"
	text += "╚═══════════════════════════════════════════════════════════════╝\n\n"
	
	text += fmt.Sprintf("会话ID: %s\n", r.report.SessionID)
	text += fmt.Sprintf("开始时间: %s\n", r.report.StartTime.Format("2006-01-02 15:04:05"))
	text += fmt.Sprintf("结束时间: %s\n", r.report.EndTime.Format("2006-01-02 15:04:05"))
	text += fmt.Sprintf("总耗时: %v\n", r.report.Duration)
	text += fmt.Sprintf("总文件数: %d\n\n", r.report.TotalFiles)
	
	// 质量分布
	text += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	text += "质量分布:\n"
	text += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	text += fmt.Sprintf("  极高品质: %d\n", r.report.QualityDistribution.ExtremelyHigh)
	text += fmt.Sprintf("  高品质:   %d\n", r.report.QualityDistribution.High)
	text += fmt.Sprintf("  中等品质: %d\n", r.report.QualityDistribution.Medium)
	text += fmt.Sprintf("  低品质:   %d\n", r.report.QualityDistribution.Low)
	text += fmt.Sprintf("  极低品质: %d\n\n", r.report.QualityDistribution.ExtremelyLow)
	
	// 压缩效果
	text += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	text += "压缩效果:\n"
	text += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	for format, stats := range r.report.CompressionEffectiveness {
		text += fmt.Sprintf("  %s:\n", format)
		text += fmt.Sprintf("    文件数: %d\n", stats.FileCount)
		text += fmt.Sprintf("    平均节省: %.1f%%\n", stats.AvgSaving*100)
		text += fmt.Sprintf("    最佳节省: %.1f%%\n", stats.BestSaving*100)
		text += fmt.Sprintf("    最差节省: %.1f%%\n", stats.WorstSaving*100)
		text += fmt.Sprintf("    平均BPP: %.2f\n\n", stats.AvgBPP)
	}
	
	// 格式分布
	text += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	text += "格式分布:\n"
	text += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	text += "  源格式:\n"
	for format, count := range r.report.FormatDistribution.Source {
		text += fmt.Sprintf("    %s: %d\n", format, count)
	}
	text += "\n  目标格式:\n"
	for format, count := range r.report.FormatDistribution.Target {
		text += fmt.Sprintf("    %s: %d\n", format, count)
	}
	
	return text
}

// GetReport returns the current report
func (r *Reporter) GetReport() *QualityReport {
	return r.report
}
