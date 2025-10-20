package concurrency

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"time"

	"pixly/pkg/core/types"

	"go.uber.org/zap"
)

// SmartConcurrencyManager 智能并发管理器
type SmartConcurrencyManager struct {
	logger *zap.Logger
	mutex  sync.RWMutex

	// 核心配置
	maxWorkers     int // 最大工作协程数
	minWorkers     int // 最小工作协程数
	currentWorkers int // 当前活跃工作协程数

	// 复杂度阈值配置
	lowComplexityThreshold  float64 // 低复杂度阈值
	highComplexityThreshold float64 // 高复杂度阈值

	// 内存监控配置
	memoryThreshold     float64       // 内存使用阈值（百分比）
	memoryCheckInterval time.Duration // 内存检查间隔

	// 动态调整参数
	scalingFactor      float64       // 并发调整因子
	backoffFactor      float64       // 回退因子
	adjustmentInterval time.Duration // 调整间隔

	// 运行时状态
	activeJobs           map[string]*JobContext // 活跃任务上下文
	jobComplexityHistory []float64              // 任务复杂度历史
	performanceMetrics   *PerformanceMetrics    // 性能指标

	// 控制通道
	jobQueue         chan *JobRequest // 任务队列
	resultQueue      chan *JobResult  // 结果队列
	adjustmentTicker *time.Ticker     // 调整定时器
	memoryTicker     *time.Ticker     // 内存监控定时器
	shutdownChan     chan struct{}    // 关闭信号

	// 统计信息
	stats *ConcurrencyStats // 并发统计
}

// JobContext 任务上下文
type JobContext struct {
	ID              string                 // 任务ID
	MediaInfo       *types.MediaInfo       // 媒体信息
	ComplexityScore float64                // 复杂度分数
	EstimatedMemory int64                  // 预估内存使用
	StartTime       time.Time              // 开始时间
	Priority        JobPriority            // 任务优先级
	ProcessingMode  types.AppMode          // 处理模式
	Metadata        map[string]interface{} // 扩展元数据
}

// JobRequest 任务请求
type JobRequest struct {
	Context    *JobContext                              // 任务上下文
	Handler    func(context.Context, *JobContext) error // 处理函数
	ResultChan chan *JobResult                          // 结果通道
}

// JobResult 任务结果
type JobResult struct {
	JobID          string                  // 任务ID
	Success        bool                    // 是否成功
	Result         *types.ProcessingResult // 处理结果
	Error          error                   // 错误信息
	Duration       time.Duration           // 处理时长
	MemoryUsed     int64                   // 实际内存使用
	ComplexityUsed float64                 // 实际复杂度
}

// JobPriority 任务优先级
type JobPriority int

const (
	PriorityLow JobPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	AverageProcessingTime time.Duration // 平均处理时间
	ThroughputPerSecond   float64       // 每秒吞吐量
	MemoryEfficiency      float64       // 内存效率
	ComplexityAccuracy    float64       // 复杂度预测准确性
	ResourceUtilization   float64       // 资源利用率
	LastUpdateTime        time.Time     // 最后更新时间
}

// ConcurrencyStats 并发统计
type ConcurrencyStats struct {
	TotalJobsProcessed int64     // 总处理任务数
	SuccessfulJobs     int64     // 成功任务数
	FailedJobs         int64     // 失败任务数
	AverageWorkers     float64   // 平均工作协程数
	PeakWorkers        int       // 峰值工作协程数
	TotalMemoryUsed    int64     // 总内存使用量
	WorkerAdjustments  int64     // 工作协程调整次数
	MemoryLimitHits    int64     // 内存限制触发次数
	StartTime          time.Time // 启动时间
}

// NewSmartConcurrencyManager 创建智能并发管理器
func NewSmartConcurrencyManager(logger *zap.Logger) *SmartConcurrencyManager {
	cpuCores := runtime.NumCPU()

	manager := &SmartConcurrencyManager{
		logger: logger,

		// README要求：扫描阶段CPU核心数x2，处理阶段动态调整
		maxWorkers:     cpuCores * 4, // 最大4倍CPU核心数
		minWorkers:     cpuCores,     // 最小等于CPU核心数
		currentWorkers: cpuCores * 2, // 初始2倍CPU核心数

		// 复杂度阈值
		lowComplexityThreshold:  30.0, // 低复杂度：小文件、简单格式
		highComplexityThreshold: 80.0, // 高复杂度：大文件、复杂转换

		// 内存监控 - README要求：防止内存溢出被系统强杀
		memoryThreshold:     75.0,            // 75%内存使用阈值
		memoryCheckInterval: 2 * time.Second, // 每2秒检查内存

		// 动态调整参数
		scalingFactor:      1.2,              // 20%调整幅度
		backoffFactor:      0.8,              // 20%回退幅度
		adjustmentInterval: 10 * time.Second, // 每10秒调整一次

		// 运行时状态
		activeJobs:           make(map[string]*JobContext),
		jobComplexityHistory: make([]float64, 0, 100), // 保留最近100个任务复杂度
		performanceMetrics:   &PerformanceMetrics{},

		// 队列初始化
		jobQueue:     make(chan *JobRequest, cpuCores*10), // 10倍缓冲
		resultQueue:  make(chan *JobResult, cpuCores*5),   // 5倍缓冲
		shutdownChan: make(chan struct{}),

		// 统计初始化
		stats: &ConcurrencyStats{
			StartTime: time.Now(),
		},
	}

	logger.Info("智能并发管理器初始化完成",
		zap.Int("cpu_cores", cpuCores),
		zap.Int("initial_workers", manager.currentWorkers),
		zap.Int("max_workers", manager.maxWorkers),
		zap.Float64("memory_threshold", manager.memoryThreshold))

	return manager
}

// Start 启动智能并发管理器
func (scm *SmartConcurrencyManager) Start(ctx context.Context) error {
	scm.logger.Info("启动智能并发管理器")

	// 启动内存监控
	scm.memoryTicker = time.NewTicker(scm.memoryCheckInterval)
	go scm.memoryMonitor(ctx)

	// 启动动态调整器
	scm.adjustmentTicker = time.NewTicker(scm.adjustmentInterval)
	go scm.dynamicAdjuster(ctx)

	// 启动工作协程池
	for i := 0; i < scm.currentWorkers; i++ {
		go scm.worker(ctx, i)
	}

	// 启动结果处理器
	go scm.resultProcessor(ctx)

	scm.logger.Info("智能并发管理器启动完成",
		zap.Int("active_workers", scm.currentWorkers))

	return nil
}

// CalculateFileComplexity 计算文件复杂度分数
func (scm *SmartConcurrencyManager) CalculateFileComplexity(mediaInfo *types.MediaInfo, mode types.AppMode) float64 {
	score := 0.0

	// 1. 文件大小权重 (40%)
	sizeScore := scm.calculateSizeComplexity(mediaInfo.Size)
	score += sizeScore * 0.4

	// 2. 格式复杂度权重 (30%)
	formatScore := scm.calculateFormatComplexity(mediaInfo.Path)
	score += formatScore * 0.3

	// 3. 处理模式权重 (20%)
	modeScore := scm.calculateModeComplexity(mode)
	score += modeScore * 0.2

	// 4. 品质等级权重 (10%)
	qualityScore := scm.calculateQualityComplexity(mediaInfo.Quality)
	score += qualityScore * 0.1

	// 确保分数在0-100范围内
	score = math.Max(0, math.Min(100, score))

	scm.logger.Debug("文件复杂度计算完成",
		zap.String("file", mediaInfo.Path),
		zap.Float64("complexity_score", score),
		zap.Float64("size_score", sizeScore),
		zap.Float64("format_score", formatScore))

	return score
}

// calculateSizeComplexity 计算文件大小复杂度
func (scm *SmartConcurrencyManager) calculateSizeComplexity(size int64) float64 {
	// 文件大小复杂度：基于对数增长
	sizeMB := float64(size) / (1024 * 1024)

	if sizeMB <= 1 {
		return 10.0 // 小文件，低复杂度
	} else if sizeMB <= 10 {
		return 20.0 + (sizeMB-1)*3.0 // 1-10MB线性增长
	} else if sizeMB <= 100 {
		return 47.0 + (sizeMB-10)*0.5 // 10-100MB缓慢增长
	} else {
		// 超大文件：对数增长，避免过度增长
		return 92.0 + math.Log10(sizeMB/100)*8.0
	}
}

// calculateFormatComplexity 计算格式复杂度
func (scm *SmartConcurrencyManager) calculateFormatComplexity(filePath string) float64 {
	ext := getFileExtension(filePath)

	// README要求的格式复杂度分类
	switch ext {
	case "jpg", "jpeg":
		return 20.0 // JPEG相对简单
	case "png":
		return 35.0 // PNG稍复杂
	case "webp":
		return 45.0 // WebP中等复杂度
	case "heif", "heic":
		return 65.0 // HEIF/HEIC较复杂
	case "tiff", "tif":
		return 55.0 // TIFF中高复杂度
	case "gif":
		return 40.0 // GIF动图处理
	case "mp4", "mov":
		return 80.0 // 视频格式高复杂度
	case "webm", "mkv":
		return 85.0 // 复杂视频容器
	case "avi":
		return 75.0 // 传统视频格式
	default:
		return 50.0 // 未知格式，中等复杂度
	}
}

// calculateModeComplexity 计算处理模式复杂度
func (scm *SmartConcurrencyManager) calculateModeComplexity(mode types.AppMode) float64 {
	switch mode {
	case types.ModeEmoji:
		return 30.0 // 表情包模式：激进压缩，相对简单
	case types.ModeAutoPlus:
		return 60.0 // 自动模式+：智能决策，中高复杂度
	case types.ModeQuality:
		return 80.0 // 品质模式：无损压缩，高复杂度
	default:
		return 50.0 // 默认中等复杂度
	}
}

// calculateQualityComplexity 计算品质复杂度
func (scm *SmartConcurrencyManager) calculateQualityComplexity(quality types.QualityLevel) float64 {
	switch quality {
	case types.QualityVeryLow:
		return 20.0 // 极低品质，处理相对简单
	case types.QualityLow:
		return 35.0
	case types.QualityMediumLow:
		return 45.0
	case types.QualityMediumHigh:
		return 60.0
	case types.QualityHigh:
		return 70.0
	case types.QualityVeryHigh:
		return 85.0 // 极高品质，需要精细处理
	default:
		return 50.0
	}
}

// SubmitJob 提交任务到智能并发处理队列
func (scm *SmartConcurrencyManager) SubmitJob(ctx context.Context, mediaInfo *types.MediaInfo, mode types.AppMode, handler func(context.Context, *JobContext) error) (*JobResult, error) {
	// 计算任务复杂度
	complexityScore := scm.CalculateFileComplexity(mediaInfo, mode)

	// 预估内存使用量
	estimatedMemory := scm.estimateMemoryUsage(mediaInfo, complexityScore)

	// 创建任务上下文
	jobContext := &JobContext{
		ID:              scm.generateJobID(),
		MediaInfo:       mediaInfo,
		ComplexityScore: complexityScore,
		EstimatedMemory: estimatedMemory,
		StartTime:       time.Now(),
		Priority:        scm.calculateJobPriority(complexityScore),
		ProcessingMode:  mode,
		Metadata:        make(map[string]interface{}),
	}

	// 创建结果通道
	resultChan := make(chan *JobResult, 1)

	// 创建任务请求
	jobRequest := &JobRequest{
		Context:    jobContext,
		Handler:    handler,
		ResultChan: resultChan,
	}

	// 检查内存限制
	if err := scm.checkMemoryAvailability(estimatedMemory); err != nil {
		return nil, fmt.Errorf("内存不足，无法提交任务: %w", err)
	}

	// 提交任务到队列
	select {
	case scm.jobQueue <- jobRequest:
		scm.logger.Debug("任务已提交到队列",
			zap.String("job_id", jobContext.ID),
			zap.Float64("complexity", complexityScore),
			zap.Int64("estimated_memory_mb", estimatedMemory/(1024*1024)))
	case <-ctx.Done():
		return nil, fmt.Errorf("上下文取消，任务提交失败")
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("任务提交超时")
	}

	// 等待处理结果
	select {
	case result := <-resultChan:
		// 更新复杂度历史
		scm.updateComplexityHistory(complexityScore, result.Duration)
		return result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("上下文取消，任务处理中断")
	}
}

// 辅助方法
func getFileExtension(filePath string) string {
	// 简化版本，实际应该使用filepath.Ext并做更复杂的处理
	parts := strings.Split(filePath, ".")
	if len(parts) > 1 {
		return strings.ToLower(parts[len(parts)-1])
	}
	return ""
}
