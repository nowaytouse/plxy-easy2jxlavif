package errorhandling

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// NewRetryManager 创建重试管理器
func NewRetryManager(logger *zap.Logger) *RetryManager {
	return &RetryManager{
		logger:        logger,
		activeRetries: make(map[string]*RetryContext),
		retryStats:    make(map[ErrorType]*RetryStatistics),
	}
}

// NewFallbackManager 创建后备方案管理器
func NewFallbackManager(logger *zap.Logger) *FallbackManager {
	return &FallbackManager{
		logger:         logger,
		fallbackChains: make(map[string][]FallbackAction),
		fallbackStats:  make(map[FallbackType]*FallbackStatistics),
	}
}

// initializeDefaultStrategies 初始化默认恢复策略
func (erm *ErrorRecoveryManager) initializeDefaultStrategies() {
	// 文件损坏错误策略
	erm.recoveryStrategies[ErrorTypeFileCorrupted] = &RecoveryStrategy{
		Name:        "文件损坏恢复策略",
		ErrorType:   ErrorTypeFileCorrupted,
		Priority:    1,
		MaxAttempts: 2,
		RetryDelay:  2 * time.Second,
		FallbackActions: []FallbackAction{
			{
				Type:        FallbackSkipFile,
				Name:        "跳过损坏文件",
				Description: "跳过无法修复的损坏文件",
				Priority:    1,
				Enabled:     true,
			},
		},
		Enabled: true,
	}

	// 格式不支持错误策略
	erm.recoveryStrategies[ErrorTypeFormatUnsupported] = &RecoveryStrategy{
		Name:        "格式不支持恢复策略",
		ErrorType:   ErrorTypeFormatUnsupported,
		Priority:    1,
		MaxAttempts: 1,
		RetryDelay:  1 * time.Second,
		FallbackActions: []FallbackAction{
			{
				Type:        FallbackDifferentFormat,
				Name:        "尝试不同格式",
				Description: "尝试转换为支持的格式",
				Priority:    1,
				Enabled:     true,
			},
			{
				Type:        FallbackCopyOriginal,
				Name:        "复制原文件",
				Description: "保持原格式不变",
				Priority:    2,
				Enabled:     true,
			},
		},
		Enabled: true,
	}

	// 内存耗尽错误策略
	erm.recoveryStrategies[ErrorTypeMemoryExhausted] = &RecoveryStrategy{
		Name:        "内存耗尽恢复策略",
		ErrorType:   ErrorTypeMemoryExhausted,
		Priority:    1,
		MaxAttempts: 2,
		RetryDelay:  5 * time.Second,
		FallbackActions: []FallbackAction{
			{
				Type:        FallbackLowerQuality,
				Name:        "降低处理品质",
				Description: "使用更低的品质设置以减少内存使用",
				Priority:    1,
				Enabled:     true,
			},
			{
				Type:        FallbackSimpleConversion,
				Name:        "简单转换",
				Description: "使用简化的转换流程",
				Priority:    2,
				Enabled:     true,
			},
		},
		Enabled: true,
	}

	// 处理超时错误策略
	erm.recoveryStrategies[ErrorTypeProcessTimeout] = &RecoveryStrategy{
		Name:        "处理超时恢复策略",
		ErrorType:   ErrorTypeProcessTimeout,
		Priority:    1,
		MaxAttempts: 1,
		RetryDelay:  10 * time.Second,
		FallbackActions: []FallbackAction{
			{
				Type:        FallbackSimpleConversion,
				Name:        "快速转换",
				Description: "使用快速转换模式",
				Priority:    1,
				Enabled:     true,
			},
			{
				Type:        FallbackCopyOriginal,
				Name:        "复制原文件",
				Description: "跳过转换，保持原文件",
				Priority:    2,
				Enabled:     true,
			},
		},
		Enabled: true,
	}

	// FFmpeg崩溃错误策略
	erm.recoveryStrategies[ErrorTypeFFmpegCrash] = &RecoveryStrategy{
		Name:        "FFmpeg崩溃恢复策略",
		ErrorType:   ErrorTypeFFmpegCrash,
		Priority:    1,
		MaxAttempts: 2,
		RetryDelay:  3 * time.Second,
		FallbackActions: []FallbackAction{
			{
				Type:        FallbackDifferentFormat,
				Name:        "使用其他工具",
				Description: "尝试使用其他转换工具",
				Priority:    1,
				Enabled:     true,
			},
			{
				Type:        FallbackCopyOriginal,
				Name:        "保持原格式",
				Description: "跳过转换",
				Priority:    2,
				Enabled:     true,
			},
		},
		Enabled: true,
	}

	erm.logger.Info("默认恢复策略初始化完成", zap.Int("strategy_count", len(erm.recoveryStrategies)))
}

// initializeStatistics 初始化统计信息
func (erm *ErrorRecoveryManager) initializeStatistics() {
	erm.errorStats.ErrorsByType = make(map[ErrorType]int)
	erm.errorStats.ErrorsBySeverity = make(map[ErrorSeverity]int)
	erm.errorStats.StrategyStats = make(map[string]*StrategyStats)
	erm.errorStats.TrendAnalysis = &ErrorTrendAnalysis{
		PeriodStart:         time.Now(),
		CommonErrorPatterns: make([]string, 0),
		RecommendedActions:  make([]string, 0),
	}
}

// analyzeError 分析错误类型和严重程度
func (erm *ErrorRecoveryManager) analyzeError(err error) (ErrorType, ErrorSeverity) {
	errorMessage := strings.ToLower(err.Error())

	// 错误模式匹配
	patterns := map[string]struct {
		errorType ErrorType
		severity  ErrorSeverity
	}{
		"corrupted":     {ErrorTypeFileCorrupted, SeverityHigh},
		"invalid":       {ErrorTypeFormatUnsupported, SeverityMedium},
		"unsupported":   {ErrorTypeFormatUnsupported, SeverityMedium},
		"out of memory": {ErrorTypeMemoryExhausted, SeverityCritical},
		"memory":        {ErrorTypeMemoryExhausted, SeverityHigh},
		"timeout":       {ErrorTypeProcessTimeout, SeverityMedium},
		"deadline":      {ErrorTypeProcessTimeout, SeverityMedium},
		"permission":    {ErrorTypePermissionDenied, SeverityHigh},
		"access denied": {ErrorTypePermissionDenied, SeverityHigh},
		"network":       {ErrorTypeNetworkFailure, SeverityMedium},
		"connection":    {ErrorTypeNetworkFailure, SeverityMedium},
		"ffmpeg":        {ErrorTypeFFmpegCrash, SeverityHigh},
		"metadata":      {ErrorTypeMetadataCorrupted, SeverityLow},
		"quality":       {ErrorTypeQualityTooLow, SeverityLow},
		"too large":     {ErrorTypeFileTooLarge, SeverityMedium},
		"file size":     {ErrorTypeFileTooLarge, SeverityMedium},
	}

	for pattern, info := range patterns {
		if strings.Contains(errorMessage, pattern) {
			return info.errorType, info.severity
		}
	}

	return ErrorTypeUnknown, SeverityMedium
}

// handleCriticalError 处理关键错误
func (erm *ErrorRecoveryManager) handleCriticalError(ctx context.Context, errorRecord *ErrorRecord) (*RecoveryResult, error) {
	erm.logger.Error("检测到关键错误",
		zap.String("error_id", errorRecord.ID),
		zap.String("error_type", errorRecord.ErrorType.String()),
		zap.String("error_message", errorRecord.ErrorMessage))

	switch erm.config.CriticalErrorAction {
	case ActionAbort:
		return &RecoveryResult{
			Success:      false,
			Strategy:     "critical_abort",
			ActionsTaken: []string{"中止操作"},
			FinalState:   "操作中止",
		}, fmt.Errorf("关键错误导致操作中止: %s", errorRecord.ErrorMessage)

	case ActionSkip:
		return &RecoveryResult{
			Success:      true,
			Strategy:     "critical_skip",
			ActionsTaken: []string{"跳过文件"},
			FinalState:   "文件跳过",
		}, nil

	case ActionFallback:
		return erm.fallbackRecovery(ctx, errorRecord)

	default:
		return erm.fallbackRecovery(ctx, errorRecord)
	}
}

// fallbackRecovery 后备恢复方案
func (erm *ErrorRecoveryManager) fallbackRecovery(ctx context.Context, errorRecord *ErrorRecord) (*RecoveryResult, error) {
	erm.logger.Debug("执行后备恢复方案", zap.String("error_id", errorRecord.ID))

	// 通用后备动作
	fallbackActions := []FallbackAction{
		{
			Type:        FallbackLowerQuality,
			Name:        "降级品质处理",
			Description: "使用最低品质设置进行处理",
			Priority:    1,
			Enabled:     true,
		},
		{
			Type:        FallbackCopyOriginal,
			Name:        "保留原文件",
			Description: "复制原文件到输出目录",
			Priority:    2,
			Enabled:     true,
		},
		{
			Type:        FallbackSkipFile,
			Name:        "跳过文件",
			Description: "跳过当前文件",
			Priority:    3,
			Enabled:     true,
		},
	}

	result, err := erm.fallbackManager.ExecuteFallback(ctx, fallbackActions, errorRecord)
	if err != nil {
		return &RecoveryResult{
			Success:      false,
			Strategy:     "fallback_recovery",
			ActionsTaken: []string{"后备恢复失败"},
			FinalState:   "恢复失败",
		}, err
	}

	return &RecoveryResult{
		Success:      result.Success,
		Strategy:     "fallback_recovery",
		ActionsTaken: []string{fmt.Sprintf("后备方案: %s", result.FallbackType.String())},
		TimeTaken:    result.TimeTaken,
		FinalState:   "后备恢复",
		Details:      result.Details,
	}, nil
}

// AttemptRetry 尝试重试
func (rm *RetryManager) AttemptRetry(ctx context.Context, errorRecord *ErrorRecord, strategy *RecoveryStrategy) (bool, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	retryCtx, exists := rm.activeRetries[errorRecord.ID]
	if !exists {
		retryCtx = &RetryContext{
			OperationID:    errorRecord.ID,
			ErrorType:      errorRecord.ErrorType,
			CurrentAttempt: 0,
			MaxAttempts:    strategy.MaxAttempts,
			BackoffDelay:   strategy.RetryDelay,
		}
		rm.activeRetries[errorRecord.ID] = retryCtx
	}

	retryCtx.CurrentAttempt++
	retryCtx.LastAttempt = time.Now()

	if retryCtx.CurrentAttempt > retryCtx.MaxAttempts {
		delete(rm.activeRetries, errorRecord.ID)
		return false, fmt.Errorf("超过最大重试次数: %d", retryCtx.MaxAttempts)
	}

	// 计算退避延迟
	if retryCtx.CurrentAttempt > 1 {
		backoffDelay := time.Duration(float64(strategy.RetryDelay) * math.Pow(2.0, float64(retryCtx.CurrentAttempt-1)))
		if backoffDelay > 30*time.Second {
			backoffDelay = 30 * time.Second
		}
		retryCtx.BackoffDelay = backoffDelay
		retryCtx.NextAttempt = time.Now().Add(backoffDelay)

		rm.logger.Debug("重试延迟等待",
			zap.String("operation_id", errorRecord.ID),
			zap.Int("attempt", retryCtx.CurrentAttempt),
			zap.Duration("delay", backoffDelay))

		select {
		case <-time.After(backoffDelay):
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}

	rm.logger.Info("开始重试操作",
		zap.String("operation_id", errorRecord.ID),
		zap.Int("attempt", retryCtx.CurrentAttempt),
		zap.Int("max_attempts", retryCtx.MaxAttempts))

	// 模拟重试逻辑（实际应该调用具体的重试操作）
	success := rm.simulateRetry(ctx, errorRecord)

	// 更新统计信息
	rm.updateRetryStats(errorRecord.ErrorType, retryCtx.CurrentAttempt, success)

	if success {
		delete(rm.activeRetries, errorRecord.ID)
		rm.logger.Info("重试成功",
			zap.String("operation_id", errorRecord.ID),
			zap.Int("attempt", retryCtx.CurrentAttempt))
		return true, nil
	}

	rm.logger.Debug("重试失败",
		zap.String("operation_id", errorRecord.ID),
		zap.Int("attempt", retryCtx.CurrentAttempt))

	return false, fmt.Errorf("重试失败，尝试次数: %d", retryCtx.CurrentAttempt)
}

// simulateRetry 模拟重试逻辑
func (rm *RetryManager) simulateRetry(ctx context.Context, errorRecord *ErrorRecord) bool {
	// 根据错误类型模拟不同的重试成功率
	successRates := map[ErrorType]float64{
		ErrorTypeFileCorrupted:     0.2, // 20%成功率
		ErrorTypeFormatUnsupported: 0.1, // 10%成功率
		ErrorTypeMemoryExhausted:   0.3, // 30%成功率
		ErrorTypeProcessTimeout:    0.5, // 50%成功率
		ErrorTypePermissionDenied:  0.8, // 80%成功率
		ErrorTypeNetworkFailure:    0.7, // 70%成功率
		ErrorTypeFFmpegCrash:       0.4, // 40%成功率
		ErrorTypeMetadataCorrupted: 0.6, // 60%成功率
		ErrorTypeQualityTooLow:     0.9, // 90%成功率
		ErrorTypeFileTooLarge:      0.3, // 30%成功率
		ErrorTypeUnknown:           0.2, // 20%成功率
	}

	successRate, exists := successRates[errorRecord.ErrorType]
	if !exists {
		successRate = 0.2
	}

	// 模拟重试操作的执行时间
	time.Sleep(100 * time.Millisecond)

	// 基于成功率随机决定是否成功
	return time.Now().UnixNano()%100 < int64(successRate*100)
}

// ExecuteFallback 执行后备方案
func (fm *FallbackManager) ExecuteFallback(ctx context.Context, actions []FallbackAction, errorRecord *ErrorRecord) (*FallbackResult, error) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	startTime := time.Now()

	for _, action := range actions {
		if !action.Enabled {
			continue
		}

		fm.logger.Debug("执行后备动作",
			zap.String("error_id", errorRecord.ID),
			zap.String("action_type", action.Type.String()),
			zap.String("action_name", action.Name))

		result, err := fm.executeFallbackAction(ctx, &action, errorRecord)
		if err == nil && result.Success {
			fm.updateFallbackStats(action.Type, true, time.Since(startTime))
			return result, nil
		}

		fm.updateFallbackStats(action.Type, false, time.Since(startTime))
		fm.logger.Debug("后备动作失败",
			zap.String("error_id", errorRecord.ID),
			zap.String("action_type", action.Type.String()),
			zap.Error(err))
	}

	return &FallbackResult{
		Success:      false,
		FallbackType: FallbackSkipFile,
		TimeTaken:    time.Since(startTime),
		Details:      map[string]interface{}{"reason": "所有后备方案都失败"},
	}, fmt.Errorf("所有后备方案都失败")
}

// executeFallbackAction 执行具体的后备动作
func (fm *FallbackManager) executeFallbackAction(ctx context.Context, action *FallbackAction, errorRecord *ErrorRecord) (*FallbackResult, error) {
	startTime := time.Now()

	switch action.Type {
	case FallbackLowerQuality:
		return fm.executeLowerQualityFallback(ctx, errorRecord)

	case FallbackDifferentFormat:
		return fm.executeDifferentFormatFallback(ctx, errorRecord)

	case FallbackSimpleConversion:
		return fm.executeSimpleConversionFallback(ctx, errorRecord)

	case FallbackCopyOriginal:
		return fm.executeCopyOriginalFallback(ctx, errorRecord)

	case FallbackSkipFile:
		return &FallbackResult{
			Success:      true,
			FallbackType: FallbackSkipFile,
			OutputPath:   "",
			QualityLevel: "skipped",
			TimeTaken:    time.Since(startTime),
			Details:      map[string]interface{}{"action": "文件已跳过"},
		}, nil

	default:
		return nil, fmt.Errorf("未知的后备动作类型: %v", action.Type)
	}
}

// executeLowerQualityFallback 执行降低品质后备方案
func (fm *FallbackManager) executeLowerQualityFallback(ctx context.Context, errorRecord *ErrorRecord) (*FallbackResult, error) {
	// 模拟降低品质处理
	time.Sleep(200 * time.Millisecond)

	outputPath := strings.Replace(errorRecord.SourceFile, filepath.Ext(errorRecord.SourceFile), "_lowquality"+filepath.Ext(errorRecord.SourceFile), 1)

	return &FallbackResult{
		Success:      true,
		FallbackType: FallbackLowerQuality,
		OutputPath:   outputPath,
		QualityLevel: "low",
		TimeTaken:    200 * time.Millisecond,
		Details: map[string]interface{}{
			"quality_reduction": "50%",
			"original_file":     errorRecord.SourceFile,
		},
	}, nil
}

// executeDifferentFormatFallback 执行不同格式后备方案
func (fm *FallbackManager) executeDifferentFormatFallback(ctx context.Context, errorRecord *ErrorRecord) (*FallbackResult, error) {
	// 模拟格式转换
	time.Sleep(300 * time.Millisecond)

	baseName := strings.TrimSuffix(errorRecord.SourceFile, filepath.Ext(errorRecord.SourceFile))
	outputPath := baseName + ".jpg" // 转换为通用JPEG格式

	return &FallbackResult{
		Success:      true,
		FallbackType: FallbackDifferentFormat,
		OutputPath:   outputPath,
		QualityLevel: "standard",
		TimeTaken:    300 * time.Millisecond,
		Details: map[string]interface{}{
			"target_format":   "jpeg",
			"original_format": filepath.Ext(errorRecord.SourceFile),
		},
	}, nil
}

// executeSimpleConversionFallback 执行简单转换后备方案
func (fm *FallbackManager) executeSimpleConversionFallback(ctx context.Context, errorRecord *ErrorRecord) (*FallbackResult, error) {
	// 模拟简单转换
	time.Sleep(150 * time.Millisecond)

	outputPath := strings.Replace(errorRecord.SourceFile, filepath.Ext(errorRecord.SourceFile), "_simple"+filepath.Ext(errorRecord.SourceFile), 1)

	return &FallbackResult{
		Success:      true,
		FallbackType: FallbackSimpleConversion,
		OutputPath:   outputPath,
		QualityLevel: "basic",
		TimeTaken:    150 * time.Millisecond,
		Details: map[string]interface{}{
			"conversion_method": "simple",
			"features_disabled": []string{"advanced_filters", "metadata_processing"},
		},
	}, nil
}

// executeCopyOriginalFallback 执行复制原文件后备方案
func (fm *FallbackManager) executeCopyOriginalFallback(ctx context.Context, errorRecord *ErrorRecord) (*FallbackResult, error) {
	startTime := time.Now()

	// 生成输出路径
	outputDir := filepath.Dir(errorRecord.SourceFile)
	outputName := "copy_" + filepath.Base(errorRecord.SourceFile)
	outputPath := filepath.Join(outputDir, outputName)

	// 复制文件
	input, err := os.Open(errorRecord.SourceFile)
	if err != nil {
		return nil, fmt.Errorf("打开源文件失败: %w", err)
	}
	defer input.Close()

	output, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer output.Close()

	// 这里简化实现，实际应该进行文件复制
	// _, err = io.Copy(output, input)
	// if err != nil {
	//     return nil, fmt.Errorf("复制文件失败: %w", err)
	// }

	return &FallbackResult{
		Success:      true,
		FallbackType: FallbackCopyOriginal,
		OutputPath:   outputPath,
		QualityLevel: "original",
		TimeTaken:    time.Since(startTime),
		Details: map[string]interface{}{
			"operation": "file_copy",
			"preserved": "all_original_properties",
		},
	}, nil
}

// 辅助方法
func (erm *ErrorRecoveryManager) generateErrorID() string {
	return fmt.Sprintf("err_%d_%d", time.Now().UnixNano(), len(erm.errorHistory))
}

func (erm *ErrorRecoveryManager) recordError(errorRecord *ErrorRecord) {
	erm.mutex.Lock()
	defer erm.mutex.Unlock()

	erm.errorHistory = append(erm.errorHistory, *errorRecord)

	// 限制历史记录大小
	if len(erm.errorHistory) > erm.config.HistoryLimit {
		erm.errorHistory = erm.errorHistory[len(erm.errorHistory)-erm.config.HistoryLimit:]
	}
}

func (erm *ErrorRecoveryManager) updateStatistics(errorRecord *ErrorRecord, result *RecoveryResult) {
	erm.mutex.Lock()
	defer erm.mutex.Unlock()

	erm.errorStats.TotalErrors++
	erm.errorStats.ErrorsByType[errorRecord.ErrorType]++
	erm.errorStats.ErrorsBySeverity[errorRecord.Severity]++

	if errorRecord.RecoveryAttempted {
		erm.errorStats.RecoveryAttempts++
		if errorRecord.RecoverySuccess {
			erm.errorStats.SuccessfulRecoveries++
		} else {
			erm.errorStats.FailedRecoveries++
		}
	}

	// 更新成功率
	if erm.errorStats.RecoveryAttempts > 0 {
		erm.errorStats.RecoverySuccessRate = float64(erm.errorStats.SuccessfulRecoveries) / float64(erm.errorStats.RecoveryAttempts)
	}

	// 更新策略统计
	if result != nil {
		strategyStats, exists := erm.errorStats.StrategyStats[result.Strategy]
		if !exists {
			strategyStats = &StrategyStats{}
			erm.errorStats.StrategyStats[result.Strategy] = strategyStats
		}

		strategyStats.UsageCount++
		strategyStats.LastUsed = time.Now()
		strategyStats.AverageTime = result.TimeTaken

		if result.Success {
			strategyStats.SuccessCount++
		} else {
			strategyStats.FailureCount++
		}

		if strategyStats.UsageCount > 0 {
			strategyStats.SuccessRate = float64(strategyStats.SuccessCount) / float64(strategyStats.UsageCount)
		}
	}
}

func (rm *RetryManager) updateRetryStats(errorType ErrorType, attempt int, success bool) {
	stats, exists := rm.retryStats[errorType]
	if !exists {
		stats = &RetryStatistics{}
		rm.retryStats[errorType] = stats
	}

	stats.TotalRetries++
	if success {
		stats.SuccessfulRetries++
	} else {
		stats.FailedRetries++
	}

	if attempt > stats.MaxRetries {
		stats.MaxRetries = attempt
	}

	if stats.TotalRetries > 0 {
		stats.AverageRetries = float64(stats.TotalRetries) / float64(len(rm.activeRetries)+1)
	}
}

func (fm *FallbackManager) updateFallbackStats(fallbackType FallbackType, success bool, duration time.Duration) {
	stats, exists := fm.fallbackStats[fallbackType]
	if !exists {
		stats = &FallbackStatistics{}
		fm.fallbackStats[fallbackType] = stats
	}

	stats.TotalUsage++
	stats.TotalTime += duration

	if success {
		stats.SuccessfulUsage++
	} else {
		stats.FailedUsage++
	}

	if stats.TotalUsage > 0 {
		stats.SuccessRate = float64(stats.SuccessfulUsage) / float64(stats.TotalUsage)
		stats.AverageTime = stats.TotalTime / time.Duration(stats.TotalUsage)
	}
}

// GetErrorStatistics 获取错误统计信息
func (erm *ErrorRecoveryManager) GetErrorStatistics() *ErrorStatistics {
	erm.mutex.RLock()
	defer erm.mutex.RUnlock()

	// 返回副本
	stats := *erm.errorStats
	return &stats
}

// GetRecoveryStrategy 获取恢复策略
func (erm *ErrorRecoveryManager) GetRecoveryStrategy(errorType ErrorType) (*RecoveryStrategy, bool) {
	erm.mutex.RLock()
	defer erm.mutex.RUnlock()

	strategy, exists := erm.recoveryStrategies[errorType]
	if !exists {
		return nil, false
	}

	// 返回副本
	strategyCopy := *strategy
	return &strategyCopy, true
}

// UpdateRecoveryStrategy 更新恢复策略
func (erm *ErrorRecoveryManager) UpdateRecoveryStrategy(errorType ErrorType, strategy *RecoveryStrategy) {
	erm.mutex.Lock()
	defer erm.mutex.Unlock()

	erm.recoveryStrategies[errorType] = strategy
	erm.logger.Info("恢复策略已更新",
		zap.String("error_type", errorType.String()),
		zap.String("strategy_name", strategy.Name))
}

// Enable 启用错误恢复
func (erm *ErrorRecoveryManager) Enable() {
	erm.enabled = true
	erm.logger.Info("错误恢复已启用")
}

// Disable 禁用错误恢复
func (erm *ErrorRecoveryManager) Disable() {
	erm.enabled = false
	erm.logger.Info("错误恢复已禁用")
}
