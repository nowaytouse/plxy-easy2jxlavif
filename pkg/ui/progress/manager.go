package progress

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"pixly/pkg/core/types"

	"github.com/fatih/color"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"go.uber.org/zap"
)

// ProgressType 进度类型
type ProgressType int

const (
	ProgressTypeScan ProgressType = iota
	ProgressTypeAssessment
	ProgressTypeConversion
)

func (pt ProgressType) String() string {
	switch pt {
	case ProgressTypeScan:
		return "扫描"
	case ProgressTypeAssessment:
		return "评估"
	case ProgressTypeConversion:
		return "转换"
	default:
		return "未知"
	}
}

// ProgressManager 统一进度管理器
type ProgressManager struct {
	container *mpb.Progress
	bars      map[ProgressType]*mpb.Bar
	mutex     sync.RWMutex
	paused    bool
	logger    *zap.Logger
	ctx       context.Context
	cancel    context.CancelFunc

	// 统计信息
	stats *ProgressStats
}

// ProgressStats 进度统计
type ProgressStats struct {
	ScanProgress       int `json:"scan_progress"`
	AssessmentProgress int `json:"assessment_progress"`
	ConversionProgress int `json:"conversion_progress"`
	TotalFound         int `json:"total_found"`
	TotalToAssess      int `json:"total_to_assess"`
	TotalToConvert     int `json:"total_to_convert"`

	// 处理结果统计
	SuccessCount   int `json:"success_count"`
	SkippedCount   int `json:"skipped_count"`
	FailedCount    int `json:"failed_count"`
	CorruptedCount int `json:"corrupted_count"`

	// 实时统计信息
	StartTime         time.Time     `json:"start_time"`
	CurrentSpeed      float64       `json:"current_speed"` // 文件/秒
	AverageSpeed      float64       `json:"average_speed"` // 文件/秒
	EstimatedTimeLeft time.Duration `json:"estimated_time_left"`
	TotalSpaceSaved   int64         `json:"total_space_saved"`
	LastUpdateTime    time.Time     `json:"last_update_time"`
	ProcessingRate    int           `json:"processing_rate"` // 每分钟处理文件数
}

// ToJSON 将进度统计转换为JSON格式
func (ps *ProgressStats) ToJSON() ([]byte, error) {
	return json.MarshalIndent(ps, "", "  ")
}

// FromJSON 从JSON数据恢复进度统计
func (ps *ProgressStats) FromJSON(data []byte) error {
	return json.Unmarshal(data, ps)
}

// NewProgressManager 创建新的进度管理器
func NewProgressManager(logger *zap.Logger) *ProgressManager {
	ctx, cancel := context.WithCancel(context.Background())

	container := mpb.NewWithContext(ctx,
		mpb.WithWidth(80),
		mpb.WithRefreshRate(100*time.Millisecond),
		mpb.WithOutput(color.Output),
	)

	now := time.Now()
	return &ProgressManager{
		container: container,
		bars:      make(map[ProgressType]*mpb.Bar),
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		stats: &ProgressStats{
			StartTime:      now,
			LastUpdateTime: now,
		},
	}
}

// CreateScanProgress 创建扫描进度条
func (pm *ProgressManager) CreateScanProgress(total int) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.paused {
		return
	}

	pm.stats.TotalFound = total

	bar := pm.container.New(int64(total),
		mpb.BarStyle().Lbound("").Filler("▓").Tip("▓").Padding("░").Rbound(""),
		mpb.PrependDecorators(
			decor.Name("🔍 扫描文件: ", decor.WC{W: 12}),
			decor.CountersNoUnit("%d/%d", decor.WC{W: 10}),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 5}),
			decor.Name(" "),
			decor.AverageETA(decor.ET_STYLE_GO, decor.WC{W: 6}),
		),
	)

	pm.bars[ProgressTypeScan] = bar
}

// CreateAssessmentProgress 创建评估进度条
func (pm *ProgressManager) CreateAssessmentProgress(total int) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.paused {
		return
	}

	pm.stats.TotalToAssess = total

	bar := pm.container.New(int64(total),
		mpb.BarStyle().Lbound("").Filler("▓").Tip("▓").Padding("░").Rbound(""),
		mpb.PrependDecorators(
			decor.Name("🧠 品质评估: ", decor.WC{W: 12}),
			decor.CountersNoUnit("%d/%d", decor.WC{W: 10}),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 5}),
			decor.Name(" "),
			decor.AverageETA(decor.ET_STYLE_GO, decor.WC{W: 6}),
		),
	)

	pm.bars[ProgressTypeAssessment] = bar
}

// CreateConversionProgress 创建转换进度条
func (pm *ProgressManager) CreateConversionProgress(total int) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.paused {
		return
	}

	pm.stats.TotalToConvert = total

	bar := pm.container.New(int64(total),
		mpb.BarStyle().Lbound("").Filler("▓").Tip("▓").Padding("░").Rbound(""),
		mpb.PrependDecorators(
			decor.Name("⚡ 转换处理: ", decor.WC{W: 12}),
			decor.CountersNoUnit("%d/%d", decor.WC{W: 10}),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 5}),
			decor.Name(" "),
			decor.AverageETA(decor.ET_STYLE_GO, decor.WC{W: 6}),
		),
	)

	pm.bars[ProgressTypeConversion] = bar
}

// UpdateProgress 更新进度
func (pm *ProgressManager) UpdateProgress(progressType ProgressType, increment int) {
	pm.mutex.RLock()
	paused := pm.paused
	pm.mutex.RUnlock()

	if paused {
		return
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	bar, exists := pm.bars[progressType]
	if !exists || bar == nil {
		return
	}

	// 更新统计信息
	switch progressType {
	case ProgressTypeScan:
		pm.stats.ScanProgress += increment
	case ProgressTypeAssessment:
		pm.stats.AssessmentProgress += increment
	case ProgressTypeConversion:
		pm.stats.ConversionProgress += increment
		// 仅在转换阶段计算速度和预估时间
		pm.updateSpeedAndETA()
	}

	// 更新进度条
	bar.IncrBy(increment)

	// 记录最后更新时间
	pm.stats.LastUpdateTime = time.Now()
}

// UpdateResult 更新处理结果统计
func (pm *ProgressManager) UpdateResult(result *types.ProcessingResult) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if result.Success {
		pm.stats.SuccessCount++
		// 统计节省的空间
		spaceSaved := result.OriginalSize - result.NewSize
		pm.stats.TotalSpaceSaved += spaceSaved
	} else {
		pm.stats.FailedCount++
	}

	// 更新实时统计
	pm.updateRealTimeStats()
}

// updateSpeedAndETA 更新速度和预估时间 - 新增方法
func (pm *ProgressManager) updateSpeedAndETA() {
	now := time.Now()
	elapsedTime := now.Sub(pm.stats.StartTime)

	if elapsedTime.Seconds() < 1 {
		return // 避免除以零
	}

	totalProcessed := pm.stats.ConversionProgress
	if totalProcessed > 0 {
		// 计算平均速度 (文件/秒)
		pm.stats.AverageSpeed = float64(totalProcessed) / elapsedTime.Seconds()

		// 计算当前速度 (基于最近10秒的处理速度)
		timeSinceLastUpdate := now.Sub(pm.stats.LastUpdateTime)
		if timeSinceLastUpdate.Seconds() > 0 {
			// 取近期速度和平均速度的加权平均
			recentSpeed := 1.0 / timeSinceLastUpdate.Seconds()
			pm.stats.CurrentSpeed = 0.7*pm.stats.AverageSpeed + 0.3*recentSpeed
		}

		// 计算每分钟处理率
		pm.stats.ProcessingRate = int(pm.stats.AverageSpeed * 60)

		// 预估剩余时间
		remainingFiles := pm.stats.TotalToConvert - totalProcessed
		if remainingFiles > 0 && pm.stats.CurrentSpeed > 0 {
			pm.stats.EstimatedTimeLeft = time.Duration(float64(remainingFiles)/pm.stats.CurrentSpeed) * time.Second
		}
	}
}

// updateRealTimeStats 更新实时统计信息 - 新增方法
func (pm *ProgressManager) updateRealTimeStats() {
	// 更新最后更新时间
	pm.stats.LastUpdateTime = time.Now()

	// 重新计算速度和ETA
	pm.updateSpeedAndETA()
}

// UpdateSkipped 更新跳过计数
func (pm *ProgressManager) UpdateSkipped(count int) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.stats.SkippedCount += count
}

// UpdateCorrupted 更新损坏文件计数
func (pm *ProgressManager) UpdateCorrupted(count int) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.stats.CorruptedCount += count
}

// Pause 暂停所有进度显示
func (pm *ProgressManager) Pause() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.paused = true
	pm.logger.Debug("进度显示已暂停")
}

// Resume 恢复所有进度显示
func (pm *ProgressManager) Resume() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.paused = false
	pm.logger.Debug("进度显示已恢复")
}

// IsPaused 检查是否暂停
func (pm *ProgressManager) IsPaused() bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.paused
}

// CompleteProgress 完成指定类型的进度条
func (pm *ProgressManager) CompleteProgress(progressType ProgressType) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	bar, exists := pm.bars[progressType]
	if !exists || bar == nil {
		return
	}

	// 将进度条设置为完成状态
	bar.SetTotal(bar.Current(), true)
	pm.logger.Debug("进度条已完成", zap.String("type", progressType.String()))
}

// Wait 等待所有进度条完成
func (pm *ProgressManager) Wait() {
	pm.container.Wait()
}

// Stop 停止进度管理器
func (pm *ProgressManager) Stop() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// 完成所有进度条
	for progressType, bar := range pm.bars {
		if bar != nil {
			bar.SetTotal(bar.Current(), true)
			pm.logger.Debug("强制完成进度条", zap.String("type", progressType.String()))
		}
	}

	// 取消context
	if pm.cancel != nil {
		pm.cancel()
	}

	pm.logger.Debug("进度管理器已停止")
}

// GetStats 获取进度统计信息
func (pm *ProgressManager) GetStats() *ProgressStats {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// 创建副本以避免并发问题
	statsCopy := *pm.stats
	return &statsCopy
}

// ShowRealTimeStats 显示实时统计信息
func (pm *ProgressManager) ShowRealTimeStats() {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if pm.paused {
		return
	}

	stats := pm.stats

	// 计算整体进度百分比
	var overallProgress float64
	if stats.TotalToConvert > 0 {
		overallProgress = float64(stats.ConversionProgress) / float64(stats.TotalToConvert) * 100
	}

	// 格式化空间节省
	spaceSavedStr := formatBytes(stats.TotalSpaceSaved)

	// 格式化ETA
	etaStr := "--:--"
	if stats.EstimatedTimeLeft > 0 {
		etaStr = formatDuration(stats.EstimatedTimeLeft)
	}

	// 显示完整的实时统计信息
	fmt.Printf("\r📊 进度: %.1f%% │ ✅ 成功: %d │ ❌ 失败: %d │ ⏭️ 跳过: %d │ 🚫 损坏: %d │ 💰 节省: %s │ ⏱️ ETA: %s",
		overallProgress,
		stats.SuccessCount,
		stats.FailedCount,
		stats.SkippedCount,
		stats.CorruptedCount,
		spaceSavedStr,
		etaStr,
	)
}

// ShowDetailedRealTimeStats 显示详细的实时统计信息 - 新增方法
func (pm *ProgressManager) ShowDetailedRealTimeStats() {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if pm.paused {
		return
	}

	stats := pm.stats

	// 计算运行时间
	elapsedTime := time.Since(stats.StartTime)

	// 格式化显示
	fmt.Println("\n📊 实时统计 " + strings.Repeat("━", 40))
	fmt.Printf("📋 处理进度: %d/%d 文件\n", stats.ConversionProgress, stats.TotalToConvert)
	fmt.Printf("⏱️ 运行时间: %s\n", formatDuration(elapsedTime))
	fmt.Printf("⚡ 平均速度: %.2f 文件/秒\n", stats.AverageSpeed)
	fmt.Printf("🚀 当前速度: %.2f 文件/秒\n", stats.CurrentSpeed)
	fmt.Printf("💰 节省空间: %s\n", formatBytes(stats.TotalSpaceSaved))

	if stats.EstimatedTimeLeft > 0 {
		fmt.Printf("🕰️ 预估剩余: %s\n", formatDuration(stats.EstimatedTimeLeft))
	}

	successRate := 0.0
	totalProcessed := stats.SuccessCount + stats.FailedCount + stats.SkippedCount
	if totalProcessed > 0 {
		successRate = float64(stats.SuccessCount) / float64(totalProcessed) * 100
	}
	fmt.Printf("🏆 成功率: %.1f%% (%d/%d)\n", successRate, stats.SuccessCount, totalProcessed)
	fmt.Println(strings.Repeat("━", 50))
}

// GenerateReport 生成最终报告
func (pm *ProgressManager) GenerateReport(stats *types.Statistics) string {
	var report strings.Builder

	report.WriteString("\n" + color.New(color.Bold).Sprint("📊 处理统计报告") + "\n")
	report.WriteString(strings.Repeat("=", 50) + "\n")

	// 文件处理统计
	report.WriteString(fmt.Sprintf("📁 总文件数: %d\n", stats.TotalFiles))
	report.WriteString(fmt.Sprintf("✅ 成功处理: %d\n", stats.SuccessFiles))
	report.WriteString(fmt.Sprintf("⏭️ 跳过文件: %d\n", stats.SkippedFiles))
	report.WriteString(fmt.Sprintf("❌ 处理失败: %d\n", stats.FailedFiles))
	report.WriteString(fmt.Sprintf("🚫 损坏文件: %d\n", stats.CorruptedFiles))

	// 空间统计
	if stats.TotalSpaceSaved > 0 {
		savedGB := float64(stats.TotalSpaceSaved) / (1024 * 1024 * 1024)
		report.WriteString(fmt.Sprintf("💰 节省空间: %.2f GB\n", savedGB))
	} else if stats.TotalSpaceSaved < 0 {
		increasedGB := float64(-stats.TotalSpaceSaved) / (1024 * 1024 * 1024)
		report.WriteString(fmt.Sprintf("⬆️ 空间增加: %.2f GB\n", increasedGB))
	}

	// 处理时间
	if stats.ProcessingTime > 0 {
		report.WriteString(fmt.Sprintf("⏱️ 处理耗时: %v\n", stats.ProcessingTime.Round(time.Second)))

		if stats.SuccessFiles > 0 {
			avgTime := stats.ProcessingTime / time.Duration(stats.SuccessFiles)
			report.WriteString(fmt.Sprintf("📈 平均速度: %v/文件\n", avgTime.Round(time.Millisecond)))
		}
	}

	// 品质分布统计
	if len(stats.QualityStats) > 0 {
		report.WriteString("\n📊 品质分布:\n")
		for quality, count := range stats.QualityStats {
			if count > 0 {
				report.WriteString(fmt.Sprintf("   %s: %d 个文件\n", quality.String(), count))
			}
		}
	}

	// 格式分布统计
	if len(stats.FormatStats) > 0 {
		report.WriteString("\n📄 格式分布:\n")
		for format, count := range stats.FormatStats {
			if count > 0 {
				report.WriteString(fmt.Sprintf("   %s: %d 个文件\n", format, count))
			}
		}
	}

	report.WriteString(strings.Repeat("=", 50) + "\n")

	return report.String()
}

// 全局进度管理器实例
var (
	globalProgressManager *ProgressManager
	globalProgressMutex   sync.RWMutex
)

// GetGlobalProgressManager 获取全局进度管理器
func GetGlobalProgressManager() *ProgressManager {
	globalProgressMutex.RLock()
	defer globalProgressMutex.RUnlock()
	return globalProgressManager
}

// SetGlobalProgressManager 设置全局进度管理器
func SetGlobalProgressManager(pm *ProgressManager) {
	globalProgressMutex.Lock()
	defer globalProgressMutex.Unlock()

	// 如果已有管理器，先停止它
	if globalProgressManager != nil {
		globalProgressManager.Stop()
	}

	globalProgressManager = pm
}

// PauseAllProgress 暂停所有进度显示
func PauseAllProgress() {
	if pm := GetGlobalProgressManager(); pm != nil {
		pm.Pause()
	}
}

// ResumeAllProgress 恢复所有进度显示
func ResumeAllProgress() {
	if pm := GetGlobalProgressManager(); pm != nil {
		pm.Resume()
	}
}

// PauseAll 暂停所有进度显示（实例方法）
func (pm *ProgressManager) PauseAll() {
	pm.Pause()
}

// ResumeAll 恢复所有进度显示（实例方法）
func (pm *ProgressManager) ResumeAll() {
	pm.Resume()
}

// formatBytes 格式化字节数为可读字符串 - 新增方法
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration 格式化时间间隔为可读字符串 - 新增方法
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "--:--"
	}

	totalSeconds := int(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// StartRealTimeDisplay 启动实时显示系统 - 新增方法
func (pm *ProgressManager) StartRealTimeDisplay() {
	go func() {
		ticker := time.NewTicker(2 * time.Second) // 每2秒更新一次
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !pm.IsPaused() && pm.stats.ConversionProgress > 0 {
					pm.ShowRealTimeStats()
				}
			case <-pm.ctx.Done():
				return
			}
		}
	}()
}
