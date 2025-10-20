package concurrency

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"pixly/pkg/core/types"

	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// worker 工作协程
func (scm *SmartConcurrencyManager) worker(ctx context.Context, workerID int) {
	scm.logger.Debug("工作协程启动", zap.Int("worker_id", workerID))

	defer func() {
		scm.logger.Debug("工作协程退出", zap.Int("worker_id", workerID))
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-scm.shutdownChan:
			return
		case jobRequest := <-scm.jobQueue:
			// 处理任务
			result := scm.processJob(ctx, jobRequest)

			// 发送结果
			select {
			case jobRequest.ResultChan <- result:
			case <-time.After(5 * time.Second):
				scm.logger.Warn("结果发送超时", zap.String("job_id", jobRequest.Context.ID))
			}
		}
	}
}

// processJob 处理单个任务
func (scm *SmartConcurrencyManager) processJob(ctx context.Context, jobRequest *JobRequest) *JobResult {
	startTime := time.Now()
	jobCtx := jobRequest.Context

	scm.mutex.Lock()
	scm.activeJobs[jobCtx.ID] = jobCtx
	scm.mutex.Unlock()

	defer func() {
		scm.mutex.Lock()
		delete(scm.activeJobs, jobCtx.ID)
		scm.mutex.Unlock()
	}()

	scm.logger.Debug("开始处理任务",
		zap.String("job_id", jobCtx.ID),
		zap.String("file_path", jobCtx.MediaInfo.Path),
		zap.Float64("complexity", jobCtx.ComplexityScore))

	// 记录内存使用（处理前）
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// 执行任务处理
	err := jobRequest.Handler(ctx, jobCtx)

	// 记录内存使用（处理后）
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	duration := time.Since(startTime)
	memoryUsed := int64(memAfter.Alloc - memBefore.Alloc)

	// 创建结果
	result := &JobResult{
		JobID:          jobCtx.ID,
		Success:        err == nil,
		Error:          err,
		Duration:       duration,
		MemoryUsed:     memoryUsed,
		ComplexityUsed: jobCtx.ComplexityScore,
	}

	if err == nil {
		// 这里应该从实际处理结果中获取，暂时创建一个基本结果
		result.Result = &types.ProcessingResult{
			OriginalPath: jobCtx.MediaInfo.Path,
			OriginalSize: jobCtx.MediaInfo.Size,
			NewSize:      jobCtx.MediaInfo.Size, // 实际应该从转换结果获取
			SpaceSaved:   0,                     // 实际应该计算
			Success:      true,
			ProcessTime:  duration,
			Mode:         jobCtx.ProcessingMode,
		}
	}

	// 发送结果到结果处理器
	select {
	case scm.resultQueue <- result:
	default:
		scm.logger.Warn("结果队列已满", zap.String("job_id", jobCtx.ID))
	}

	scm.logger.Debug("任务处理完成",
		zap.String("job_id", jobCtx.ID),
		zap.Bool("success", result.Success),
		zap.Duration("duration", duration),
		zap.Int64("memory_used_kb", memoryUsed/1024))

	return result
}

// resultProcessor 结果处理器
func (scm *SmartConcurrencyManager) resultProcessor(ctx context.Context) {
	scm.logger.Debug("结果处理器启动")

	for {
		select {
		case <-ctx.Done():
			return
		case <-scm.shutdownChan:
			return
		case result := <-scm.resultQueue:
			scm.processResult(result)
		}
	}
}

// processResult 处理任务结果，更新统计信息
func (scm *SmartConcurrencyManager) processResult(result *JobResult) {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	// 更新统计信息
	scm.stats.TotalJobsProcessed++
	if result.Success {
		scm.stats.SuccessfulJobs++
	} else {
		scm.stats.FailedJobs++
	}

	scm.stats.TotalMemoryUsed += result.MemoryUsed

	// 更新性能指标
	scm.updatePerformanceMetrics(result)

	scm.logger.Debug("结果处理完成",
		zap.String("job_id", result.JobID),
		zap.Bool("success", result.Success),
		zap.Int64("total_jobs", scm.stats.TotalJobsProcessed))
}

// memoryMonitor 内存监控器 - README要求：防止内存溢出被系统强杀
func (scm *SmartConcurrencyManager) memoryMonitor(ctx context.Context) {
	scm.logger.Debug("内存监控器启动")

	for {
		select {
		case <-ctx.Done():
			return
		case <-scm.shutdownChan:
			return
		case <-scm.memoryTicker.C:
			scm.checkSystemMemory()
		}
	}
}

// checkSystemMemory 检查系统内存使用情况
func (scm *SmartConcurrencyManager) checkSystemMemory() {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		scm.logger.Warn("获取内存信息失败", zap.Error(err))
		return
	}

	usagePercent := memInfo.UsedPercent

	scm.logger.Debug("内存监控检查",
		zap.Float64("usage_percent", usagePercent),
		zap.Float64("threshold", scm.memoryThreshold),
		zap.Uint64("used_gb", memInfo.Used/(1024*1024*1024)),
		zap.Uint64("total_gb", memInfo.Total/(1024*1024*1024)))

	// README要求：内存监控机制，防止超大文件导致内存溢出
	if usagePercent > scm.memoryThreshold {
		scm.handleHighMemoryUsage(usagePercent)
	} else if usagePercent < scm.memoryThreshold-20 && scm.currentWorkers < scm.maxWorkers {
		// 内存充足时，考虑增加工作协程
		scm.considerIncreasingWorkers()
	}
}

// handleHighMemoryUsage 处理高内存使用情况
func (scm *SmartConcurrencyManager) handleHighMemoryUsage(currentUsage float64) {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	scm.stats.MemoryLimitHits++

	scm.logger.Warn("检测到高内存使用",
		zap.Float64("current_usage", currentUsage),
		zap.Float64("threshold", scm.memoryThreshold),
		zap.Int("current_workers", scm.currentWorkers))

	// 紧急措施：减少工作协程数
	if scm.currentWorkers > scm.minWorkers {
		newWorkers := int(float64(scm.currentWorkers) * scm.backoffFactor)
		if newWorkers < scm.minWorkers {
			newWorkers = scm.minWorkers
		}

		scm.adjustWorkerCount(newWorkers)

		scm.logger.Warn("由于内存压力减少工作协程数",
			zap.Int("old_workers", scm.currentWorkers),
			zap.Int("new_workers", newWorkers))

		// 强制垃圾回收
		runtime.GC()
	}
}

// dynamicAdjuster 动态调整器 - README要求：基于文件复杂度动态调整并发数
func (scm *SmartConcurrencyManager) dynamicAdjuster(ctx context.Context) {
	scm.logger.Debug("动态调整器启动")

	for {
		select {
		case <-ctx.Done():
			return
		case <-scm.shutdownChan:
			return
		case <-scm.adjustmentTicker.C:
			scm.performDynamicAdjustment()
		}
	}
}

// performDynamicAdjustment 执行动态调整
func (scm *SmartConcurrencyManager) performDynamicAdjustment() {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	// 计算当前系统负载
	activeJobCount := len(scm.activeJobs)
	avgComplexity := scm.calculateAverageComplexity()
	queueLength := len(scm.jobQueue)

	scm.logger.Debug("动态调整分析",
		zap.Int("active_jobs", activeJobCount),
		zap.Float64("avg_complexity", avgComplexity),
		zap.Int("queue_length", queueLength),
		zap.Int("current_workers", scm.currentWorkers))

	// 决策逻辑：基于复杂度和队列情况调整
	targetWorkers := scm.calculateOptimalWorkerCount(avgComplexity, queueLength, activeJobCount)

	if targetWorkers != scm.currentWorkers {
		scm.adjustWorkerCount(targetWorkers)
		scm.stats.WorkerAdjustments++

		scm.logger.Info("动态调整工作协程数",
			zap.Int("old_workers", scm.currentWorkers),
			zap.Int("new_workers", targetWorkers),
			zap.Float64("avg_complexity", avgComplexity))
	}
}

// calculateOptimalWorkerCount 计算最优工作协程数
func (scm *SmartConcurrencyManager) calculateOptimalWorkerCount(avgComplexity float64, queueLength, activeJobs int) int {
	baseWorkers := scm.currentWorkers

	// 1. 基于复杂度调整
	if avgComplexity > scm.highComplexityThreshold {
		// 高复杂度任务：减少并发数，避免资源竞争
		baseWorkers = int(float64(baseWorkers) * scm.backoffFactor)
	} else if avgComplexity < scm.lowComplexityThreshold {
		// 低复杂度任务：可以增加并发数
		baseWorkers = int(float64(baseWorkers) * scm.scalingFactor)
	}

	// 2. 基于队列长度调整
	if queueLength > scm.currentWorkers*2 {
		// 队列积压严重，适度增加工作协程
		baseWorkers = int(float64(baseWorkers) * 1.1)
	} else if queueLength == 0 && activeJobs < scm.currentWorkers/2 {
		// 队列空闲，工作协程利用率低，可以减少
		baseWorkers = int(float64(baseWorkers) * 0.9)
	}

	// 3. 应用边界限制
	if baseWorkers < scm.minWorkers {
		baseWorkers = scm.minWorkers
	} else if baseWorkers > scm.maxWorkers {
		baseWorkers = scm.maxWorkers
	}

	return baseWorkers
}

// adjustWorkerCount 调整工作协程数量
func (scm *SmartConcurrencyManager) adjustWorkerCount(targetWorkers int) {
	currentWorkers := scm.currentWorkers

	if targetWorkers > currentWorkers {
		// 增加工作协程
		for i := currentWorkers; i < targetWorkers; i++ {
			go scm.worker(context.Background(), i)
		}
	}
	// 注意：Go协程无法直接强制停止，减少工作协程通过自然完成任务后不再启动新协程实现

	scm.currentWorkers = targetWorkers

	// 更新峰值记录
	if targetWorkers > scm.stats.PeakWorkers {
		scm.stats.PeakWorkers = targetWorkers
	}
}

// 辅助方法实现
func (scm *SmartConcurrencyManager) generateJobID() string {
	return fmt.Sprintf("job_%d_%d", time.Now().UnixNano(), len(scm.activeJobs))
}

func (scm *SmartConcurrencyManager) estimateMemoryUsage(mediaInfo *types.MediaInfo, complexityScore float64) int64 {
	// 基于文件大小和复杂度估算内存使用
	baseMem := mediaInfo.Size * 2 // 假设需要2倍文件大小的内存

	// 复杂度加权
	complexityMultiplier := 1.0 + (complexityScore / 100.0)

	return int64(float64(baseMem) * complexityMultiplier)
}

func (scm *SmartConcurrencyManager) calculateJobPriority(complexityScore float64) JobPriority {
	if complexityScore >= 90 {
		return PriorityCritical
	} else if complexityScore >= 70 {
		return PriorityHigh
	} else if complexityScore >= 40 {
		return PriorityNormal
	} else {
		return PriorityLow
	}
}

func (scm *SmartConcurrencyManager) checkMemoryAvailability(estimatedMemory int64) error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	if memInfo.Available < uint64(estimatedMemory) {
		return fmt.Errorf("可用内存不足：需要 %d MB，可用 %d MB",
			estimatedMemory/(1024*1024),
			memInfo.Available/(1024*1024))
	}

	return nil
}

func (scm *SmartConcurrencyManager) updateComplexityHistory(complexityScore float64, duration time.Duration) {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	scm.jobComplexityHistory = append(scm.jobComplexityHistory, complexityScore)

	// 保持历史记录在合理范围内
	if len(scm.jobComplexityHistory) > 100 {
		scm.jobComplexityHistory = scm.jobComplexityHistory[1:]
	}
}

func (scm *SmartConcurrencyManager) calculateAverageComplexity() float64 {
	if len(scm.jobComplexityHistory) == 0 {
		return 50.0 // 默认中等复杂度
	}

	sum := 0.0
	for _, complexity := range scm.jobComplexityHistory {
		sum += complexity
	}

	return sum / float64(len(scm.jobComplexityHistory))
}

func (scm *SmartConcurrencyManager) updatePerformanceMetrics(result *JobResult) {
	// 更新性能指标（简化版本）
	scm.performanceMetrics.LastUpdateTime = time.Now()

	// 这里应该包含更复杂的性能指标计算逻辑
	// 如吞吐量、内存效率、复杂度预测准确性等
}

func (scm *SmartConcurrencyManager) considerIncreasingWorkers() {
	// 在内存充足时考虑增加工作协程的逻辑
	if scm.currentWorkers < scm.maxWorkers && len(scm.jobQueue) > 0 {
		newWorkers := scm.currentWorkers + 1
		scm.adjustWorkerCount(newWorkers)

		scm.logger.Debug("内存充足，增加工作协程",
			zap.Int("new_workers", newWorkers))
	}
}

// GetStats 获取并发统计信息
func (scm *SmartConcurrencyManager) GetStats() *ConcurrencyStats {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()

	// 计算平均工作协程数
	if scm.stats.TotalJobsProcessed > 0 {
		scm.stats.AverageWorkers = float64(scm.currentWorkers)
	}

	return scm.stats
}

// Stop 停止智能并发管理器
func (scm *SmartConcurrencyManager) Stop() error {
	scm.logger.Info("正在停止智能并发管理器")

	// 停止定时器
	if scm.adjustmentTicker != nil {
		scm.adjustmentTicker.Stop()
	}
	if scm.memoryTicker != nil {
		scm.memoryTicker.Stop()
	}

	// 发送关闭信号
	close(scm.shutdownChan)

	// 等待活跃任务完成（最多等待30秒）
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		scm.mutex.RLock()
		activeCount := len(scm.activeJobs)
		scm.mutex.RUnlock()

		if activeCount == 0 {
			break
		}

		select {
		case <-timeout:
			scm.logger.Warn("等待活跃任务完成超时", zap.Int("remaining_jobs", activeCount))
			return fmt.Errorf("停止超时，仍有 %d 个活跃任务", activeCount)
		case <-ticker.C:
			scm.logger.Debug("等待活跃任务完成", zap.Int("remaining_jobs", activeCount))
		}
	}

	scm.logger.Info("智能并发管理器已停止")
	return nil
}
