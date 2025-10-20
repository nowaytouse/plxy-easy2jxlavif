package progressui

import (
	"fmt"
	"sync"
	"time"

	"pixly/pkg/core/types"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"go.uber.org/zap"
)

// AdvancedProgressUI README要求的mpb/v8实时精确进度条显示系统
type AdvancedProgressUI struct {
	logger *zap.Logger
	mutex  sync.RWMutex

	// README要求：使用mpb/v8实现实时精确的进度条显示
	container *mpb.Progress

	// 进度条管理
	scanBar       *mpb.Bar // 扫描进度条
	analysisBar   *mpb.Bar // 分析进度条
	processingBar *mpb.Bar // 处理进度条
	overallBar    *mpb.Bar // 总体进度条

	// 统计信息显示
	statsDisplay *StatsDisplay

	// 配置参数
	refreshRate  time.Duration
	enableColors bool
	showETA      bool
	showSpeed    bool
	showDetailed bool

	// 实时统计
	stats      *UIStats
	startTime  time.Time
	lastUpdate time.Time

	// 状态管理
	isActive     bool
	currentPhase ProcessingPhase
	totalPhases  int
}

// ProcessingPhase 处理阶段
type ProcessingPhase int

const (
	PhaseIdle       ProcessingPhase = iota
	PhaseScanning                   // 扫描阶段
	PhaseAnalyzing                  // 分析阶段
	PhaseProcessing                 // 处理阶段
	PhaseCompleted                  // 完成阶段
)

func (pp ProcessingPhase) String() string {
	switch pp {
	case PhaseScanning:
		return "扫描文件"
	case PhaseAnalyzing:
		return "分析品质"
	case PhaseProcessing:
		return "处理转换"
	case PhaseCompleted:
		return "处理完成"
	default:
		return "等待中"
	}
}

// UIStats UI统计信息
type UIStats struct {
	// 文件统计
	TotalFiles     int64 `json:"total_files"`
	ScannedFiles   int64 `json:"scanned_files"`
	AnalyzedFiles  int64 `json:"analyzed_files"`
	ProcessedFiles int64 `json:"processed_files"`
	SuccessFiles   int64 `json:"success_files"`
	SkippedFiles   int64 `json:"skipped_files"`
	FailedFiles    int64 `json:"failed_files"`

	// 性能统计
	ScanSpeed       float64 `json:"scan_speed"`       // 文件/秒
	ProcessingSpeed float64 `json:"processing_speed"` // 文件/秒
	ThroughputMB    float64 `json:"throughput_mb"`    // MB/秒

	// 空间统计 - README要求的统计报告格式
	SpaceIncreased int64 `json:"space_increased"` // 增加的空间 ⬆️
	SpaceDecreased int64 `json:"space_decreased"` // 减小的空间 ⬇️
	SpaceSaved     int64 `json:"space_saved"`     // 节省的空间 💰

	// 时间统计
	ElapsedTime     time.Duration `json:"elapsed_time"`
	EstimatedTotal  time.Duration `json:"estimated_total"`
	EstimatedRemain time.Duration `json:"estimated_remain"`

	// 品质分布 - README要求的详细报告
	QualityDistrib map[types.QualityLevel]int64 `json:"quality_distribution"`
}

// StatsDisplay 统计信息显示器
type StatsDisplay struct {
	mutex        sync.RWMutex
	stats        *UIStats
	lastStats    *UIStats
	displayLines []string
	updateTicker *time.Ticker
}

// NewAdvancedProgressUI 创建高级进度UI
func NewAdvancedProgressUI(logger *zap.Logger) *AdvancedProgressUI {
	ui := &AdvancedProgressUI{
		logger:       logger,
		container:    mpb.New(mpb.WithWidth(80), mpb.WithRefreshRate(100*time.Millisecond)),
		refreshRate:  100 * time.Millisecond, // README要求：实时更新
		enableColors: true,
		showETA:      true,
		showSpeed:    true,
		showDetailed: true,
		totalPhases:  4, // 扫描、分析、处理、完成
		stats: &UIStats{
			QualityDistrib: make(map[types.QualityLevel]int64),
		},
		statsDisplay: &StatsDisplay{
			displayLines: make([]string, 0),
		},
	}

	ui.statsDisplay.stats = ui.stats
	ui.statsDisplay.updateTicker = time.NewTicker(ui.refreshRate)

	logger.Info("高级进度UI初始化完成",
		zap.Duration("refresh_rate", ui.refreshRate),
		zap.Bool("colors_enabled", ui.enableColors))

	return ui
}

// StartScanningPhase 开始扫描阶段 - README统一扫描架构
func (ui *AdvancedProgressUI) StartScanningPhase(totalFiles int64) {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	ui.isActive = true
	ui.currentPhase = PhaseScanning
	ui.startTime = time.Now()
	ui.stats.TotalFiles = totalFiles

	// 创建扫描进度条
	ui.scanBar = ui.container.AddBar(totalFiles,
		mpb.PrependDecorators(
			decor.Name("🔍 扫描: ", decor.WC{W: 10}),
			decor.CountersNoUnit("%d/%d", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 5}),
			decor.Name(" | "),
			decor.EwmaETA(decor.ET_STYLE_GO, 30, decor.WC{W: 4}),
			decor.Name(" | "),
			decor.EwmaSpeed(decor.SizeB1024(0), "%.1f files/s", 30),
		),
	)

	// 创建总体进度条
	ui.overallBar = ui.container.AddBar(int64(ui.totalPhases),
		mpb.PrependDecorators(
			decor.Name("📊 总进度: ", decor.WC{W: 10}),
			decor.CountersNoUnit("%d/%d 阶段", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 5}),
			decor.Name(" | "),
			decor.Elapsed(decor.ET_STYLE_GO, decor.WC{W: 4}),
		),
	)

	ui.logger.Info("开始扫描阶段",
		zap.Int64("total_files", totalFiles),
		zap.String("phase", ui.currentPhase.String()))
}

// UpdateScanProgress 更新扫描进度
func (ui *AdvancedProgressUI) UpdateScanProgress(scannedCount int64) {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	ui.stats.ScannedFiles = scannedCount

	if ui.scanBar != nil {
		ui.scanBar.SetCurrent(scannedCount)
	}

	// 计算扫描速度
	elapsed := time.Since(ui.startTime)
	if elapsed > 0 {
		ui.stats.ScanSpeed = float64(scannedCount) / elapsed.Seconds()
	}

	ui.updateGeneralStats()
}

// StartAnalysisPhase 开始分析阶段 - README智能品质判断
func (ui *AdvancedProgressUI) StartAnalysisPhase(totalFiles int64) {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	ui.currentPhase = PhaseAnalyzing

	// 完成扫描进度条
	if ui.scanBar != nil {
		ui.scanBar.SetTotal(ui.stats.ScannedFiles, true)
	}

	// 更新总体进度
	if ui.overallBar != nil {
		ui.overallBar.Increment()
	}

	// 创建分析进度条
	ui.analysisBar = ui.container.AddBar(totalFiles,
		mpb.PrependDecorators(
			decor.Name("🧠 分析: ", decor.WC{W: 10}),
			decor.CountersNoUnit("%d/%d", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 5}),
			decor.Name(" | "),
			decor.EwmaETA(decor.ET_STYLE_GO, 30, decor.WC{W: 4}),
			decor.Name(" | 品质评估"),
		),
	)

	ui.logger.Info("开始分析阶段",
		zap.Int64("total_files", totalFiles),
		zap.String("phase", ui.currentPhase.String()))
}

// UpdateAnalysisProgress 更新分析进度
func (ui *AdvancedProgressUI) UpdateAnalysisProgress(analyzedCount int64, qualityStats map[types.QualityLevel]int64) {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	ui.stats.AnalyzedFiles = analyzedCount
	ui.stats.QualityDistrib = qualityStats

	if ui.analysisBar != nil {
		ui.analysisBar.SetCurrent(analyzedCount)
	}

	ui.updateGeneralStats()
}

// StartProcessingPhase 开始处理阶段 - README核心处理
func (ui *AdvancedProgressUI) StartProcessingPhase(totalFiles int64) {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	ui.currentPhase = PhaseProcessing

	// 完成分析进度条
	if ui.analysisBar != nil {
		ui.analysisBar.SetTotal(ui.stats.AnalyzedFiles, true)
	}

	// 更新总体进度
	if ui.overallBar != nil {
		ui.overallBar.Increment()
	}

	// 创建处理进度条
	ui.processingBar = ui.container.AddBar(totalFiles,
		mpb.PrependDecorators(
			decor.Name("⚡ 处理: ", decor.WC{W: 10}),
			decor.CountersNoUnit("%d/%d", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 5}),
			decor.Name(" | "),
			decor.EwmaETA(decor.ET_STYLE_GO, 60, decor.WC{W: 4}),
			decor.Name(" | "),
			decor.EwmaSpeed(decor.SizeB1024(0), "%.1f files/s", 60),
			decor.Name(" | "),
			decor.Any(func(statistics decor.Statistics) string {
				ui.mutex.RLock()
				defer ui.mutex.RUnlock()
				if ui.stats.ThroughputMB > 0 {
					return fmt.Sprintf("%.1f MB/s", ui.stats.ThroughputMB)
				}
				return "0 MB/s"
			}),
		),
	)

	ui.logger.Info("开始处理阶段",
		zap.Int64("total_files", totalFiles),
		zap.String("phase", ui.currentPhase.String()))
}

// UpdateProcessingProgress 更新处理进度
func (ui *AdvancedProgressUI) UpdateProcessingProgress(processed, success, skipped, failed int64, throughputMB float64) {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	ui.stats.ProcessedFiles = processed
	ui.stats.SuccessFiles = success
	ui.stats.SkippedFiles = skipped
	ui.stats.FailedFiles = failed
	ui.stats.ThroughputMB = throughputMB

	if ui.processingBar != nil {
		ui.processingBar.SetCurrent(processed)
	}

	// 计算处理速度
	elapsed := time.Since(ui.startTime)
	if elapsed > 0 {
		ui.stats.ProcessingSpeed = float64(processed) / elapsed.Seconds()
	}

	ui.updateGeneralStats()
}

// UpdateSpaceStats 更新空间统计 - README要求的统计报告格式
func (ui *AdvancedProgressUI) UpdateSpaceStats(increased, decreased int64) {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	ui.stats.SpaceIncreased += increased
	ui.stats.SpaceDecreased += decreased
	ui.stats.SpaceSaved = ui.stats.SpaceDecreased - ui.stats.SpaceIncreased
}

// CompleteProcessing 完成处理阶段
func (ui *AdvancedProgressUI) CompleteProcessing() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	ui.currentPhase = PhaseCompleted

	// 完成处理进度条
	if ui.processingBar != nil {
		ui.processingBar.SetTotal(ui.stats.ProcessedFiles, true)
	}

	// 完成总体进度
	if ui.overallBar != nil {
		ui.overallBar.Increment()
		ui.overallBar.SetTotal(int64(ui.totalPhases), true)
	}

	ui.logger.Info("处理阶段完成",
		zap.Int64("processed_files", ui.stats.ProcessedFiles),
		zap.Int64("success_files", ui.stats.SuccessFiles))
}

// GenerateStatisticsReport 生成统计报告 - README要求的报告格式
func (ui *AdvancedProgressUI) GenerateStatisticsReport() string {
	ui.mutex.RLock()
	defer ui.mutex.RUnlock()

	// README要求的报告格式：(增加) ⬆️ 150 MB - (减小) ⬇️ 2500 MB = 💰节省: 2350 MB
	report := fmt.Sprintf("\n🎉 处理完成统计报告：\n")
	report += fmt.Sprintf("📁 总文件数: %d\n", ui.stats.TotalFiles)
	report += fmt.Sprintf("✅ 成功处理: %d\n", ui.stats.SuccessFiles)
	report += fmt.Sprintf("⚠️  跳过文件: %d\n", ui.stats.SkippedFiles)
	report += fmt.Sprintf("❌ 失败文件: %d\n", ui.stats.FailedFiles)
	report += fmt.Sprintf("⏱️  处理时间: %v\n", ui.stats.ElapsedTime.Round(time.Second))
	report += fmt.Sprintf("⚡ 处理速度: %.1f 文件/秒\n", ui.stats.ProcessingSpeed)

	// README要求的空间节省格式
	if ui.stats.SpaceIncreased > 0 || ui.stats.SpaceDecreased > 0 {
		report += fmt.Sprintf("\n💾 空间统计：\n")
		report += fmt.Sprintf("(增加) ⬆️ %.0f MB - (减小) ⬇️ %.0f MB = 💰节省: %.0f MB\n",
			float64(ui.stats.SpaceIncreased)/(1024*1024),
			float64(ui.stats.SpaceDecreased)/(1024*1024),
			float64(ui.stats.SpaceSaved)/(1024*1024))
	}

	// 品质分布统计
	if len(ui.stats.QualityDistrib) > 0 {
		report += "\n🎯 品质分布：\n"
		for quality, count := range ui.stats.QualityDistrib {
			if count > 0 {
				report += fmt.Sprintf("  %s: %d 文件\n", quality.String(), count)
			}
		}
	}

	return report
}

// 辅助方法
func (ui *AdvancedProgressUI) updateGeneralStats() {
	ui.stats.ElapsedTime = time.Since(ui.startTime)
	ui.lastUpdate = time.Now()

	// 估算剩余时间
	if ui.stats.ProcessedFiles > 0 && ui.stats.TotalFiles > ui.stats.ProcessedFiles {
		avgTimePerFile := ui.stats.ElapsedTime / time.Duration(ui.stats.ProcessedFiles)
		remainingFiles := ui.stats.TotalFiles - ui.stats.ProcessedFiles
		ui.stats.EstimatedRemain = avgTimePerFile * time.Duration(remainingFiles)
		ui.stats.EstimatedTotal = ui.stats.ElapsedTime + ui.stats.EstimatedRemain
	}
}

// ShowRealtimeStats 显示实时统计信息
func (ui *AdvancedProgressUI) ShowRealtimeStats() {
	if !ui.showDetailed {
		return
	}

	ui.mutex.RLock()
	defer ui.mutex.RUnlock()

	// 这里可以添加额外的实时统计显示逻辑
	// 例如在另一个goroutine中定期输出详细统计
}

// Stop 停止进度UI
func (ui *AdvancedProgressUI) Stop() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	ui.isActive = false

	// 停止统计显示更新器
	if ui.statsDisplay.updateTicker != nil {
		ui.statsDisplay.updateTicker.Stop()
	}

	// 等待mpb容器完成
	ui.container.Wait()

	ui.logger.Info("进度UI已停止")
}

// GetStats 获取当前统计信息
func (ui *AdvancedProgressUI) GetStats() *UIStats {
	ui.mutex.RLock()
	defer ui.mutex.RUnlock()

	// 返回统计信息的副本
	statsCopy := *ui.stats
	statsCopy.QualityDistrib = make(map[types.QualityLevel]int64)
	for k, v := range ui.stats.QualityDistrib {
		statsCopy.QualityDistrib[k] = v
	}

	return &statsCopy
}

// IsActive 检查UI是否处于活跃状态
func (ui *AdvancedProgressUI) IsActive() bool {
	ui.mutex.RLock()
	defer ui.mutex.RUnlock()
	return ui.isActive
}

// GetCurrentPhase 获取当前处理阶段
func (ui *AdvancedProgressUI) GetCurrentPhase() ProcessingPhase {
	ui.mutex.RLock()
	defer ui.mutex.RUnlock()
	return ui.currentPhase
}
