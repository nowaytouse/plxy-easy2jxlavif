package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pixly/pkg/monitor"

	"go.uber.org/zap"
)

// PerformanceReporter 性能报告生成器
// 负责生成详细的性能分析报告（JSON和文本格式）
type PerformanceReporter struct {
	logger     *zap.Logger
	reportDir  string // 报告保存目录（默认~/.pixly/reports/）
	enableJSON bool   // 生成JSON报告
	enableText bool   // 生成文本报告
}

// NewPerformanceReporter 创建性能报告生成器
func NewPerformanceReporter(logger *zap.Logger, reportDir string, enableJSON, enableText bool) *PerformanceReporter {
	// 确保报告目录存在
	if reportDir == "" {
		homeDir, _ := os.UserHomeDir()
		reportDir = filepath.Join(homeDir, ".pixly", "reports")
	}

	os.MkdirAll(reportDir, 0755)

	return &PerformanceReporter{
		logger:     logger,
		reportDir:  reportDir,
		enableJSON: enableJSON,
		enableText: enableText,
	}
}

// GenerateReport 生成性能报告
func (pr *PerformanceReporter) GenerateReport(summary *monitor.PerformanceSummary) error {
	timestamp := time.Now().Format("20060102_150405")

	// 生成JSON报告
	if pr.enableJSON {
		jsonPath := filepath.Join(pr.reportDir, fmt.Sprintf("performance_%s.json", timestamp))
		if err := pr.generateJSONReport(jsonPath, summary); err != nil {
			pr.logger.Error("生成JSON报告失败", zap.Error(err))
		} else {
			pr.logger.Info("📊 性能报告已生成（JSON）", zap.String("path", jsonPath))
		}
	}

	// 生成文本报告
	if pr.enableText {
		txtPath := filepath.Join(pr.reportDir, fmt.Sprintf("performance_%s.txt", timestamp))
		if err := pr.generateTextReport(txtPath, summary); err != nil {
			pr.logger.Error("生成文本报告失败", zap.Error(err))
		} else {
			pr.logger.Info("📊 性能报告已生成（文本）", zap.String("path", txtPath))
		}
	}

	return nil
}

// generateJSONReport 生成JSON格式报告
func (pr *PerformanceReporter) generateJSONReport(filePath string, summary *monitor.PerformanceSummary) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // 美化JSON
	return encoder.Encode(summary)
}

// generateTextReport 生成文本格式报告
func (pr *PerformanceReporter) generateTextReport(filePath string, summary *monitor.PerformanceSummary) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 生成报告内容
	content := pr.formatTextReport(summary)
	_, err = file.WriteString(content)
	return err
}

// formatTextReport 格式化文本报告
func (pr *PerformanceReporter) formatTextReport(summary *monitor.PerformanceSummary) string {
	report := fmt.Sprintf(`
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   📊 Pixly性能报告                                           ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝

生成时间: %s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⏱️  会话信息
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

开始时间: %s
结束时间: %s
总耗时:   %s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📁 文件统计
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

总文件数:     %d
成功处理:     %d (%.1f%%)
处理失败:     %d (%.1f%%)
跳过文件:     %d (%.1f%%)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚡ 性能指标
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

吞吐量:       %.2f 文件/秒
处理速度:     %.2f MB/秒

CPU峰值:      %.1f%%
CPU平均:      %.1f%%

内存峰值:     %.1f%%
内存平均:     %.1f%%

磁盘读取:     %s
磁盘写入:     %s

GC次数:       %d 次
GC总暂停:     %s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔧 Worker调整历史
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

初始Worker:   %d
最终Worker:   %d
平均Worker:   %.1f
调整次数:     %d

%s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

报告生成完成！

位置: %s
`,
		time.Now().Format("2006-01-02 15:04:05"),
		summary.StartTime.Format("2006-01-02 15:04:05"),
		summary.EndTime.Format("2006-01-02 15:04:05"),
		formatDuration(summary.Duration),
		summary.TotalFiles,
		summary.ProcessedOK, float64(summary.ProcessedOK)/float64(summary.TotalFiles)*100,
		summary.ProcessedFail, float64(summary.ProcessedFail)/float64(summary.TotalFiles)*100,
		summary.Skipped, float64(summary.Skipped)/float64(summary.TotalFiles)*100,
		summary.AvgThroughput,
		summary.AvgProcessingRate,
		summary.PeakCPU*100,
		summary.AvgCPU*100,
		summary.PeakMemory*100,
		summary.AvgMemory*100,
		formatBytes(summary.TotalDiskRead),
		formatBytes(summary.TotalDiskWrite),
		summary.GCCount,
		summary.TotalGCPause,
		summary.InitialWorkers,
		summary.FinalWorkers,
		summary.AvgWorkers,
		len(summary.WorkerAdjustments),
		formatWorkerAdjustments(summary.WorkerAdjustments),
		pr.reportDir,
	)

	return report
}

// formatWorkerAdjustments 格式化Worker调整历史
func formatWorkerAdjustments(adjustments []monitor.WorkerAdjustment) string {
	if len(adjustments) == 0 {
		return "无调整记录\n"
	}

	result := ""
	for i, adj := range adjustments {
		icon := "⬆️"
		if adj.Action == "decrease" {
			icon = "⬇️"
		} else if adj.Action == "maintain" {
			icon = "➡️"
		}

		result += fmt.Sprintf(
			"%s [%s] %s: %d → %d (%s)\n",
			icon,
			adj.Timestamp.Format("15:04:05"),
			adj.Action,
			adj.OldWorkers,
			adj.NewWorkers,
			adj.Reason,
		)

		// 最多显示前10条
		if i >= 9 {
			remaining := len(adjustments) - 10
			if remaining > 0 {
				result += fmt.Sprintf("... 还有 %d 条调整记录\n", remaining)
			}
			break
		}
	}

	return result
}

// formatDuration 格式化时长
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		return fmt.Sprintf("%dh%dm%ds", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
	}
}

// formatBytes 格式化字节数
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// GenerateQuickSummary 生成快速摘要（用于终端输出）
func GenerateQuickSummary(summary *monitor.PerformanceSummary) string {
	successRate := float64(summary.ProcessedOK) / float64(summary.TotalFiles) * 100

	return fmt.Sprintf(`
🎊 转换完成！

📊 统计:
  • 总文件: %d
  • 成功: %d (%.1f%%)
  • 失败: %d
  • 耗时: %s

⚡ 性能:
  • 吞吐: %.1f 文件/秒
  • CPU峰值: %.1f%%
  • 内存峰值: %.1f%%
  • Worker调整: %d 次
`,
		summary.TotalFiles,
		summary.ProcessedOK, successRate,
		summary.ProcessedFail,
		formatDuration(summary.Duration),
		summary.AvgThroughput,
		summary.PeakCPU*100,
		summary.PeakMemory*100,
		len(summary.WorkerAdjustments),
	)
}


