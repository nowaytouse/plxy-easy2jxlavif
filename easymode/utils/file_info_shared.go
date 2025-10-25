// utils/file_info_shared.go - 共享文件处理信息结构体
//
// 功能说明：
// - 提供统一的文件处理信息结构体
// - 支持详细的处理日志记录
// - 可扩展的字段设计
//
// 作者: AI Assistant
// 版本: v1.0.0
// 创建: 2025-10-26

package utils

import (
	"time"
)

// SharedFileProcessInfo 共享的文件处理信息结构体（可扩展版本）
// 记录文件处理过程中的各种信息
type SharedFileProcessInfo struct {
	FilePath          string
	FileSize          int64
	FileType          string
	IsAnimated        bool                   // 可选：是否为动画
	ProcessingTime    time.Duration
	ConversionMode    string
	Success           bool
	ErrorMsg          string
	RetryCount        int                    // 可选：重试次数
	StartTime         time.Time
	EndTime           time.Time
	ErrorType         string
	MemoryUsage       uint64                 // 可选：内存使用
	CPUPercent        float64                // 可选：CPU使用率
	QualityMetrics    map[string]float64     // 可选：质量指标
	PerformanceScore  float64                // 可选：性能评分
	ToolVersion       string                 // 可选：工具版本
}

// NewFileProcessInfo 创建新的文件处理信息实例
func NewFileProcessInfo(filePath string, fileSize int64) *SharedFileProcessInfo {
	return &SharedFileProcessInfo{
		FilePath:       filePath,
		FileSize:       fileSize,
		FileType:       "",
		StartTime:      time.Now(),
		QualityMetrics: make(map[string]float64),
	}
}

// MarkSuccess 标记处理成功
func (f *SharedFileProcessInfo) MarkSuccess(conversionMode string) {
	f.Success = true
	f.ConversionMode = conversionMode
	f.EndTime = time.Now()
	f.ProcessingTime = f.EndTime.Sub(f.StartTime)
}

// MarkFailed 标记处理失败
func (f *SharedFileProcessInfo) MarkFailed(errorMsg, errorType string) {
	f.Success = false
	f.ErrorMsg = errorMsg
	f.ErrorType = errorType
	f.EndTime = time.Now()
	f.ProcessingTime = f.EndTime.Sub(f.StartTime)
}

// AddRetry 增加重试次数
func (f *SharedFileProcessInfo) AddRetry() {
	f.RetryCount++
}

// SetQualityMetric 设置质量指标
func (f *SharedFileProcessInfo) SetQualityMetric(key string, value float64) {
	if f.QualityMetrics == nil {
		f.QualityMetrics = make(map[string]float64)
	}
	f.QualityMetrics[key] = value
}

