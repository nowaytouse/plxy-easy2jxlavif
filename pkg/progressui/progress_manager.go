package progressui

import (
	"fmt"
	"sync"
	"time"

	"github.com/vbauerster/mpb/v8"
	"go.uber.org/zap"
)

// ProgressManager 进度管理器 - README要求的实时精确进度显示
//
// 核心功能：
//   - 使用mpb/v8实现高性能进度条显示
//   - 支持多任务并发进度跟踪
//   - 实时显示处理速度、预估剩余时间
//   - 支持任务暂停、恢复、取消操作
//   - 提供详细的进度统计和分析
//
// 设计原则：
//   - 用户友好：直观的进度显示和状态提示
//   - 性能优化：最小化对主要处理流程的影响
//   - 精确追踪：精确到文件级别的进度监控
//   - 实时响应：即时更新进度和状态变化
//   - 可定制化：支持不同风格和布局的进度显示
type ProgressManager struct {
	logger        *zap.Logger
	container     *mpb.Progress
	trackers      map[string]*ProgressTracker
	trackersMutex sync.RWMutex
	enabled       bool
	config        *ProgressConfig
	stats         *ProgressStats
	callbacks     *ProgressCallbacks
}

// ProgressTracker 单个进度跟踪器
type ProgressTracker struct {
	ID                string                 `json:"id"`                  // 跟踪器ID
	Name              string                 `json:"name"`                // 显示名称
	TaskType          TaskType               `json:"task_type"`           // 任务类型
	Bar               *mpb.Bar               `json:"-"`                   // mpb进度条
	TotalItems        int64                  `json:"total_items"`         // 总项目数
	ProcessedItems    int64                  `json:"processed_items"`     // 已处理项目数
	SuccessfulItems   int64                  `json:"successful_items"`    // 成功项目数
	FailedItems       int64                  `json:"failed_items"`        // 失败项目数
	SkippedItems      int64                  `json:"skipped_items"`       // 跳过项目数
	StartTime         time.Time              `json:"start_time"`          // 开始时间
	LastUpdate        time.Time              `json:"last_update"`         // 最后更新时间
	Status            ProgressStatus         `json:"status"`              // 进度状态
	Speed             float64                `json:"speed"`               // 处理速度（项目/秒）
	EstimatedTimeLeft time.Duration          `json:"estimated_time_left"` // 预估剩余时间
	Metadata          map[string]interface{} `json:"metadata"`            // 元数据
	Paused            bool                   `json:"paused"`              // 是否暂停
	Cancelled         bool                   `json:"cancelled"`           // 是否取消
}

// ProgressConfig 进度配置
type ProgressConfig struct {
	EnableRealTimeStats    bool          `json:"enable_realtime_stats"`    // 启用实时统计
	EnableSpeedCalculation bool          `json:"enable_speed_calculation"` // 启用速度计算
	UpdateInterval         time.Duration `json:"update_interval"`          // 更新间隔
	BarWidth               int           `json:"bar_width"`                // 进度条宽度
	ShowPercentage         bool          `json:"show_percentage"`          // 显示百分比
	ShowSpeed              bool          `json:"show_speed"`               // 显示速度
	ShowETA                bool          `json:"show_eta"`                 // 显示预估时间
	ShowElapsed            bool          `json:"show_elapsed"`             // 显示已用时间
	Theme                  ProgressTheme `json:"theme"`                    // 进度条主题
	LogLevel               LogLevel      `json:"log_level"`                // 日志级别
}

// ProgressStats 进度统计
type ProgressStats struct {
	TotalTrackers      int           `json:"total_trackers"`      // 总跟踪器数
	ActiveTrackers     int           `json:"active_trackers"`     // 活跃跟踪器数
	CompletedTrackers  int           `json:"completed_trackers"`  // 完成跟踪器数
	FailedTrackers     int           `json:"failed_trackers"`     // 失败跟踪器数
	TotalItems         int64         `json:"total_items"`         // 总项目数
	ProcessedItems     int64         `json:"processed_items"`     // 已处理项目数
	SuccessfulItems    int64         `json:"successful_items"`    // 成功项目数
	FailedItems        int64         `json:"failed_items"`        // 失败项目数
	SkippedItems       int64         `json:"skipped_items"`       // 跳过项目数
	AverageSpeed       float64       `json:"average_speed"`       // 平均速度
	PeakSpeed          float64       `json:"peak_speed"`          // 峰值速度
	TotalElapsed       time.Duration `json:"total_elapsed"`       // 总耗时
	EstimatedRemaining time.Duration `json:"estimated_remaining"` // 预估剩余时间
}

// ProgressCallbacks 进度回调
type ProgressCallbacks struct {
	OnStart       func(*ProgressTracker)                   `json:"-"` // 开始回调
	OnUpdate      func(*ProgressTracker, int64)            `json:"-"` // 更新回调
	OnComplete    func(*ProgressTracker, *ProgressResult)  `json:"-"` // 完成回调
	OnError       func(*ProgressTracker, error)            `json:"-"` // 错误回调
	OnPause       func(*ProgressTracker)                   `json:"-"` // 暂停回调
	OnResume      func(*ProgressTracker)                   `json:"-"` // 恢复回调
	OnCancel      func(*ProgressTracker)                   `json:"-"` // 取消回调
	OnSpeedChange func(*ProgressTracker, float64, float64) `json:"-"` // 速度变化回调
}

// ProgressResult 进度结果
type ProgressResult struct {
	TrackerID       string        `json:"tracker_id"`
	TotalItems      int64         `json:"total_items"`
	ProcessedItems  int64         `json:"processed_items"`
	SuccessfulItems int64         `json:"successful_items"`
	FailedItems     int64         `json:"failed_items"`
	SkippedItems    int64         `json:"skipped_items"`
	TotalTime       time.Duration `json:"total_time"`
	AverageSpeed    float64       `json:"average_speed"`
	Success         bool          `json:"success"`
	ErrorMessage    string        `json:"error_message,omitempty"`
}

// 枚举定义
type TaskType int
type ProgressStatus int
type ProgressTheme int
type LogLevel int

const (
	// 任务类型
	TaskTypeFileScanning       TaskType = iota // 文件扫描
	TaskTypeFileProcessing                     // 文件处理
	TaskTypeFileConversion                     // 文件转换
	TaskTypeMetadataExtraction                 // 元数据提取
	TaskTypeQualityAnalysis                    // 品质分析
	TaskTypeBatchOperation                     // 批量操作
	TaskTypeBackup                             // 备份操作
	TaskTypeCleanup                            // 清理操作
)

const (
	// 进度状态
	StatusIdle      ProgressStatus = iota // 空闲
	StatusRunning                         // 运行中
	StatusPaused                          // 暂停
	StatusCompleted                       // 完成
	StatusFailed                          // 失败
	StatusCancelled                       // 取消
)

const (
	// 进度条主题
	ThemeDefault  ProgressTheme = iota // 默认主题
	ThemeMinimal                       // 简约主题
	ThemeDetailed                      // 详细主题
	ThemeColorful                      // 彩色主题
)

const (
	// 日志级别
	LogLevelNone  LogLevel = iota // 无日志
	LogLevelError                 // 仅错误
	LogLevelWarn                  // 警告及以上
	LogLevelInfo                  // 信息及以上
	LogLevelDebug                 // 调试及以上
)

// NewProgressManager 创建进度管理器
func NewProgressManager(logger *zap.Logger, config *ProgressConfig) *ProgressManager {
	if config == nil {
		config = &ProgressConfig{
			EnableRealTimeStats:    true,
			EnableSpeedCalculation: true,
			UpdateInterval:         100 * time.Millisecond,
			BarWidth:               50,
			ShowPercentage:         true,
			ShowSpeed:              true,
			ShowETA:                true,
			ShowElapsed:            true,
			Theme:                  ThemeDefault,
			LogLevel:               LogLevelInfo,
		}
	}

	// 创建mpb容器
	container := mpb.New(
		mpb.WithWidth(config.BarWidth),
		mpb.WithRefreshRate(config.UpdateInterval),
	)

	manager := &ProgressManager{
		logger:    logger,
		container: container,
		trackers:  make(map[string]*ProgressTracker),
		enabled:   true,
		config:    config,
		stats:     &ProgressStats{},
		callbacks: &ProgressCallbacks{},
	}

	logger.Info("进度管理器初始化完成",
		zap.Int("bar_width", config.BarWidth),
		zap.Duration("update_interval", config.UpdateInterval),
		zap.String("theme", config.Theme.String()))

	return manager
}

// CreateTracker 创建进度跟踪器 - README核心功能
func (pm *ProgressManager) CreateTracker(id, name string, taskType TaskType, totalItems int64) (*ProgressTracker, error) {
	pm.trackersMutex.Lock()
	defer pm.trackersMutex.Unlock()

	if _, exists := pm.trackers[id]; exists {
		return nil, fmt.Errorf("跟踪器ID已存在: %s", id)
	}

	// 创建进度条装饰器
	decorators := pm.createDecorators(name, taskType)

	// 创建mpb进度条
	bar := pm.container.AddBar(totalItems,
		mpb.PrependDecorators(decorators.prepend...),
		mpb.AppendDecorators(decorators.append...),
	)

	tracker := &ProgressTracker{
		ID:                id,
		Name:              name,
		TaskType:          taskType,
		Bar:               bar,
		TotalItems:        totalItems,
		ProcessedItems:    0,
		SuccessfulItems:   0,
		FailedItems:       0,
		SkippedItems:      0,
		StartTime:         time.Now(),
		LastUpdate:        time.Now(),
		Status:            StatusIdle,
		Speed:             0,
		EstimatedTimeLeft: 0,
		Metadata:          make(map[string]interface{}),
		Paused:            false,
		Cancelled:         false,
	}

	pm.trackers[id] = tracker
	pm.updateGlobalStats()

	pm.logger.Info("创建进度跟踪器",
		zap.String("id", id),
		zap.String("name", name),
		zap.String("task_type", taskType.String()),
		zap.Int64("total_items", totalItems))

	// 调用开始回调
	if pm.callbacks.OnStart != nil {
		pm.callbacks.OnStart(tracker)
	}

	return tracker, nil
}

// UpdateProgress 更新进度 - README核心功能：实时精确进度显示
func (pm *ProgressManager) UpdateProgress(trackerID string, increment int64, success bool) error {
	pm.trackersMutex.Lock()
	defer pm.trackersMutex.Unlock()

	tracker, exists := pm.trackers[trackerID]
	if !exists {
		return fmt.Errorf("跟踪器不存在: %s", trackerID)
	}

	if tracker.Cancelled || tracker.Status == StatusCompleted {
		return fmt.Errorf("跟踪器已完成或取消: %s", trackerID)
	}

	// 更新处理计数
	tracker.ProcessedItems += increment
	if success {
		tracker.SuccessfulItems += increment
	} else {
		tracker.FailedItems += increment
	}

	// 更新mpb进度条
	tracker.Bar.IncrBy(int(increment))

	// 计算速度和预估时间
	pm.calculateSpeed(tracker)
	pm.calculateETA(tracker)

	// 更新状态
	if tracker.Status == StatusIdle {
		tracker.Status = StatusRunning
	}

	tracker.LastUpdate = time.Now()

	// 检查是否完成
	if tracker.ProcessedItems >= tracker.TotalItems {
		tracker.Status = StatusCompleted
		pm.completeTracker(tracker)
	}

	pm.updateGlobalStats()

	// 调用更新回调
	if pm.callbacks.OnUpdate != nil {
		pm.callbacks.OnUpdate(tracker, increment)
	}

	pm.logger.Debug("进度更新",
		zap.String("tracker_id", trackerID),
		zap.Int64("increment", increment),
		zap.Bool("success", success),
		zap.Int64("processed", tracker.ProcessedItems),
		zap.Int64("total", tracker.TotalItems),
		zap.Float64("speed", tracker.Speed))

	return nil
}

// SkipItems 跳过项目
func (pm *ProgressManager) SkipItems(trackerID string, skipCount int64) error {
	pm.trackersMutex.Lock()
	defer pm.trackersMutex.Unlock()

	tracker, exists := pm.trackers[trackerID]
	if !exists {
		return fmt.Errorf("跟踪器不存在: %s", trackerID)
	}

	tracker.SkippedItems += skipCount
	tracker.ProcessedItems += skipCount

	// 更新mpb进度条
	tracker.Bar.IncrBy(int(skipCount))

	// 更新计算
	pm.calculateSpeed(tracker)
	pm.calculateETA(tracker)

	tracker.LastUpdate = time.Now()

	// 检查是否完成
	if tracker.ProcessedItems >= tracker.TotalItems {
		tracker.Status = StatusCompleted
		pm.completeTracker(tracker)
	}

	pm.updateGlobalStats()

	pm.logger.Debug("跳过项目",
		zap.String("tracker_id", trackerID),
		zap.Int64("skip_count", skipCount),
		zap.Int64("total_skipped", tracker.SkippedItems))

	return nil
}

// PauseTracker 暂停跟踪器
func (pm *ProgressManager) PauseTracker(trackerID string) error {
	pm.trackersMutex.Lock()
	defer pm.trackersMutex.Unlock()

	tracker, exists := pm.trackers[trackerID]
	if !exists {
		return fmt.Errorf("跟踪器不存在: %s", trackerID)
	}

	if tracker.Status != StatusRunning {
		return fmt.Errorf("跟踪器状态不允许暂停: %s", tracker.Status.String())
	}

	tracker.Paused = true
	tracker.Status = StatusPaused

	// 调用暂停回调
	if pm.callbacks.OnPause != nil {
		pm.callbacks.OnPause(tracker)
	}

	pm.logger.Info("暂停进度跟踪器", zap.String("tracker_id", trackerID))
	return nil
}

// ResumeTracker 恢复跟踪器
func (pm *ProgressManager) ResumeTracker(trackerID string) error {
	pm.trackersMutex.Lock()
	defer pm.trackersMutex.Unlock()

	tracker, exists := pm.trackers[trackerID]
	if !exists {
		return fmt.Errorf("跟踪器不存在: %s", trackerID)
	}

	if tracker.Status != StatusPaused {
		return fmt.Errorf("跟踪器状态不允许恢复: %s", tracker.Status.String())
	}

	tracker.Paused = false
	tracker.Status = StatusRunning

	// 调用恢复回调
	if pm.callbacks.OnResume != nil {
		pm.callbacks.OnResume(tracker)
	}

	pm.logger.Info("恢复进度跟踪器", zap.String("tracker_id", trackerID))
	return nil
}

// CancelTracker 取消跟踪器
func (pm *ProgressManager) CancelTracker(trackerID string) error {
	pm.trackersMutex.Lock()
	defer pm.trackersMutex.Unlock()

	tracker, exists := pm.trackers[trackerID]
	if !exists {
		return fmt.Errorf("跟踪器不存在: %s", trackerID)
	}

	if tracker.Status == StatusCompleted || tracker.Status == StatusCancelled {
		return fmt.Errorf("跟踪器已完成或取消: %s", tracker.Status.String())
	}

	tracker.Cancelled = true
	tracker.Status = StatusCancelled

	// 完成进度条显示
	tracker.Bar.Abort(false)

	// 调用取消回调
	if pm.callbacks.OnCancel != nil {
		pm.callbacks.OnCancel(tracker)
	}

	pm.updateGlobalStats()

	pm.logger.Info("取消进度跟踪器", zap.String("tracker_id", trackerID))
	return nil
}

// 字符串方法
func (tt TaskType) String() string {
	switch tt {
	case TaskTypeFileScanning:
		return "file_scanning"
	case TaskTypeFileProcessing:
		return "file_processing"
	case TaskTypeFileConversion:
		return "file_conversion"
	case TaskTypeMetadataExtraction:
		return "metadata_extraction"
	case TaskTypeQualityAnalysis:
		return "quality_analysis"
	case TaskTypeBatchOperation:
		return "batch_operation"
	case TaskTypeBackup:
		return "backup"
	case TaskTypeCleanup:
		return "cleanup"
	default:
		return "unknown"
	}
}

func (ps ProgressStatus) String() string {
	switch ps {
	case StatusIdle:
		return "idle"
	case StatusRunning:
		return "running"
	case StatusPaused:
		return "paused"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

func (pt ProgressTheme) String() string {
	switch pt {
	case ThemeDefault:
		return "default"
	case ThemeMinimal:
		return "minimal"
	case ThemeDetailed:
		return "detailed"
	case ThemeColorful:
		return "colorful"
	default:
		return "unknown"
	}
}
