// utils/stats_shared.go - 共享统计结构体和方法
//
// 功能说明：
// - 提供统一的Stats结构体
// - 提供统一的Stats方法
// - 支持多种统计场景
//
// 作者: AI Assistant
// 版本: v1.0.0
// 创建: 2025-10-26

package utils

import (
	"sync"
	"time"
)

// SharedStats 共享的统计结构体（可扩展版本）
// 包含所有工具可能需要的统计字段
type SharedStats struct {
	sync.RWMutex
	ImagesProcessed    int
	ImagesFailed       int
	ImagesSkipped      int
	VideosSkipped      int                // 可选：视频跳过计数
	OtherSkipped       int                // 可选：其他类型跳过计数
	TotalBytesBefore   int64
	TotalBytesAfter    int64
	StartTime          time.Time
	DetailedLogs       []SharedFileProcessInfo
	ByExt              map[string]int
	PeakMemoryUsage    int64
	TotalRetries       int
	RecoveryActions    int                      // 可选：恢复操作计数
	ErrorTypes         map[string]int
	PerformanceMetrics map[string]float64       // 可选：性能指标
}

// NewSharedStats 创建新的共享统计实例
func NewSharedStats() *SharedStats {
	return &SharedStats{
		StartTime:          time.Now(),
		ByExt:              make(map[string]int),
		ErrorTypes:         make(map[string]int),
		PerformanceMetrics: make(map[string]float64),
		DetailedLogs:       make([]SharedFileProcessInfo, 0),
	}
}

// AddProcessed 记录成功处理的文件
func (s *SharedStats) AddProcessed(sizeBefore, sizeAfter int64) {
	s.Lock()
	defer s.Unlock()
	s.ImagesProcessed++
	s.TotalBytesBefore += sizeBefore
	s.TotalBytesAfter += sizeAfter
}

// AddFailed 记录处理失败的文件
func (s *SharedStats) AddFailed() {
	s.Lock()
	defer s.Unlock()
	s.ImagesFailed++
}

// AddSkipped 记录跳过的文件
func (s *SharedStats) AddSkipped() {
	s.Lock()
	defer s.Unlock()
	s.ImagesSkipped++
}

// AddVideoSkipped 记录跳过的视频文件
func (s *SharedStats) AddVideoSkipped() {
	s.Lock()
	defer s.Unlock()
	s.VideosSkipped++
}

// AddOtherSkipped 记录跳过的其他文件
func (s *SharedStats) AddOtherSkipped() {
	s.Lock()
	defer s.Unlock()
	s.OtherSkipped++
}

// AddByExt 按文件扩展名统计
func (s *SharedStats) AddByExt(ext string) {
	s.Lock()
	defer s.Unlock()
	s.ByExt[ext]++
}

// AddDetailedLog 添加详细日志
func (s *SharedStats) AddDetailedLog(info SharedFileProcessInfo) {
	s.Lock()
	defer s.Unlock()
	s.DetailedLogs = append(s.DetailedLogs, info)
}

// AddRetry 记录重试次数
func (s *SharedStats) AddRetry() {
	s.Lock()
	defer s.Unlock()
	s.TotalRetries++
}

// AddRecovery 记录恢复操作
func (s *SharedStats) AddRecovery() {
	s.Lock()
	defer s.Unlock()
	s.RecoveryActions++
}

// AddErrorType 记录错误类型
func (s *SharedStats) AddErrorType(errorType string) {
	s.Lock()
	defer s.Unlock()
	s.ErrorTypes[errorType]++
}

// UpdatePeakMemory 更新峰值内存使用
func (s *SharedStats) UpdatePeakMemory(current int64) {
	s.Lock()
	defer s.Unlock()
	if current > s.PeakMemoryUsage {
		s.PeakMemoryUsage = current
	}
}

// SetPerformanceMetric 设置性能指标
func (s *SharedStats) SetPerformanceMetric(key string, value float64) {
	s.Lock()
	defer s.Unlock()
	s.PerformanceMetrics[key] = value
}

// GetCompressionRatio 获取压缩比率
func (s *SharedStats) GetCompressionRatio() float64 {
	s.RLock()
	defer s.RUnlock()
	if s.TotalBytesBefore == 0 {
		return 0
	}
	return float64(s.TotalBytesAfter) / float64(s.TotalBytesBefore)
}

// GetSuccessRate 获取成功率
func (s *SharedStats) GetSuccessRate() float64 {
	s.RLock()
	defer s.RUnlock()
	total := s.ImagesProcessed + s.ImagesFailed
	if total == 0 {
		return 0
	}
	return float64(s.ImagesProcessed) / float64(total) * 100
}

// GetTotalProcessed 获取总处理数（成功+失败）
func (s *SharedStats) GetTotalProcessed() int {
	s.RLock()
	defer s.RUnlock()
	return s.ImagesProcessed + s.ImagesFailed
}

// GetElapsedTime 获取已用时间
func (s *SharedStats) GetElapsedTime() time.Duration {
	s.RLock()
	defer s.RUnlock()
	return time.Since(s.StartTime)
}

