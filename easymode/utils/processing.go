// utils/processing.go - 共享处理逻辑模块
//
// 功能说明：
// - 提取所有工具中重复的处理逻辑
// - 统一Options、Stats、FileProcessInfo等结构体
// - 提供公共的文件处理函数

package utils

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ProcessOptions 处理选项（通用）
type ProcessOptions struct {
	Workers           int
	InputDir          string
	OutputDir         string
	SkipExist         bool
	DryRun            bool
	InPlace           bool // 原地转换：转换成功后删除原文件
	TimeoutSeconds    int
	Retries           int
	MaxMemory         int64
	MaxFileSize       int64
	EnableHealthCheck bool
	Quality           int  // 质量参数（1-100）
	Speed             int  // 速度参数
	CopyMetadata      bool // 是否复制元数据
	PreserveTimes     bool // 是否保留时间戳
}

// FileProcessInfo 文件处理信息
type FileProcessInfo struct {
	FilePath       string
	FileSize       int64
	FileType       string
	IsAnimated     bool
	ProcessingTime time.Duration
	ConversionMode string
	Success        bool
	ErrorMsg       string
	RetryCount     int
	StartTime      time.Time
	EndTime        time.Time
	ErrorType      string
	SizeBefore     int64
	SizeAfter      int64
}

// ProcessStats 处理统计信息
type ProcessStats struct {
	mu              sync.RWMutex
	Processed       int
	Failed          int
	Skipped         int
	VideoSkipped    int
	OtherSkipped    int
	TotalSizeBefore int64
	TotalSizeAfter  int64
	StartTime       time.Time
	DetailedLogs    []FileProcessInfo
	ByExt           map[string]int
	PeakMemoryUsage int64
	TotalRetries    int
	ErrorTypes      map[string]int
}

// NewProcessStats 创建新的统计对象
func NewProcessStats() *ProcessStats {
	return &ProcessStats{
		StartTime:  time.Now(),
		ByExt:      make(map[string]int),
		ErrorTypes: make(map[string]int),
	}
}

// AddProcessed 添加成功处理的文件
func (s *ProcessStats) AddProcessed(sizeBefore, sizeAfter int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Processed++
	s.TotalSizeBefore += sizeBefore
	s.TotalSizeAfter += sizeAfter
}

// AddFailed 添加失败的文件
func (s *ProcessStats) AddFailed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Failed++
}

// AddSkipped 添加跳过的文件
func (s *ProcessStats) AddSkipped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Skipped++
}

// AddVideoSkipped 添加跳过的视频文件
func (s *ProcessStats) AddVideoSkipped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.VideoSkipped++
	s.Skipped++
}

// AddOtherSkipped 添加跳过的其他文件
func (s *ProcessStats) AddOtherSkipped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.OtherSkipped++
	s.Skipped++
}

// AddByExt 按扩展名统计
func (s *ProcessStats) AddByExt(ext string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ByExt[ext]++
}

// AddDetailedLog 添加详细日志
func (s *ProcessStats) AddDetailedLog(info FileProcessInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DetailedLogs = append(s.DetailedLogs, info)
}

// AddRetry 添加重试次数
func (s *ProcessStats) AddRetry() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalRetries++
}

// AddErrorType 添加错误类型统计
func (s *ProcessStats) AddErrorType(errorType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorTypes[errorType]++
}

// UpdatePeakMemory 更新峰值内存使用
func (s *ProcessStats) UpdatePeakMemory(current int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current > s.PeakMemoryUsage {
		s.PeakMemoryUsage = current
	}
}

// GetCompressionRatio 获取压缩比
func (s *ProcessStats) GetCompressionRatio() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.TotalSizeBefore == 0 {
		return 0
	}
	return float64(s.TotalSizeAfter) / float64(s.TotalSizeBefore)
}

// GetSuccessRate 获取成功率
func (s *ProcessStats) GetSuccessRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := s.Processed + s.Failed
	if total == 0 {
		return 0
	}
	return float64(s.Processed) / float64(total) * 100
}

// GetTotalProcessed 获取总处理数
func (s *ProcessStats) GetTotalProcessed() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Processed + s.Failed
}

// ClassifyError 对错误进行分类
func ClassifyError(err error) string {
	if err == nil {
		return "unknown"
	}
	errStr := err.Error()
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "超时") {
		return "timeout"
	} else if strings.Contains(errStr, "memory") || strings.Contains(errStr, "内存") {
		return "memory"
	} else if strings.Contains(errStr, "permission") || strings.Contains(errStr, "权限") {
		return "permission"
	} else if strings.Contains(errStr, "format") || strings.Contains(errStr, "格式") {
		return "format"
	} else if strings.Contains(errStr, "not found") || strings.Contains(errStr, "不存在") {
		return "not_found"
	} else if strings.Contains(errStr, "validation") || strings.Contains(errStr, "验证") {
		return "validation"
	}
	return "unknown"
}

// FormatBytes 格式化字节大小
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
