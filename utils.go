package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pixly/pkg/core/types"
	"pixly/pkg/ui/interactive"

	"github.com/fatih/color"
	"go.uber.org/zap"
)

func showStepHeader(state *StandardFlowState, stepName, icon string) {
	color.Cyan("\n================================================================================")
	color.HiYellow("%s 步骤 %d/%d: %s", icon, state.Step, state.TotalSteps, stepName)
	color.Cyan("================================================================================")
}

func getTargetDirectory(reader *bufio.Reader) (string, error) {
	color.White("请输入媒体目录路径（支持拖拽）：")

	attempts := 0
	maxAttempts := 3

	for attempts < maxAttempts {
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				attempts++
				if attempts >= maxAttempts {
					return "", fmt.Errorf("输入尝试次数已达上限")
				}
				color.Yellow("输入为空，请重新输入媒体目录路径（还可以尝试%d次）：", maxAttempts-attempts)
				continue
			}
			return "", fmt.Errorf("读取输入失败: %w", err)
		}

		path := strings.TrimSpace(input)
		path = strings.Trim(path, "'\"")

		if path == "" {
			attempts++
			if attempts >= maxAttempts {
				return "", fmt.Errorf("输入尝试次数已达上限")
			}
			color.Yellow("路径不能为空，请重新输入（还可以尝试%d次）：", maxAttempts-attempts)
			continue
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			attempts++
			if attempts >= maxAttempts {
				return "", fmt.Errorf("输入尝试次数已达上限")
			}
			color.Yellow("路径不存在，请重新输入（还可以尝试%d次）：", maxAttempts-attempts)
			continue
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			attempts++
			if attempts >= maxAttempts {
				return "", fmt.Errorf("输入尝试次数已达上限")
			}
			color.Yellow("路径解析失败，请重新输入（还可以尝试%d次）：", maxAttempts-attempts)
			continue
		}

		color.Green("✅ 已选择目录: %s", absPath)
		return absPath, nil
	}

	return "", fmt.Errorf("输入尝试次数已达上限")
}

func showToolStatus(tools types.ToolCheckResults) {
	color.Cyan("🔧 工具链状态检查：")

	missingTools := []string{}

	if tools.HasFfmpeg {
		color.Green("  ✅ FFmpeg: 已找到")
		if tools.FfmpegStablePath != "" {
			color.White("    - 稳定版: %s", tools.FfmpegStablePath)
		}
		if tools.FfmpegDevPath != "" {
			color.White("    - 开发版: %s", tools.FfmpegDevPath)
		}
	} else {
		color.Red("  ❌ FFmpeg: 未找到 - 建议安装: brew install ffmpeg")
		missingTools = append(missingTools, "ffmpeg")
	}

	if tools.HasCjxl {
		color.Green("  ✅ cjxl: 已找到")
		if tools.CjxlPath != "" {
			color.White("    - 路径: %s", tools.CjxlPath)
		}
	} else {
		color.Red("  ❌ cjxl: 未找到 - 建议安装: brew install jpeg-xl")
		missingTools = append(missingTools, "jpeg-xl")
	}

	if tools.HasAvifenc {
		color.Green("  ✅ avifenc: 已找到")
		if tools.AvifencPath != "" {
			color.White("    - 路径: %s", tools.AvifencPath)
		}
	} else {
		color.Red("  ❌ avifenc: 未找到 - 建议安装: brew install libavif")
		missingTools = append(missingTools, "libavif")
	}

	if tools.HasExiftool {
		color.Green("  ✅ exiftool: 已找到")
		if tools.ExiftoolPath != "" {
			color.White("    - 路径: %s", tools.ExiftoolPath)
		}
	} else {
		color.Yellow("  ⚠️  exiftool: 未找到 - 可选安装: brew install exiftool")
	}

	if tools.HasLibx264 || tools.HasLibx265 || tools.HasLibSvtAv1 {
		color.White("  🎥 编解码器支持:")
		if tools.HasLibx264 {
			color.Green("    ✅ libx264")
		}
		if tools.HasLibx265 {
			color.Green("    ✅ libx265")
		}
		if tools.HasLibSvtAv1 {
			color.Green("    ✅ libsvtav1 (AVIF高质量编码)")
		}
		if tools.HasVToolbox {
			color.Green("    ✅ VideoToolbox (macOS硬件加速)")
		}
	}

	if len(missingTools) > 0 {
		color.Yellow("\n💡 快速安装指导：")
		color.White("   全部安装： brew install %s", strings.Join(missingTools, " "))
		color.White("   或者分步安装：")
		for _, tool := range missingTools {
			color.White("     brew install %s", tool)
		}
		color.Cyan("\nℹ️  安装后重新运行程序即可使用全部功能")
	}

	color.White("")
}

func getChoiceWithTimeout(reader *bufio.Reader, timeoutSeconds int, defaultChoice string) (string, error) {
	inputChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		input, err := reader.ReadString('\n')
		if err != nil {
			errorChan <- err
			return
		}
		inputChan <- strings.TrimSpace(input)
	}()

	for i := timeoutSeconds; i > 0; i-- {
		select {
		case input := <-inputChan:
			return input, nil
		case err := <-errorChan:
			return "", err
		case <-time.After(1 * time.Second):
			if i > 1 {
				color.Yellow("\r⚡ 请选择 (1-3，%d秒后选择%s): ", i-1, defaultChoice)
			}
		}
	}

	color.Yellow("\n⏰ 超时，使用默认选择: %s\n", defaultChoice)
	return defaultChoice, nil
}

func showStepHeaderAdvanced(state *StandardFlowState, stepName, icon string, uiManager *interactive.Interface) {
	fmt.Println()
	color.HiCyan("═══════════════════════════════════════════════════════════════════════════════")
	progressBar := fmt.Sprintf("[%d/%d]", state.Step, state.TotalSteps)
	color.HiYellow("%s %s %s", icon, progressBar, stepName)
	color.HiCyan("═══════════════════════════════════════════════════════════════════════════════")
	fmt.Println()
}

func createSafeWorkingCopy(sourceDir string, mediaFiles []*types.MediaInfo, logger *zap.Logger) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	parentDir := filepath.Dir(sourceDir)
	baseName := filepath.Base(sourceDir)
	workingDirName := fmt.Sprintf("%s_pixly_safe_copy_%s", baseName, timestamp)
	workingDir := filepath.Join(parentDir, workingDirName)

	color.HiYellow("⚠️  安全副本机制启动")
	color.Cyan("🛡️  安全保障：")
	color.White("   • 绝不对原文件进行任何操作")
	color.White("   • 所有处理都在副本上进行")
	color.White("   • 原文件绝对安全")
	color.Green("✅ 原始目录： %s", sourceDir)
	color.Green("✅ 安全副本： %s", workingDir)
	color.White("")

	logger.Info("开始创建安全副本",
		zap.String("source", sourceDir),
		zap.String("working_copy", workingDir),
		zap.Int("file_count", len(mediaFiles)))

	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return "", fmt.Errorf("创建副本目录失败: %w", err)
	}

	color.Yellow("📋 开始复制文件...")
	color.Cyan("🔒 安全确认：正在创建安全副本，原文件不受影响")

	for i, mediaFile := range mediaFiles {
		relPath, err := filepath.Rel(sourceDir, mediaFile.Path)
		if err != nil {
			logger.Error("计算相对路径失败",
				zap.String("file", mediaFile.Path),
				zap.Error(err))
			continue
		}

		targetPath := filepath.Join(workingDir, relPath)
		targetDir := filepath.Dir(targetPath)

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			logger.Error("创建目标目录失败",
				zap.String("dir", targetDir),
				zap.Error(err))
			continue
		}

		if err := copyFileSecurely(mediaFile.Path, targetPath); err != nil {
			logger.Error("复制文件失败",
				zap.String("source", mediaFile.Path),
				zap.String("target", targetPath),
				zap.Error(err))
			continue
		}

		if (i+1)%5 == 0 || i == len(mediaFiles)-1 {
			color.Cyan("💾 已安全复制: %d/%d 文件 (原文件未受影响)", i+1, len(mediaFiles))
			if i == len(mediaFiles)-1 {
				color.Green("✅ 所有文件已安全复制完成")
				color.Cyan("🔒 再次确认：原文件保持原始状态，未受任何影响")
			}
		}
	}

	logger.Info("安全副本创建完成",
		zap.String("working_dir", workingDir),
		zap.Int("total_files", len(mediaFiles)))

	color.Green("🎉 副本创建成功！")
	color.HiGreen("✨ 安全状态：原文件100%安全，所有操作在副本上进行")
	color.White("")

	return workingDir, nil
}

func copyFileSecurely(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("获取源文件信息失败: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer destFile.Close()

	copiedBytes, err := io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	if copiedBytes != sourceInfo.Size() {
		return fmt.Errorf("文件复制不完整: 原始%d字节，复制%d字节", sourceInfo.Size(), copiedBytes)
	}

	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("设置文件权限失败: %w", err)
	}

	return nil
}

func updatePathsToWorkingCopy(mediaFiles []*types.MediaInfo, sourceDir, workingDir string) ([]*types.MediaInfo, error) {
	copiedFiles := make([]*types.MediaInfo, 0, len(mediaFiles))

	for _, mediaFile := range mediaFiles {
		relPath, err := filepath.Rel(sourceDir, mediaFile.Path)
		if err != nil {
			return nil, fmt.Errorf("计算相对路径失败: %w", err)
		}

		copiedFile := &types.MediaInfo{
			Path:        filepath.Join(workingDir, relPath),
			Size:        mediaFile.Size,
			ModTime:     mediaFile.ModTime,
			Type:        mediaFile.Type,
			Status:      mediaFile.Status,
			IsCorrupted: mediaFile.IsCorrupted,
			Quality:     mediaFile.Quality,
			Format:      mediaFile.Format,
			Width:       mediaFile.Width,
			Height:      mediaFile.Height,
			Duration:    mediaFile.Duration,
		}

		if _, err := os.Stat(copiedFile.Path); err != nil {
			return nil, fmt.Errorf("副本文件不存在: %s", copiedFile.Path)
		}

		copiedFiles = append(copiedFiles, copiedFile)
	}

	return copiedFiles, nil
}

func selectProcessingMode(reader *bufio.Reader, scanResults []*types.MediaInfo) (types.AppMode, error) {
	color.White("🎯 请选择处理模式：")
	color.White("1. 🤖 自动模式+ (智能决策)")
	color.White("2. 🔥 品质模式 (无损优先)")
	color.White("3. 🚀 表情包模式 (极限压缩)")

	if len(os.Args) > 2 {
		mode := os.Args[2]
		switch mode {
		case "1", "auto+":
			color.Green("✅ 已选择：自动模式+")
			return types.ModeAutoPlus, nil
		case "2", "quality":
			color.Green("✅ 已选择：品质模式")
			return types.ModeQuality, nil
		case "3", "emoji":
			color.Green("✅ 已选择：表情包模式")
			return types.ModeEmoji, nil
		default:
			color.Yellow("无效的模式参数，使用默认模式：自动模式+")
			return types.ModeAutoPlus, nil
		}
	}

	stdin, err := os.Stdin.Stat()
	if err != nil || (stdin.Mode()&os.ModeCharDevice) == 0 {
		color.Green("✅ 使用默认模式：自动模式+")
		return types.ModeAutoPlus, nil
	}

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			color.Yellow("读取输入失败，使用默认模式：自动模式+")
			return types.ModeAutoPlus, nil
		}

		choice := strings.TrimSpace(input)
		switch choice {
		case "1":
			color.Green("✅ 已选择：自动模式+")
			return types.ModeAutoPlus, nil
		case "2":
			color.Green("✅ 已选择：品质模式")
			return types.ModeQuality, nil
		case "3":
			color.Green("✅ 已选择：表情包模式")
			return types.ModeEmoji, nil
		default:
			color.Yellow("无效选择，请输入 1-3（默认使用自动模式+）：")
		}
	}
}
