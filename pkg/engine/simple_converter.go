package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"pixly/pkg/core/types"
	"pixly/pkg/processmonitor"

	"go.uber.org/zap"
)

// SimpleConverter 简化的转换器 - 调用真实的转换工具（如 cjxl, ffmpeg, avifenc）。
// 它封装了与外部工具交互的逻辑，并处理命令的构建和执行。
type SimpleConverter struct {
	logger         *zap.Logger
	toolPaths      types.ToolCheckResults // 存储外部工具的路径和能力信息。
	processMonitor *processmonitor.ProcessMonitor // 用于监控外部进程，防止卡死。
}

// NewSimpleConverter 创建简化转换器的新实例。
// nonInteractive 参数用于控制进程监控器是否以非交互模式运行。
func NewSimpleConverter(logger *zap.Logger, toolPaths types.ToolCheckResults, nonInteractive bool) *SimpleConverter {
	return &SimpleConverter{
		logger:         logger,
		toolPaths:      toolPaths,
		processMonitor: processmonitor.NewProcessMonitor(logger, nonInteractive), // 传入 nonInteractive 标志。
	}
}

// ConvertToJXL 将源文件转换为 JXL 格式。
// lossless 参数决定是进行无损转换还是有损转换。
func (sc *SimpleConverter) ConvertToJXL(ctx context.Context, sourcePath, targetPath string, lossless bool) (*types.ProcessingResult, error) {
	startTime := time.Now()

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("获取源文件信息失败: %w", err)
	}

	// 检查 cjxl 工具是否可用。
	if !sc.toolPaths.HasCjxl {
		return nil, fmt.Errorf("cjxl工具不可用")
	}

	// 构建 cjxl 命令的参数。
	var args []string
	args = append(args, sourcePath, targetPath)

	if lossless {
		// 无损模式：根据源文件类型选择不同的无损参数。
		ext := strings.ToLower(filepath.Ext(sourcePath))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".jpe" || ext == ".jfif" {
			// JPEG 无损模式：使用 --lossless_jpeg=1 选项。
			args = append(args, "--lossless_jpeg=1")
		} else {
			// 其他格式无损模式：使用 -q 100 (最高质量)。
			args = append(args, "-q", "100")
		}
		args = append(args, "-e", "7") // 适中的努力值，平衡压缩时间和文件大小。
	} else {
		// 平衡模式（有损）：使用 -q 85 (中等质量) 和 -e 8 (较高努力值)。
		args = append(args, "-q", "85", "-e", "8")
	}

	// 创建进程上下文，用于进程监控。
	processCtx := &processmonitor.ProcessContext{
		Operation:       "jxl_conversion",
		SourceFile:      sourcePath,
		FileSize:        sourceInfo.Size(),
		FileFormat:      "jxl",
		ComplexityLevel: processmonitor.ComplexityMedium,
		Priority:        processmonitor.PriorityNormal,
		Metadata:        map[string]string{"lossless": fmt.Sprintf("%t", lossless)},
	}

	// 执行 cjxl 转换命令。
	cmd := exec.CommandContext(ctx, "cjxl", args...)
	err = sc.processMonitor.MonitorCommand(ctx, cmd, processCtx)
	if err != nil {
		return &types.ProcessingResult{
			OriginalPath: sourcePath,
			OriginalSize: sourceInfo.Size(),
			Success:      false,
			Error:        fmt.Sprintf("JXL转换失败: %v", err),
			ProcessTime:  time.Since(startTime),
		}, nil
	}

	// 获取转换后文件的大小。
	var newSize int64
	if targetInfo, err := os.Stat(targetPath); err == nil {
		newSize = targetInfo.Size()
	} else {
		newSize = sourceInfo.Size() // 如果无法获取新文件大小，假设没有变化。
	}

	// 计算节省的空间。
	spaceSaved := sourceInfo.Size() - newSize

	return &types.ProcessingResult{
		OriginalPath: sourcePath,
		NewPath:      targetPath,
		OriginalSize: sourceInfo.Size(),
		NewSize:      newSize,
		SpaceSaved:   spaceSaved,
		Success:      true,
		ProcessTime:  time.Since(startTime),
	}, nil
}

// ConvertToAVIF 将源文件转换为 AVIF 格式。
// mode 参数决定转换的质量模式（如 "compressed", "balanced", "lossless"）。
// mediaType 参数用于区分静态图片和动图，以便选择合适的工具和参数。
func (sc *SimpleConverter) ConvertToAVIF(ctx context.Context, sourcePath, targetPath string, mode string, mediaType types.MediaType) (*types.ProcessingResult, error) {
	startTime := time.Now()

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("获取源文件信息失败: %w", err)
	}

	// 检查 AVIF 转换工具（ffmpeg 或 avifenc）是否可用。
	if !sc.toolPaths.HasFfmpeg && !sc.toolPaths.HasAvifenc {
		return nil, fmt.Errorf("AVIF转换工具不可用（需要FFmpeg或avifenc）")
	}

	var tool string
	var args []string

	// 优先使用 avifenc 处理静态图片，因为它通常对静态图片有更好的优化。
	if sc.toolPaths.HasAvifenc && mediaType == types.MediaTypeImage {
		tool = "avifenc"
		args = append(args, sourcePath, targetPath)

		// 根据模式设置 avifenc 参数。
		switch mode {
		case "compressed":
			args = append(args, "-q", "50", "-s", "6") // 较低质量，较快速度。
		case "balanced":
			args = append(args, "-q", "35", "-s", "8") // 平衡质量和速度。
		default: // 默认为 "lossless" 或其他未指定模式。
			args = append(args, "-q", "25", "-s", "10") // 较高质量，较慢速度。
		}
	} else if sc.toolPaths.HasFfmpeg {
		// 回退到 FFmpeg 处理，尤其适用于动图和 avifenc 不可用的情况。
		tool = "ffmpeg"
		// 构建 FFmpeg 命令参数。
		// 关键修复：使用 "-c:v av1" 来指定 AV1 编码器。
		// 之前的尝试中，"-c:v libaom-av1" 和 "-codec:v libaom-av1" 均因特定 FFmpeg 构建问题而失败。
		// 经验证，直接使用编解码器名称 "av1" 配合 "-c:v" 是有效的。
		args = []string{"-i", sourcePath, "-c:v", "av1", "-crf", "30", "-y", targetPath}
		// TODO: 根据 mode 参数调整 FFmpeg 的 CRF 值或其他 AV1 参数。
	} else {
		return nil, fmt.Errorf("没有可用的AVIF转换工具")
	}

	// 创建进程上下文，用于进程监控。
	processCtx := &processmonitor.ProcessContext{
		Operation:       "avif_conversion",
		SourceFile:      sourcePath,
		FileSize:        sourceInfo.Size(),
		FileFormat:      "avif",
		ComplexityLevel: processmonitor.ComplexityHigh, // AVIF 转换通常复杂度较高。
		Priority:        processmonitor.PriorityNormal,
		Metadata:        map[string]string{"tool": tool, "mode": mode},
	}

	// 执行转换命令。
	var cmd *exec.Cmd
	if tool == "avifenc" {
		cmd = exec.CommandContext(ctx, sc.toolPaths.AvifencPath, args...)
	} else {
		// 使用工具检查器提供的 FFmpeg 路径。
		// 之前为了调试硬编码了路径，现在恢复为动态获取，因为路径解析不是问题。
		ffmpegPath := sc.toolPaths.FfmpegStablePath
		if ffmpegPath == "" {
			ffmpegPath = sc.toolPaths.FfmpegDevPath
		}
		cmd = exec.CommandContext(ctx, ffmpegPath, args...)
	}

	err = sc.processMonitor.MonitorCommand(ctx, cmd, processCtx)
	if err != nil {
		return &types.ProcessingResult{
			OriginalPath: sourcePath,
			OriginalSize: sourceInfo.Size(),
			Success:      false,
			Error:        fmt.Sprintf("AVIF转换失败: %v", err),
			ProcessTime:  time.Since(startTime),
		}, nil
	}

	// 获取转换后文件的大小。
	var newSize int64
	if targetInfo, err := os.Stat(targetPath); err == nil {
		newSize = targetInfo.Size()
	} else {
		newSize = sourceInfo.Size()
	}

	spaceSaved := sourceInfo.Size() - newSize

	return &types.ProcessingResult{
		OriginalPath: sourcePath,
		NewPath:      targetPath,
		OriginalSize: sourceInfo.Size(),
		NewSize:      newSize,
		SpaceSaved:   spaceSaved,
		Success:      true,
		ProcessTime:  time.Since(startTime),
	}, nil
}

// RemuxVideo 对视频文件进行重封装（Remux）。
// 重封装通常是无损的，只改变容器格式而不重新编码视频流。
func (sc *SimpleConverter) RemuxVideo(ctx context.Context, sourcePath, targetPath string) (*types.ProcessingResult, error) {
	startTime := time.Now()

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("获取源文件信息失败: %w", err)
	}

	// 检查 FFmpeg 工具是否可用。
	if !sc.toolPaths.HasFfmpeg {
		return nil, fmt.Errorf("FFmpeg不可用，无法进行视频重包装")
	}

	// 构建 FFmpeg 重封装命令。
	// "-c copy" 表示直接复制视频和音频流，不进行重新编码，因此是无损的。
	args := []string{"-i", sourcePath, "-c", "copy", "-y", targetPath}

	// 创建进程上下文，用于进程监控。
	processCtx := &processmonitor.ProcessContext{
		Operation:       "video_remux",
		SourceFile:      sourcePath,
		FileSize:        sourceInfo.Size(),
		FileFormat:      "mov", // 假设目标格式为 MOV。
		ComplexityLevel: processmonitor.ComplexityLow, // 重封装复杂度较低。
		Priority:        processmonitor.PriorityNormal,
		Metadata:        map[string]string{"operation": "remux"},
	}

	// 执行 FFmpeg 重封装命令。
	// 使用工具检查器提供的 FFmpeg 路径。
	ffmpegPath := sc.toolPaths.FfmpegStablePath
	if ffmpegPath == "" {
		ffmpegPath = sc.toolPaths.FfmpegDevPath
	}

	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	err = sc.processMonitor.MonitorCommand(ctx, cmd, processCtx)
	if err != nil {
		return &types.ProcessingResult{
			OriginalPath: sourcePath,
			OriginalSize: sourceInfo.Size(),
			Success:      false,
			Error:        fmt.Sprintf("视频重包装失败: %v", err),
			ProcessTime:  time.Since(startTime),
		}, nil
	}

	// 获取转换后文件的大小。
	var newSize int64
	if targetInfo, err := os.Stat(targetPath); err == nil {
		newSize = targetInfo.Size()
	} else {
		newSize = sourceInfo.Size()
	}

	spaceSaved := sourceInfo.Size() - newSize

	return &types.ProcessingResult{
		OriginalPath: sourcePath,
		NewPath:      targetPath,
		OriginalSize: sourceInfo.Size(),
		NewSize:      newSize,
		SpaceSaved:   spaceSaved,
		Success:      true,
		ProcessTime:  time.Since(startTime),
	}, nil
}