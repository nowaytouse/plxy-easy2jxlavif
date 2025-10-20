package formatsupport

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FormatSupportManager 格式支持管理器 - README要求的完整格式支持和智能跳过机制
//
// 核心功能：
//   - 完整的格式支持白名单管理
//   - 智能格式检测和验证
//   - 自动跳过不支持的格式
//   - 格式转换建议和路径规划
//   - 格式兼容性分析和报告
//
// 设计原则：
//   - README严格规范：完全按照README规定的格式支持实现
//   - 智能识别：准确识别文件格式，避免误判
//   - 用户友好：清晰的格式支持状态和建议
//   - 性能优化：高效的格式检测和缓存机制
//   - 可扩展性：方便添加新格式支持
type FormatSupportManager struct {
	logger           *zap.Logger
	config           *FormatConfig
	supportedFormats map[string]*FormatInfo
	formatCategories map[FormatCategory][]string
	conversionMatrix map[string][]string
	formatCache      map[string]*CachedFormatResult
	stats            *FormatStatistics
	mutex            sync.RWMutex
	enabled          bool
}

// FormatConfig 格式配置
type FormatConfig struct {
	EnableWhitelist       bool          `json:"enable_whitelist"`        // 启用白名单
	EnableSmartSkip       bool          `json:"enable_smart_skip"`       // 启用智能跳过
	EnableFormatDetection bool          `json:"enable_format_detection"` // 启用格式检测
	EnableConversionHints bool          `json:"enable_conversion_hints"` // 启用转换提示
	CacheTimeout          time.Duration `json:"cache_timeout"`           // 缓存超时
	StrictMode            bool          `json:"strict_mode"`             // 严格模式
	AllowExperimental     bool          `json:"allow_experimental"`      // 允许实验性格式
	DefaultAction         DefaultAction `json:"default_action"`          // 默认动作
}

// FormatInfo 格式信息
type FormatInfo struct {
	Name              string           `json:"name"`               // 格式名称
	Extensions        []string         `json:"extensions"`         // 文件扩展名
	MimeTypes         []string         `json:"mime_types"`         // MIME类型
	Category          FormatCategory   `json:"category"`           // 格式类别
	SupportLevel      SupportLevel     `json:"support_level"`      // 支持级别
	InputSupported    bool             `json:"input_supported"`    // 输入支持
	OutputSupported   bool             `json:"output_supported"`   // 输出支持
	Quality           QualitySupport   `json:"quality"`            // 品质支持
	Features          []FormatFeature  `json:"features"`           // 格式特性
	Limitations       []string         `json:"limitations"`        // 限制说明
	RecommendedUse    string           `json:"recommended_use"`    // 推荐用途
	ConversionTargets []string         `json:"conversion_targets"` // 转换目标
	ProcessingHints   *ProcessingHints `json:"processing_hints"`   // 处理提示
	LastUpdated       time.Time        `json:"last_updated"`       // 最后更新时间
	Experimental      bool             `json:"experimental"`       // 实验性格式
}

// ProcessingHints 处理提示
type ProcessingHints struct {
	PreferredQuality   []int    `json:"preferred_quality"`   // 推荐品质设置
	OptimalSizes       []string `json:"optimal_sizes"`       // 最佳尺寸
	PerformanceNotes   []string `json:"performance_notes"`   // 性能说明
	CompatibilityNotes []string `json:"compatibility_notes"` // 兼容性说明
	BestPractices      []string `json:"best_practices"`      // 最佳实践
}

// CachedFormatResult 缓存的格式结果
type CachedFormatResult struct {
	DetectedFormat  string                 `json:"detected_format"`
	SupportStatus   SupportStatus          `json:"support_status"`
	Recommendations []string               `json:"recommendations"`
	ConversionPaths []ConversionPath       `json:"conversion_paths"`
	CacheTime       time.Time              `json:"cache_time"`
	ExpiresAt       time.Time              `json:"expires_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// ConversionPath 转换路径
type ConversionPath struct {
	TargetFormat    string        `json:"target_format"`
	Method          string        `json:"method"`
	QualityImpact   QualityImpact `json:"quality_impact"`
	PerformanceHint string        `json:"performance_hint"`
	RequiredTools   []string      `json:"required_tools"`
	EstimatedTime   time.Duration `json:"estimated_time"`
	Confidence      float64       `json:"confidence"`
}

// FormatStatistics 格式统计
type FormatStatistics struct {
	TotalFormatsSupported  int                       `json:"total_formats_supported"`
	FormatsByCategory      map[FormatCategory]int    `json:"formats_by_category"`
	FormatsBySupportLevel  map[SupportLevel]int      `json:"formats_by_support_level"`
	DetectionAttempts      int                       `json:"detection_attempts"`
	SuccessfulDetections   int                       `json:"successful_detections"`
	FailedDetections       int                       `json:"failed_detections"`
	SkippedFiles           int                       `json:"skipped_files"`
	ConversionsRecommended int                       `json:"conversions_recommended"`
	PopularFormats         map[string]int            `json:"popular_formats"`
	ConversionMatrix       map[string]map[string]int `json:"conversion_matrix"`
	PerformanceMetrics     *PerformanceMetrics       `json:"performance_metrics"`
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	AverageDetectionTime time.Duration `json:"average_detection_time"`
	FastestDetection     time.Duration `json:"fastest_detection"`
	SlowestDetection     time.Duration `json:"slowest_detection"`
	CacheHitRate         float64       `json:"cache_hit_rate"`
	TotalDetectionTime   time.Duration `json:"total_detection_time"`
}

// 枚举定义
type FormatCategory int
type SupportLevel int
type QualitySupport int
type FormatFeature int
type SupportStatus int
type QualityImpact int
type DefaultAction int

const (
	// 格式类别
	CategoryImage    FormatCategory = iota // 图像格式
	CategoryVideo                          // 视频格式
	CategoryAudio                          // 音频格式
	CategoryRaw                            // 原始格式
	CategoryVector                         // 矢量格式
	CategoryDocument                       // 文档格式
	CategoryArchive                        // 压缩格式
	CategoryOther                          // 其他格式
)

const (
	// 支持级别
	SupportFull         SupportLevel = iota // 完全支持
	SupportPartial                          // 部分支持
	SupportBasic                            // 基础支持
	SupportExperimental                     // 实验性支持
	SupportLegacy                           // 传统支持
	SupportNone                             // 不支持
)

const (
	// 品质支持
	QualityLossless    QualitySupport = iota // 无损
	QualityHighLossy                         // 高品质有损
	QualityMediumLossy                       // 中等品质有损
	QualityLowLossy                          // 低品质有损
	QualityVariable                          // 可变品质
)

const (
	// 格式特性
	FeatureTransparency FormatFeature = iota // 透明度支持
	FeatureAnimation                         // 动画支持
	FeatureMetadata                          // 元数据支持
	FeatureCompression                       // 压缩支持
	FeatureColorProfile                      // 色彩配置文件支持
	FeatureHDR                               // HDR支持
	FeatureLayered                           // 分层支持
	FeatureProgressive                       // 渐进式支持
)

const (
	// 支持状态
	StatusSupported    SupportStatus = iota // 支持
	StatusUnsupported                       // 不支持
	StatusDeprecated                        // 已弃用
	StatusExperimental                      // 实验性
	StatusConvertible                       // 可转换
	StatusUnknown                           // 未知
)

const (
	// 品质影响
	ImpactNone     QualityImpact = iota // 无影响
	ImpactMinimal                       // 最小影响
	ImpactModerate                      // 中等影响
	ImpactHigh                          // 高影响
	ImpactSevere                        // 严重影响
)

const (
	// 默认动作
	ActionAllow   DefaultAction = iota // 允许
	ActionSkip                         // 跳过
	ActionConvert                      // 转换
	ActionPrompt                       // 提示用户
)

// NewFormatSupportManager 创建格式支持管理器
func NewFormatSupportManager(logger *zap.Logger, config *FormatConfig) *FormatSupportManager {
	if config == nil {
		config = &FormatConfig{
			EnableWhitelist:       true,
			EnableSmartSkip:       true,
			EnableFormatDetection: true,
			EnableConversionHints: true,
			CacheTimeout:          1 * time.Hour,
			StrictMode:            false,
			AllowExperimental:     false,
			DefaultAction:         ActionSkip,
		}
	}

	manager := &FormatSupportManager{
		logger:           logger,
		config:           config,
		supportedFormats: make(map[string]*FormatInfo),
		formatCategories: make(map[FormatCategory][]string),
		conversionMatrix: make(map[string][]string),
		formatCache:      make(map[string]*CachedFormatResult),
		stats:            &FormatStatistics{},
		enabled:          true,
	}

	// 初始化支持的格式
	manager.initializeSupportedFormats()

	// 初始化转换矩阵
	manager.initializeConversionMatrix()

	// 初始化统计信息
	manager.initializeStatistics()

	logger.Info("格式支持管理器初始化完成",
		zap.Int("supported_formats", len(manager.supportedFormats)),
		zap.Bool("whitelist_enabled", config.EnableWhitelist),
		zap.Bool("smart_skip_enabled", config.EnableSmartSkip))

	return manager
}

// CheckFormatSupport 检查格式支持 - README核心功能
func (fsm *FormatSupportManager) CheckFormatSupport(ctx context.Context, filePath string) (*FormatSupportResult, error) {
	if !fsm.enabled {
		return &FormatSupportResult{
			Supported: true,
			Status:    StatusSupported,
			Action:    ActionAllow,
		}, nil
	}

	startTime := time.Now()
	fsm.stats.DetectionAttempts++

	// 检查缓存
	if cached := fsm.getCachedResult(filePath); cached != nil {
		fsm.logger.Debug("使用缓存的格式检查结果", zap.String("file", filepath.Base(filePath)))
		return fsm.convertCachedResult(cached), nil
	}

	fsm.logger.Debug("开始格式支持检查", zap.String("file", filepath.Base(filePath)))

	// 1. 基于文件扩展名的初步检测
	detectedFormat := fsm.detectFormatByExtension(filePath)

	// 2. 深度格式检测（如果启用）
	if fsm.config.EnableFormatDetection && detectedFormat == "" {
		detectedFormat = fsm.detectFormatByContent(ctx, filePath)
	}

	// 3. 获取格式信息
	formatInfo, exists := fsm.supportedFormats[detectedFormat]
	if !exists {
		return fsm.handleUnsupportedFormat(filePath, detectedFormat)
	}

	// 4. 检查支持级别
	supportStatus := fsm.evaluateSupportStatus(formatInfo)

	// 5. 生成建议和转换路径
	recommendations := fsm.generateRecommendations(formatInfo, filePath)
	conversionPaths := fsm.findConversionPaths(detectedFormat)

	// 6. 确定处理动作
	action := fsm.determineAction(supportStatus, formatInfo)

	result := &FormatSupportResult{
		FilePath:        filePath,
		DetectedFormat:  detectedFormat,
		FormatInfo:      formatInfo,
		Supported:       supportStatus == StatusSupported,
		Status:          supportStatus,
		Action:          action,
		Recommendations: recommendations,
		ConversionPaths: conversionPaths,
		DetectionTime:   time.Since(startTime),
	}

	// 7. 缓存结果
	fsm.cacheResult(filePath, result)

	// 8. 更新统计信息
	fsm.updateStatistics(result)

	fsm.logger.Info("格式支持检查完成",
		zap.String("file", filepath.Base(filePath)),
		zap.String("detected_format", detectedFormat),
		zap.String("status", supportStatus.String()),
		zap.String("action", action.String()),
		zap.Duration("detection_time", result.DetectionTime))

	return result, nil
}

// FormatSupportResult 格式支持结果
type FormatSupportResult struct {
	FilePath        string           `json:"file_path"`
	DetectedFormat  string           `json:"detected_format"`
	FormatInfo      *FormatInfo      `json:"format_info"`
	Supported       bool             `json:"supported"`
	Status          SupportStatus    `json:"status"`
	Action          DefaultAction    `json:"action"`
	Recommendations []string         `json:"recommendations"`
	ConversionPaths []ConversionPath `json:"conversion_paths"`
	DetectionTime   time.Duration    `json:"detection_time"`
	CacheHit        bool             `json:"cache_hit"`
}

// initializeSupportedFormats 初始化支持的格式 - README规定的完整格式支持
func (fsm *FormatSupportManager) initializeSupportedFormats() {
	// 图像格式 - README要求的核心格式
	fsm.addFormat(&FormatInfo{
		Name:              "JPEG",
		Extensions:        []string{".jpg", ".jpeg"},
		MimeTypes:         []string{"image/jpeg"},
		Category:          CategoryImage,
		SupportLevel:      SupportFull,
		InputSupported:    true,
		OutputSupported:   true,
		Quality:           QualityHighLossy,
		Features:          []FormatFeature{FeatureCompression, FeatureMetadata, FeatureProgressive},
		RecommendedUse:    "通用图像格式，兼容性最佳",
		ConversionTargets: []string{"webp", "avif", "jxl"},
		ProcessingHints: &ProcessingHints{
			PreferredQuality:   []int{85, 90, 95},
			OptimalSizes:       []string{"1920x1080", "3840x2160"},
			PerformanceNotes:   []string{"快速处理", "内存友好"},
			CompatibilityNotes: []string{"全平台支持"},
			BestPractices:      []string{"使用85-95品质", "保留EXIF元数据"},
		},
	})

	fsm.addFormat(&FormatInfo{
		Name:              "WebP",
		Extensions:        []string{".webp"},
		MimeTypes:         []string{"image/webp"},
		Category:          CategoryImage,
		SupportLevel:      SupportFull,
		InputSupported:    true,
		OutputSupported:   true,
		Quality:           QualityVariable,
		Features:          []FormatFeature{FeatureTransparency, FeatureAnimation, FeatureCompression},
		RecommendedUse:    "现代Web图像格式，压缩效率高",
		ConversionTargets: []string{"jpeg", "png", "avif"},
	})

	fsm.addFormat(&FormatInfo{
		Name:              "AVIF",
		Extensions:        []string{".avif"},
		MimeTypes:         []string{"image/avif"},
		Category:          CategoryImage,
		SupportLevel:      SupportFull,
		InputSupported:    true,
		OutputSupported:   true,
		Quality:           QualityVariable,
		Features:          []FormatFeature{FeatureTransparency, FeatureHDR, FeatureCompression},
		RecommendedUse:    "次世代图像格式，最佳压缩比",
		ConversionTargets: []string{"jpeg", "webp", "jxl"},
	})

	fsm.addFormat(&FormatInfo{
		Name:              "JPEG XL",
		Extensions:        []string{".jxl"},
		MimeTypes:         []string{"image/jxl"},
		Category:          CategoryImage,
		SupportLevel:      SupportFull,
		InputSupported:    true,
		OutputSupported:   true,
		Quality:           QualityVariable,
		Features:          []FormatFeature{FeatureTransparency, FeatureHDR, FeatureProgressive, FeatureAnimation},
		RecommendedUse:    "全能图像格式，无损和有损兼顾",
		ConversionTargets: []string{"jpeg", "webp", "avif"},
	})

	fsm.addFormat(&FormatInfo{
		Name:              "PNG",
		Extensions:        []string{".png"},
		MimeTypes:         []string{"image/png"},
		Category:          CategoryImage,
		SupportLevel:      SupportFull,
		InputSupported:    true,
		OutputSupported:   true,
		Quality:           QualityLossless,
		Features:          []FormatFeature{FeatureTransparency, FeatureMetadata},
		RecommendedUse:    "无损图像格式，支持透明度",
		ConversionTargets: []string{"webp", "avif", "jpeg"},
	})

	fsm.addFormat(&FormatInfo{
		Name:              "HEIF/HEIC",
		Extensions:        []string{".heif", ".heic"},
		MimeTypes:         []string{"image/heif", "image/heic"},
		Category:          CategoryImage,
		SupportLevel:      SupportFull,
		InputSupported:    true,
		OutputSupported:   true,
		Quality:           QualityHighLossy,
		Features:          []FormatFeature{FeatureTransparency, FeatureHDR, FeatureMetadata},
		RecommendedUse:    "苹果设备主流格式，高压缩比",
		ConversionTargets: []string{"jpeg", "webp", "avif"},
	})

	// 视频格式
	fsm.addFormat(&FormatInfo{
		Name:              "MP4",
		Extensions:        []string{".mp4", ".m4v"},
		MimeTypes:         []string{"video/mp4"},
		Category:          CategoryVideo,
		SupportLevel:      SupportFull,
		InputSupported:    true,
		OutputSupported:   true,
		Quality:           QualityVariable,
		Features:          []FormatFeature{FeatureCompression, FeatureMetadata, FeatureHDR},
		RecommendedUse:    "通用视频格式，兼容性最佳",
		ConversionTargets: []string{"webm", "mov", "avi"},
	})

	fsm.addFormat(&FormatInfo{
		Name:              "WebM",
		Extensions:        []string{".webm"},
		MimeTypes:         []string{"video/webm"},
		Category:          CategoryVideo,
		SupportLevel:      SupportFull,
		InputSupported:    true,
		OutputSupported:   true,
		Quality:           QualityVariable,
		Features:          []FormatFeature{FeatureCompression},
		RecommendedUse:    "Web视频格式，开源免费",
		ConversionTargets: []string{"mp4", "mov"},
	})

	// 更多格式可以继续添加...

	fsm.logger.Info("格式支持列表初始化完成",
		zap.Int("total_formats", len(fsm.supportedFormats)))
}
