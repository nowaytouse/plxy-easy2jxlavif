package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

// NewConversionEngine 创建转换引擎
func NewConversionEngine(logger *zap.Logger, config *ui.Config) (*ConversionEngine, error) {
	// 简化：直接创建优化器，假设工具已安装
	toolPaths := types.ToolCheckResults{
		CjxlPath:         "cjxl",
		AvifencPath:      "avifenc",
		FfmpegStablePath: "ffmpeg",
	}

	// 创建优化器（暂时禁用知识库）
	dbPath := "" // 暂时不使用知识库
	optimizer := engine.NewBalanceOptimizer(logger, toolPaths, dbPath)
	optimizer.EnableKnowledge(false) // 暂时禁用知识库

	return &ConversionEngine{
		optimizer: optimizer,
		logger:    logger,
		config:    config,
	}, nil
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

// ConvertDirectory 转换整个目录
func (ce *ConversionEngine) ConvertDirectory(
	ctx context.Context,
	inputDir string,
	outputDir string,
	inPlace bool,
) (*ConversionResult, error) {

	result := &ConversionResult{
		Errors: make([]string, 0),
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

	// 逐个转换
	for i, file := range files {
		select {
		case <-ctx.Done():
			pterm.Warning.Println("\n⚠️  转换被取消")
			result.Duration = time.Since(startTime)
			return result, ctx.Err()
		default:
		}

		// 更新进度条消息
		if progressBar != nil {
			progressBar.SetMessage(fmt.Sprintf("处理 %s (%d/%d)",
				filepath.Base(file), i+1, len(files)))
		}

		// 执行转换
		convertResult, err := ce.convertSingleFile(ctx, file, outputDir, inPlace)

		if err != nil {
			result.FailCount++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filepath.Base(file), err))
			ce.logger.Warn("转换失败",
				zap.String("file", filepath.Base(file)),
				zap.Error(err))
		} else if convertResult != nil {
			if convertResult.Skipped {
				result.SkipCount++
			} else {
				result.SuccessCount++
				result.TotalOrigSize += convertResult.OriginalSize
				result.TotalNewSize += convertResult.NewSize
			}
		}

		// 更新进度
		if progressBar != nil {
			progressBar.Increment()
		}
	}

	// 完成进度条
	if progressBar != nil {
		progressBar.Finish()
	}

	// 计算总结
	result.Duration = time.Since(startTime)
	if result.TotalOrigSize > 0 {
		result.TotalSaved = result.TotalOrigSize - result.TotalNewSize
		result.SavedPercentage = float64(result.TotalSaved) / float64(result.TotalOrigSize) * 100
	}

	return result, nil
}

// SingleFileResult 单文件转换结果
type SingleFileResult struct {
	OriginalSize int64
	NewSize      int64
	Skipped      bool
	Error        error
}

// convertSingleFile 转换单个文件
func (ce *ConversionEngine) convertSingleFile(
	ctx context.Context,
	filePath string,
	outputDir string,
	inPlace bool,
) (*SingleFileResult, error) {

	// 获取原始文件大小
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	originalSize := fileInfo.Size()

	// 检测媒体类型
	mediaType := types.MediaTypeUnknown
	ext := filepath.Ext(filePath)
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		mediaType = types.MediaTypeImage
	case ".mp4", ".mov", ".avi":
		mediaType = types.MediaTypeVideo
	}

	// 执行优化
	optimizeResult, err := ce.optimizer.OptimizeFile(ctx, filePath, mediaType)
	if err != nil {
		return nil, err
	}

	if !optimizeResult.Success {
		return &SingleFileResult{
			OriginalSize: originalSize,
			Skipped:      true,
			Error:        optimizeResult.Error,
		}, nil
	}

	// 处理输出文件
	var finalPath string
	if inPlace {
		// 原地替换
		finalPath = filePath
		// 删除原文件，重命名新文件
		if err := os.Remove(filePath); err != nil {
			return nil, fmt.Errorf("删除原文件失败: %w", err)
		}
		if err := os.Rename(optimizeResult.OutputPath, finalPath); err != nil {
			return nil, fmt.Errorf("重命名失败: %w", err)
		}
	} else {
		// 复制到输出目录
		if outputDir != "" {
			// 创建输出目录结构
			relPath, _ := filepath.Rel(filepath.Dir(filePath), filePath)
			finalPath = filepath.Join(outputDir, relPath)

			if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
				return nil, fmt.Errorf("创建输出目录失败: %w", err)
			}

			// 移动文件
			if err := os.Rename(optimizeResult.OutputPath, finalPath); err != nil {
				return nil, fmt.Errorf("移动文件失败: %w", err)
			}
		} else {
			finalPath = optimizeResult.OutputPath
		}
	}

	return &SingleFileResult{
		OriginalSize: originalSize,
		NewSize:      optimizeResult.NewSize,
		Skipped:      false,
	}, nil
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

// Close 关闭引擎
func (ce *ConversionEngine) Close() error {
	if ce.optimizer != nil {
		return ce.optimizer.Close()
	}
	return nil
}
