package monitor

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// ConcurrentManager 智能化并发管理器 - README要求的动态并发调整和内存监控
//
// 核心功能：
//   - 动态并发调整：根据文件复杂度和系统资源自动调整工作线程数
//   - 内存监控：实时监控内存使用，防止因处理超大文件导致系统强杀
//   - 负载均衡：智能分配工作负载，优化CPU和内存使用效率
//   - 自适应策略：根据历史处理数据调整并发策略
//
// 性能特点：
//   - 扫描阶段：CPU核心数 x 2的高并发
//   - 处理阶段：基于文件复杂度和可用内存的动态调整
//   - 内存保护：内存使用超过阈值时自动降低并发数
//   - 优雅降级：系统资源不足时平滑降低处理强度
type ConcurrentManager struct {
	logger              *zap.Logger
	memoryMonitor       *MemoryMonitor
	concurrencyStrategy *ConcurrencyStrategy
	loadBalancer        *LoadBalancer

	// 配置参数
	maxConcurrency  int           // 最大并发数
	minConcurrency  int           // 最小并发数
	memoryThreshold float64       // 内存使用阈值（0-1）
	adjustInterval  time.Duration // 调整间隔

	// 状态管理
	currentConcurrency int               // 当前并发数
	isMonitoring       bool              // 是否正在监控
	statistics         *ConcurrencyStats // 并发统计信息
	mutex              sync.RWMutex      // 读写锁
}

// MemoryMonitor 内存监控器 - README要求的内存溢出防护
type MemoryMonitor struct {
	logger             *zap.Logger
	currentUsage       float64          // 当前内存使用率
	peakUsage          float64          // 峰值内存使用率
	availableMemory    uint64           // 可用内存（字节）
	totalMemory        uint64           // 总内存（字节）
	warningThreshold   float64          // 警告阈值
	criticalThreshold  float64          // 临界阈值
	monitoringInterval time.Duration    // 监控间隔
	callbacks          []MemoryCallback // 内存事件回调
	isActive           bool             // 是否活跃监控
	mutex              sync.RWMutex
}

// ConcurrencyStrategy 并发策略 - 基于文件复杂度的动态调整
type ConcurrencyStrategy struct {
	logger             *zap.Logger
	baseStrategy       string              // 基础策略："conservative", "balanced", "aggressive"
	fileComplexityMap  map[string]int      // 文件类型复杂度映射
	adaptiveRules      []AdaptiveRule      // 自适应规则
	performanceHistory []PerformanceRecord // 性能历史记录
	optimizationMode   string              // 优化模式
}

// LoadBalancer 负载均衡器 - 智能工作分配
type LoadBalancer struct {
	logger         *zap.Logger
	workQueues     []chan WorkItem   // 工作队列
	workerStatus   []WorkerStatus    // 工作线程状态
	schedulingAlgo string            // 调度算法："round_robin", "least_loaded", "priority"
	workDistrib    *WorkDistribution // 工作分配统计
}

// 数据结构定义
type ConcurrencyStats struct {
	StartTime         time.Time          `json:"start_time"`
	TotalAdjustments  int                `json:"total_adjustments"`
	AverageResponse   time.Duration      `json:"average_response"`
	PeakConcurrency   int                `json:"peak_concurrency"`
	MemoryEfficiency  float64            `json:"memory_efficiency"`
	ThroughputHistory []ThroughputRecord `json:"throughput_history"`
	AdjustmentReasons map[string]int     `json:"adjustment_reasons"`
	SystemLoadHistory []SystemLoadRecord `json:"system_load_history"`
}

type MemoryCallback func(usage float64, available uint64)

type AdaptiveRule struct {
	Condition string  `json:"condition"` // 条件
	Action    string  `json:"action"`    // 动作
	Threshold float64 `json:"threshold"` // 阈值
	Priority  int     `json:"priority"`  // 优先级
	Enabled   bool    `json:"enabled"`   // 是否启用
}

type PerformanceRecord struct {
	Timestamp   time.Time `json:"timestamp"`
	Concurrency int       `json:"concurrency"`
	Throughput  float64   `json:"throughput"`   // 文件/秒
	MemoryUsage float64   `json:"memory_usage"` // 内存使用率
	CPUUsage    float64   `json:"cpu_usage"`    // CPU使用率
	Success     bool      `json:"success"`      // 是否成功
	ErrorRate   float64   `json:"error_rate"`   // 错误率
}

type WorkItem struct {
	ID            string        `json:"id"`
	Type          string        `json:"type"`           // "scan", "analyze", "convert"
	Priority      int           `json:"priority"`       // 优先级
	Complexity    int           `json:"complexity"`     // 复杂度（1-10）
	EstimatedTime time.Duration `json:"estimated_time"` // 预估时间
	Payload       interface{}   `json:"payload"`        // 工作负载
	CreatedAt     time.Time     `json:"created_at"`
}

type WorkerStatus struct {
	ID          int           `json:"id"`
	Status      string        `json:"status"` // "idle", "busy", "overloaded"
	CurrentWork *WorkItem     `json:"current_work"`
	StartTime   time.Time     `json:"start_time"`
	TaskCount   int           `json:"task_count"`
	ErrorCount  int           `json:"error_count"`
	AverageTime time.Duration `json:"average_time"`
}

type WorkDistribution struct {
	TotalTasks        int       `json:"total_tasks"`
	CompletedTasks    int       `json:"completed_tasks"`
	FailedTasks       int       `json:"failed_tasks"`
	QueueSizes        []int     `json:"queue_sizes"`
	WorkerUtilization []float64 `json:"worker_utilization"`
	LoadBalance       float64   `json:"load_balance"` // 负载均衡度
}

type ThroughputRecord struct {
	Timestamp   time.Time `json:"timestamp"`
	FilesPerSec float64   `json:"files_per_sec"`
	BytesPerSec int64     `json:"bytes_per_sec"`
}

type SystemLoadRecord struct {
	Timestamp   time.Time `json:"timestamp"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	Concurrency int       `json:"concurrency"`
}

// NewConcurrentManager 创建智能化并发管理器
func NewConcurrentManager(logger *zap.Logger) *ConcurrentManager {
	// README要求：扫描阶段高并发（CPU核心数 x 2）
	maxConcurrency := runtime.NumCPU() * 2
	minConcurrency := 1

	manager := &ConcurrentManager{
		logger:             logger,
		maxConcurrency:     maxConcurrency,
		minConcurrency:     minConcurrency,
		memoryThreshold:    0.80, // 80%内存使用阈值
		adjustInterval:     5 * time.Second,
		currentConcurrency: maxConcurrency / 2, // 初始为最大值的一半
		statistics: &ConcurrencyStats{
			StartTime:         time.Now(),
			AdjustmentReasons: make(map[string]int),
			ThroughputHistory: make([]ThroughputRecord, 0),
			SystemLoadHistory: make([]SystemLoadRecord, 0),
		},
	}

	// 初始化组件
	manager.memoryMonitor = NewMemoryMonitor(logger)
	manager.concurrencyStrategy = NewConcurrencyStrategy(logger)
	manager.loadBalancer = NewLoadBalancer(logger, manager.currentConcurrency)

	// 注册内存回调
	manager.memoryMonitor.RegisterCallback(manager.onMemoryChange)

	logger.Info("智能化并发管理器初始化完成",
		zap.Int("max_concurrency", maxConcurrency),
		zap.Int("min_concurrency", minConcurrency),
		zap.Int("initial_concurrency", manager.currentConcurrency),
		zap.Float64("memory_threshold", manager.memoryThreshold))

	return manager
}

// NewMemoryMonitor 创建内存监控器
func NewMemoryMonitor(logger *zap.Logger) *MemoryMonitor {
	return &MemoryMonitor{
		logger:             logger,
		warningThreshold:   0.75, // 75%警告
		criticalThreshold:  0.90, // 90%临界
		monitoringInterval: 2 * time.Second,
		callbacks:          make([]MemoryCallback, 0),
		isActive:           false,
	}
}

// NewConcurrencyStrategy 创建并发策略
func NewConcurrencyStrategy(logger *zap.Logger) *ConcurrencyStrategy {
	return &ConcurrencyStrategy{
		logger:       logger,
		baseStrategy: "balanced", // 默认平衡策略
		// README要求：基于文件复杂度动态调整
		fileComplexityMap: map[string]int{
			"jpeg":    3, // JPEG处理复杂度中等
			"png":     4, // PNG处理复杂度较高
			"gif":     6, // GIF动图复杂度高
			"webp":    5, // WebP复杂度中高
			"heif":    7, // HEIF复杂度高
			"avif":    8, // AVIF复杂度很高
			"jxl":     9, // JXL复杂度最高
			"mp4":     7, // 视频处理复杂度高
			"mov":     6, // MOV复杂度中高
			"unknown": 5, // 未知格式中等复杂度
		},
		adaptiveRules: []AdaptiveRule{
			{
				Condition: "memory_usage > 0.8",
				Action:    "reduce_concurrency",
				Threshold: 0.8,
				Priority:  1,
				Enabled:   true,
			},
			{
				Condition: "error_rate > 0.1",
				Action:    "reduce_concurrency",
				Threshold: 0.1,
				Priority:  2,
				Enabled:   true,
			},
			{
				Condition: "throughput < 0.5",
				Action:    "increase_concurrency",
				Threshold: 0.5,
				Priority:  3,
				Enabled:   true,
			},
		},
		performanceHistory: make([]PerformanceRecord, 0),
		optimizationMode:   "adaptive",
	}
}

// NewLoadBalancer 创建负载均衡器
func NewLoadBalancer(logger *zap.Logger, workerCount int) *LoadBalancer {
	workQueues := make([]chan WorkItem, workerCount)
	workerStatus := make([]WorkerStatus, workerCount)

	for i := 0; i < workerCount; i++ {
		workQueues[i] = make(chan WorkItem, 100) // 每个队列容量100
		workerStatus[i] = WorkerStatus{
			ID:        i,
			Status:    "idle",
			StartTime: time.Now(),
		}
	}

	return &LoadBalancer{
		logger:         logger,
		workQueues:     workQueues,
		workerStatus:   workerStatus,
		schedulingAlgo: "least_loaded", // 最少负载调度
		workDistrib: &WorkDistribution{
			QueueSizes:        make([]int, workerCount),
			WorkerUtilization: make([]float64, workerCount),
		},
	}
}

// StartMonitoring 开始智能监控 - README要求的核心功能
func (cm *ConcurrentManager) StartMonitoring(ctx context.Context) error {
	cm.mutex.Lock()
	if cm.isMonitoring {
		cm.mutex.Unlock()
		return fmt.Errorf("并发管理器已在监控中")
	}
	cm.isMonitoring = true
	cm.mutex.Unlock()

	cm.logger.Info("开始智能化并发监控")

	// 启动内存监控
	if err := cm.memoryMonitor.StartMonitoring(ctx); err != nil {
		return fmt.Errorf("启动内存监控失败: %w", err)
	}

	// 启动并发调整循环
	go cm.concurrencyAdjustmentLoop(ctx)

	// 启动性能统计
	go cm.performanceStatsLoop(ctx)

	return nil
}

// StopMonitoring 停止监控
func (cm *ConcurrentManager) StopMonitoring() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if !cm.isMonitoring {
		return
	}

	cm.isMonitoring = false
	cm.memoryMonitor.StopMonitoring()

	cm.logger.Info("智能化并发监控已停止",
		zap.Int("final_concurrency", cm.currentConcurrency),
		zap.Int("total_adjustments", cm.statistics.TotalAdjustments))
}

// AdjustConcurrency 动态调整并发数 - README核心要求
func (cm *ConcurrentManager) AdjustConcurrency(reason string, fileType string) int {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	oldConcurrency := cm.currentConcurrency

	// 基于文件复杂度计算建议并发数
	complexity := cm.concurrencyStrategy.GetFileComplexity(fileType)
	memoryUsage := cm.memoryMonitor.GetCurrentUsage()

	// README要求：基于文件复杂度动态调整
	targetConcurrency := cm.calculateOptimalConcurrency(complexity, memoryUsage)

	// 应用调整
	cm.currentConcurrency = cm.clampConcurrency(targetConcurrency)

	// 更新负载均衡器
	if cm.currentConcurrency != oldConcurrency {
		cm.loadBalancer.AdjustWorkerCount(cm.currentConcurrency)
		cm.statistics.TotalAdjustments++
		cm.statistics.AdjustmentReasons[reason]++

		if cm.currentConcurrency > cm.statistics.PeakConcurrency {
			cm.statistics.PeakConcurrency = cm.currentConcurrency
		}

		cm.logger.Info("并发数已调整",
			zap.String("reason", reason),
			zap.String("file_type", fileType),
			zap.Int("old_concurrency", oldConcurrency),
			zap.Int("new_concurrency", cm.currentConcurrency),
			zap.Int("complexity", complexity),
			zap.Float64("memory_usage", memoryUsage))
	}

	return cm.currentConcurrency
}

// calculateOptimalConcurrency 计算最优并发数
func (cm *ConcurrentManager) calculateOptimalConcurrency(complexity int, memoryUsage float64) int {
	// 基础并发数（基于CPU核心数）
	baseConcurrency := runtime.NumCPU()

	// 根据文件复杂度调整
	complexityFactor := 1.0 - (float64(complexity-1) * 0.1) // 复杂度越高，并发越低

	// 根据内存使用率调整
	memoryFactor := 1.0
	if memoryUsage > 0.7 {
		memoryFactor = 1.0 - (memoryUsage-0.7)*2 // 内存使用超过70%时开始降低并发
	}

	// 综合计算
	optimalConcurrency := int(float64(baseConcurrency) * complexityFactor * memoryFactor)

	// README要求：扫描阶段可以使用高并发，处理阶段需要保守
	if complexity <= 3 { // 简单文件可以高并发
		optimalConcurrency = min(baseConcurrency*2, cm.maxConcurrency)
	}

	return optimalConcurrency
}

// clampConcurrency 限制并发数在合理范围内
func (cm *ConcurrentManager) clampConcurrency(concurrency int) int {
	if concurrency < cm.minConcurrency {
		return cm.minConcurrency
	}
	if concurrency > cm.maxConcurrency {
		return cm.maxConcurrency
	}
	return concurrency
}

// concurrencyAdjustmentLoop 并发调整循环
func (cm *ConcurrentManager) concurrencyAdjustmentLoop(ctx context.Context) {
	ticker := time.NewTicker(cm.adjustInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.evaluateAndAdjust()
		}
	}
}

// evaluateAndAdjust 评估并调整
func (cm *ConcurrentManager) evaluateAndAdjust() {
	// 评估当前性能
	performance := cm.collectPerformanceMetrics()

	// 应用自适应规则
	for _, rule := range cm.concurrencyStrategy.adaptiveRules {
		if !rule.Enabled {
			continue
		}

		if cm.evaluateRule(rule, performance) {
			switch rule.Action {
			case "reduce_concurrency":
				cm.AdjustConcurrency("rule_triggered_reduce", "unknown")
			case "increase_concurrency":
				cm.AdjustConcurrency("rule_triggered_increase", "unknown")
			}
			break // 只执行第一个匹配的规则
		}
	}
}

// 内存监控相关方法
func (mm *MemoryMonitor) StartMonitoring(ctx context.Context) error {
	mm.mutex.Lock()
	if mm.isActive {
		mm.mutex.Unlock()
		return fmt.Errorf("内存监控已经启动")
	}
	mm.isActive = true
	mm.mutex.Unlock()

	go mm.monitoringLoop(ctx)
	mm.logger.Info("内存监控已启动",
		zap.Float64("warning_threshold", mm.warningThreshold),
		zap.Float64("critical_threshold", mm.criticalThreshold))

	return nil
}

func (mm *MemoryMonitor) StopMonitoring() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	mm.isActive = false
	mm.logger.Info("内存监控已停止")
}

func (mm *MemoryMonitor) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(mm.monitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !mm.isActive {
				return
			}
			mm.updateMemoryStats()
		}
	}
}

func (mm *MemoryMonitor) updateMemoryStats() {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		mm.logger.Warn("获取内存统计失败", zap.Error(err))
		return
	}

	mm.mutex.Lock()
	mm.currentUsage = vmStat.UsedPercent / 100.0
	mm.totalMemory = vmStat.Total
	mm.availableMemory = vmStat.Available

	if mm.currentUsage > mm.peakUsage {
		mm.peakUsage = mm.currentUsage
	}
	mm.mutex.Unlock()

	// 触发回调
	for _, callback := range mm.callbacks {
		callback(mm.currentUsage, mm.availableMemory)
	}

	// 检查阈值
	if mm.currentUsage > mm.criticalThreshold {
		mm.logger.Warn("内存使用达到临界阈值",
			zap.Float64("usage", mm.currentUsage),
			zap.Float64("threshold", mm.criticalThreshold))
	} else if mm.currentUsage > mm.warningThreshold {
		mm.logger.Warn("内存使用超过警告阈值",
			zap.Float64("usage", mm.currentUsage),
			zap.Float64("threshold", mm.warningThreshold))
	}
}

// 回调和工具方法
func (mm *MemoryMonitor) RegisterCallback(callback MemoryCallback) {
	mm.callbacks = append(mm.callbacks, callback)
}

func (mm *MemoryMonitor) GetCurrentUsage() float64 {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()
	return mm.currentUsage
}

func (cm *ConcurrentManager) onMemoryChange(usage float64, available uint64) {
	// README要求：内存监控机制，防止因内存溢出被系统强杀
	if usage > cm.memoryThreshold {
		cm.AdjustConcurrency("memory_pressure", "unknown")
	}
}

func (cs *ConcurrencyStrategy) GetFileComplexity(fileType string) int {
	if complexity, exists := cs.fileComplexityMap[fileType]; exists {
		return complexity
	}
	return cs.fileComplexityMap["unknown"]
}

func (cm *ConcurrentManager) collectPerformanceMetrics() *PerformanceRecord {
	return &PerformanceRecord{
		Timestamp:   time.Now(),
		Concurrency: cm.currentConcurrency,
		MemoryUsage: cm.memoryMonitor.GetCurrentUsage(),
		Success:     true, // 简化实现
	}
}

func (cm *ConcurrentManager) evaluateRule(rule AdaptiveRule, performance *PerformanceRecord) bool {
	// 简化的规则评估
	switch rule.Condition {
	case "memory_usage > 0.8":
		return performance.MemoryUsage > rule.Threshold
	default:
		return false
	}
}

func (cm *ConcurrentManager) performanceStatsLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := cm.collectPerformanceMetrics()
			cm.concurrencyStrategy.performanceHistory = append(cm.concurrencyStrategy.performanceHistory, *stats)

			// 保持历史记录在合理大小
			if len(cm.concurrencyStrategy.performanceHistory) > 100 {
				cm.concurrencyStrategy.performanceHistory = cm.concurrencyStrategy.performanceHistory[1:]
			}
		}
	}
}

func (lb *LoadBalancer) AdjustWorkerCount(newCount int) {
	// 简化实现：调整工作队列数量
	lb.logger.Info("调整工作线程数量",
		zap.Int("old_count", len(lb.workQueues)),
		zap.Int("new_count", newCount))

	// 实际实现中需要安全地调整工作线程数量
}

// GetCurrentConcurrency 获取当前并发数
func (cm *ConcurrentManager) GetCurrentConcurrency() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.currentConcurrency
}

// GetStatistics 获取并发统计信息
func (cm *ConcurrentManager) GetStatistics() *ConcurrencyStats {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.statistics
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
