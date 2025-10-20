package main

import (
	"fmt"
	"log"

	"pixly/pkg/core/types"
	"pixly/pkg/tools"

	"github.com/fatih/color"
	"go.uber.org/zap"
)

func main() {
	// 初始化日志
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	fmt.Println("🧪 Pixly 工具链检查测试程序")
	fmt.Println("================================")

	// 创建工具检查器
	toolChecker := tools.NewChecker(logger)

	// 执行工具检查
	toolPaths, err := toolChecker.CheckAll()
	if err != nil {
		color.Red("❌ 工具链检查失败: %v", err)
		color.Yellow("⚠️ 某些工具未找到，可能影响转换效果")
	} else {
		color.Green("✅ 工具链检查完成")
	}

	// 显示工具状态
	showToolStatus(toolPaths)
}

// showToolStatus 显示工具链状态
func showToolStatus(tools types.ToolCheckResults) {
	color.Cyan("🔧 工具链状态检查：")

	// FFmpeg状态
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
	}

	// JPEG XL (cjxl)状态
	if tools.HasCjxl {
		color.Green("  ✅ cjxl: 已找到")
		if tools.CjxlPath != "" {
			color.White("    - 路径: %s", tools.CjxlPath)
		}
	} else {
		color.Red("  ❌ cjxl: 未找到 - 建议安装: brew install jpeg-xl")
	}

	// AVIF编码器状态
	if tools.HasAvifenc {
		color.Green("  ✅ avifenc: 已找到")
		if tools.AvifencPath != "" {
			color.White("    - 路径: %s", tools.AvifencPath)
		}
	} else {
		color.Red("  ❌ avifenc: 未找到 - 建议安装: brew install libavif")
	}

	// ExifTool状态
	if tools.HasExiftool {
		color.Green("  ✅ exiftool: 已找到")
		if tools.ExiftoolPath != "" {
			color.White("    - 路径: %s", tools.ExiftoolPath)
		}
	} else {
		color.Yellow("  ⚠️ exiftool: 未找到 - 可选安装: brew install exiftool")
	}

	// 编解码器支持
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

	color.White("")
}
