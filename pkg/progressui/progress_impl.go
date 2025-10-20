package progressui

import (
	"fmt"
	"strings"
	"time"

	"github.com/vbauerster/mpb/v8/decor"
	"go.uber.org/zap"
)

// ProgressDecorators 进度条装饰器
type ProgressDecorators struct {
	prepend []decor.Decorator
	append  []decor.Decorator
}

// createDecorators 创建进度条装饰器 - README要求的实时精确显示
func (pm *ProgressManager) createDecorators(name string, taskType TaskType) *ProgressDecorators {
	decorators := &ProgressDecorators{
		prepend: make([]decor.Decorator, 0),
		append:  make([]decor.Decorator, 0),
	}

	// 任务名称和类型
	taskName := fmt.Sprintf("[%s] %s", taskType.String(), name)
	decorators.prepend = append(decorators.prepend,
		decor.Name(taskName, decor.WCSyncSpaceR),
	)

	// 根据主题配置装饰器
	switch pm.config.Theme {
	case ThemeMinimal:
		pm.createMinimalDecorators(decorators)
	case ThemeDetailed:
		pm.createDetailedDecorators(decorators)
	case ThemeColorful:
		pm.createColorfulDecorators(decorators)
	default:
		pm.createDefaultDecorators(decorators)
	}

	return decorators
}

// createDefaultDecorators 创建默认主题装饰器
func (pm *ProgressManager) createDefaultDecorators(decorators *ProgressDecorators) {
	// 百分比
	if pm.config.ShowPercentage {
		decorators.append = append(decorators.append,
			decor.Percentage(decor.WCSyncSpace),
		)
	}

	// 计数器
	decorators.append = append(decorators.append,
		decor.CountersNoUnit("%d / %d", decor.WCSyncSpace),
	)

	// 速度
	if pm.config.ShowSpeed {
		decorators.append = append(decorators.append,
			decor.EwmaSpeed(0, "%.1f/s", 60, decor.WCSyncSpace),
		)
	}

	// 预估时间
	if pm.config.ShowETA {
		decorators.append = append(decorators.append,
			decor.EwmaETA(decor.ET_STYLE_GO, 60, decor.WCSyncSpace),
		)
	}

	// 已用时间
	if pm.config.ShowElapsed {
		decorators.append = append(decorators.append,
			decor.Elapsed(decor.ET_STYLE_MMSS, decor.WCSyncSpace),
		)
	}
}

// createMinimalDecorators 创建简约主题装饰器
func (pm *ProgressManager) createMinimalDecorators(decorators *ProgressDecorators) {
	// 仅显示百分比和计数器
	if pm.config.ShowPercentage {
		decorators.append = append(decorators.append,
			decor.Percentage(decor.WCSyncSpace),
		)
	}

	decorators.append = append(decorators.append,
		decor.CountersNoUnit("%d/%d", decor.WCSyncSpace),
	)
}

// createDetailedDecorators 创建详细主题装饰器
func (pm *ProgressManager) createDetailedDecorators(decorators *ProgressDecorators) {
	// 详细信息：百分比、计数器、速度、ETA、已用时间、成功/失败统计
	if pm.config.ShowPercentage {
		decorators.append = append(decorators.append,
			decor.Percentage(decor.WCSyncSpace),
		)
	}

	decorators.append = append(decorators.append,
		decor.CountersNoUnit("(%d/%d)", decor.WCSyncSpace),
	)

	if pm.config.ShowSpeed {
		decorators.append = append(decorators.append,
			decor.EwmaSpeed(0, "Speed:%.1f/s", 60, decor.WCSyncSpace),
		)
	}

	if pm.config.ShowETA {
		decorators.append = append(decorators.append,
			decor.EwmaETA(decor.ET_STYLE_GO, 60, decor.WCSyncSpace),
		)
	}

	if pm.config.ShowElapsed {
		decorators.append = append(decorators.append,
			decor.Elapsed(decor.ET_STYLE_HHMMSS, decor.WCSyncSpace),
		)
	}
}

// createColorfulDecorators 创建彩色主题装饰器
func (pm *ProgressManager) createColorfulDecorators(decorators *ProgressDecorators) {
	// 彩色显示（这里简化实现，实际可以使用颜色代码）
	if pm.config.ShowPercentage {
		decorators.append = append(decorators.append,
			decor.Percentage(decor.WCSyncSpace),
		)
	}

	decorators.append = append(decorators.append,
		decor.CountersNoUnit("✅%d/❌%d", decor.WCSyncSpace),
	)

	if pm.config.ShowSpeed {
		decorators.append = append(decorators.append,
			decor.EwmaSpeed(0, "🚀%.1f/s", 60, decor.WCSyncSpace),
		)
	}

	if pm.config.ShowETA {
		decorators.append = append(decorators.append,
			decor.EwmaETA(decor.ET_STYLE_GO, 60, decor.WCSyncSpace),
		)
	}
}

// calculateSpeed 计算处理速度 - README要求的实时速度显示
func (pm *ProgressManager) calculateSpeed(tracker *ProgressTracker) {
	if !pm.config.EnableSpeedCalculation || tracker.Paused {
		return
	}

	now := time.Now()
	elapsed := now.Sub(tracker.StartTime)

	if elapsed <= 0 {
		tracker.Speed = 0
		return
	}

	// 计算平均速度（项目/秒）
	tracker.Speed = float64(tracker.ProcessedItems) / elapsed.Seconds()

	// 更新全局峰值速度
	if tracker.Speed > pm.stats.PeakSpeed {
		pm.stats.PeakSpeed = tracker.Speed
	}

	// 调用速度变化回调
	if pm.callbacks.OnSpeedChange != nil {
		oldSpeed := pm.stats.AverageSpeed
		pm.callbacks.OnSpeedChange(tracker, oldSpeed, tracker.Speed)
	}
}

// calculateETA 计算预估剩余时间 - README要求的精确时间预估
func (pm *ProgressManager) calculateETA(tracker *ProgressTracker) {
	if tracker.Speed <= 0 || tracker.Paused {
		tracker.EstimatedTimeLeft = 0
		return
	}

	remainingItems := tracker.TotalItems - tracker.ProcessedItems
	if remainingItems <= 0 {
		tracker.EstimatedTimeLeft = 0
		return
	}

	// 基于当前速度计算预估时间
	etaSeconds := float64(remainingItems) / tracker.Speed
	tracker.EstimatedTimeLeft = time.Duration(etaSeconds) * time.Second
}

// completeTracker 完成跟踪器
func (pm *ProgressManager) completeTracker(tracker *ProgressTracker) {
	endTime := time.Now()
	totalTime := endTime.Sub(tracker.StartTime)

	// 创建结果
	result := &ProgressResult{
		TrackerID:       tracker.ID,
		TotalItems:      tracker.TotalItems,
		ProcessedItems:  tracker.ProcessedItems,
		SuccessfulItems: tracker.SuccessfulItems,
		FailedItems:     tracker.FailedItems,
		SkippedItems:    tracker.SkippedItems,
		TotalTime:       totalTime,
		AverageSpeed:    tracker.Speed,
		Success:         tracker.FailedItems == 0 && !tracker.Cancelled,
	}

	if tracker.FailedItems > 0 {
		result.ErrorMessage = fmt.Sprintf("处理失败项目数: %d", tracker.FailedItems)
	}

	// 调用完成回调
	if pm.callbacks.OnComplete != nil {
		pm.callbacks.OnComplete(tracker, result)
	}

	pm.logger.Info("进度跟踪器完成",
		zap.String("tracker_id", tracker.ID),
		zap.String("name", tracker.Name),
		zap.Int64("total_items", result.TotalItems),
		zap.Int64("successful_items", result.SuccessfulItems),
		zap.Int64("failed_items", result.FailedItems),
		zap.Int64("skipped_items", result.SkippedItems),
		zap.Duration("total_time", result.TotalTime),
		zap.Float64("average_speed", result.AverageSpeed),
		zap.Bool("success", result.Success))
}

// updateGlobalStats 更新全局统计信息
func (pm *ProgressManager) updateGlobalStats() {
	if !pm.config.EnableRealTimeStats {
		return
	}

	stats := &ProgressStats{}

	var totalSpeed float64
	var activeCount int

	for _, tracker := range pm.trackers {
		stats.TotalTrackers++
		stats.TotalItems += tracker.TotalItems
		stats.ProcessedItems += tracker.ProcessedItems
		stats.SuccessfulItems += tracker.SuccessfulItems
		stats.FailedItems += tracker.FailedItems
		stats.SkippedItems += tracker.SkippedItems

		switch tracker.Status {
		case StatusRunning, StatusPaused:
			stats.ActiveTrackers++
			activeCount++
			totalSpeed += tracker.Speed
		case StatusCompleted:
			stats.CompletedTrackers++
		case StatusFailed, StatusCancelled:
			stats.FailedTrackers++
		}

		elapsed := time.Since(tracker.StartTime)
		if elapsed > stats.TotalElapsed {
			stats.TotalElapsed = elapsed
		}
	}

	// 计算平均速度
	if activeCount > 0 {
		stats.AverageSpeed = totalSpeed / float64(activeCount)
	}

	// 计算预估剩余时间
	if stats.AverageSpeed > 0 {
		remainingItems := stats.TotalItems - stats.ProcessedItems
		etaSeconds := float64(remainingItems) / stats.AverageSpeed
		stats.EstimatedRemaining = time.Duration(etaSeconds) * time.Second
	}

	pm.stats = stats
}

// GetTracker 获取跟踪器
func (pm *ProgressManager) GetTracker(trackerID string) (*ProgressTracker, error) {
	pm.trackersMutex.RLock()
	defer pm.trackersMutex.RUnlock()

	tracker, exists := pm.trackers[trackerID]
	if !exists {
		return nil, fmt.Errorf("跟踪器不存在: %s", trackerID)
	}

	// 返回副本以避免并发修改
	trackerCopy := *tracker
	return &trackerCopy, nil
}

// GetAllTrackers 获取所有跟踪器
func (pm *ProgressManager) GetAllTrackers() map[string]*ProgressTracker {
	pm.trackersMutex.RLock()
	defer pm.trackersMutex.RUnlock()

	trackers := make(map[string]*ProgressTracker)
	for id, tracker := range pm.trackers {
		// 返回副本
		trackerCopy := *tracker
		trackers[id] = &trackerCopy
	}

	return trackers
}

// GetStats 获取全局统计信息
func (pm *ProgressManager) GetStats() *ProgressStats {
	pm.trackersMutex.RLock()
	defer pm.trackersMutex.RUnlock()

	// 返回副本
	statsCopy := *pm.stats
	return &statsCopy
}

// SetCallbacks 设置回调函数
func (pm *ProgressManager) SetCallbacks(callbacks *ProgressCallbacks) {
	pm.callbacks = callbacks
}

// SetConfig 更新配置
func (pm *ProgressManager) SetConfig(config *ProgressConfig) {
	pm.config = config
	pm.logger.Info("进度管理器配置已更新")
}

// Enable 启用进度显示
func (pm *ProgressManager) Enable() {
	pm.enabled = true
	pm.logger.Info("进度显示已启用")
}

// Disable 禁用进度显示
func (pm *ProgressManager) Disable() {
	pm.enabled = false
	pm.logger.Info("进度显示已禁用")
}

// Wait 等待所有进度条完成
func (pm *ProgressManager) Wait() {
	if pm.container != nil {
		pm.container.Wait()
	}
}

// Shutdown 关闭进度管理器
func (pm *ProgressManager) Shutdown() {
	pm.trackersMutex.Lock()
	defer pm.trackersMutex.Unlock()

	// 取消所有活跃的跟踪器
	for id, tracker := range pm.trackers {
		if tracker.Status == StatusRunning || tracker.Status == StatusPaused {
			tracker.Status = StatusCancelled
			tracker.Cancelled = true
			tracker.Bar.Abort(false)
			pm.logger.Debug("关闭时取消跟踪器", zap.String("tracker_id", id))
		}
	}

	// 等待所有进度条完成
	if pm.container != nil {
		pm.container.Wait()
	}

	pm.logger.Info("进度管理器已关闭",
		zap.Int("total_trackers", len(pm.trackers)),
		zap.Int("completed_trackers", pm.stats.CompletedTrackers),
		zap.Int("cancelled_trackers", len(pm.trackers)-pm.stats.CompletedTrackers))
}

// GetProgressSummary 获取进度摘要信息
func (pm *ProgressManager) GetProgressSummary() string {
	pm.trackersMutex.RLock()
	defer pm.trackersMutex.RUnlock()

	var summary strings.Builder
	summary.WriteString("📊 进度摘要:\n")
	summary.WriteString(fmt.Sprintf("总跟踪器: %d | 活跃: %d | 完成: %d | 失败: %d\n",
		pm.stats.TotalTrackers,
		pm.stats.ActiveTrackers,
		pm.stats.CompletedTrackers,
		pm.stats.FailedTrackers))

	summary.WriteString(fmt.Sprintf("总项目: %d | 已处理: %d | 成功: %d | 失败: %d | 跳过: %d\n",
		pm.stats.TotalItems,
		pm.stats.ProcessedItems,
		pm.stats.SuccessfulItems,
		pm.stats.FailedItems,
		pm.stats.SkippedItems))

	if pm.stats.ProcessedItems > 0 {
		successRate := float64(pm.stats.SuccessfulItems) / float64(pm.stats.ProcessedItems) * 100
		summary.WriteString(fmt.Sprintf("成功率: %.1f%% | 平均速度: %.1f项/秒 | 峰值速度: %.1f项/秒\n",
			successRate,
			pm.stats.AverageSpeed,
			pm.stats.PeakSpeed))
	}

	if pm.stats.EstimatedRemaining > 0 {
		summary.WriteString(fmt.Sprintf("预估剩余时间: %v | 总耗时: %v",
			pm.stats.EstimatedRemaining,
			pm.stats.TotalElapsed))
	}

	return summary.String()
}

// RemoveTracker 移除跟踪器
func (pm *ProgressManager) RemoveTracker(trackerID string) error {
	pm.trackersMutex.Lock()
	defer pm.trackersMutex.Unlock()

	tracker, exists := pm.trackers[trackerID]
	if !exists {
		return fmt.Errorf("跟踪器不存在: %s", trackerID)
	}

	// 确保跟踪器已完成或取消
	if tracker.Status == StatusRunning || tracker.Status == StatusPaused {
		return fmt.Errorf("无法移除活跃的跟踪器: %s", trackerID)
	}

	delete(pm.trackers, trackerID)
	pm.updateGlobalStats()

	pm.logger.Debug("移除进度跟踪器", zap.String("tracker_id", trackerID))
	return nil
}

// ClearCompletedTrackers 清理已完成的跟踪器
func (pm *ProgressManager) ClearCompletedTrackers() int {
	pm.trackersMutex.Lock()
	defer pm.trackersMutex.Unlock()

	var removedCount int
	for id, tracker := range pm.trackers {
		if tracker.Status == StatusCompleted || tracker.Status == StatusFailed || tracker.Status == StatusCancelled {
			delete(pm.trackers, id)
			removedCount++
		}
	}

	if removedCount > 0 {
		pm.updateGlobalStats()
		pm.logger.Info("清理已完成的跟踪器", zap.Int("removed_count", removedCount))
	}

	return removedCount
}

// UpdateTrackerMetadata 更新跟踪器元数据
func (pm *ProgressManager) UpdateTrackerMetadata(trackerID string, metadata map[string]interface{}) error {
	pm.trackersMutex.Lock()
	defer pm.trackersMutex.Unlock()

	tracker, exists := pm.trackers[trackerID]
	if !exists {
		return fmt.Errorf("跟踪器不存在: %s", trackerID)
	}

	for key, value := range metadata {
		tracker.Metadata[key] = value
	}

	pm.logger.Debug("更新跟踪器元数据",
		zap.String("tracker_id", trackerID),
		zap.Int("metadata_count", len(metadata)))

	return nil
}
