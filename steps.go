package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pixly/pkg/batchdecision"
	"pixly/pkg/core/types"
	"pixly/pkg/engine"
	"pixly/pkg/engine/quality"
	"pixly/pkg/progress"
	"pixly/pkg/scanner"
	"pixly/pkg/security"
	"pixly/pkg/tools"
	"pixly/pkg/ui/interactive"

	"github.com/fatih/color"
	"go.uber.org/zap"
)

// executeStandardFlow 执行README标准化7步流程
func executeStandardFlow(ctx context.Context, state *StandardFlowState, logger *zap.Logger, uiManager *interactive.Interface, unifiedProgress *progress.UnifiedProgress, isInteractive bool) error {
	reader := bufio.NewReader(os.Stdin)

	// 步骤1：启动与输入 - 选择目录
	state.Step = 1
	if state.TargetDir == "" {
		showStepHeaderAdvanced(state, "启动与输入", "📁", uiManager)
		targetDir, err := uiManager.GetTargetDirectory()
		if err != nil {
			return fmt.Errorf("步骤1失败 - 选择目录: %w", err)
		}
		state.TargetDir = targetDir
	} else {
		showStepHeaderAdvanced(state, "启动与输入", "📁", uiManager)
		uiManager.ShowInfo("使用指定的媒体目录路径：")
		uiManager.ShowSuccess(fmt.Sprintf("✅ 已选择目录: %s", state.TargetDir))
	}
	logger.Info("步骤1完成：选择目录", zap.String("dir", state.TargetDir))

	// 步骤2：安全检查
	state.Step = 2
	showStepHeaderAdvanced(state, "安全检查", "🔒", uiManager)
	uiManager.ShowInfo("🔍 正在进行安全检查...")
	if err := performSecurityCheck(state.TargetDir, logger); err != nil {
		return fmt.Errorf("步骤2失败 - 安全检查: %w", err)
	}
	state.SecurityPassed = true
	uiManager.ShowSuccess("✅ 安全检查通过")
	logger.Info("步骤2完成：安全检查通过")

	// 步骤3：统一扫描与分析
	state.Step = 3
	showStepHeaderAdvanced(state, "统一扫描与分析", "🔍", uiManager)
	uiManager.ShowInfo("🔍 正在扫描媒体文件...")
	unifiedProgress.StartStep(progress.StepScan, 100, "🔍 统一扫描与分析")
	scanResults, err := performUnifiedScan(state.TargetDir, logger)
	if err != nil {
		return fmt.Errorf("步骤3失败 - 统一扫描: %w", err)
	}
	unifiedProgress.CompleteStep()
	state.ScanComplete = true
	uiManager.ShowSuccess(fmt.Sprintf("✅ 扫描完成，发现 %d 个文件", len(scanResults)))
	logger.Info("步骤3完成：统一扫描", zap.Int("files", len(scanResults)))

	// 步骤4：问题文件决策 (批量决策)
	state.Step = 4
	showStepHeaderAdvanced(state, "问题文件决策", "🚨", uiManager)
	uiManager.ShowInfo("🚨 正在进行批量决策...")
	if err := performBatchDecisions(reader, scanResults, logger, isInteractive); err != nil {
		return fmt.Errorf("步骤4失败 - 批量决策: %w", err)
	}
	uiManager.ShowSuccess("✅ 批量决策完成")
	logger.Info("步骤4完成：批量决策处理")

	// 步骤5：处理模式选择
	state.Step = 5
	showStepHeaderAdvanced(state, "处理模式选择", "🎯", uiManager)
	var mode types.AppMode
	if isInteractive {
		mode, err = uiManager.SelectMode()
		if err != nil {
			return fmt.Errorf("步骤5失败 - 模式选择: %w", err)
		}
	} else {
		mode = types.ModeAutoPlus // 非交互模式下使用默认值
		uiManager.ShowInfo("使用默认处理模式：自动模式+")
	}
	state.ModeSelected = mode
	uiManager.ShowSuccess(fmt.Sprintf("✅ 已选择 %s", mode.String()))
	logger.Info("步骤5完成：选择模式", zap.String("mode", mode.String()))

	// 步骤6：核心处理
	state.Step = 6
	showStepHeaderAdvanced(state, "核心处理", "⚡", uiManager)
	uiManager.ShowInfo("⚡ 正在进行核心处理...")
	unifiedProgress.StartStep(progress.StepProcessing, int64(len(scanResults)), "⚡ 核心处理")
	results, err := performCoreProcessing(ctx, state.TargetDir, mode, scanResults, logger)
	if err != nil {
		return fmt.Errorf("步骤6失败 - 核心处理: %w", err)
	}
	unifiedProgress.CompleteStep()
	state.ProcessingDone = true
	uiManager.ShowSuccess("✅ 核心处理完成")
	logger.Info("步骤6完成：核心处理")

	// 步骤7：统计报告
	state.Step = 7
	showStepHeaderAdvanced(state, "统计报告", "📊", uiManager)
	uiManager.ShowInfo("📊 正在生成统计报告...")
	if err := generateStatisticsReport(results, logger); err != nil {
		return fmt.Errorf("步骤7失败 - 统计报告: %w", err)
	}
	state.ReportGenerated = true
	uiManager.ShowSuccess("✅ 统计报告生成完成")
	logger.Info("步骤7完成：统计报告生成")

	return nil
}

func performSecurityCheck(targetDir string, logger *zap.Logger) error {
	color.White("🔒 正在执行安全检查...")

	checker := security.NewSecurityChecker(logger)
	result, err := checker.PerformSecurityCheck(targetDir)
	if err != nil {
		color.Red("❌ 安全检查错误: %v", err)
		color.Yellow("💡 解决方案：")
		color.White("   1. 检查目录读取权限")
		color.White("   2. 确保目录路径正确")
		color.White("   3. 检查磁盘空间是否足够")
		return err
	}

	if !result.Passed {
		color.Red("❌ 安全检查失败")
		color.Yellow("🚨 发现安全问题：")
		for _, issue := range result.Issues {
			color.Red("  - %s: %s", issue.Type.String(), issue.Message)
		}
		color.Yellow("💡 建议：")
		color.White("   1. 选择其他安全目录")
		color.White("   2. 检查目录权限设置")
		color.White("   3. 避免在系统关键目录中操作")
		return fmt.Errorf("安全检查失败")
	}

	color.Green("✅ 安全检查通过")
	return nil
}

func performUnifiedScan(targetDir string, logger *zap.Logger) ([]*types.MediaInfo, error) {
	color.White("🔍 正在执行统一扫描...")

	scanner := scanner.NewScanner(logger)
	ctx := context.Background()
	files, err := scanner.ScanDirectory(ctx, targetDir)
	if err != nil {
		color.Red("❌ 扫描失败: %v", err)
		color.Yellow("💡 解决方案：")
		color.White("   1. 检查目录访问权限")
		color.White("   2. 确保目录包含媒体文件")
		color.White("   3. 检查磁盘空间和内存")
		color.White("   4. 稍后重试或重启程序")
		return nil, err
	}

	var mediaInfos []*types.MediaInfo
	for _, file := range files {
		if !file.IsDir {
			mediaInfo := &types.MediaInfo{
				Path:    file.Path,
				Size:    file.Size,
				ModTime: time.Unix(file.ModTime, 0),
				Type:    getMediaType(file.Path),
				Status:  types.StatusPending,
			}
			mediaInfos = append(mediaInfos, mediaInfo)
		}
	}

	if len(mediaInfos) == 0 {
		color.Yellow("⚠️  未找到媒体文件")
		color.Yellow("💡 建议：")
		color.White("   1. 检查目录是否包含图片或视频文件")
		color.White("   2. 支持的格式: JPG, PNG, GIF, MP4, MOV, WEBM, AVIF, JXL 等")
		color.White("   3. 检查文件是否被隐藏或加密")
		return mediaInfos, nil
	}

	color.Green("✅ 扫描完成，发现 %d 个媒体文件", len(mediaInfos))
	return mediaInfos, nil
}

func performBatchDecisions(reader *bufio.Reader, scanResults []*types.MediaInfo, logger *zap.Logger, isInteractive bool) error {
	batchManager := batchdecision.NewBatchDecisionManager(logger, isInteractive)

	var corruptedCount, lowQualityCount int
	for _, info := range scanResults {
		if info.IsCorrupted {
			corruptedCount++
			batchManager.AddCorruptedFile(info.Path, batchdecision.CorruptionDataCorrupt, info.ErrorMessage, false)
		} else if info.Quality == types.QualityVeryLow {
			lowQualityCount++
			lowQualityFile := &batchdecision.LowQualityFile{
				FilePath:        info.Path,
				QualityScore:    info.QualityScore,
				QualityIssues:   []string{"低品质"},
				FileSize:        info.Size,
				DetectedAt:      time.Now(),
				CanConvert:      true,
				RecommendedMode: batchdecision.ProcessingModeAuto,
			}
			batchManager.AddLowQualityFile(lowQualityFile)
		}
	}

	if corruptedCount == 0 && lowQualityCount == 0 {
		color.Green("✅ 没有发现问题文件，跳过批量决策")
		return nil
	}

	ctx := context.Background()
	result, err := batchManager.ProcessBatchDecisions(ctx)
	if err != nil {
		return fmt.Errorf("批量决策失败: %w", err)
	}

	color.Green("✅ 批量决策完成，处理了 %d 个文件", result.Summary.TotalFiles)
	color.Cyan("📊 成功: %d, 失败: %d, 成功率: %.1f%%",
		result.Summary.SuccessfulFiles,
		result.Summary.FailedFiles,
		result.Summary.SuccessRate*100)

	return nil
}

func performCoreProcessing(ctx context.Context, targetDir string, mode types.AppMode, scanResults []*types.MediaInfo, logger *zap.Logger) ([]*types.ProcessingResult, error) {
	color.White("⚡ 开始核心处理...")

	isTestMode := os.Getenv("PIXLY_TEST_MODE") == "true" || os.Getenv("TEST_MODE") == "true"

	var processingResults []*types.MediaInfo
	var workingDir string

	if isTestMode {
		color.Yellow("🧪 测试模式：创建安全副本保护原始测试数据")
		var err error
		workingDir, err = createSafeWorkingCopy(targetDir, scanResults, logger)
		if err != nil {
			return nil, fmt.Errorf("创建测试副本失败: %w", err)
		}
		color.Green("✅ 测试副本创建完成: %s", workingDir)

		copiedResults, err := updatePathsToWorkingCopy(scanResults, targetDir, workingDir)
		if err != nil {
			return nil, fmt.Errorf("更新副本路径失败: %w", err)
		}
		processingResults = copiedResults
	} else {
		color.Green("🚀 正常模式：直接优化原文件，提升处理效率")
		color.Cyan("💡 提示：程序会自动备份重要文件，您可以放心使用")
		processingResults = scanResults
		workingDir = targetDir
	}

	toolChecker := tools.NewChecker(logger)
	toolPaths, err := toolChecker.CheckAll()
	if err != nil {
		logger.Warn("工具链检查警告", zap.Error(err))
		color.Yellow("⚠️  工具链检查不完整，可能影响转换效果")
	} else {
		color.Green("✅ 工具链检查通过")
	}

	showToolStatus(toolPaths)

	qualityEngine := quality.NewQualityEngine(logger, "", "", true)
	processingManager := engine.NewProcessingModeManager(logger, toolPaths, qualityEngine)

	color.Yellow("🎯 使用模式: %s", mode.String())
	if isTestMode {
		color.Cyan("🔒 测试模式：正在对副本文件进行处理，原始测试数据绝对安全")
	} else {
		color.Cyan("⚡ 正常模式：直接优化文件，自动备份重要数据")
	}
	results, err := processingManager.ProcessFiles(ctx, mode, processingResults)
	if err != nil {
		return nil, fmt.Errorf("处理模式执行失败: %w", err)
	}

	color.Green("✅ 核心处理完成，处理了 %d 个文件", len(results))
	return results, nil
}

func generateStatisticsReport(results []*types.ProcessingResult, logger *zap.Logger) error {
	color.White("Generating statistics report...")

	successCount := 0
	var totalSpaceSaved int64

	for _, result := range results {
		if result.Success {
			successCount++
			totalSpaceSaved += result.SpaceSaved
		}
	}

	color.Cyan("\nProcessing complete. Statistics report:")
	color.Green("Successfully processed: %d files", successCount)
	color.Blue("Space saved: %.2f MB", float64(totalSpaceSaved)/(1024*1024))

	return nil
}

func getMediaType(path string) types.MediaType {
	extension := strings.ToLower(filepath.Ext(path))
	switch extension {
	case ".jpg", ".jpeg", ".png", ".webp", ".bmp", ".tiff":
		return types.MediaTypeImage
	case ".gif", ".apng", ".mng":
		return types.MediaTypeAnimated
	case ".mp4", ".mov", ".mkv", ".avi", ".webm":
		return types.MediaTypeVideo
	default:
		return types.MediaTypeUnknown
	}
}
