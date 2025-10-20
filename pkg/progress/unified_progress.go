package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"go.uber.org/zap"
)

// UnifiedProgress 统一进度条系统 - README要求的mpb/v8精确实时进度显示
//
// 核心功能：
//   - 替换现有的3套混乱进度条实现
//   - 使用mpb/v8提供高性能实时进度显示
//   - 支持7步标准化流程的进度跟踪
//   - 清晰的状态管理和用户体验
//
// 设计原则：
//   - 简洁明了：统一的API接口
//   - 性能优化：最小化资源占用
//   - 用户友好：直观的进度显示
//   - 状态清晰：精确的阶段追踪
type UnifiedProgress struct {
	logger    *zap.Logger
	container *mpb.Progress
	mutex     sync.RWMutex

	// 7步标准化流程的进度条
	stepBars map[ProcessStep]*mpb.Bar

	// 统计信息
	stats       *ProgressStats
	startTime   time.Time
	isActive    bool
	currentStep ProcessStep
}

// ProcessStep 处理步骤 - 对应README的7步标准化流程
type ProcessStep int

const (
	StepInput      ProcessStep = iota // 1. 启动与输入
	StepSecurity                      // 2. 安全检查
	StepScan                          // 3. 统一扫描与分析
	StepDecision                      // 4. 问题文件决策
	StepModeSelect                    // 5. 处理模式选择
	StepProcessing                    // 6. 核心处理
	StepReport                        // 7. 统计报告
)

func (ps ProcessStep) String() string {
	switch ps {
	case StepInput:
		return "📁 启动与输入"
	case StepSecurity:
		return "🔒 安全检查"
	case StepScan:
		return "🔍 统一扫描"
	case StepDecision:
		return "🚨 批量决策"
	case StepModeSelect:
		return "⚙️ 模式选择"
	case StepProcessing:
		return "⚡ 核心处理"
	case StepReport:
		return "📊 生成报告"
	default:
		return "未知步骤"
	}
}

// ProgressStats 进度统计信息
type ProgressStats struct {
	// 基础统计
	TotalFiles      int64 `json:"total_files"`
	ProcessedFiles  int64 `json:"processed_files"`
	SuccessfulFiles int64 `json:"successful_files"`
	FailedFiles     int64 `json:"failed_files"`
	SkippedFiles    int64 `json:"skipped_files"`

	// 时间统计
	StartTime      time.Time     `json:"start_time"`
	CurrentTime    time.Time     `json:"current_time"`
	ElapsedTime    time.Duration `json:"elapsed_time"`
	EstimatedTotal time.Duration `json:"estimated_total"`

	// 速度统计
	FilesPerSecond  float64 `json:"files_per_second"`
	MegaBytesPerSec float64 `json:"megabytes_per_second"`

	// 空间统计
	TotalSizeProcessed int64 `json:"total_size_processed"`
	SpaceSaved         int64 `json:"space_saved"`
	SpaceUsed          int64 `json:"space_used"`

	// 当前状态
	CurrentStep     ProcessStep `json:"current_step"`
	StepProgress    float64     `json:"step_progress"`
	OverallProgress float64     `json:"overall_progress"`
}

// NewUnifiedProgress 创建新的统一进度条系统
func NewUnifiedProgress(logger *zap.Logger) *UnifiedProgress {
	// 创建mpb容器，使用README要求的配置
	container := mpb.New(
		mpb.WithWidth(80), // 合适的宽度
		mpb.WithRefreshRate(100*time.Millisecond), // 实时更新频率
	)

	up := &UnifiedProgress{
		logger:      logger,
		container:   container,
		stepBars:    make(map[ProcessStep]*mpb.Bar),
		stats:       &ProgressStats{},
		startTime:   time.Now(),
		isActive:    true,
		currentStep: StepInput,
	}

	up.stats.StartTime = up.startTime
	up.stats.CurrentStep = StepInput

	logger.Info("统一进度条系统初始化完成",
		zap.String("version", "mpb/v8"),
		zap.Bool("active", up.isActive))

	return up
}

// StartStep 开始新的处理步骤
func (up *UnifiedProgress) StartStep(step ProcessStep, totalItems int64, description string) {
	up.mutex.Lock()
	defer up.mutex.Unlock()

	up.currentStep = step
	up.stats.CurrentStep = step

	// 如果描述为空，使用默认描述
	if description == "" {
		description = step.String()
	}

	// 创建该步骤的进度条
	bar := up.container.AddBar(totalItems,
		mpb.PrependDecorators(
			decor.Name(fmt.Sprintf("%s: ", description), decor.WC{W: 15}),
			decor.CountersNoUnit("%d/%d", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 5}),
			decor.Name(" | "),
			decor.EwmaETA(decor.ET_STYLE_GO, 60, decor.WC{W: 6}),
			decor.Name(" | "),
			decor.EwmaSpeed(decor.SizeB1024(0), "%.1f/s", 60),
		),
	)

	up.stepBars[step] = bar

	up.logger.Info("开始处理步骤",
		zap.String("step", step.String()),
		zap.Int64("total_items", totalItems),
		zap.String("description", description))
}

// UpdateStep 更新当前步骤的进度
func (up *UnifiedProgress) UpdateStep(increment int64, success bool) {
	up.mutex.Lock()
	defer up.mutex.Unlock()

	bar, exists := up.stepBars[up.currentStep]
	if !exists {
		up.logger.Warn("尝试更新不存在的步骤进度条",
			zap.String("step", up.currentStep.String()))
		return
	}

	// 更新进度条
	bar.IncrBy(int(increment))

	// 更新统计信息
	up.stats.ProcessedFiles += increment
	if success {
		up.stats.SuccessfulFiles += increment
	} else {
		up.stats.FailedFiles += increment
	}

	// 计算实时统计
	up.updateRealTimeStats()
}

// SkipItems 跳过项目
func (up *UnifiedProgress) SkipItems(count int64) {
	up.mutex.Lock()
	defer up.mutex.Unlock()

	bar, exists := up.stepBars[up.currentStep]
	if exists {
		bar.IncrBy(int(count))
	}

	up.stats.SkippedFiles += count
	up.stats.ProcessedFiles += count
	up.updateRealTimeStats()

	up.logger.Debug("跳过项目",
		zap.Int64("count", count),
		zap.String("step", up.currentStep.String()))
}

// CompleteStep 完成当前步骤
func (up *UnifiedProgress) CompleteStep() {
	up.mutex.Lock()
	defer up.mutex.Unlock()

	bar, exists := up.stepBars[up.currentStep]
	if exists {
		// 将进度条设置为完成状态
		bar.SetTotal(bar.Current(), true)
	}

	up.logger.Info("步骤完成",
		zap.String("step", up.currentStep.String()),
		zap.Int64("processed", up.stats.ProcessedFiles))
}

// SetTotalFiles 设置总文件数量
func (up *UnifiedProgress) SetTotalFiles(total int64) {
	up.mutex.Lock()
	defer up.mutex.Unlock()

	up.stats.TotalFiles = total
	up.updateRealTimeStats()

	up.logger.Info("设置总文件数量", zap.Int64("total", total))
}

// UpdateSpaceStats 更新空间统计
func (up *UnifiedProgress) UpdateSpaceStats(sizeProcessed, spaceSaved, spaceUsed int64) {
	up.mutex.Lock()
	defer up.mutex.Unlock()

	up.stats.TotalSizeProcessed += sizeProcessed
	up.stats.SpaceSaved += spaceSaved
	up.stats.SpaceUsed += spaceUsed

	up.updateRealTimeStats()
}

// GetCurrentStats 获取当前统计信息
func (up *UnifiedProgress) GetCurrentStats() *ProgressStats {
	up.mutex.RLock()
	defer up.mutex.RUnlock()

	// 返回统计信息的副本
	statsCopy := *up.stats
	return &statsCopy
}

// ShowSummary 显示处理摘要
func (up *UnifiedProgress) ShowSummary() {
	up.mutex.RLock()
	defer up.mutex.RUnlock()

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("📊 处理完成摘要\n")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("📁 总文件数: %d\n", up.stats.TotalFiles)
	fmt.Printf("✅ 成功处理: %d\n", up.stats.SuccessfulFiles)
	fmt.Printf("❌ 处理失败: %d\n", up.stats.FailedFiles)
	fmt.Printf("⏭️ 跳过文件: %d\n", up.stats.SkippedFiles)
	fmt.Printf("⏱️ 总耗时: %v\n", up.stats.ElapsedTime)
	fmt.Printf("⚡ 平均速度: %.2f 文件/秒\n", up.stats.FilesPerSecond)

	if up.stats.SpaceSaved > 0 {
		fmt.Printf("💰 节省空间: %.2f MB\n", float64(up.stats.SpaceSaved)/(1024*1024))
	}
	if up.stats.SpaceUsed > 0 {
		fmt.Printf("📈 使用空间: %.2f MB\n", float64(up.stats.SpaceUsed)/(1024*1024))
	}

	fmt.Println(strings.Repeat("=", 80))
}

// Wait 等待所有进度条完成
func (up *UnifiedProgress) Wait() {
	up.container.Wait()
}

// Stop 停止进度条系统
func (up *UnifiedProgress) Stop() {
	up.mutex.Lock()
	defer up.mutex.Unlock()

	up.isActive = false

	// 完成所有未完成的进度条
	for step, bar := range up.stepBars {
		if bar != nil {
			bar.SetTotal(bar.Current(), true)
			up.logger.Debug("强制完成进度条", zap.String("step", step.String()))
		}
	}

	up.logger.Info("统一进度条系统已停止")
}

// IsActive 检查是否处于活跃状态
func (up *UnifiedProgress) IsActive() bool {
	up.mutex.RLock()
	defer up.mutex.RUnlock()
	return up.isActive
}

// updateRealTimeStats 更新实时统计信息（内部方法，需要调用方持有锁）
func (up *UnifiedProgress) updateRealTimeStats() {
	now := time.Now()
	up.stats.CurrentTime = now
	up.stats.ElapsedTime = now.Sub(up.stats.StartTime)

	// 计算处理速度
	if up.stats.ElapsedTime.Seconds() > 0 {
		up.stats.FilesPerSecond = float64(up.stats.ProcessedFiles) / up.stats.ElapsedTime.Seconds()
		up.stats.MegaBytesPerSec = float64(up.stats.TotalSizeProcessed) / (1024 * 1024) / up.stats.ElapsedTime.Seconds()
	}

	// 计算总体进度
	if up.stats.TotalFiles > 0 {
		up.stats.OverallProgress = float64(up.stats.ProcessedFiles) / float64(up.stats.TotalFiles) * 100
	}

	// 估算剩余时间
	if up.stats.FilesPerSecond > 0 && up.stats.TotalFiles > up.stats.ProcessedFiles {
		remainingFiles := up.stats.TotalFiles - up.stats.ProcessedFiles
		remainingSeconds := float64(remainingFiles) / up.stats.FilesPerSecond
		up.stats.EstimatedTotal = up.stats.ElapsedTime + time.Duration(remainingSeconds)*time.Second
	}
}
