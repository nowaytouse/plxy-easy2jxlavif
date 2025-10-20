package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"pixly/pkg/core/types"
)

// Config 应用配置
type Config struct {
	// Processing directory
	TargetDir string `json:"target_dir"`

	// Processing mode
	Mode                string `json:"mode"` // "auto+", "quality", "sticker"
	ConcurrentJobs      int    `json:"concurrent_jobs"`
	EnableBackups       bool   `json:"enable_backups"`
	HwAccel             bool   `json:"hw_accel"`
	MaxRetries          int    `json:"max_retries"`
	LogLevel            string `json:"log_level"`
	SortOrder           string `json:"sort_order"`
	CRF                 int    `json:"crf"`
	StickerTargetFormat string `json:"sticker_target_format"`
	Overwrite           bool   `json:"overwrite"`
	DebugMode           bool   `json:"debug_mode"`
	DryRun              bool   `json:"dry_run"`

	// Performance settings
	MaxWorkers     int     `json:"max_workers"`
	MemoryLimit    uint64  `json:"memory_limit_gb"`
	CPUUtilization float64 `json:"cpu_utilization"`

	// Timeout settings
	VideoTimeout    int `json:"video_timeout_seconds"`
	AnimatedTimeout int `json:"animated_timeout_seconds"`
	ImageTimeout    int `json:"image_timeout_seconds"`
	DebugTimeout    int `json:"debug_timeout_seconds"`

	// Quality assessment
	EnableQualityAssessment bool    `json:"enable_quality_assessment"`
	HighQualityThreshold    float64 `json:"high_quality_threshold"`
	LowQualityThreshold     float64 `json:"low_quality_threshold"`

	// Processing options
	CreateBackups      bool `json:"create_backups"`
	KeepBackups        bool `json:"keep_backups"` // 是否保留备份文件
	EnableMetadataCopy bool `json:"enable_metadata_copy"`
	EnableExtensionFix bool `json:"enable_extension_fix"`
	EnableMemoryWatch  bool `json:"enable_memory_watch"`

	// Output options
	JXLEffort     int  `json:"jxl_effort"`
	AVIFSpeed     int  `json:"avif_speed"`
	WebPQuality   int  `json:"webp_quality"`
	ShowDetailLog bool `json:"show_detail_log"`

	// UI settings
	UseColorOutput   bool   `json:"use_color_output"`
	ShowProgressBars bool   `json:"show_progress_bars"`
	UILanguage       string `json:"ui_language"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	// 计算默认并发数：CPU核心数的75%，但不少于1，不超过8
	numCPU := runtime.NumCPU()
	defaultConcurrency := max(1, min(8, int(float64(numCPU)*0.75)))

	return &Config{
		// Processing mode
		Mode:                "auto+",
		ConcurrentJobs:      defaultConcurrency, // 使用计算出的默认值
		EnableBackups:       true,
		HwAccel:             true,
		MaxRetries:          2,
		LogLevel:            "info",
		SortOrder:           "size",
		CRF:                 28,
		StickerTargetFormat: "avif",
		Overwrite:           false,
		DebugMode:           false,
		DryRun:              false,

		// Performance
		MaxWorkers:     min(7, max(1, int(float64(runtime.NumCPU())*0.85))),
		MemoryLimit:    8, // GB
		CPUUtilization: 0.85,

		// Timeouts
		VideoTimeout:    90,
		AnimatedTimeout: 60,
		ImageTimeout:    30,
		DebugTimeout:    30,

		// Quality
		EnableQualityAssessment: true,
		HighQualityThreshold:    2.5,
		LowQualityThreshold:     0.6,

		// Processing
		CreateBackups:      true,
		KeepBackups:        false, // 默认不保留备份文件
		EnableMetadataCopy: true,
		EnableExtensionFix: true,
		EnableMemoryWatch:  true,

		// Output
		JXLEffort:     7,
		AVIFSpeed:     6,
		WebPQuality:   80,
		ShowDetailLog: false,

		// UI
		UseColorOutput:   true,
		ShowProgressBars: true,
		UILanguage:       "zh-CN",
	}
}

// ValidateConfig 验证配置
func (c *Config) ValidateConfig() error {
	// 验证ConcurrentJobs（并发任务数）
	if c.ConcurrentJobs < 1 || c.ConcurrentJobs > 32 {
		return fmt.Errorf("无效的并发任务数: %d (应在 1-32 之间)", c.ConcurrentJobs)
	}

	// 验证MaxWorkers（工作线程数）
	if c.MaxWorkers < 1 || c.MaxWorkers > runtime.NumCPU()*2 {
		return fmt.Errorf("无效的工作线程数: %d (应在 1-%d 之间)", c.MaxWorkers, runtime.NumCPU()*2)
	}

	if c.MemoryLimit < 1 || c.MemoryLimit > 64 {
		return fmt.Errorf("无效的内存限制: %d GB (应在 1-64 之间)", c.MemoryLimit)
	}

	if c.JXLEffort < 1 || c.JXLEffort > 9 {
		return fmt.Errorf("无效的JXL压缩级别: %d (应在 1-9 之间)", c.JXLEffort)
	}

	if c.AVIFSpeed < 0 || c.AVIFSpeed > 10 {
		return fmt.Errorf("无效的AVIF速度: %d (应在 0-10 之间)", c.AVIFSpeed)
	}

	// 验证MaxRetries
	if c.MaxRetries < 0 || c.MaxRetries > 10 {
		return fmt.Errorf("无效的最大重试次数: %d (应在 0-10 之间)", c.MaxRetries)
	}

	// 验证CRF值
	if c.CRF < 0 || c.CRF > 51 {
		return fmt.Errorf("无效的CRF值: %d (应在 0-51 之间)", c.CRF)
	}

	return nil
}

// Validate 验证配置（全局函数）
func Validate(c *Config) error {
	return c.ValidateConfig()
}

// NormalizeConfig 标准化配置，修复无效值
func NormalizeConfig(c *Config) {
	if c == nil {
		return
	}

	// 修复ConcurrentJobs
	if c.ConcurrentJobs <= 0 {
		numCPU := runtime.NumCPU()
		c.ConcurrentJobs = max(1, min(8, int(float64(numCPU)*0.75)))
	}
	if c.ConcurrentJobs > 32 {
		c.ConcurrentJobs = 32
	}

	// 修复MaxWorkers
	if c.MaxWorkers <= 0 {
		c.MaxWorkers = min(7, max(1, int(float64(runtime.NumCPU())*0.85)))
	}
	if c.MaxWorkers > runtime.NumCPU()*2 {
		c.MaxWorkers = runtime.NumCPU() * 2
	}

	// 修复MaxRetries
	if c.MaxRetries < 0 {
		c.MaxRetries = 2
	}
	if c.MaxRetries > 10 {
		c.MaxRetries = 10
	}

	// 修复CRF值
	if c.CRF < 0 || c.CRF > 51 {
		c.CRF = 28
	}

	// 修复其他参数
	if c.MemoryLimit == 0 {
		c.MemoryLimit = 8
	}

	if c.JXLEffort < 1 || c.JXLEffort > 9 {
		c.JXLEffort = 7
	}

	if c.AVIFSpeed < 0 || c.AVIFSpeed > 10 {
		c.AVIFSpeed = 6
	}

	// 修复字符串字段
	if c.Mode == "" {
		c.Mode = "auto+"
	}

	if c.LogLevel == "" {
		c.LogLevel = "info"
	}

	if c.SortOrder == "" {
		c.SortOrder = "size"
	}

	if c.StickerTargetFormat == "" {
		c.StickerTargetFormat = "avif"
	}
}

// ValidateAndNormalize 验证并标准化配置
func ValidateAndNormalize(c *Config) error {
	if c == nil {
		return fmt.Errorf("配置不能为空")
	}

	// 先标准化
	NormalizeConfig(c)

	// 再验证
	return c.ValidateConfig()
}

// GetTimeoutForMedia 根据媒体类型获取超时时间
func (c *Config) GetTimeoutForMedia(mediaType types.MediaType) int {
	switch mediaType {
	case types.MediaTypeVideo:
		return c.VideoTimeout
	case types.MediaTypeAnimated:
		return c.AnimatedTimeout
	case types.MediaTypeImage:
		return c.ImageTimeout
	default:
		return c.ImageTimeout
	}
}

// GetDataDir 获取数据目录
func GetDataDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法获取可执行文件路径: %w", err)
	}

	dataDir := filepath.Join(filepath.Dir(execPath), ".pixly")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("无法创建数据目录: %w", err)
	}

	return dataDir, nil
}

// GetLogDir 获取日志目录
func GetLogDir() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}

	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("无法创建日志目录: %w", err)
	}

	return logDir, nil
}

// GetCacheDir 获取缓存目录
func GetCacheDir() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(dataDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("无法创建缓存目录: %w", err)
	}

	return cacheDir, nil
}

// GetStateDBPath 获取状态数据库路径
func GetStateDBPath() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dataDir, "state.db"), nil
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
