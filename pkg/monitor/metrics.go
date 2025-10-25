package monitor

import "time"

// PerformanceMetrics 性能指标结构体
// 包含系统和进程的所有监控指标
type PerformanceMetrics struct {
	// === CPU监控 ===
	CPUUsage      float64 // CPU使用率（0-1）
	CPUCores      int     // CPU核心数
	LoadAverage1  float64 // 1分钟负载
	LoadAverage5  float64 // 5分钟负载
	LoadAverage15 float64 // 15分钟负载

	// === 内存监控 ===
	MemoryUsage     float64 // 内存使用率（0-1）
	MemoryTotal     uint64  // 总内存（bytes）
	MemoryAvailable uint64  // 可用内存（bytes）
	MemoryUsed      uint64  // 已用内存（bytes）
	SwapUsage       float64 // 交换区使用率（0-1）
	SwapTotal       uint64  // 交换区总量（bytes）
	SwapUsed        uint64  // 交换区已用（bytes）

	// === 磁盘监控 ===
	DiskUsagePercent float64 // 磁盘使用率（0-1）
	DiskReadBytes    uint64  // 累计读取字节数
	DiskWriteBytes   uint64  // 累计写入字节数
	DiskReadSpeed    float64 // 当前读取速度（MB/s）
	DiskWriteSpeed   float64 // 当前写入速度（MB/s）
	DiskIOPS         float64 // 每秒I/O操作数

	// === 进程监控 ===
	GoroutineCount int           // 协程数量
	ThreadCount    int           // 线程数量
	ProcessMemory  uint64        // 进程内存使用（bytes）
	GCCount        uint32        // GC次数
	GCPauseTime    time.Duration // GC累计暂停时间
	GCPauseAvg     time.Duration // GC平均暂停时间

	// === 性能指标 ===
	Throughput     float64       // 吞吐量（文件/秒）
	ProcessingRate float64       // 处理速度（MB/秒）
	AverageTime    time.Duration // 平均处理时间
	ErrorRate      float64       // 错误率（0-1）
	QueueLength    int           // 等待队列长度

	// === Worker状态 ===
	CurrentWorkers int // 当前worker数量
	MaxWorkers     int // 最大worker数量
	IdleWorkers    int // 空闲worker数量
	BusyWorkers    int // 繁忙worker数量

	// === 时间戳 ===
	Timestamp time.Time     // 采集时间戳
	Uptime    time.Duration // 程序运行时间
}

// WorkerAdjustment Worker调整记录
type WorkerAdjustment struct {
	Timestamp  time.Time     // 调整时间
	Action     string        // increase, decrease, maintain
	Reason     string        // 调整原因
	OldWorkers int           // 调整前worker数
	NewWorkers int           // 调整后worker数
	Metrics    *PerformanceMetrics // 调整时的性能指标
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Enable            bool          // 是否启用监控
	Interval          time.Duration // 采集间隔（默认3秒）
	EnableCPU         bool          // CPU监控
	EnableMemory      bool          // 内存监控
	EnableDisk        bool          // 磁盘监控
	EnableProcess     bool          // 进程监控
	EnableUI          bool          // 显示UI面板
	UIPosition        string        // top, bottom, floating
	UIRefreshInterval time.Duration // UI刷新间隔
}

// DefaultMonitorConfig 默认监控配置
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		Enable:            true,
		Interval:          3 * time.Second,
		EnableCPU:         true,
		EnableMemory:      true,
		EnableDisk:        true,
		EnableProcess:     true,
		EnableUI:          true,
		UIPosition:        "top",
		UIRefreshInterval: 3 * time.Second,
	}
}

// PerformanceSummary 性能摘要（用于报告）
type PerformanceSummary struct {
	// 时间
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`

	// 文件统计
	TotalFiles    int `json:"total_files"`
	ProcessedOK   int `json:"processed_ok"`
	ProcessedFail int `json:"processed_fail"`
	Skipped       int `json:"skipped"`

	// 性能统计
	AvgThroughput     float64       `json:"avg_throughput"`      // 平均吞吐量（文件/秒）
	AvgProcessingRate float64       `json:"avg_processing_rate"` // 平均处理速度（MB/秒）
	PeakCPU           float64       `json:"peak_cpu"`            // CPU峰值
	PeakMemory        float64       `json:"peak_memory"`         // 内存峰值
	AvgCPU            float64       `json:"avg_cpu"`             // 平均CPU
	AvgMemory         float64       `json:"avg_memory"`          // 平均内存
	TotalDiskRead     uint64        `json:"total_disk_read"`     // 总读取量
	TotalDiskWrite    uint64        `json:"total_disk_write"`    // 总写入量
	GCCount           uint32        `json:"gc_count"`            // GC次数
	TotalGCPause      time.Duration `json:"total_gc_pause"`      // GC总暂停

	// Worker调整
	WorkerAdjustments []WorkerAdjustment `json:"worker_adjustments"` // 调整历史
	InitialWorkers    int                `json:"initial_workers"`    // 初始worker数
	FinalWorkers      int                `json:"final_workers"`      // 最终worker数
	AvgWorkers        float64            `json:"avg_workers"`        // 平均worker数
}


