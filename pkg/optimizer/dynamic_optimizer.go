package optimizer

import (
	"sync/atomic"
	"time"

	"pixly/pkg/monitor"

	"go.uber.org/zap"
)

// DynamicOptimizer 动态优化器
// 根据系统资源使用情况自动调整worker数量，实现自适应优化
type DynamicOptimizer struct {
	logger *zap.Logger
	config *OptimizerConfig

	// Worker控制
	currentWorkers int32 // 当前worker数量（原子操作）
	maxWorkers     int32 // 最大worker数量
	minWorkers     int32 // 最小worker数量

	// 阈值设置
	memoryThreshold float64 // 内存使用率阈值（默认0.75）
	cpuThreshold    float64 // CPU使用率阈值（默认0.80）
	diskThreshold   float64 // 磁盘使用率阈值（默认0.85）

	// 调整策略
	adjustmentFactor float64       // 调整系数（默认1.2）
	cooldown         time.Duration // 冷却时间（默认10秒）
	lastAdjustment   time.Time     // 上次调整时间

	// 历史记录
	adjustments []monitor.WorkerAdjustment // 调整历史
}

// OptimizerConfig 优化器配置
type OptimizerConfig struct {
	Enable           bool          // 是否启用动态优化
	InitialWorkers   int           // 初始worker数（默认8）
	MinWorkers       int           // 最小worker数（默认2）
	MaxWorkers       int           // 最大worker数（默认16）
	MemoryThreshold  float64       // 内存阈值（默认0.75）
	CPUThreshold     float64       // CPU阈值（默认0.80）
	DiskThreshold    float64       // 磁盘阈值（默认0.85）
	AdjustmentFactor float64       // 调整系数（默认1.2）
	Cooldown         time.Duration // 冷却时间（默认10秒）
}

// DefaultOptimizerConfig 默认优化器配置
func DefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		Enable:           true,
		InitialWorkers:   8,
		MinWorkers:       2,
		MaxWorkers:       16,
		MemoryThreshold:  0.75, // 75%
		CPUThreshold:     0.80, // 80%
		DiskThreshold:    0.85, // 85%
		AdjustmentFactor: 1.2,  // 每次调整20%
		Cooldown:         10 * time.Second,
	}
}

// NewDynamicOptimizer 创建动态优化器
func NewDynamicOptimizer(logger *zap.Logger, config *OptimizerConfig) *DynamicOptimizer {
	if config == nil {
		config = DefaultOptimizerConfig()
	}

	do := &DynamicOptimizer{
		logger:           logger,
		config:           config,
		maxWorkers:       int32(config.MaxWorkers),
		minWorkers:       int32(config.MinWorkers),
		memoryThreshold:  config.MemoryThreshold,
		cpuThreshold:     config.CPUThreshold,
		diskThreshold:    config.DiskThreshold,
		adjustmentFactor: config.AdjustmentFactor,
		cooldown:         config.Cooldown,
		adjustments:      make([]monitor.WorkerAdjustment, 0, 50),
	}

	// 设置初始worker数
	atomic.StoreInt32(&do.currentWorkers, int32(config.InitialWorkers))

	return do
}

// AdjustWorkers 根据性能指标调整worker数量
func (do *DynamicOptimizer) AdjustWorkers(metrics *monitor.PerformanceMetrics) {
	if !do.config.Enable {
		return
	}

	// 检查冷却时间
	if time.Since(do.lastAdjustment) < do.cooldown {
		return // 在冷却期内，不调整
	}

	currentWorkers := atomic.LoadInt32(&do.currentWorkers)
	oldWorkers := int(currentWorkers)
	newWorkers := oldWorkers
	action := "maintain"
	reason := ""

	// === 资源压力检测（降低worker）===

	// 1. 内存压力
	if metrics.MemoryUsage > do.memoryThreshold {
		newWorkers = int(float64(currentWorkers) / do.adjustmentFactor)
		if newWorkers < int(do.minWorkers) {
			newWorkers = int(do.minWorkers)
		}
		action = "decrease"
		reason = "内存压力过高"
		do.logger.Warn("检测到内存压力，降低worker数量",
			zap.Float64("memory_usage", metrics.MemoryUsage),
			zap.Int("old_workers", oldWorkers),
			zap.Int("new_workers", newWorkers))
	}

	// 2. CPU负载过高
	if metrics.CPUUsage > do.cpuThreshold {
		newWorkers = int(float64(currentWorkers) / do.adjustmentFactor)
		if newWorkers < int(do.minWorkers) {
			newWorkers = int(do.minWorkers)
		}
		action = "decrease"
		reason = "CPU负载过高"
		do.logger.Warn("检测到CPU负载过高，降低worker数量",
			zap.Float64("cpu_usage", metrics.CPUUsage),
			zap.Int("old_workers", oldWorkers),
			zap.Int("new_workers", newWorkers))
	}

	// 3. 磁盘I/O瓶颈
	if metrics.DiskUsagePercent > do.diskThreshold {
		newWorkers = int(float64(currentWorkers) / do.adjustmentFactor)
		if newWorkers < int(do.minWorkers) {
			newWorkers = int(do.minWorkers)
		}
		action = "decrease"
		reason = "磁盘I/O瓶颈"
		do.logger.Warn("检测到磁盘I/O瓶颈，降低worker数量",
			zap.Float64("disk_usage", metrics.DiskUsagePercent),
			zap.Int("old_workers", oldWorkers),
			zap.Int("new_workers", newWorkers))
	}

	// === 资源充足检测（增加worker）===

	// 只有在没有压力的情况下才考虑增加
	if action == "maintain" {
		// 内存充足 && CPU充足 && worker未达上限
		if metrics.MemoryUsage < 0.50 && metrics.CPUUsage < 0.60 && currentWorkers < do.maxWorkers {
			newWorkers = int(float64(currentWorkers) * do.adjustmentFactor)
			if newWorkers > int(do.maxWorkers) {
				newWorkers = int(do.maxWorkers)
			}
			action = "increase"
			reason = "资源充足，提升并发"
			do.logger.Info("资源充足，增加worker数量",
				zap.Float64("memory_usage", metrics.MemoryUsage),
				zap.Float64("cpu_usage", metrics.CPUUsage),
				zap.Int("old_workers", oldWorkers),
				zap.Int("new_workers", newWorkers))
		}
	}

	// 应用调整
	if newWorkers != oldWorkers {
		atomic.StoreInt32(&do.currentWorkers, int32(newWorkers))
		do.lastAdjustment = time.Now()

		// 记录调整历史
		adjustment := monitor.WorkerAdjustment{
			Timestamp:  time.Now(),
			Action:     action,
			Reason:     reason,
			OldWorkers: oldWorkers,
			NewWorkers: newWorkers,
			Metrics:    metrics,
		}
		do.adjustments = append(do.adjustments, adjustment)

		do.logger.Info("⚡ Worker数量已调整",
			zap.String("action", action),
			zap.String("reason", reason),
			zap.Int("old", oldWorkers),
			zap.Int("new", newWorkers))
	}
}

// GetCurrentWorkers 获取当前worker数量（线程安全）
func (do *DynamicOptimizer) GetCurrentWorkers() int {
	return int(atomic.LoadInt32(&do.currentWorkers))
}

// SetWorkers 手动设置worker数量（用于初始化或强制调整）
func (do *DynamicOptimizer) SetWorkers(count int) {
	if count < int(do.minWorkers) {
		count = int(do.minWorkers)
	}
	if count > int(do.maxWorkers) {
		count = int(do.maxWorkers)
	}

	atomic.StoreInt32(&do.currentWorkers, int32(count))
	do.logger.Info("手动设置worker数量", zap.Int("workers", count))
}

// GetAdjustmentHistory 获取调整历史
func (do *DynamicOptimizer) GetAdjustmentHistory() []monitor.WorkerAdjustment {
	// 返回副本
	history := make([]monitor.WorkerAdjustment, len(do.adjustments))
	copy(history, do.adjustments)
	return history
}

// GetAdjustmentStats 获取调整统计
func (do *DynamicOptimizer) GetAdjustmentStats() (increases, decreases, total int) {
	for _, adj := range do.adjustments {
		total++
		if adj.Action == "increase" {
			increases++
		} else if adj.Action == "decrease" {
			decreases++
		}
	}
	return
}

// ShouldThrottle 判断是否应该限流（用于外部判断）
func (do *DynamicOptimizer) ShouldThrottle(metrics *monitor.PerformanceMetrics) bool {
	// 任一资源超过阈值即限流
	return metrics.MemoryUsage > do.memoryThreshold ||
		metrics.CPUUsage > do.cpuThreshold ||
		metrics.DiskUsagePercent > do.diskThreshold
}

// GetRecommendedWorkers 获取推荐worker数量（基于当前指标）
func (do *DynamicOptimizer) GetRecommendedWorkers(metrics *monitor.PerformanceMetrics) int {
	currentWorkers := do.GetCurrentWorkers()

	// 如果有资源压力，建议减少
	if do.ShouldThrottle(metrics) {
		recommended := int(float64(currentWorkers) / do.adjustmentFactor)
		if recommended < int(do.minWorkers) {
			recommended = int(do.minWorkers)
		}
		return recommended
	}

	// 如果资源充足，建议增加
	if metrics.MemoryUsage < 0.50 && metrics.CPUUsage < 0.60 {
		recommended := int(float64(currentWorkers) * do.adjustmentFactor)
		if recommended > int(do.maxWorkers) {
			recommended = int(do.maxWorkers)
		}
		return recommended
	}

	// 资源适中，维持当前
	return currentWorkers
}


