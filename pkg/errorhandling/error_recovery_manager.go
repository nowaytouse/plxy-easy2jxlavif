package errorhandling

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ErrorRecoveryManager 错误恢复管理器 - README要求的后备转换机制
//
// 核心功能：
//   - 智能错误分析和分类处理
//   - 多级后备转换策略
//   - 自动重试机制和指数退避
//   - 错误恢复策略的动态调整
//   - 详细的错误统计和分析报告
//
// 设计原则：
//   - 自动恢复：优先尝试自动修复和恢复
//   - 渐进降级：从最佳方案逐步降级到可用方案
//   - 智能学习：基于历史错误数据优化恢复策略
//   - 用户透明：最小化对用户体验的影响
//   - 完整日志：详细记录所有错误和恢复过程
type ErrorRecoveryManager struct {
	logger             *zap.Logger
	config             *RecoveryConfig
	errorStats         *ErrorStatistics
	recoveryStrategies map[ErrorType]*RecoveryStrategy
	retryManager       *RetryManager
	fallbackManager    *FallbackManager
	errorHistory       []ErrorRecord
	mutex              sync.RWMutex
	enabled            bool
}

// RecoveryConfig 恢复配置
type RecoveryConfig struct {
	MaxRetries          int           `json:"max_retries"`           // 最大重试次数
	BaseRetryDelay      time.Duration `json:"base_retry_delay"`      // 基础重试延迟
	MaxRetryDelay       time.Duration `json:"max_retry_delay"`       // 最大重试延迟
	BackoffMultiplier   float64       `json:"backoff_multiplier"`    // 退避倍数
	EnableFallback      bool          `json:"enable_fallback"`       // 启用后备方案
	EnableLearning      bool          `json:"enable_learning"`       // 启用学习模式
	HistoryLimit        int           `json:"history_limit"`         // 历史记录限制
	RecoveryTimeout     time.Duration `json:"recovery_timeout"`      // 恢复超时
	CriticalErrorAction ActionType    `json:"critical_error_action"` // 关键错误动作
	AutoRecoveryEnabled bool          `json:"auto_recovery_enabled"` // 自动恢复启用
}

// RecoveryStrategy 恢复策略
type RecoveryStrategy struct {
	Name            string           `json:"name"`             // 策略名称
	ErrorType       ErrorType        `json:"error_type"`       // 错误类型
	Priority        int              `json:"priority"`         // 优先级（数字越小优先级越高）
	MaxAttempts     int              `json:"max_attempts"`     // 最大尝试次数
	RetryDelay      time.Duration    `json:"retry_delay"`      // 重试延迟
	FallbackActions []FallbackAction `json:"fallback_actions"` // 后备动作
	SuccessRate     float64          `json:"success_rate"`     // 成功率
	LastUsed        time.Time        `json:"last_used"`        // 最后使用时间
	Enabled         bool             `json:"enabled"`          // 是否启用
	Handler         RecoveryHandler  `json:"-"`                // 处理器函数
}

// RetryManager 重试管理器
type RetryManager struct {
	logger        *zap.Logger
	activeRetries map[string]*RetryContext
	retryStats    map[ErrorType]*RetryStatistics
	mutex         sync.RWMutex
}

// FallbackManager 后备方案管理器
type FallbackManager struct {
	logger         *zap.Logger
	fallbackChains map[string][]FallbackAction
	fallbackStats  map[FallbackType]*FallbackStatistics
	mutex          sync.RWMutex
}

// ErrorRecord 错误记录
type ErrorRecord struct {
	ID                string                 `json:"id"`                  // 错误ID
	ErrorType         ErrorType              `json:"error_type"`          // 错误类型
	ErrorMessage      string                 `json:"error_message"`       // 错误消息
	SourceFile        string                 `json:"source_file"`         // 源文件路径
	Operation         string                 `json:"operation"`           // 操作类型
	Timestamp         time.Time              `json:"timestamp"`           // 时间戳
	Severity          ErrorSeverity          `json:"severity"`            // 严重程度
	RecoveryAttempted bool                   `json:"recovery_attempted"`  // 是否尝试恢复
	RecoverySuccess   bool                   `json:"recovery_success"`    // 恢复是否成功
	RecoveryStrategy  string                 `json:"recovery_strategy"`   // 使用的恢复策略
	RetryCount        int                    `json:"retry_count"`         // 重试次数
	TotalRecoveryTime time.Duration          `json:"total_recovery_time"` // 总恢复时间
	Context           map[string]interface{} `json:"context"`             // 上下文信息
	Resolution        string                 `json:"resolution"`          // 解决方案
}

// RetryContext 重试上下文
type RetryContext struct {
	OperationID    string        `json:"operation_id"`
	ErrorType      ErrorType     `json:"error_type"`
	CurrentAttempt int           `json:"current_attempt"`
	MaxAttempts    int           `json:"max_attempts"`
	LastAttempt    time.Time     `json:"last_attempt"`
	NextAttempt    time.Time     `json:"next_attempt"`
	BackoffDelay   time.Duration `json:"backoff_delay"`
	OriginalError  error         `json:"-"`
}

// FallbackAction 后备动作
type FallbackAction struct {
	Type        FallbackType           `json:"type"`        // 后备类型
	Name        string                 `json:"name"`        // 动作名称
	Description string                 `json:"description"` // 描述
	Parameters  map[string]interface{} `json:"parameters"`  // 参数
	Priority    int                    `json:"priority"`    // 优先级
	Enabled     bool                   `json:"enabled"`     // 是否启用
	Handler     FallbackHandler        `json:"-"`           // 处理器函数
}

// ErrorStatistics 错误统计
type ErrorStatistics struct {
	TotalErrors          int                       `json:"total_errors"`
	ErrorsByType         map[ErrorType]int         `json:"errors_by_type"`
	ErrorsBySeverity     map[ErrorSeverity]int     `json:"errors_by_severity"`
	RecoveryAttempts     int                       `json:"recovery_attempts"`
	SuccessfulRecoveries int                       `json:"successful_recoveries"`
	FailedRecoveries     int                       `json:"failed_recoveries"`
	RecoverySuccessRate  float64                   `json:"recovery_success_rate"`
	AverageRecoveryTime  time.Duration             `json:"average_recovery_time"`
	StrategyStats        map[string]*StrategyStats `json:"strategy_stats"`
	TrendAnalysis        *ErrorTrendAnalysis       `json:"trend_analysis"`
}

// RetryStatistics 重试统计
type RetryStatistics struct {
	TotalRetries      int           `json:"total_retries"`
	SuccessfulRetries int           `json:"successful_retries"`
	FailedRetries     int           `json:"failed_retries"`
	AverageRetries    float64       `json:"average_retries"`
	MaxRetries        int           `json:"max_retries"`
	AverageDelay      time.Duration `json:"average_delay"`
	TotalRetryTime    time.Duration `json:"total_retry_time"`
}

// FallbackStatistics 后备统计
type FallbackStatistics struct {
	TotalUsage      int           `json:"total_usage"`
	SuccessfulUsage int           `json:"successful_usage"`
	FailedUsage     int           `json:"failed_usage"`
	SuccessRate     float64       `json:"success_rate"`
	AverageTime     time.Duration `json:"average_time"`
	TotalTime       time.Duration `json:"total_time"`
}

// StrategyStats 策略统计
type StrategyStats struct {
	UsageCount   int           `json:"usage_count"`
	SuccessCount int           `json:"success_count"`
	FailureCount int           `json:"failure_count"`
	SuccessRate  float64       `json:"success_rate"`
	AverageTime  time.Duration `json:"average_time"`
	LastUsed     time.Time     `json:"last_used"`
}

// ErrorTrendAnalysis 错误趋势分析
type ErrorTrendAnalysis struct {
	PeriodStart           time.Time `json:"period_start"`
	PeriodEnd             time.Time `json:"period_end"`
	ErrorRateChange       float64   `json:"error_rate_change"`       // 错误率变化
	RecoveryRateChange    float64   `json:"recovery_rate_change"`    // 恢复率变化
	CommonErrorPatterns   []string  `json:"common_error_patterns"`   // 常见错误模式
	MostEffectiveStrategy string    `json:"most_effective_strategy"` // 最有效策略
	RecommendedActions    []string  `json:"recommended_actions"`     // 推荐动作
}

// 函数类型定义
type RecoveryHandler func(context.Context, *ErrorRecord) (*RecoveryResult, error)
type FallbackHandler func(context.Context, *FallbackAction, map[string]interface{}) (*FallbackResult, error)

// RecoveryResult 恢复结果
type RecoveryResult struct {
	Success        bool                   `json:"success"`
	Strategy       string                 `json:"strategy"`
	ActionsTaken   []string               `json:"actions_taken"`
	TimeTaken      time.Duration          `json:"time_taken"`
	FinalState     string                 `json:"final_state"`
	Details        map[string]interface{} `json:"details"`
	Recommendation string                 `json:"recommendation"`
}

// FallbackResult 后备结果
type FallbackResult struct {
	Success      bool                   `json:"success"`
	FallbackType FallbackType           `json:"fallback_type"`
	OutputPath   string                 `json:"output_path"`
	QualityLevel string                 `json:"quality_level"`
	TimeTaken    time.Duration          `json:"time_taken"`
	Details      map[string]interface{} `json:"details"`
}

// 枚举定义
type ErrorType int
type ErrorSeverity int
type ActionType int
type FallbackType int

const (
	// 错误类型
	ErrorTypeFileCorrupted     ErrorType = iota // 文件损坏
	ErrorTypeFormatUnsupported                  // 格式不支持
	ErrorTypeMemoryExhausted                    // 内存耗尽
	ErrorTypeProcessTimeout                     // 处理超时
	ErrorTypePermissionDenied                   // 权限拒绝
	ErrorTypeNetworkFailure                     // 网络失败
	ErrorTypeFFmpegCrash                        // FFmpeg崩溃
	ErrorTypeMetadataCorrupted                  // 元数据损坏
	ErrorTypeQualityTooLow                      // 品质过低
	ErrorTypeFileTooLarge                       // 文件过大
	ErrorTypeUnknown                            // 未知错误
)

const (
	// 错误严重程度
	SeverityLow      ErrorSeverity = iota // 低
	SeverityMedium                        // 中
	SeverityHigh                          // 高
	SeverityCritical                      // 关键
)

const (
	// 动作类型
	ActionRetry    ActionType = iota // 重试
	ActionFallback                   // 后备方案
	ActionSkip                       // 跳过
	ActionAbort                      // 中止
)

const (
	// 后备类型
	FallbackLowerQuality     FallbackType = iota // 降低品质
	FallbackDifferentFormat                      // 不同格式
	FallbackSimpleConversion                     // 简单转换
	FallbackCopyOriginal                         // 复制原文件
	FallbackSkipFile                             // 跳过文件
)

// NewErrorRecoveryManager 创建错误恢复管理器
func NewErrorRecoveryManager(logger *zap.Logger, config *RecoveryConfig) *ErrorRecoveryManager {
	if config == nil {
		config = &RecoveryConfig{
			MaxRetries:          3,
			BaseRetryDelay:      1 * time.Second,
			MaxRetryDelay:       30 * time.Second,
			BackoffMultiplier:   2.0,
			EnableFallback:      true,
			EnableLearning:      true,
			HistoryLimit:        1000,
			RecoveryTimeout:     5 * time.Minute,
			CriticalErrorAction: ActionFallback,
			AutoRecoveryEnabled: true,
		}
	}

	manager := &ErrorRecoveryManager{
		logger:             logger,
		config:             config,
		errorStats:         &ErrorStatistics{},
		recoveryStrategies: make(map[ErrorType]*RecoveryStrategy),
		errorHistory:       make([]ErrorRecord, 0),
		enabled:            true,
	}

	// 初始化子管理器
	manager.retryManager = NewRetryManager(logger)
	manager.fallbackManager = NewFallbackManager(logger)

	// 初始化默认恢复策略
	manager.initializeDefaultStrategies()

	// 初始化统计信息
	manager.initializeStatistics()

	logger.Info("错误恢复管理器初始化完成",
		zap.Int("max_retries", config.MaxRetries),
		zap.Duration("base_retry_delay", config.BaseRetryDelay),
		zap.Bool("auto_recovery_enabled", config.AutoRecoveryEnabled))

	return manager
}

// HandleError 处理错误 - README核心功能：智能错误恢复
func (erm *ErrorRecoveryManager) HandleError(ctx context.Context, err error, operation, filePath string) (*RecoveryResult, error) {
	if !erm.enabled {
		return nil, err
	}

	startTime := time.Now()

	// 分析错误类型
	errorType, severity := erm.analyzeError(err)

	// 创建错误记录
	errorRecord := &ErrorRecord{
		ID:           erm.generateErrorID(),
		ErrorType:    errorType,
		ErrorMessage: err.Error(),
		SourceFile:   filePath,
		Operation:    operation,
		Timestamp:    startTime,
		Severity:     severity,
		Context:      make(map[string]interface{}),
	}

	erm.logger.Info("处理错误",
		zap.String("error_id", errorRecord.ID),
		zap.String("error_type", errorType.String()),
		zap.String("severity", severity.String()),
		zap.String("operation", operation),
		zap.String("file", filePath))

	// 检查是否为关键错误
	if severity == SeverityCritical {
		return erm.handleCriticalError(ctx, errorRecord)
	}

	// 尝试恢复
	result, err := erm.attemptRecovery(ctx, errorRecord)

	// 更新错误记录
	errorRecord.RecoveryAttempted = true
	errorRecord.RecoverySuccess = result != nil && result.Success
	errorRecord.TotalRecoveryTime = time.Since(startTime)

	if result != nil {
		errorRecord.RecoveryStrategy = result.Strategy
		errorRecord.Resolution = result.FinalState
	}

	// 记录错误
	erm.recordError(errorRecord)

	// 更新统计信息
	erm.updateStatistics(errorRecord, result)

	if result != nil && result.Success {
		erm.logger.Info("错误恢复成功",
			zap.String("error_id", errorRecord.ID),
			zap.String("strategy", result.Strategy),
			zap.Duration("recovery_time", errorRecord.TotalRecoveryTime))
		return result, nil
	}

	erm.logger.Error("错误恢复失败",
		zap.String("error_id", errorRecord.ID),
		zap.String("error_type", errorType.String()),
		zap.Duration("recovery_time", errorRecord.TotalRecoveryTime))

	return result, err
}

// attemptRecovery 尝试恢复 - README核心功能：多级后备转换策略
func (erm *ErrorRecoveryManager) attemptRecovery(ctx context.Context, errorRecord *ErrorRecord) (*RecoveryResult, error) {
	strategy, exists := erm.recoveryStrategies[errorRecord.ErrorType]
	if !exists {
		return erm.fallbackRecovery(ctx, errorRecord)
	}

	erm.logger.Debug("开始恢复尝试",
		zap.String("error_id", errorRecord.ID),
		zap.String("strategy", strategy.Name))

	// 第一阶段：重试机制
	if strategy.MaxAttempts > 0 {
		retryResult, err := erm.retryManager.AttemptRetry(ctx, errorRecord, strategy)
		if err == nil && retryResult {
			return &RecoveryResult{
				Success:      true,
				Strategy:     strategy.Name,
				ActionsTaken: []string{"重试成功"},
				TimeTaken:    time.Since(errorRecord.Timestamp),
				FinalState:   "重试解决",
			}, nil
		}
	}

	// 第二阶段：后备方案
	if erm.config.EnableFallback && len(strategy.FallbackActions) > 0 {
		fallbackResult, err := erm.fallbackManager.ExecuteFallback(ctx, strategy.FallbackActions, errorRecord)
		if err == nil && fallbackResult != nil && fallbackResult.Success {
			return &RecoveryResult{
				Success:      true,
				Strategy:     strategy.Name,
				ActionsTaken: []string{fmt.Sprintf("后备方案: %s", fallbackResult.FallbackType.String())},
				TimeTaken:    fallbackResult.TimeTaken,
				FinalState:   "后备方案解决",
				Details:      fallbackResult.Details,
			}, nil
		}
	}

	// 第三阶段：通用后备恢复
	return erm.fallbackRecovery(ctx, errorRecord)
}

// 字符串方法
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeFileCorrupted:
		return "file_corrupted"
	case ErrorTypeFormatUnsupported:
		return "format_unsupported"
	case ErrorTypeMemoryExhausted:
		return "memory_exhausted"
	case ErrorTypeProcessTimeout:
		return "process_timeout"
	case ErrorTypePermissionDenied:
		return "permission_denied"
	case ErrorTypeNetworkFailure:
		return "network_failure"
	case ErrorTypeFFmpegCrash:
		return "ffmpeg_crash"
	case ErrorTypeMetadataCorrupted:
		return "metadata_corrupted"
	case ErrorTypeQualityTooLow:
		return "quality_too_low"
	case ErrorTypeFileTooLarge:
		return "file_too_large"
	case ErrorTypeUnknown:
		return "unknown"
	default:
		return "undefined"
	}
}

func (es ErrorSeverity) String() string {
	switch es {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "undefined"
	}
}

func (ft FallbackType) String() string {
	switch ft {
	case FallbackLowerQuality:
		return "lower_quality"
	case FallbackDifferentFormat:
		return "different_format"
	case FallbackSimpleConversion:
		return "simple_conversion"
	case FallbackCopyOriginal:
		return "copy_original"
	case FallbackSkipFile:
		return "skip_file"
	default:
		return "undefined"
	}
}
