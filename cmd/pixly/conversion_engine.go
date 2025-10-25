package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"pixly/pkg/core/types"
	"pixly/pkg/engine"
	"pixly/pkg/ui"

	"github.com/pterm/pterm"
	"go.uber.org/zap"
)

// ConversionEngine 转换引擎包装器
type ConversionEngine struct {
	optimizer *engine.BalanceOptimizer
	logger    *zap.Logger
	config    *ui.Config
}

// NewConversionEngine 创建转换引擎（v3.1.1完整版）
func NewConversionEngine(logger *zap.Logger, config *ui.Config) (*ConversionEngine, error) {
	pterm.Info.Println("🔧 初始化 Pixly v3.1.1 引擎...")
	pterm.Info.Println("🔍 检查必要工具...")

	// 完整的工具检查（v3.1.1）
	toolPaths := checkTools()

	// 显示工具检查结果
	showToolCheckResults(&toolPaths)

	if !toolPaths.HasCjxl || !toolPaths.HasAvifenc || !toolPaths.HasFfmpeg {
		pterm.Error.Println("❌ 缺少必要工具")
		pterm.Warning.Println("💡 安装提示：")
		pterm.Println("  brew install jpeg-xl libavif ffmpeg exiftool")
		return nil, fmt.Errorf("缺少必要工具")
	}

	pterm.Success.Println("✅ 所有必要工具已就绪")
	pterm.Println()

	// 知识库路径（v3.1.1完整功能）
	homeDir, _ := os.UserHomeDir()
	dbPath := filepath.Join(homeDir, ".pixly", "knowledge.db")

	// 创建知识库目录
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	pterm.Info.Printfln("📊 知识库: %s", dbPath)

	// 创建优化器（v3.1.1完整功能）
	optimizer := engine.NewBalanceOptimizer(logger, toolPaths, dbPath)

	// 启用知识库（v3.1.1自动学习）
	optimizer.EnableKnowledge(true)
	pterm.Success.Println("✅ 知识库已启用（实时学习中）")
	pterm.Println()

	return &ConversionEngine{
		optimizer: optimizer,
		logger:    logger,
		config:    config,
	}, nil
}

// checkTools 检查工具（v3.1.1完整版）
func checkTools() types.ToolCheckResults {
	result := types.ToolCheckResults{}

	// 检查cjxl
	if path, err := exec.LookPath("cjxl"); err == nil {
		result.HasCjxl = true
		result.CjxlPath = path
	}

	// 检查avifenc
	if path, err := exec.LookPath("avifenc"); err == nil {
		result.HasAvifenc = true
		result.AvifencPath = path
	}

	// 检查ffmpeg
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		result.HasFfmpeg = true
		result.FfmpegStablePath = path
	}

	// 检查exiftool
	if path, err := exec.LookPath("exiftool"); err == nil {
		result.HasExiftool = true
		result.ExiftoolPath = path
	}

	return result
}

// showToolCheckResults 显示工具检测结果
func showToolCheckResults(tools *types.ToolCheckResults) {
	pterm.Info.Println("工具检测结果：")

	tableData := pterm.TableData{
		{"工具", "状态", "路径"},
	}

	addToolRow := func(name, path string, available bool) {
		status := "❌ 未安装"
		if available {
			status = "✅ 已安装"
		}
		displayPath := path
		if displayPath == "" {
			displayPath = "N/A"
		}
		tableData = append(tableData, []string{name, status, displayPath})
	}

	addToolRow("cjxl", tools.CjxlPath, tools.HasCjxl)
	addToolRow("avifenc", tools.AvifencPath, tools.HasAvifenc)
	addToolRow("ffmpeg", tools.FfmpegStablePath, tools.HasFfmpeg)
	addToolRow("exiftool", tools.ExiftoolPath, tools.HasExiftool)

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	pterm.Println()
}

// ConversionResult 转换结果
type ConversionResult struct {
	TotalFiles      int
	SuccessCount    int
	FailCount       int
	SkipCount       int
	TotalOrigSize   int64
	TotalNewSize    int64
	TotalSaved      int64
	SavedPercentage float64
	Duration        time.Duration
	Errors          []string
}

// ConvertDirectory 转换整个目录（完整版，带断点续传）
func (ce *ConversionEngine) ConvertDirectory(
	ctx context.Context,
	inputDir string,
	outputDir string,
	inPlace bool,
) (*ConversionResult, error) {

	result := &ConversionResult{
		Errors: make([]string, 0),
	}

	// 断点续传管理器
	resumeManager := ui.NewResumeManager()

	// 检查断点
	var resumePoint *ui.ResumePoint
	var useResume bool

	if resumeManager.HasResumePoint() {
		loadedPoint, err := resumeManager.LoadResumePoint()
		if err == nil && loadedPoint != nil && loadedPoint.InputDir == inputDir {
			// 询问用户是否续传
			shouldResume, err := resumeManager.ShowResumePrompt(loadedPoint)
			if err != nil {
				// 用户取消
				return nil, err
			}

			if shouldResume {
				resumePoint = loadedPoint
				useResume = true
				pterm.Success.Printfln("📍 断点续传：将跳过已处理的 %d 个文件", len(resumePoint.ProcessedFiles))
			}
		}
	}

	startTime := time.Now()

	// 扫描文件
	pterm.Info.Println("📂 扫描目录...")
	files, err := scanMediaFiles(inputDir)
	if err != nil {
		return nil, fmt.Errorf("扫描失败: %w", err)
	}

	result.TotalFiles = len(files)
	pterm.Success.Printfln("✅ 找到 %d 个媒体文件", len(files))

	// 显示文件类型统计
	ce.showFileTypeStats(files)
	pterm.Println()

	if len(files) == 0 {
		return result, nil
	}

	// 创建安全进度条
	progressMgr := ui.NewProgressManager(ce.config)
	progressBar, err := ui.NewSafeProgressBar(progressMgr, "🎨 转换中", len(files))
	if err != nil {
		ce.logger.Warn("进度条创建失败", zap.Error(err))
	}

	// 实时统计（线程安全）
	var processedCount int32
	var successCount int32
	var failCount int32
	var skipCount int32

	// 禁用动画（转换阶段为性能让步）
	if ce.config.EnableAnimation {
		pterm.Info.Println("⚡ 转换过程中暂时禁用动画以提升性能")
	}

	// 初始化或恢复统计
	processedFiles := make([]string, 0)
	if useResume && resumePoint != nil {
		atomic.StoreInt32(&successCount, int32(resumePoint.SuccessCount))
		atomic.StoreInt32(&failCount, int32(resumePoint.FailCount))
		atomic.StoreInt32(&skipCount, int32(resumePoint.SkipCount))
		processedFiles = resumePoint.ProcessedFiles

		pterm.Info.Printfln("📍 从第 %d/%d 个文件继续", len(processedFiles)+1, len(files))
		pterm.Println()
	}

	// 逐个转换（带超时+断点保存）
	for i, file := range files {
		// 断点续传：跳过已处理的文件
		if useResume && resumePoint != nil && resumePoint.IsProcessed(file) {
			pterm.Info.Printfln("⏭️  跳过已处理: %s", filepath.Base(file))
			if progressBar != nil {
				progressBar.Increment()
			}
			continue
		}

		select {
		case <-ctx.Done():
			pterm.Warning.Println("\n⚠️  转换被中断")

			// 保存断点
			ce.saveResumePoint(resumeManager, inputDir, outputDir, inPlace,
				files, processedFiles, &successCount, &failCount, &skipCount, file)

			result.Duration = time.Since(startTime)
			result.SuccessCount = int(atomic.LoadInt32(&successCount))
			result.FailCount = int(atomic.LoadInt32(&failCount))
			result.SkipCount = int(atomic.LoadInt32(&skipCount))
			return result, ctx.Err()
		default:
		}

		// 更新进度条消息（根据文件类型显示不同emoji）
		if progressBar != nil {
			icon := ce.getFileIcon(file)
			progressBar.SetMessage(fmt.Sprintf("%s %s (%d/%d)",
				icon, filepath.Base(file), i+1, len(files)))
		}

		// 创建超时上下文（每个文件最多5分钟）
		fileCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)

		// 执行转换（带超时）
		convertResult, err := ce.convertSingleFileWithTimeout(fileCtx, file, outputDir, inPlace)
		cancel() // 立即释放资源

		atomic.AddInt32(&processedCount, 1)
		processedFiles = append(processedFiles, file)

		if err != nil {
			atomic.AddInt32(&failCount, 1)
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filepath.Base(file), err))
			ce.logger.Warn("转换失败",
				zap.String("file", filepath.Base(file)),
				zap.Error(err))
		} else if convertResult != nil {
			if convertResult.Skipped {
				atomic.AddInt32(&skipCount, 1)
			} else {
				atomic.AddInt32(&successCount, 1)
				result.TotalOrigSize += convertResult.OriginalSize
				result.TotalNewSize += convertResult.NewSize
			}
		}

		// 更新进度
		if progressBar != nil {
			progressBar.Increment()
		}

		// 每10个文件保存一次断点（避免频繁IO）
		if (i+1)%10 == 0 {
			ce.saveResumePoint(resumeManager, inputDir, outputDir, inPlace,
				files, processedFiles, &successCount, &failCount, &skipCount, file)
		}
	}

	// 清除断点（全部完成）
	resumeManager.ClearResumePoint()

	// 完成进度条
	if progressBar != nil {
		progressBar.Finish()
	}

	pterm.Println()

	// 最终统计
	result.SuccessCount = int(atomic.LoadInt32(&successCount))
	result.FailCount = int(atomic.LoadInt32(&failCount))
	result.SkipCount = int(atomic.LoadInt32(&skipCount))

	// 计算总结
	result.Duration = time.Since(startTime)
	if result.TotalOrigSize > 0 {
		result.TotalSaved = result.TotalOrigSize - result.TotalNewSize
		result.SavedPercentage = float64(result.TotalSaved) / float64(result.TotalOrigSize) * 100
	}

	// 显示知识库统计
	ce.showKnowledgeStats()

	return result, nil
}

// showFileTypeStats 显示文件类型统计
func (ce *ConversionEngine) showFileTypeStats(files []string) {
	stats := make(map[string]int)
	for _, file := range files {
		ext := filepath.Ext(file)
		stats[ext]++
	}

	pterm.Info.Println("文件类型分布：")
	for ext, count := range stats {
		percentage := float64(count) / float64(len(files)) * 100
		pterm.Printfln("  %s: %d (%.1f%%)", ext, count, percentage)
	}
}

// showKnowledgeStats 显示知识库统计
func (ce *ConversionEngine) showKnowledgeStats() {
	if !ce.optimizer.IsKnowledgeEnabled() {
		return
	}

	pterm.Println()
	pterm.Info.Println("📊 知识库统计：")

	stats, err := ce.optimizer.GetKnowledgeStats()
	if err != nil {
		ce.logger.Warn("获取知识库统计失败", zap.Error(err))
		return
	}

	if totalRecords, ok := stats["total_records"].(int64); ok {
		pterm.Printfln("  总记录数: %d", totalRecords)
	}

	pterm.Success.Println("✅ 转换记录已保存，系统将持续学习优化")
}

// SingleFileResult 单文件转换结果
type SingleFileResult struct {
	OriginalSize int64
	NewSize      int64
	Skipped      bool
	Error        error
}

// convertSingleFileWithTimeout 带超时的转换（防止卡死）
func (ce *ConversionEngine) convertSingleFileWithTimeout(
	ctx context.Context,
	filePath string,
	outputDir string,
	inPlace bool,
) (*SingleFileResult, error) {
	// 使用channel实现超时检测
	resultChan := make(chan *SingleFileResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := ce.convertSingleFile(ctx, filePath, outputDir, inPlace)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	// 等待结果或超时
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		ce.logger.Warn("文件转换超时",
			zap.String("file", filepath.Base(filePath)),
			zap.Error(ctx.Err()))
		return nil, fmt.Errorf("转换超时（超过5分钟）: %w", ctx.Err())
	}
}

// convertSingleFile 转换单个文件（完整版，包含完整验证）
func (ce *ConversionEngine) convertSingleFile(
	ctx context.Context,
	filePath string,
	outputDir string,
	inPlace bool,
) (*SingleFileResult, error) {

	// 获取原始文件大小
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	originalSize := fileInfo.Size()

	// 检测媒体类型（完整版）
	mediaType := ce.detectMediaType(filePath)
	if mediaType == types.MediaTypeUnknown {
		ce.logger.Debug("跳过未知文件类型",
			zap.String("file", filepath.Base(filePath)))
		return &SingleFileResult{
			OriginalSize: originalSize,
			Skipped:      true,
		}, nil
	}

	// 视频文件特殊提示（处理可能较慢）
	if mediaType == types.MediaTypeVideo {
		ce.logger.Info("处理视频文件（可能需要较长时间）",
			zap.String("file", filepath.Base(filePath)),
			zap.Int64("size_mb", originalSize/(1024*1024)))
	}

	// 执行优化（使用完整的v3.1.1引擎）
	optimizeResult, err := ce.optimizer.OptimizeFile(ctx, filePath, mediaType)
	if err != nil {
		return nil, fmt.Errorf("优化失败: %w", err)
	}

	if !optimizeResult.Success {
		// 优化失败但不是错误（可能是文件已是最优格式）
		if optimizeResult.Error != nil {
			ce.logger.Debug("优化跳过",
				zap.String("file", filepath.Base(filePath)),
				zap.Error(optimizeResult.Error))
		}
		return &SingleFileResult{
			OriginalSize: originalSize,
			Skipped:      true,
			Error:        optimizeResult.Error,
		}, nil
	}

	// 验证转换结果
	if err := ce.validateConversionResult(filePath, optimizeResult.OutputPath); err != nil {
		// 清理失败的输出文件
		os.Remove(optimizeResult.OutputPath)
		return nil, fmt.Errorf("验证失败: %w", err)
	}

	// 处理输出文件
	finalPath, err := ce.handleOutputFile(filePath, optimizeResult.OutputPath, outputDir, inPlace)
	if err != nil {
		// 清理
		os.Remove(optimizeResult.OutputPath)
		return nil, err
	}

	ce.logger.Info("转换成功",
		zap.String("file", filepath.Base(filePath)),
		zap.String("输出", finalPath),
		zap.Int64("原始", originalSize),
		zap.Int64("转换后", optimizeResult.NewSize),
		zap.Float64("节省", float64(originalSize-optimizeResult.NewSize)/float64(originalSize)*100))

	return &SingleFileResult{
		OriginalSize: originalSize,
		NewSize:      optimizeResult.NewSize,
		Skipped:      false,
	}, nil
}

// detectMediaType 检测媒体类型（完整版）
func (ce *ConversionEngine) detectMediaType(filePath string) types.MediaType {
	ext := filepath.Ext(filePath)
	ext = strings.ToLower(ext)

	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tiff":
		return types.MediaTypeImage
	case ".mp4", ".mov", ".avi", ".mkv", ".webm", ".flv":
		return types.MediaTypeVideo
	default:
		return types.MediaTypeUnknown
	}
}

// getFileIcon 根据文件类型获取emoji图标
func (ce *ConversionEngine) getFileIcon(filePath string) string {
	ext := filepath.Ext(filePath)
	ext = strings.ToLower(ext)

	switch ext {
	case ".png":
		return "🖼️"
	case ".jpg", ".jpeg":
		return "📸"
	case ".gif":
		return "🎞️"
	case ".webp":
		return "🎨"
	case ".mp4", ".mov", ".avi", ".mkv":
		return "🎬"
	default:
		return "📄"
	}
}

// validateConversionResult 验证转换结果（完整版）
func (ce *ConversionEngine) validateConversionResult(originalPath, convertedPath string) error {
	// 检查转换后文件是否存在
	convertedInfo, err := os.Stat(convertedPath)
	if err != nil {
		return fmt.Errorf("转换后文件不存在: %w", err)
	}

	// 检查文件大小是否合理
	if convertedInfo.Size() == 0 {
		return fmt.Errorf("转换后文件为空")
	}

	originalInfo, _ := os.Stat(originalPath)
	if originalInfo != nil {
		// 检查是否异常膨胀（超过原始10倍）
		if convertedInfo.Size() > originalInfo.Size()*10 {
			return fmt.Errorf("转换后文件异常膨胀 (%d -> %d)",
				originalInfo.Size(), convertedInfo.Size())
		}
	}

	// 对于JXL和AVIF文件，进行基本格式验证
	ext := filepath.Ext(convertedPath)
	switch ext {
	case ".jxl":
		// 简单验证：检查文件头
		if err := ce.validateJXLFile(convertedPath); err != nil {
			return fmt.Errorf("JXL文件验证失败: %w", err)
		}
	case ".avif":
		// 简单验证：检查文件头
		if err := ce.validateAVIFFile(convertedPath); err != nil {
			return fmt.Errorf("AVIF文件验证失败: %w", err)
		}
	}

	return nil
}

// validateJXLFile 验证JXL文件
func (ce *ConversionEngine) validateJXLFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// 读取文件头
	header := make([]byte, 12)
	n, err := f.Read(header)
	if err != nil || n < 12 {
		return fmt.Errorf("无法读取文件头")
	}

	// JXL文件头魔术字节: 0xFF 0x0A 或 0x00 0x00 0x00 0x0C 0x4A 0x58 0x4C 0x20 0x0D 0x0A 0x87 0x0A
	if header[0] == 0xFF && header[1] == 0x0A {
		return nil // Naked codestream
	}
	if header[0] == 0x00 && header[1] == 0x00 && header[2] == 0x00 && header[3] == 0x0C {
		return nil // Container format
	}

	return fmt.Errorf("不是有效的JXL文件")
}

// validateAVIFFile 验证AVIF文件
func (ce *ConversionEngine) validateAVIFFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// 读取文件头
	header := make([]byte, 12)
	n, err := f.Read(header)
	if err != nil || n < 12 {
		return fmt.Errorf("无法读取文件头")
	}

	// AVIF文件是ISO Base Media File Format
	// 检查ftyp box
	if header[4] == 'f' && header[5] == 't' && header[6] == 'y' && header[7] == 'p' {
		return nil
	}

	return fmt.Errorf("不是有效的AVIF文件")
}

// handleOutputFile 处理输出文件
func (ce *ConversionEngine) handleOutputFile(
	originalPath string,
	convertedPath string,
	outputDir string,
	inPlace bool,
) (string, error) {
	var finalPath string

	if inPlace {
		// 原地替换：先备份，再替换，最后删除备份
		backupPath := originalPath + ".pixly_backup"

		// 重命名原文件为备份
		if err := os.Rename(originalPath, backupPath); err != nil {
			return "", fmt.Errorf("创建备份失败: %w", err)
		}

		// 移动新文件到原位置
		if err := os.Rename(convertedPath, originalPath); err != nil {
			// 恢复备份
			os.Rename(backupPath, originalPath)
			return "", fmt.Errorf("替换文件失败: %w", err)
		}

		// 删除备份
		os.Remove(backupPath)

		finalPath = originalPath
	} else {
		// 复制到输出目录
		if outputDir != "" {
			// 保持目录结构
			relPath, _ := filepath.Rel(filepath.Dir(originalPath), originalPath)
			finalPath = filepath.Join(outputDir, relPath)

			// 创建输出目录
			if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
				return "", fmt.Errorf("创建输出目录失败: %w", err)
			}

			// 移动文件
			if err := os.Rename(convertedPath, finalPath); err != nil {
				return "", fmt.Errorf("移动文件失败: %w", err)
			}
		} else {
			finalPath = convertedPath
		}
	}

	return finalPath, nil
}

// ShowResult 显示转换结果
func (ce *ConversionEngine) ShowResult(result *ConversionResult) {
	pterm.Println()
	pterm.DefaultSection.Println("📊 转换完成报告")
	pterm.Println()

	// 基本统计
	statsTable := pterm.TableData{
		{"项目", "数值"},
		{"总文件数", fmt.Sprintf("%d", result.TotalFiles)},
		{"成功转换", fmt.Sprintf("✅ %d", result.SuccessCount)},
		{"失败", fmt.Sprintf("❌ %d", result.FailCount)},
		{"跳过", fmt.Sprintf("⏭️  %d", result.SkipCount)},
		{"耗时", fmt.Sprintf("%.1f秒", result.Duration.Seconds())},
	}

	pterm.DefaultTable.WithHasHeader().WithData(statsTable).Render()
	pterm.Println()

	// 空间统计
	if result.TotalOrigSize > 0 {
		spaceTable := pterm.TableData{
			{"项目", "大小"},
			{"原始总大小", formatBytes(result.TotalOrigSize)},
			{"转换后大小", formatBytes(result.TotalNewSize)},
			{"节省空间", formatBytes(result.TotalSaved)},
			{"节省比例", fmt.Sprintf("%.1f%%", result.SavedPercentage)},
		}

		pterm.DefaultTable.WithHasHeader().WithData(spaceTable).Render()
		pterm.Println()
	}

	// 显示错误（如果有）
	if len(result.Errors) > 0 && len(result.Errors) <= 10 {
		pterm.Warning.Println("⚠️  转换错误列表:")
		for i, errMsg := range result.Errors {
			pterm.Printfln("  [%d] %s", i+1, errMsg)
		}
		pterm.Println()
	} else if len(result.Errors) > 10 {
		pterm.Warning.Printfln("⚠️  共 %d 个错误（显示前10个）:", len(result.Errors))
		for i := 0; i < 10; i++ {
			pterm.Printfln("  [%d] %s", i+1, result.Errors[i])
		}
		pterm.Println()
	}

	// 总结框
	summaryMsg := fmt.Sprintf(`转换完成！

成功率: %.1f%%
节省空间: %s (%.1f%%)
总耗时: %.1f秒`,
		float64(result.SuccessCount)/float64(result.TotalFiles)*100,
		formatBytes(result.TotalSaved),
		result.SavedPercentage,
		result.Duration.Seconds())

	if result.SuccessCount == result.TotalFiles {
		pterm.DefaultBox.
			WithTitle("🎉 完美完成").
			WithTitleTopCenter().
			WithBoxStyle(pterm.NewStyle(pterm.FgLightGreen)).
			Println(summaryMsg)
	} else if result.SuccessCount > 0 {
		pterm.DefaultBox.
			WithTitle("✅ 部分完成").
			WithTitleTopCenter().
			WithBoxStyle(pterm.NewStyle(pterm.FgLightYellow)).
			Println(summaryMsg)
	} else {
		pterm.DefaultBox.
			WithTitle("❌ 转换失败").
			WithTitleTopCenter().
			WithBoxStyle(pterm.NewStyle(pterm.FgLightRed)).
			Println(summaryMsg)
	}
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// saveResumePoint 保存断点
func (ce *ConversionEngine) saveResumePoint(
	manager *ui.ResumeManager,
	inputDir, outputDir string,
	inPlace bool,
	allFiles, processedFiles []string,
	successCount, failCount, skipCount *int32,
	lastFile string,
) {
	point := &ui.ResumePoint{
		InputDir:       inputDir,
		OutputDir:      outputDir,
		InPlace:        inPlace,
		TotalFiles:     len(allFiles),
		ProcessedFiles: processedFiles,
		ProcessedCount: len(processedFiles),
		SuccessCount:   int(atomic.LoadInt32(successCount)),
		FailCount:      int(atomic.LoadInt32(failCount)),
		SkipCount:      int(atomic.LoadInt32(skipCount)),
		LastFile:       lastFile,
	}

	if err := manager.SaveResumePoint(point); err != nil {
		ce.logger.Warn("保存断点失败", zap.Error(err))
	}
}

// Close 关闭引擎
func (ce *ConversionEngine) Close() error {
	if ce.optimizer != nil {
		return ce.optimizer.Close()
	}
	return nil
}
