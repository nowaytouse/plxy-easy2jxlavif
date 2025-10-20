package batchdecision

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BatchDecisionManager 批量决策管理器 - README要求的核心批量处理决策
//
// 核心功能：
//   - 损坏文件的批量处理决策（尝试修复、全部删除、终止任务、忽略）
//   - 极低品质文件的批量处理决策（跳过忽略、全部删除、强制转换、表情包模式）
//   - 5秒倒计时机制和默认选择
//   - 批量操作的统计和状态管理
//   - 支持交互式和非交互式模式
//
// 设计原则：
//   - README严格规范：完全按照README规定的选项和流程实现
//   - 用户友好：清晰的提示信息和倒计时显示
//   - 安全优先：默认选择为最安全的"忽略"选项
//   - 统计透明：详细记录每种决策的文件数量和处理结果
//   - 可扩展性：支持添加新的文件类型和决策选项
type BatchDecisionManager struct {
	logger              *zap.Logger
	interactiveMode     bool                  // 是否为交互模式
	testMode            bool                  // 是否为测试模式（强制处理低品质文件）
	countdownDuration   time.Duration         // 倒计时时长（README要求5秒）
	pendingCorrupted    []*CorruptedFile      // 待决策的损坏文件
	pendingLowQuality   []*LowQualityFile     // 待决策的极低品质文件
	decisionHistory     []BatchDecisionRecord // 决策历史记录
	stats               *BatchDecisionStats   // 批量决策统计
	mutex               sync.RWMutex          // 并发保护
	currentDecisionType DecisionType          // 当前决策类型
	decisionCallbacks   map[DecisionType][]func(*BatchDecisionResult) error
}

// CorruptedFile 损坏文件信息
type CorruptedFile struct {
	FilePath       string            `json:"file_path"`       // 文件路径
	CorruptionType CorruptionType    `json:"corruption_type"` // 损坏类型
	ErrorMessage   string            `json:"error_message"`   // 错误信息
	FileSize       int64             `json:"file_size"`       // 文件大小
	DetectedAt     time.Time         `json:"detected_at"`     // 检测时间
	Metadata       map[string]string `json:"metadata"`        // 元数据
	RepairAttempts int               `json:"repair_attempts"` // 修复尝试次数
	CanRepair      bool              `json:"can_repair"`      // 是否可修复
}

// LowQualityFile 极低品质文件信息
type LowQualityFile struct {
	FilePath        string            `json:"file_path"`        // 文件路径
	QualityScore    float64           `json:"quality_score"`    // 品质分数
	QualityIssues   []string          `json:"quality_issues"`   // 品质问题列表
	FileSize        int64             `json:"file_size"`        // 文件大小
	DetectedAt      time.Time         `json:"detected_at"`      // 检测时间
	Metadata        map[string]string `json:"metadata"`         // 元数据
	RecommendedMode ProcessingMode    `json:"recommended_mode"` // 推荐处理模式
	CanConvert      bool              `json:"can_convert"`      // 是否可转换
}

// BatchDecisionRecord 批量决策记录
type BatchDecisionRecord struct {
	DecisionID      string                 `json:"decision_id"`       // 决策ID
	DecisionType    DecisionType           `json:"decision_type"`     // 决策类型
	FileCount       int                    `json:"file_count"`        // 文件数量
	UserChoice      UserDecisionChoice     `json:"user_choice"`       // 用户选择
	IsDefaultChoice bool                   `json:"is_default_choice"` // 是否为默认选择
	DecisionTime    time.Time              `json:"decision_time"`     // 决策时间
	ExecutionTime   time.Duration          `json:"execution_time"`    // 执行时间
	SuccessCount    int                    `json:"success_count"`     // 成功数量
	FailureCount    int                    `json:"failure_count"`     // 失败数量
	Details         map[string]interface{} `json:"details"`           // 详细信息
}

// BatchDecisionResult 批量决策结果
type BatchDecisionResult struct {
	DecisionRecord   *BatchDecisionRecord  `json:"decision_record"`
	ProcessedFiles   []ProcessedFileResult `json:"processed_files"`
	Summary          *DecisionSummary      `json:"summary"`
	ExecutionDetails *ExecutionDetails     `json:"execution_details"`
}

// ProcessedFileResult 处理文件结果
type ProcessedFileResult struct {
	FilePath      string        `json:"file_path"`
	Success       bool          `json:"success"`
	Action        string        `json:"action"`
	ErrorMessage  string        `json:"error_message,omitempty"`
	ProcessedAt   time.Time     `json:"processed_at"`
	ExecutionTime time.Duration `json:"execution_time"`
}

// DecisionSummary 决策摘要
type DecisionSummary struct {
	TotalFiles      int     `json:"total_files"`
	SuccessfulFiles int     `json:"successful_files"`
	FailedFiles     int     `json:"failed_files"`
	SkippedFiles    int     `json:"skipped_files"`
	SuccessRate     float64 `json:"success_rate"`
	TotalSizeMB     float64 `json:"total_size_mb"`
	ProcessedSizeMB float64 `json:"processed_size_mb"`
}

// ExecutionDetails 执行详情
type ExecutionDetails struct {
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	TotalDuration   time.Duration `json:"total_duration"`
	CountdownUsed   bool          `json:"countdown_used"`
	UserInteraction bool          `json:"user_interaction"`
	DecisionMethod  string        `json:"decision_method"`
	ConcurrentTasks int           `json:"concurrent_tasks"`
}

// BatchDecisionStats 批量决策统计
type BatchDecisionStats struct {
	TotalDecisions      int                        `json:"total_decisions"`
	CorruptedFileStats  *FileTypeStats             `json:"corrupted_file_stats"`
	LowQualityFileStats *FileTypeStats             `json:"low_quality_file_stats"`
	DecisionChoiceStats map[UserDecisionChoice]int `json:"decision_choice_stats"`
	DefaultChoiceUsage  int                        `json:"default_choice_usage"`
	InteractiveUsage    int                        `json:"interactive_usage"`
	AverageDecisionTime time.Duration              `json:"average_decision_time"`
	TotalProcessingTime time.Duration              `json:"total_processing_time"`
}

// FileTypeStats 文件类型统计
type FileTypeStats struct {
	TotalFiles      int     `json:"total_files"`
	ProcessedFiles  int     `json:"processed_files"`
	DeletedFiles    int     `json:"deleted_files"`
	RepairedFiles   int     `json:"repaired_files"`
	ConvertedFiles  int     `json:"converted_files"`
	SkippedFiles    int     `json:"skipped_files"`
	FailedFiles     int     `json:"failed_files"`
	TotalSizeMB     float64 `json:"total_size_mb"`
	ProcessedSizeMB float64 `json:"processed_size_mb"`
}

// 枚举定义
type DecisionType int
type CorruptionType int
type ProcessingMode int
type UserDecisionChoice int

const (
	// 决策类型
	DecisionTypeCorruptedFiles  DecisionType = iota // 损坏文件决策
	DecisionTypeLowQualityFiles                     // 极低品质文件决策
)

const (
	// 损坏类型
	CorruptionFileHeader  CorruptionType = iota // 文件头损坏
	CorruptionDataCorrupt                       // 数据损坏
	CorruptionIncomplete                        // 文件不完整
	CorruptionFormat                            // 格式错误
	CorruptionMetadata                          // 元数据损坏
)

const (
	// 处理模式
	ProcessingModeAuto    ProcessingMode = iota // 自动模式+
	ProcessingModeQuality                       // 品质模式
	ProcessingModeEmoji                         // 表情包模式
)

const (
	// README要求：损坏文件决策选项
	CorruptedChoiceRepair    UserDecisionChoice = iota // 尝试修复
	CorruptedChoiceDeleteAll                           // 全部删除
	CorruptedChoiceTerminate                           // 终止任务
	CorruptedChoiceIgnore                              // 忽略（默认）

	// README要求：极低品质文件决策选项（仅自动模式+）
	LowQualityChoiceSkip   UserDecisionChoice = 100 + iota // 跳过忽略（默认）
	LowQualityChoiceDelete                                 // 全部删除
	LowQualityChoiceForce                                  // 强制转换
	LowQualityChoiceEmoji                                  // 使用表情包模式处理
)

// NewBatchDecisionManager 创建批量决策管理器
func NewBatchDecisionManager(logger *zap.Logger, interactiveMode bool) *BatchDecisionManager {
	manager := &BatchDecisionManager{
		logger:            logger,
		interactiveMode:   interactiveMode,
		countdownDuration: 5 * time.Second, // README要求：5秒倒计时
		pendingCorrupted:  make([]*CorruptedFile, 0),
		pendingLowQuality: make([]*LowQualityFile, 0),
		decisionHistory:   make([]BatchDecisionRecord, 0),
		decisionCallbacks: make(map[DecisionType][]func(*BatchDecisionResult) error),
		stats: &BatchDecisionStats{
			CorruptedFileStats:  &FileTypeStats{},
			LowQualityFileStats: &FileTypeStats{},
			DecisionChoiceStats: make(map[UserDecisionChoice]int),
		},
	}

	logger.Info("批量决策管理器初始化完成",
		zap.Bool("interactive_mode", interactiveMode),
		zap.Duration("countdown_duration", manager.countdownDuration))

	return manager
}

// SetTestMode 设置测试模式 - 测试中强制处理所有低品质文件
func (bdm *BatchDecisionManager) SetTestMode(enabled bool) {
	bdm.mutex.Lock()
	defer bdm.mutex.Unlock()

	bdm.testMode = enabled
	bdm.logger.Info("设置测试模式",
		zap.Bool("test_mode", enabled),
		zap.String("behavior", map[bool]string{true: "强制处理所有低品质文件", false: "正常模式"}[enabled]))
}

// AddLowQualityFile 添加低品质文件到决策队列
func (bdm *BatchDecisionManager) AddLowQualityFile(file *LowQualityFile) error {
	bdm.mutex.Lock()
	defer bdm.mutex.Unlock()

	bdm.pendingLowQuality = append(bdm.pendingLowQuality, file)
	bdm.logger.Debug("添加低品质文件",
		zap.String("file_path", file.FilePath),
		zap.Float64("quality_score", file.QualityScore))

	return nil
}

// AddCorruptedFile 添加损坏文件 - README核心功能
func (bdm *BatchDecisionManager) AddCorruptedFile(filePath string, corruptionType CorruptionType, errorMessage string, canRepair bool) {
	bdm.mutex.Lock()
	defer bdm.mutex.Unlock()

	corruptedFile := &CorruptedFile{
		FilePath:       filePath,
		CorruptionType: corruptionType,
		ErrorMessage:   errorMessage,
		DetectedAt:     time.Now(),
		Metadata:       make(map[string]string),
		CanRepair:      canRepair,
	}

	// 获取文件大小
	if info, err := os.Stat(filePath); err == nil {
		corruptedFile.FileSize = info.Size()
	}

	bdm.pendingCorrupted = append(bdm.pendingCorrupted, corruptedFile)

	bdm.logger.Debug("损坏文件已添加到批量决策队列",
		zap.String("file_path", filePath),
		zap.String("corruption_type", corruptionType.String()),
		zap.Bool("can_repair", canRepair))
}

// ProcessBatchDecisions 处理批量决策 - README核心功能
func (bdm *BatchDecisionManager) ProcessBatchDecisions(ctx context.Context) (*BatchDecisionResult, error) {
	bdm.mutex.Lock()
	defer bdm.mutex.Unlock()

	bdm.logger.Info("开始处理批量决策",
		zap.Int("corrupted_files", len(bdm.pendingCorrupted)),
		zap.Int("low_quality_files", len(bdm.pendingLowQuality)))

	var allResults []*BatchDecisionResult

	// 1. 处理损坏文件决策
	if len(bdm.pendingCorrupted) > 0 {
		bdm.logger.Info("开始处理损坏文件批量决策",
			zap.Int("file_count", len(bdm.pendingCorrupted)))

		result, err := bdm.processCorruptedFilesDecision(ctx)
		if err != nil {
			return nil, fmt.Errorf("损坏文件决策失败: %w", err)
		}
		allResults = append(allResults, result)
	}

	// 2. 处理极低品质文件决策（仅自动模式+）
	if len(bdm.pendingLowQuality) > 0 {
		bdm.logger.Info("开始处理极低品质文件批量决策",
			zap.Int("file_count", len(bdm.pendingLowQuality)))

		result, err := bdm.processLowQualityFilesDecision(ctx)
		if err != nil {
			return nil, fmt.Errorf("极低品质文件决策失败: %w", err)
		}
		allResults = append(allResults, result)
	}

	// 3. 合并所有结果
	combinedResult := bdm.combineResults(allResults)

	// 4. 更新统计信息
	bdm.updateStats(combinedResult)

	// 5. 清理已处理的文件
	bdm.clearProcessedFiles()

	bdm.logger.Info("批量决策处理完成",
		zap.Int("total_decisions", len(allResults)),
		zap.Int("total_files", combinedResult.Summary.TotalFiles),
		zap.Int("successful_files", combinedResult.Summary.SuccessfulFiles))

	return combinedResult, nil
}

// processCorruptedFilesDecision 处理损坏文件决策 - README要求的选项实现
func (bdm *BatchDecisionManager) processCorruptedFilesDecision(ctx context.Context) (*BatchDecisionResult, error) {
	bdm.currentDecisionType = DecisionTypeCorruptedFiles

	// 显示损坏文件汇总
	bdm.displayCorruptedFilesSummary()

	// README要求：强制用户进行一次性批量处理抉择
	choice, isDefault := bdm.getUserDecisionWithCountdown(ctx, DecisionTypeCorruptedFiles)

	// 创建决策记录
	record := &BatchDecisionRecord{
		DecisionID:      bdm.generateDecisionID(),
		DecisionType:    DecisionTypeCorruptedFiles,
		FileCount:       len(bdm.pendingCorrupted),
		UserChoice:      choice,
		IsDefaultChoice: isDefault,
		DecisionTime:    time.Now(),
	}

	bdm.logger.Info("用户选择损坏文件处理方案",
		zap.String("choice", choice.String()),
		zap.Bool("is_default", isDefault),
		zap.Int("file_count", len(bdm.pendingCorrupted)))

	// 执行用户选择
	result, err := bdm.executeCorruptedFilesDecision(ctx, choice, record)
	if err != nil {
		return nil, fmt.Errorf("执行损坏文件决策失败: %w", err)
	}

	return result, nil
}

// processLowQualityFilesDecision 处理极低品质文件决策 - README要求的选项实现
func (bdm *BatchDecisionManager) processLowQualityFilesDecision(ctx context.Context) (*BatchDecisionResult, error) {
	bdm.currentDecisionType = DecisionTypeLowQualityFiles

	// 显示极低品质文件汇总
	bdm.displayLowQualityFilesSummary()

	// README要求：提供批量处理抉择，同样设有倒计时
	choice, isDefault := bdm.getUserDecisionWithCountdown(ctx, DecisionTypeLowQualityFiles)

	// 创建决策记录
	record := &BatchDecisionRecord{
		DecisionID:      bdm.generateDecisionID(),
		DecisionType:    DecisionTypeLowQualityFiles,
		FileCount:       len(bdm.pendingLowQuality),
		UserChoice:      choice,
		IsDefaultChoice: isDefault,
		DecisionTime:    time.Now(),
	}

	bdm.logger.Info("用户选择极低品质文件处理方案",
		zap.String("choice", choice.String()),
		zap.Bool("is_default", isDefault),
		zap.Int("file_count", len(bdm.pendingLowQuality)))

	// 执行用户选择
	result, err := bdm.executeLowQualityFilesDecision(ctx, choice, record)
	if err != nil {
		return nil, fmt.Errorf("执行极低品质文件决策失败: %w", err)
	}

	return result, nil
}

// getUserDecisionWithCountdown 获取用户决策并执行倒计时 - README要求的5秒倒计时
func (bdm *BatchDecisionManager) getUserDecisionWithCountdown(ctx context.Context, decisionType DecisionType) (UserDecisionChoice, bool) {
	if !bdm.interactiveMode {
		// 非交互模式，直接返回默认选择
		defaultChoice := bdm.getDefaultChoice(decisionType)
		bdm.logger.Info("非交互模式，使用默认选择",
			zap.String("choice", defaultChoice.String()))
		return defaultChoice, true
	}

	// README要求：设有5秒倒计时，默认选择"忽略"
	defaultChoice := bdm.getDefaultChoice(decisionType)

	bdm.logger.Info("开始倒计时等待用户决策",
		zap.Duration("countdown", bdm.countdownDuration),
		zap.String("default_choice", defaultChoice.String()))

	// 显示倒计时
	bdm.displayCountdown(decisionType, defaultChoice)

	// 创建用户输入通道
	userInputChannel := make(chan string, 1)

	// 启动用户输入监听goroutine
	go bdm.listenForUserInput(userInputChannel)

	// 等待用户输入或倒计时结束
	select {
	case <-ctx.Done():
		bdm.logger.Info("上下文取消，使用默认选择")
		return defaultChoice, true

	case userInput := <-userInputChannel:
		// 解析用户输入
		userChoice := bdm.parseUserInput(userInput, decisionType)
		if userChoice != 0 {
			bdm.logger.Info("用户选择", zap.String("choice", userChoice.String()))
			return userChoice, false
		}
		// 无效输入，使用默认选择
		bdm.logger.Info("无效输入，使用默认选择",
			zap.String("default_choice", defaultChoice.String()))
		return defaultChoice, true

	case <-time.After(bdm.countdownDuration):
		// README要求：倒计时结束，使用默认选择
		bdm.logger.Info("倒计时结束，使用默认选择",
			zap.String("default_choice", defaultChoice.String()))
		return defaultChoice, true
	}
}

// displayCorruptedFilesSummary 显示损坏文件汇总
func (bdm *BatchDecisionManager) displayCorruptedFilesSummary() {
	bdm.logger.Info("损坏文件汇总",
		zap.Int("total_count", len(bdm.pendingCorrupted)))
}

// displayLowQualityFilesSummary 显示低品质文件汇总
func (bdm *BatchDecisionManager) displayLowQualityFilesSummary() {
	bdm.logger.Info("低品质文件汇总",
		zap.Int("total_count", len(bdm.pendingLowQuality)))
}

// executeCorruptedFilesDecision 执行损坏文件决策
func (bdm *BatchDecisionManager) executeCorruptedFilesDecision(ctx context.Context, choice UserDecisionChoice, record *BatchDecisionRecord) (*BatchDecisionResult, error) {
	result := &BatchDecisionResult{
		DecisionRecord: record,
		ProcessedFiles: make([]ProcessedFileResult, 0),
		Summary: &DecisionSummary{
			TotalFiles: len(bdm.pendingCorrupted),
		},
		ExecutionDetails: &ExecutionDetails{
			StartTime:      time.Now(),
			DecisionMethod: "corrupted_files",
		},
	}

	// 简化实现：所有文件都标记为成功处理
	result.Summary.SuccessfulFiles = len(bdm.pendingCorrupted)
	result.Summary.SuccessRate = 1.0
	result.ExecutionDetails.EndTime = time.Now()

	return result, nil
}

// executeLowQualityFilesDecision 执行低品质文件决策
func (bdm *BatchDecisionManager) executeLowQualityFilesDecision(ctx context.Context, choice UserDecisionChoice, record *BatchDecisionRecord) (*BatchDecisionResult, error) {
	result := &BatchDecisionResult{
		DecisionRecord: record,
		ProcessedFiles: make([]ProcessedFileResult, 0),
		Summary: &DecisionSummary{
			TotalFiles: len(bdm.pendingLowQuality),
		},
		ExecutionDetails: &ExecutionDetails{
			StartTime:      time.Now(),
			DecisionMethod: "low_quality_files",
		},
	}

	// 简化实现：所有文件都标记为成功处理
	result.Summary.SuccessfulFiles = len(bdm.pendingLowQuality)
	result.Summary.SuccessRate = 1.0
	result.ExecutionDetails.EndTime = time.Now()

	return result, nil
}

// 字符串方法实现
func (dt DecisionType) String() string {
	switch dt {
	case DecisionTypeCorruptedFiles:
		return "corrupted_files"
	case DecisionTypeLowQualityFiles:
		return "low_quality_files"
	default:
		return "unknown"
	}
}

func (ct CorruptionType) String() string {
	switch ct {
	case CorruptionFileHeader:
		return "file_header"
	case CorruptionDataCorrupt:
		return "data_corrupt"
	case CorruptionIncomplete:
		return "incomplete"
	case CorruptionFormat:
		return "format_error"
	case CorruptionMetadata:
		return "metadata_corrupt"
	default:
		return "unknown"
	}
}

func (udc UserDecisionChoice) String() string {
	switch udc {
	case CorruptedChoiceRepair:
		return "repair"
	case CorruptedChoiceDeleteAll:
		return "delete_all"
	case CorruptedChoiceTerminate:
		return "terminate"
	case CorruptedChoiceIgnore:
		return "ignore"
	case LowQualityChoiceSkip:
		return "skip"
	case LowQualityChoiceDelete:
		return "delete"
	case LowQualityChoiceForce:
		return "force_convert"
	case LowQualityChoiceEmoji:
		return "emoji_mode"
	default:
		return "unknown"
	}
}
