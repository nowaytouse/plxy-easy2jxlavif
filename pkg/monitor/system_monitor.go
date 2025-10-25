package monitor

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// SystemMonitor 系统监控器
// 负责实时监控系统资源使用情况（CPU、内存、磁盘等）
type SystemMonitor struct {
	logger *zap.Logger
	config *MonitorConfig

	// 监控状态
	mu         sync.RWMutex
	isRunning  bool
	ctx        context.Context
	cancel     context.CancelFunc
	startTime  time.Time

	// 性能指标
	currentMetrics  *PerformanceMetrics
	metricsHistory  []*PerformanceMetrics
	maxHistorySize  int // 保留最近N个采样点

	// 统计信息
	peakCPU       float64
	peakMemory    float64
	totalSamples  int
	sumCPU        float64
	sumMemory     float64

	// 磁盘I/O基线（用于计算速度）
	lastDiskRead  uint64
	lastDiskWrite uint64
	lastSampleTime time.Time
}

// NewSystemMonitor 创建系统监控器
func NewSystemMonitor(logger *zap.Logger, config *MonitorConfig) *SystemMonitor {
	if config == nil {
		config = DefaultMonitorConfig()
	}

	return &SystemMonitor{
		logger:         logger,
		config:         config,
		metricsHistory: make([]*PerformanceMetrics, 0, 100),
		maxHistorySize: 100, // 保留最近100个采样点（5分钟）
		startTime:      time.Now(),
	}
}

// Start 启动监控协程
func (sm *SystemMonitor) Start(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isRunning {
		return nil // 已经在运行
	}

	sm.ctx, sm.cancel = context.WithCancel(ctx)
	sm.isRunning = true
	sm.startTime = time.Now()

	// 启动监控协程
	go sm.monitorLoop()

	sm.logger.Info("系统监控已启动",
		zap.Duration("interval", sm.config.Interval))

	return nil
}

// Stop 停止监控
func (sm *SystemMonitor) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isRunning {
		return
	}

	sm.cancel()
	sm.isRunning = false

	sm.logger.Info("系统监控已停止",
		zap.Duration("uptime", time.Since(sm.startTime)),
		zap.Int("total_samples", sm.totalSamples))
}

// monitorLoop 监控主循环
func (sm *SystemMonitor) monitorLoop() {
	ticker := time.NewTicker(sm.config.Interval)
	defer ticker.Stop()

	// 初始化磁盘I/O基线
	if sm.config.EnableDisk {
		sm.initDiskIOBaseline()
	}

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			// 采集性能指标
			metrics := sm.collectMetrics()
			sm.updateMetrics(metrics)
		}
	}
}

// collectMetrics 采集性能指标
func (sm *SystemMonitor) collectMetrics() *PerformanceMetrics {
	metrics := &PerformanceMetrics{
		Timestamp: time.Now(),
		Uptime:    time.Since(sm.startTime),
	}

	// CPU监控
	if sm.config.EnableCPU {
		sm.collectCPUMetrics(metrics)
	}

	// 内存监控
	if sm.config.EnableMemory {
		sm.collectMemoryMetrics(metrics)
	}

	// 磁盘监控
	if sm.config.EnableDisk {
		sm.collectDiskMetrics(metrics)
	}

	// 进程监控
	if sm.config.EnableProcess {
		sm.collectProcessMetrics(metrics)
	}

	return metrics
}

// collectCPUMetrics 采集CPU指标
func (sm *SystemMonitor) collectCPUMetrics(metrics *PerformanceMetrics) {
	// CPU使用率
	if percent, err := cpu.Percent(0, false); err == nil && len(percent) > 0 {
		metrics.CPUUsage = percent[0] / 100.0 // 转换为0-1范围
	}

	// CPU核心数
	if count, err := cpu.Counts(true); err == nil {
		metrics.CPUCores = count
	}

	// 负载平均
	if loadAvg, err := load.Avg(); err == nil {
		metrics.LoadAverage1 = loadAvg.Load1
		metrics.LoadAverage5 = loadAvg.Load5
		metrics.LoadAverage15 = loadAvg.Load15
	}
}

// collectMemoryMetrics 采集内存指标
func (sm *SystemMonitor) collectMemoryMetrics(metrics *PerformanceMetrics) {
	// 虚拟内存
	if vmStat, err := mem.VirtualMemory(); err == nil {
		metrics.MemoryUsage = vmStat.UsedPercent / 100.0 // 转换为0-1范围
		metrics.MemoryTotal = vmStat.Total
		metrics.MemoryAvailable = vmStat.Available
		metrics.MemoryUsed = vmStat.Used
	}

	// 交换区
	if swapStat, err := mem.SwapMemory(); err == nil {
		metrics.SwapUsage = swapStat.UsedPercent / 100.0 // 转换为0-1范围
		metrics.SwapTotal = swapStat.Total
		metrics.SwapUsed = swapStat.Used
	}
}

// collectDiskMetrics 采集磁盘指标
func (sm *SystemMonitor) collectDiskMetrics(metrics *PerformanceMetrics) {
	// 磁盘使用率（根目录）
	if usage, err := disk.Usage("/"); err == nil {
		metrics.DiskUsagePercent = usage.UsedPercent / 100.0 // 转换为0-1范围
	}

	// 磁盘I/O统计
	if ioCounters, err := disk.IOCounters(); err == nil {
		var totalRead, totalWrite uint64
		for _, counter := range ioCounters {
			totalRead += counter.ReadBytes
			totalWrite += counter.WriteBytes
		}
		metrics.DiskReadBytes = totalRead
		metrics.DiskWriteBytes = totalWrite

		// 计算读写速度（基于上次采样）
		if !sm.lastSampleTime.IsZero() {
			elapsed := time.Since(sm.lastSampleTime).Seconds()
			if elapsed > 0 {
				readDelta := float64(totalRead - sm.lastDiskRead)
				writeDelta := float64(totalWrite - sm.lastDiskWrite)
				
				metrics.DiskReadSpeed = (readDelta / elapsed) / 1024 / 1024  // MB/s
				metrics.DiskWriteSpeed = (writeDelta / elapsed) / 1024 / 1024 // MB/s
			}
		}

		// 更新基线
		sm.lastDiskRead = totalRead
		sm.lastDiskWrite = totalWrite
		sm.lastSampleTime = time.Now()
	}
}

// collectProcessMetrics 采集进程指标
func (sm *SystemMonitor) collectProcessMetrics(metrics *PerformanceMetrics) {
	// 协程数
	metrics.GoroutineCount = runtime.NumGoroutine()

	// 内存统计
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	metrics.ProcessMemory = memStats.Alloc
	metrics.GCCount = memStats.NumGC
	metrics.GCPauseTime = time.Duration(memStats.PauseTotalNs)
	
	if memStats.NumGC > 0 {
		metrics.GCPauseAvg = time.Duration(memStats.PauseTotalNs / uint64(memStats.NumGC))
	}
}

// initDiskIOBaseline 初始化磁盘I/O基线
func (sm *SystemMonitor) initDiskIOBaseline() {
	if ioCounters, err := disk.IOCounters(); err == nil {
		var totalRead, totalWrite uint64
		for _, counter := range ioCounters {
			totalRead += counter.ReadBytes
			totalWrite += counter.WriteBytes
		}
		sm.lastDiskRead = totalRead
		sm.lastDiskWrite = totalWrite
		sm.lastSampleTime = time.Now()
	}
}

// updateMetrics 更新指标并维护历史记录
func (sm *SystemMonitor) updateMetrics(metrics *PerformanceMetrics) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 更新当前指标
	sm.currentMetrics = metrics

	// 更新统计
	sm.totalSamples++
	sm.sumCPU += metrics.CPUUsage
	sm.sumMemory += metrics.MemoryUsage

	// 更新峰值
	if metrics.CPUUsage > sm.peakCPU {
		sm.peakCPU = metrics.CPUUsage
	}
	if metrics.MemoryUsage > sm.peakMemory {
		sm.peakMemory = metrics.MemoryUsage
	}

	// 添加到历史记录
	sm.metricsHistory = append(sm.metricsHistory, metrics)

	// 限制历史记录大小
	if len(sm.metricsHistory) > sm.maxHistorySize {
		sm.metricsHistory = sm.metricsHistory[1:]
	}
}

// GetCurrentMetrics 获取当前指标（线程安全）
func (sm *SystemMonitor) GetCurrentMetrics() *PerformanceMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.currentMetrics == nil {
		return &PerformanceMetrics{}
	}

	// 返回副本
	metricsCopy := *sm.currentMetrics
	return &metricsCopy
}

// GetPeakMetrics 获取峰值指标
func (sm *SystemMonitor) GetPeakMetrics() (peakCPU, peakMemory float64) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.peakCPU, sm.peakMemory
}

// GetAverageMetrics 获取平均指标
func (sm *SystemMonitor) GetAverageMetrics() (avgCPU, avgMemory float64) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.totalSamples == 0 {
		return 0, 0
	}

	avgCPU = sm.sumCPU / float64(sm.totalSamples)
	avgMemory = sm.sumMemory / float64(sm.totalSamples)
	return
}

// GetMetricsHistory 获取历史指标
func (sm *SystemMonitor) GetMetricsHistory() []*PerformanceMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 返回副本
	history := make([]*PerformanceMetrics, len(sm.metricsHistory))
	copy(history, sm.metricsHistory)
	return history
}

// GetSummary 获取性能摘要
func (sm *SystemMonitor) GetSummary(totalFiles, processedOK, processedFail, skipped int) *PerformanceSummary {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	avgCPU, avgMemory := sm.sumCPU / float64(sm.totalSamples), sm.sumMemory / float64(sm.totalSamples)

	summary := &PerformanceSummary{
		StartTime:     sm.startTime,
		EndTime:       time.Now(),
		Duration:      time.Since(sm.startTime),
		TotalFiles:    totalFiles,
		ProcessedOK:   processedOK,
		ProcessedFail: processedFail,
		Skipped:       skipped,
		PeakCPU:       sm.peakCPU,
		PeakMemory:    sm.peakMemory,
		AvgCPU:        avgCPU,
		AvgMemory:     avgMemory,
	}

	// 计算吞吐量和处理速度
	if summary.Duration.Seconds() > 0 {
		summary.AvgThroughput = float64(processedOK) / summary.Duration.Seconds()
	}

	// 从当前指标获取磁盘和GC数据
	if sm.currentMetrics != nil {
		summary.TotalDiskRead = sm.currentMetrics.DiskReadBytes
		summary.TotalDiskWrite = sm.currentMetrics.DiskWriteBytes
		summary.GCCount = sm.currentMetrics.GCCount
		summary.TotalGCPause = sm.currentMetrics.GCPauseTime
	}

	return summary
}


