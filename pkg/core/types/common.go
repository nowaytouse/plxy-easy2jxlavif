package types

import (
	"context"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

// AppMode 应用运行模式
type AppMode int

const (
	ModeAutoPlus AppMode = iota // 自动模式+
	ModeQuality                 // 品质模式
	ModeEmoji                   // 表情包模式
)

func (m AppMode) String() string {
	switch m {
	case ModeAutoPlus:
		return "自动模式+"
	case ModeQuality:
		return "品质模式"
	case ModeEmoji:
		return "表情包模式"
	default:
		return "未知模式"
	}
}

// MediaType 媒体类型
type MediaType int

const (
	MediaTypeUnknown  MediaType = iota
	MediaTypeImage              // 静图
	MediaTypeAnimated           // 动图
	MediaTypeVideo              // 视频
)

// 新模块化系统常量别名
const (
	Static   = MediaTypeImage    // 静态图片
	Animated = MediaTypeAnimated // 动画图片
	Video    = MediaTypeVideo    // 视频
)

func (mt MediaType) String() string {
	switch mt {
	case MediaTypeImage:
		return "静图"
	case MediaTypeAnimated:
		return "动图"
	case MediaTypeVideo:
		return "视频"
	default:
		return "未知"
	}
}

// QualityLevel 品质等级
type QualityLevel int

const (
	QualityUnknown    QualityLevel = iota
	QualityVeryLow                 // 极低品质
	QualityLow                     // 低品质
	QualityMediumLow               // 中低品质
	QualityMediumHigh              // 中高品质
	QualityHigh                    // 高品质
	QualityVeryHigh                // 极高品质
	QualityCorrupted               // 损坏文件
)

func (ql QualityLevel) String() string {
	switch ql {
	case QualityVeryLow:
		return "极低品质"
	case QualityLow:
		return "低品质"
	case QualityMediumLow:
		return "中低品质"
	case QualityMediumHigh:
		return "中高品质"
	case QualityHigh:
		return "高品质"
	case QualityVeryHigh:
		return "极高品质"
	case QualityCorrupted:
		return "损坏文件"
	default:
		return "未知品质"
	}
}

// ProcessingStatus 处理状态
type ProcessingStatus int

const (
	StatusPending ProcessingStatus = iota
	StatusScanning
	StatusAssessing
	StatusConverting
	StatusCompleted
	StatusSkipped
	StatusFailed
	StatusCorrupted
)

func (s ProcessingStatus) String() string {
	switch s {
	case StatusPending:
		return "待处理"
	case StatusScanning:
		return "扫描中"
	case StatusAssessing:
		return "评估中"
	case StatusConverting:
		return "转换中"
	case StatusCompleted:
		return "已完成"
	case StatusSkipped:
		return "已跳过"
	case StatusFailed:
		return "处理失败"
	case StatusCorrupted:
		return "文件损坏"
	default:
		return "未知状态"
	}
}

// MediaInfo 媒体文件信息
type MediaInfo struct {
	Path           string           `json:"path"`
	Size           int64            `json:"size"`
	ModTime        time.Time        `json:"mod_time"`
	Type           MediaType        `json:"type"`
	Format         string           `json:"format"`
	Quality        QualityLevel     `json:"quality"`
	Status         ProcessingStatus `json:"status"`
	IsCorrupted    bool             `json:"is_corrupted"`
	Width          int              `json:"width,omitempty"`
	Height         int              `json:"height,omitempty"`
	Duration       float64          `json:"duration,omitempty"`
	PixelDensity   float64          `json:"pixel_density,omitempty"`
	JpegQuality    int              `json:"jpeg_quality,omitempty"`
	LastProcessed  time.Time        `json:"last_processed,omitempty"`
	ProcessingTime time.Duration    `json:"processing_time,omitempty"`
	// 新增字段
	CreateTime time.Time `json:"create_time"` // 创建时间
	// 新增字段
	ModifyTime time.Time `json:"modify_time"` // 修改时间
	// 新增字段
	ICCProfile string `json:"icc_profile,omitempty"` // ICC配置信息
	// 批量决策相关字段
	PreferredMode AppMode `json:"preferred_mode,omitempty"` // 用户指定的优先处理模式
	QualityScore  float64 `json:"quality_score,omitempty"`  // 质量分数 (0-10)
	ErrorMessage  string  `json:"error_message,omitempty"`  // 错误信息
}

// ProcessingResult 处理结果
type ProcessingResult struct {
	OriginalPath string        `json:"original_path"`
	NewPath      string        `json:"new_path,omitempty"`
	OriginalSize int64         `json:"original_size"`
	NewSize      int64         `json:"new_size"`
	SpaceSaved   int64         `json:"space_saved"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	ProcessTime  time.Duration `json:"process_time"`
	Mode         AppMode       `json:"mode"`
}

// Statistics 统计信息
type Statistics struct {
	TotalFiles      int                  `json:"total_files"`
	ProcessedFiles  int                  `json:"processed_files"`
	SuccessFiles    int                  `json:"success_files"`
	SkippedFiles    int                  `json:"skipped_files"`
	FailedFiles     int                  `json:"failed_files"`
	CorruptedFiles  int                  `json:"corrupted_files"`
	TotalSpaceSaved int64                `json:"total_space_saved"`
	ProcessingTime  time.Duration        `json:"processing_time"`
	QualityStats    map[QualityLevel]int `json:"quality_stats"`
	FormatStats     map[string]int       `json:"format_stats"`
}

// ToolCheckResults 工具检查结果
type ToolCheckResults struct {
	HasCjxl          bool   `json:"has_cjxl"`
	CjxlPath         string `json:"cjxl_path"`
	HasAvifenc       bool   `json:"has_avifenc"`
	AvifencPath      string `json:"avifenc_path"`
	HasExiftool      bool   `json:"has_exiftool"`
	ExiftoolPath     string `json:"exiftool_path"`
	HasFfmpeg        bool   `json:"has_ffmpeg"`
	FfmpegDevPath    string `json:"ffmpeg_dev_path"`
	FfmpegStablePath string `json:"ffmpeg_stable_path"`
	HasVideotoolbox  bool   `json:"has_videotoolbox"`
	HasLibx264       bool   `json:"has_libx264"`
	HasLibx265       bool   `json:"has_libx265"`
	HasLibaom        bool   `json:"has_libaom"`
	HasLibdav1d      bool   `json:"has_libdav1d"`
	HasLibjxl        bool   `json:"has_libjxl"`
	HasBrotli        bool   `json:"has_brotli"`
	// 新增缺少的字段
	HasLibSvtAv1       bool   `json:"has_libsvtav1"`
	HasVToolbox        bool   `json:"has_vtoolbox"`
	EmbeddedFfmpegNote string `json:"embedded_ffmpeg_note"`
}

// AppContext 应用程序上下文
type AppContext struct {
	// Core context
	Context    context.Context    `json:"-"`
	CancelFunc context.CancelFunc `json:"-"`

	// Configuration
	Mode       AppMode          `json:"mode"`
	Tools      ToolCheckResults `json:"tools"`
	WorkerPool *ants.Pool       `json:"-"`
	Logger     *zap.Logger      `json:"-"`

	// State management
	MediaFiles    []*MediaInfo        `json:"media_files"`
	Results       []*ProcessingResult `json:"results"`
	Stats         *Statistics         `json:"stats"`
	ProgressMutex sync.RWMutex        `json:"-"`
	StateMutex    sync.RWMutex        `json:"-"`

	// Processing control
	MaxWorkers     int    `json:"max_workers"`
	MemoryLimit    uint64 `json:"memory_limit"`
	EnableMemWatch bool   `json:"enable_mem_watch"`
	ProcessingDir  string `json:"processing_dir"`

	// Progress tracking
	ScanProgress       int `json:"scan_progress"`
	AssessmentProgress int `json:"assessment_progress"`
	ConversionProgress int `json:"conversion_progress"`
	TotalFound         int `json:"total_found"`
	TotalToProcess     int `json:"total_to_process"`

	// UI control
	ProgressPaused bool `json:"progress_paused"`
}

// NewAppContext 创建新的应用上下文
func NewAppContext() *AppContext {
	ctx, cancel := context.WithCancel(context.Background())

	return &AppContext{
		Context:    ctx,
		CancelFunc: cancel,
		Stats: &Statistics{
			QualityStats: make(map[QualityLevel]int),
			FormatStats:  make(map[string]int),
		},
		MediaFiles:     make([]*MediaInfo, 0),
		Results:        make([]*ProcessingResult, 0),
		MaxWorkers:     7, // 默认并发数
		EnableMemWatch: true,
	}
}

// Cleanup 清理资源
func (ctx *AppContext) Cleanup() {
	if ctx.WorkerPool != nil {
		ctx.WorkerPool.Release()
	}
	if ctx.Logger != nil {
		ctx.Logger.Sync()
	}
	if ctx.CancelFunc != nil {
		ctx.CancelFunc()
	}
}

// 转换相关类型定义

// TargetFormat 目标格式
type TargetFormat string

const (
	TargetFormatJXL  TargetFormat = "jxl"  // JPEG XL
	TargetFormatAVIF TargetFormat = "avif" // AVIF
	TargetFormatMOV  TargetFormat = "mov"  // MOV
)

// ConversionType 转换类型
type ConversionType string

const (
	ConversionTypeLossless ConversionType = "Lossless" // 无损
	ConversionTypeLossy    ConversionType = "Lossy"    // 有损
)

// Action 操作类型
type Action string

const (
	ActionConvert Action = "convert" // 转换
	ActionSkip    Action = "skip"    // 跳过
	ActionDelete  Action = "delete"  // 删除
)

// FileTask 文件任务
type FileTask struct {
	// 基本信息
	Path         string    `json:"path"`
	OriginalPath string    `json:"original_path,omitempty"`
	Size         int64     `json:"size"`
	Type         MediaType `json:"type"`
	Ext          string    `json:"ext"`
	MimeType     string    `json:"mime_type,omitempty"`

	// 转换相关
	TargetFormat   TargetFormat   `json:"target_format"`
	ConversionType ConversionType `json:"conversion_type"`
	Action         Action         `json:"action"`
	Quality        QualityLevel   `json:"quality"` // 使用已有的QualityLevel

	// 选项
	IsStickerMode           bool `json:"is_sticker_mode,omitempty"`
	UseBalancedOptimization bool `json:"use_balanced_optimization,omitempty"`

	// 超时控制
	BaselineTimeout time.Duration `json:"baseline_timeout,omitempty"`

	// 日志器
	Logger *zap.Logger `json:"-"`
}

// ConversionResult 转换结果
type ConversionResult struct {
	OriginalPath string           `json:"original_path"`
	OriginalSize int64            `json:"original_size"`
	NewSize      int64            `json:"new_size"`
	FinalPath    string           `json:"final_path,omitempty"`
	Status       ProcessingStatus `json:"status"` // 使用已有的ProcessingStatus
	Error        error            `json:"error,omitempty"`
	Task         *FileTask        `json:"task,omitempty"`
}

// 状态映射常量 - 用于新模块化系统
const (
	StatusSuccess = StatusCompleted // 成功映射到已完成
	StatusDeleted = StatusSkipped   // 已删除映射到已跳过
)

// ConversionTask 转换任务 - 兼容旧系统
type ConversionTask struct {
	SourcePath   string                 `json:"source_path"`
	TargetPath   string                 `json:"target_path,omitempty"`
	TargetFormat string                 `json:"target_format"`
	Mode         string                 `json:"mode"`
	Status       string                 `json:"status"`
	Quality      string                 `json:"quality"`
	MediaType    string                 `json:"media_type"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// RoutingDecision 路由决策
type RoutingDecision struct {
	Strategy     string       `json:"strategy"`
	TargetFormat string       `json:"target_format"`
	QualityLevel QualityLevel `json:"quality_level"`
	Reason       string       `json:"reason,omitempty"`
}
