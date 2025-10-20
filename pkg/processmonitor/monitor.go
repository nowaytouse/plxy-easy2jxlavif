package processmonitor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"bufio"

	"go.uber.org/zap"
	"github.com/shirou/gopsutil/v3/process"
)

// ProcessMonitor 进程监控器 - README要求的防卡死机制
//
// 核心功能：
//   - 根据媒体属性动态估算处理时限
//   - 监控子进程活动，防止卡死
//   - 若进程在30秒内无任何有效计算，则判断为卡死
//   - 提供强制终止和用户提示机制
//   - 支持进程组监控和级联终止
//
// 设计原则：
//   - 智能时限估算：基于文件大小、复杂度等动态调整超时时间
//   - 活动检测：通过多种指标判断进程是否在有效工作
//   - 安全终止：优雅终止 → 强制终止 → 进程组清理的三级终止机制
//   - 资源保护：防止僵尸进程和资源泄露
//   - 用户交互：卡死时提供用户选择（等待、终止、忽略）
type ProcessMonitor struct {
	logger            *zap.Logger
	processes         map[string]*MonitoredProcess // 正在监控的进程
	processMutex      sync.RWMutex                 // 进程映射锁
	baseTimeout       time.Duration                // 基础超时时间
	activityTimeout   time.Duration                // 活动检测超时（README要求30秒）
	estimator         *TimeoutEstimator            // 时限估算器
	terminationPolicy TerminationPolicy            // 终止策略
	statsEnabled      bool                         // 是否启用统计
	stats             *MonitoringStats             // 监控统计
	nonInteractive    bool                         // 是否为非交互模式
}

// MonitoredProcess 被监控的进程
type MonitoredProcess struct {
	ID               string            `json:"id"`                // 进程唯一ID
	Cmd              *exec.Cmd         `json:"-"`                 // 命令对象
	PID              int               `json:"pid"`               // 进程ID
	StartTime        time.Time         `json:"start_time"`        // 启动时间
	LastActivity     time.Time         `json:"last_activity"`     // 最后活动时间
	EstimatedTimeout time.Duration     `json:"estimated_timeout"` // 预估超时时间
	ActualTimeout    time.Duration     `json:"actual_timeout"`    // 实际设置的超时时间
	Context          *ProcessContext   `json:"context"`           // 进程上下文
	Status           ProcessStatus     `json:"status"`            // 进程状态
	ActivityCheckers []ActivityChecker `json:"-"`                 // 活动检查器
	ResourceUsage    *ResourceUsage    `json:"resource_usage"`    // 资源使用情况
	ErrorMessage     string            `json:"error_message"`     // 错误信息
	TerminationCount int               `json:"termination_count"` // 终止尝试次数
	UserChoice       UserChoice        `json:"user_choice"`       // 用户选择
	gopsProcess      *process.Process  // gopsutil 的 Process 对象
}

// Process 返回 gopsutil 的 Process 对象，如果不存在则创建。
func (mp *MonitoredProcess) Process() (*process.Process, error) {
	if mp.gopsProcess == nil {
		p, err := process.NewProcess(int32(mp.PID))
		if err != nil {
			return nil, err
		}
		mp.gopsProcess = p
	}
	return mp.gopsProcess, nil
}

// ProcessContext 进程上下文信息
type ProcessContext struct {
	SourceFile      string            `json:"source_file"`      // 源文件路径
	TargetFile      string            `json:"target_file"`      // 目标文件路径
	FileSize        int64             `json:"file_size"`        // 文件大小
	FileFormat      string            `json:"file_format"`      // 文件格式
	Operation       string            `json:"operation"`        // 操作类型
	ComplexityLevel ComplexityLevel   `json:"complexity_level"` // 复杂度等级
	Priority        Priority          `json:"priority"`         // 优先级
	Metadata        map[string]string `json:"metadata"`         // 附加元数据
}

// ActivityChecker 活动检查器接口
type ActivityChecker interface {
	CheckActivity(process *MonitoredProcess) bool
	GetName() string
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	CPUPercent     float64   `json:"cpu_percent"`      // CPU使用率
	MemoryMB       float64   `json:"memory_mb"`        // 内存使用(MB)
	DiskReadMB     float64   `json:"disk_read_mb"`     // 磁盘读取(MB)
	DiskWriteMB    float64   `json:"disk_write_mb"`    // 磁盘写入(MB)
	NetworkBytesTx int64     `json:"network_bytes_tx"` // 网络发送字节
	NetworkBytesRx int64     `json:"network_bytes_rx"` // 网络接收字节
	LastUpdate     time.Time `json:"last_update"`      // 最后更新时间
}

// TimeoutEstimator 超时时间估算器
type TimeoutEstimator struct {
	logger               *zap.Logger
	baseSizeTimeout      map[string]time.Duration    // 按文件大小的基础超时
	complexityMultiplier map[ComplexityLevel]float64 // 复杂度倍数
	formatTimeMultiplier map[string]float64          // 格式处理时间倍数
	historicalData       []ProcessingRecord          // 历史处理记录
	adaptiveEnabled      bool                        // 是否启用自适应估算
	minTimeout           time.Duration               // 最小超时时间
	maxTimeout           time.Duration               // 最大超时时间
}

// ProcessingRecord 处理记录
type ProcessingRecord struct {
	FileSize        int64           `json:"file_size"`
	Format          string          `json:"format"`
	Operation       string          `json:"operation"`
	ComplexityLevel ComplexityLevel `json:"complexity_level"`
	ActualDuration  time.Duration   `json:"actual_duration"`
	Successful      bool            `json:"successful"`
	Timestamp       time.Time       `json:"timestamp"`
}

// MonitoringStats 监控统计
type MonitoringStats struct {
	TotalProcesses      int                        `json:"total_processes"`
	CompletedProcesses  int                        `json:"completed_processes"`
	TerminatedProcesses int                        `json:"terminated_processes"`
	HungProcesses       int                        `json:"hung_processes"`
	AverageProcessTime  time.Duration              `json:"average_process_time"`
	TimeoutsByType      map[string]int             `json:"timeouts_by_type"`
	ResourcePeaks       map[string]float64         `json:"resource_peaks"`
	OperationStats      map[string]*OperationStats `json:"operation_stats"`
}

// OperationStats 操作统计
type OperationStats struct {
	Count           int           `json:"count"`
	AverageDuration time.Duration `json:"average_duration"`
	SuccessRate     float64       `json:"success_rate"`
	TimeoutRate     float64       `json:"timeout_rate"`
}

// 枚举定义
type ProcessStatus int
type ComplexityLevel int
type Priority int
type TerminationPolicy int
type UserChoice int

const (
	// 进程状态
	StatusStarting ProcessStatus = iota
	StatusRunning
	StatusActive
	StatusIdle
	StatusHung
	StatusCompleted
	StatusTerminated
	StatusFailed
)

const (
	// 复杂度等级
	ComplexityLow ComplexityLevel = iota
	ComplexityMedium
	ComplexityHigh
	ComplexityVeryHigh
)

const (
	// 优先级
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

const (
	// 终止策略
	TerminationGraceful TerminationPolicy = iota // 优雅终止
	TerminationForce                             // 强制终止
	TerminationAsk                               // 询问用户
)

const (
	// 用户选择
	UserChoiceNone UserChoice = iota
	UserChoiceWait
	UserChoiceTerminate
	UserChoiceIgnore
)

// NewProcessMonitor 创建进程监控器
func NewProcessMonitor(logger *zap.Logger, nonInteractive bool) *ProcessMonitor {
	monitor := &ProcessMonitor{
		logger:            logger,
		processes:         make(map[string]*MonitoredProcess),
		baseTimeout:       5 * time.Minute,  // 基础超时5分钟
		activityTimeout:   30 * time.Second, // README要求：30秒活动检测
		terminationPolicy: TerminationAsk,   // 默认询问用户
		statsEnabled:      true,
		stats: &MonitoringStats{
			TimeoutsByType: make(map[string]int),
			ResourcePeaks:  make(map[string]float64),
			OperationStats: make(map[string]*OperationStats),
		},
		nonInteractive:    nonInteractive, // 设置非交互模式
	}

	// 初始化超时估算器
	monitor.estimator = NewTimeoutEstimator(logger)

	return monitor
}

// NewTimeoutEstimator 创建超时估算器
func NewTimeoutEstimator(logger *zap.Logger) *TimeoutEstimator {
	return &TimeoutEstimator{
		logger: logger,
		// README要求：根据媒体属性动态估算处理时限
		baseSizeTimeout: map[string]time.Duration{
			"small":  2 * time.Minute,  // <10MB
			"medium": 5 * time.Minute,  // 10MB-100MB
			"large":  15 * time.Minute, // 100MB-1GB
			"huge":   60 * time.Minute, // >1GB
		},
		complexityMultiplier: map[ComplexityLevel]float64{
			ComplexityLow:      1.0,
			ComplexityMedium:   2.0,
			ComplexityHigh:     4.0,
			ComplexityVeryHigh: 8.0,
		},
		formatTimeMultiplier: map[string]float64{
			"jpeg": 1.0,
			"png":  1.5,
			"webp": 2.0,
			"heif": 2.5,
			"avif": 3.0,
			"jxl":  2.2,
			"mp4":  4.0,
			"mov":  3.5,
		},
		historicalData:  make([]ProcessingRecord, 0),
		adaptiveEnabled: true,
		minTimeout:      30 * time.Second, // 最小30秒
		maxTimeout:      2 * time.Hour,    // 最大2小时
	}
}

// MonitorCommand 监控命令执行 - README核心功能
func (pm *ProcessMonitor) MonitorCommand(ctx context.Context, cmd *exec.Cmd, processCtx *ProcessContext) error {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	processID := pm.generateProcessID()

	// 创建被监控进程
	monitoredProcess := &MonitoredProcess{
		ID:               processID,
		Cmd:              cmd,
		StartTime:        time.Now(),
		LastActivity:     time.Now(),
		Context:          processCtx,
		Status:           StatusStarting,
		ActivityCheckers: pm.createActivityCheckers(),
		ResourceUsage:    &ResourceUsage{},
		UserChoice:       UserChoiceNone,
	}

	// 估算超时时间
	estimatedTimeout := pm.estimator.EstimateTimeout(processCtx)
	monitoredProcess.EstimatedTimeout = estimatedTimeout
	monitoredProcess.ActualTimeout = estimatedTimeout

	// 注册进程
	pm.processMutex.Lock()
	pm.processes[processID] = monitoredProcess
	pm.processMutex.Unlock()

	pm.logger.Info("开始监控进程",
		zap.String("process_id", processID),
		zap.String("operation", processCtx.Operation),
		zap.String("source_file", filepath.Base(processCtx.SourceFile)),
		zap.Duration("estimated_timeout", estimatedTimeout))

	// 启动进程
	if err := cmd.Start(); err != nil {
		pm.processMutex.Lock()
		delete(pm.processes, processID)
		pm.processMutex.Unlock()
		return fmt.Errorf("启动进程失败: %w", err)
	}

	monitoredProcess.PID = cmd.Process.Pid
	monitoredProcess.Status = StatusRunning

	// 启动监控协程
	go pm.monitorProcessLifecycle(ctx, monitoredProcess)

	// 等待进程完成
	err := cmd.Wait()

	if err != nil {
		pm.logger.Error("ffmpeg command failed",
			zap.Error(err),
			zap.String("stdout", stdout.String()),
			zap.String("stderr", stderr.String()))
	}

	// 处理结果
	pm.processMutex.Lock()
	if process, exists := pm.processes[processID]; exists {
		if err != nil {
			process.Status = StatusFailed
			process.ErrorMessage = fmt.Sprintf("error: %v, stderr: %s", err, stderr.String())
		} else {
			process.Status = StatusCompleted
		}

		// 记录处理记录用于学习
		if pm.estimator.adaptiveEnabled {
			pm.recordProcessingData(process, err == nil)
		}
	}
	pm.processMutex.Unlock()

	// 清理进程
	pm.removeProcess(processID)

	pm.logger.Info("进程监控完成",
		zap.String("process_id", processID),
		zap.Bool("success", err == nil),
		zap.Duration("actual_duration", time.Since(monitoredProcess.StartTime)))

	if err != nil {
		return fmt.Errorf("error: %v, stderr: %s", err, stderr.String())
	}

	return nil
}

// monitorProcessLifecycle 监控进程生命周期
func (pm *ProcessMonitor) monitorProcessLifecycle(ctx context.Context, process *MonitoredProcess) {
	ticker := time.NewTicker(1 * time.Second) // 每秒检查一次
	defer ticker.Stop()

	activityDeadline := time.Now().Add(pm.activityTimeout)
	overallDeadline := time.Now().Add(process.ActualTimeout)

	for {
		select {
		case <-ctx.Done():
			pm.logger.Debug("监控上下文取消", zap.String("process_id", process.ID))
			return

		case <-ticker.C:
			pm.processMutex.Lock()
			currentProcess, exists := pm.processes[process.ID]
			if !exists {
				pm.processMutex.Unlock()
				return // 进程已完成或被移除
			}

			// 更新资源使用情况
			pm.updateResourceUsage(currentProcess)

			// 检查活动状态
			isActive := pm.checkProcessActivity(currentProcess)
			if isActive {
				currentProcess.LastActivity = time.Now()
				currentProcess.Status = StatusActive
				activityDeadline = time.Now().Add(pm.activityTimeout) // 重置活动超时
			} else {
				// 检查是否超过活动超时（README要求：30秒无活动判断为卡死）
				if time.Now().After(activityDeadline) {
					currentProcess.Status = StatusHung
					pm.logger.Warn("检测到进程卡死",
						zap.String("process_id", process.ID),
						zap.Duration("inactive_time", time.Since(currentProcess.LastActivity)))

					pm.processMutex.Unlock()
					pm.handleHungProcess(currentProcess)
					return
				}
			}

			// 检查总体超时
			if time.Now().After(overallDeadline) {
				pm.logger.Warn("进程总体超时",
					zap.String("process_id", process.ID),
					zap.Duration("timeout", process.ActualTimeout))

				pm.processMutex.Unlock()
				pm.handleTimeoutProcess(currentProcess)
				return
			}

			pm.processMutex.Unlock()
		}
	}
}

// checkProcessActivity 检查进程活动 - README核心功能
func (pm *ProcessMonitor) checkProcessActivity(process *MonitoredProcess) bool {
	// 使用多个活动检查器
	for _, checker := range process.ActivityCheckers {
		if checker.CheckActivity(process) {
			pm.logger.Debug("检测到进程活动",
				zap.String("process_id", process.ID),
				zap.String("checker", checker.GetName()))
			return true
		}
	}

	return false
}

// handleHungProcess 处理卡死进程 - README要求的用户交互
func (pm *ProcessMonitor) handleHungProcess(process *MonitoredProcess) {
	pm.logger.Warn("进程疑似卡死，根据策略处理",
		zap.String("process_id", process.ID),
		zap.String("operation", process.Context.Operation),
		zap.Duration("inactive_time", time.Since(process.LastActivity)))

	switch pm.terminationPolicy {
	case TerminationGraceful:
		pm.terminateProcess(process, false)

	case TerminationForce:
		pm.terminateProcess(process, true)

	case TerminationAsk:
		// README要求：提示用户或强制终止
		choice := pm.askUserForAction(process)
		process.UserChoice = choice

		switch choice {
		case UserChoiceWait:
			pm.logger.Info("用户选择继续等待", zap.String("process_id", process.ID))
			// 扩展超时时间
			process.ActualTimeout += 10 * time.Minute
			return

		case UserChoiceTerminate:
			pm.logger.Info("用户选择终止进程", zap.String("process_id", process.ID))
			pm.terminateProcess(process, true)

		case UserChoiceIgnore:
			pm.logger.Info("用户选择忽略警告", zap.String("process_id", process.ID))
			// 禁用进一步的卡死检测
			process.Status = StatusRunning
			return
		}
	}

	// 更新统计
	if pm.statsEnabled {
		pm.stats.HungProcesses++
		pm.stats.TimeoutsByType["hung"]++
	}
}

// handleTimeoutProcess 处理超时进程
func (pm *ProcessMonitor) handleTimeoutProcess(process *MonitoredProcess) {
	pm.logger.Error("进程超时，强制终止",
		zap.String("process_id", process.ID),
		zap.Duration("timeout", process.ActualTimeout))

	pm.terminateProcess(process, true)

	if pm.statsEnabled {
		pm.stats.TimeoutsByType["overall"]++
	}
}

// terminateProcess 终止进程 - 三级终止机制
func (pm *ProcessMonitor) terminateProcess(process *MonitoredProcess, force bool) {
	process.TerminationCount++

	if process.Cmd == nil || process.Cmd.Process == nil {
		pm.logger.Warn("进程对象为空，无法终止", zap.String("process_id", process.ID))
		return
	}

	pm.logger.Info("开始终止进程",
		zap.String("process_id", process.ID),
		zap.Int("pid", process.PID),
		zap.Bool("force", force))

	if !force && process.TerminationCount == 1 {
		// 第一次尝试：优雅终止
		if err := process.Cmd.Process.Signal(os.Interrupt); err != nil {
			pm.logger.Warn("优雅终止失败，尝试强制终止",
				zap.String("process_id", process.ID),
				zap.Error(err))
			force = true
		} else {
			pm.logger.Debug("发送中断信号", zap.String("process_id", process.ID))
			// 等待5秒后检查是否终止
			time.AfterFunc(5*time.Second, func() {
				pm.processMutex.RLock()
				if _, exists := pm.processes[process.ID]; exists {
					pm.processMutex.RUnlock()
					pm.terminateProcess(process, true) // 强制终止
				} else {
					pm.processMutex.RUnlock()
				}
			})
			return
		}
	}

	if force {
		// 强制终止
		if err := process.Cmd.Process.Kill(); err != nil {
			pm.logger.Error("强制终止进程失败",
				zap.String("process_id", process.ID),
				zap.Error(err))
		} else {
			pm.logger.Info("进程已强制终止", zap.String("process_id", process.ID))
		}
	}

	process.Status = StatusTerminated

	if pm.statsEnabled {
		pm.stats.TerminatedProcesses++
	}
}

// EstimateTimeout 估算超时时间 - README要求的动态时限估算
func (te *TimeoutEstimator) EstimateTimeout(ctx *ProcessContext) time.Duration {
	// 基于文件大小确定基础超时
	sizeCategory := te.categorizeFileSize(ctx.FileSize)
	baseTimeout := te.baseSizeTimeout[sizeCategory]

	// 应用复杂度倍数
	complexityMultiplier := te.complexityMultiplier[ctx.ComplexityLevel]

	// 应用格式倍数
	formatMultiplier := 1.0
	if multiplier, exists := te.formatTimeMultiplier[ctx.FileFormat]; exists {
		formatMultiplier = multiplier
	}

	// 计算估算时间
	estimatedTimeout := time.Duration(float64(baseTimeout) * complexityMultiplier * formatMultiplier)

	// 应用自适应学习（如果启用）
	if te.adaptiveEnabled {
		adaptiveMultiplier := te.getAdaptiveMultiplier(ctx)
		estimatedTimeout = time.Duration(float64(estimatedTimeout) * adaptiveMultiplier)
	}

	// 限制在合理范围内
	if estimatedTimeout < te.minTimeout {
		estimatedTimeout = te.minTimeout
	}
	if estimatedTimeout > te.maxTimeout {
		estimatedTimeout = te.maxTimeout
	}

	te.logger.Debug("估算处理超时时间",
		zap.String("operation", ctx.Operation),
		zap.Int64("file_size", ctx.FileSize),
		zap.String("size_category", sizeCategory),
		zap.String("format", ctx.FileFormat),
		zap.Float64("complexity_multiplier", complexityMultiplier),
		zap.Float64("format_multiplier", formatMultiplier),
		zap.Duration("estimated_timeout", estimatedTimeout))

	return estimatedTimeout
}

// 辅助方法
func (pm *ProcessMonitor) generateProcessID() string {
	return fmt.Sprintf("proc_%d_%d", time.Now().UnixNano(), len(pm.processes))
}

// createActivityCheckers 创建并返回一组 ActivityChecker。
// 实际应用中，这里会包含 CPU、磁盘 I/O、内存等多种检查器。
func (pm *ProcessMonitor) createActivityCheckers() []ActivityChecker {
	return []ActivityChecker{
		NewCPUActivityChecker(),
		NewDiskIOActivityChecker(),
		NewMemoryActivityChecker(),
	}
}

// updateResourceUsage 更新进程的资源使用情况。
func (pm *ProcessMonitor) updateResourceUsage(mp *MonitoredProcess) {
	p, err := mp.Process()
	if err != nil {
		pm.logger.Error("获取进程对象失败", zap.Error(err), zap.Int("pid", mp.PID))
		return
	}

	// CPU 使用率
	if cpuPercent, err := p.CPUPercentWithContext(context.Background()); err == nil {
		mp.ResourceUsage.CPUPercent = cpuPercent
	}

	// 内存使用
	if memInfo, err := p.MemoryInfoWithContext(context.Background()); err == nil {
		mp.ResourceUsage.MemoryMB = float64(memInfo.RSS) / (1024 * 1024) // 转换为 MB
	}

	// 磁盘 I/O
	if ioCounters, err := p.IOCountersWithContext(context.Background()); err == nil {
		mp.ResourceUsage.DiskReadMB = float64(ioCounters.ReadBytes) / (1024 * 1024)
		mp.ResourceUsage.DiskWriteMB = float64(ioCounters.WriteBytes) / (1024 * 1024)
	}

	// 网络 I/O (如果需要)
	// if netCounters, err := p.NetIOCountersWithContext(context.Background()); err == nil && len(netCounters) > 0 {
	// 	mp.ResourceUsage.NetworkBytesTx = netCounters[0].BytesSent
	// 	mp.ResourceUsage.NetworkBytesRx = netCounters[0].BytesRecv
	// }

	mp.ResourceUsage.LastUpdate = time.Now()
}

func (pm *ProcessMonitor) askUserForAction(process *MonitoredProcess) UserChoice {
	// 如果是非交互模式，则直接返回默认选择（例如等待）。
	if pm.nonInteractive {
		pm.logger.Info("非交互模式下，进程卡死默认选择等待", zap.String("process_id", process.ID))
		return UserChoiceWait
	}

	// 交互模式下，提示用户进行选择。
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("进程 %s (操作: %s, 文件: %s) 疑似卡死。请选择操作：(w)等待 (t)终止 (i)忽略: ", 
			process.ID, process.Context.Operation, filepath.Base(process.Context.SourceFile)) 
		
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "w":
			return UserChoiceWait
		case "t":
			return UserChoiceTerminate
		case "i":
			return UserChoiceIgnore
		default:
			fmt.Println("无效输入，请重新选择。")
		}
	}
}

// SetNonInteractive 设置监控器是否为非交互模式
func (pm *ProcessMonitor) SetNonInteractive(nonInteractive bool) {
	pm.nonInteractive = nonInteractive
}

// removeProcess 移除进程记录
func (pm *ProcessMonitor) removeProcess(processID string) {
	pm.processMutex.Lock()
	defer pm.processMutex.Unlock()
	
	if process, exists := pm.processes[processID]; exists {
		// 更新统计
		if pm.statsEnabled {
			pm.stats.CompletedProcesses++
			if process.Status == StatusCompleted {
				// 处理成功完成的统计
				if opStats, exists := pm.stats.OperationStats[process.Context.Operation]; exists {
					opStats.Count++
					opStats.AverageDuration = time.Duration(int64(opStats.AverageDuration) + int64(time.Since(process.StartTime))) / 2
					opStats.SuccessRate += 1.0
				} else {
					pm.stats.OperationStats[process.Context.Operation] = &OperationStats{
						Count:           1,
						AverageDuration: time.Since(process.StartTime),
						SuccessRate:     1.0,
					}
				}
			} else if process.Status == StatusTerminated || process.Status == StatusFailed {
				// 处理失败的统计
				if opStats, exists := pm.stats.OperationStats[process.Context.Operation]; exists {
					opStats.Count++
					opStats.SuccessRate -= 1.0
				} else {
					pm.stats.OperationStats[process.Context.Operation] = &OperationStats{
						Count:           1,
						AverageDuration: 0,
						SuccessRate:     0.0,
					}
				}
			}
		}
		delete(pm.processes, processID)
	}
}

// categorizeFileSize 根据文件大小分类
func (te *TimeoutEstimator) categorizeFileSize(fileSize int64) string {
	if fileSize < 10*1024*1024 { // < 10MB
		return "small"
	} else if fileSize < 100*1024*1024 { // < 100MB
		return "medium"
	} else if fileSize < 1024*1024*1024 { // < 1GB
		return "large"
	} else {
		return "huge"
	}
}

// getAdaptiveMultiplier 获取自适应倍数
func (te *TimeoutEstimator) getAdaptiveMultiplier(ctx *ProcessContext) float64 {
	// 简化的自适应逻辑：如果没有历史数据，返回1.0
	if len(te.historicalData) == 0 {
		return 1.0
	}
	
	// 计算与历史相似文件的平均处理时间倍数
	var totalTime time.Duration
	var count int
	for _, record := range te.historicalData {
		// 找到类似大小和格式的文件
		sizeDiff := abs64(record.FileSize - ctx.FileSize)
		sizeThreshold := ctx.FileSize / 10 // 10% 大小差异阈值
		if record.Format == ctx.FileFormat && sizeDiff <= sizeThreshold {
			totalTime += record.ActualDuration
			count++
		}
	}
	
	if count == 0 {
		return 1.0
	}
	
	expectedDuration := time.Duration(float64(totalTime) / float64(count))
	estimatedDuration := te.EstimateTimeout(ctx)
	if estimatedDuration == 0 {
		return 1.0
	}
	
	return float64(expectedDuration) / float64(estimatedDuration)
}

// recordProcessingData 记录处理数据用于学习
func (pm *ProcessMonitor) recordProcessingData(process *MonitoredProcess, success bool) {
	if pm.estimator == nil {
		return
	}
	
	record := ProcessingRecord{
		FileSize:        process.Context.FileSize,
		Format:          process.Context.FileFormat,
		Operation:       process.Context.Operation,
		ComplexityLevel: process.Context.ComplexityLevel,
		ActualDuration:  time.Since(process.StartTime),
		Successful:      success,
		Timestamp:       time.Now(),
	}
	
	pm.estimator.historicalData = append(pm.estimator.historicalData, record)
	
	// 限制历史数据长度，避免内存无限增长
	maxHistory := 1000
	if len(pm.estimator.historicalData) > maxHistory {
		pm.estimator.historicalData = pm.estimator.historicalData[len(pm.estimator.historicalData)-maxHistory:]
	}
}

// abs64 计算int64的绝对值
func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}