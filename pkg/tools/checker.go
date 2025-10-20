package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"pixly/pkg/core/types"

	"go.uber.org/zap"
)

// Checker 工具检查器
type Checker struct {
	logger *zap.Logger
}

// NewChecker 创建工具检查器
func NewChecker(logger *zap.Logger) *Checker {
	return &Checker{
		logger: logger,
	}
}

// CheckAll 检查所有工具依赖
func (c *Checker) CheckAll() (types.ToolCheckResults, error) {
	c.logger.Info("🔍 开始检查工具依赖")

	var tools types.ToolCheckResults

	// 检查 Homebrew
	if err := c.checkHomebrew(); err != nil {
		c.logger.Warn("Homebrew 检查失败，但不阻断运行", zap.Error(err))
	}

	// 检查 FFmpeg (最重要 - README要求)
	if err := c.checkFFmpeg(&tools); err != nil {
		c.logger.Warn("FFmpeg 检查失败", zap.Error(err))
		// 不直接返回错误，继续检查其他工具
	}

	// 检查 JPEG-XL (cjxl) - 品质模式必需
	if err := c.checkJPEGXL(&tools); err != nil {
		c.logger.Warn("JPEG-XL 检查失败", zap.Error(err))
	}

	// 检查 AVIF编码器 (avifenc) - 表情包模式必需
	if err := c.checkAvifenc(&tools); err != nil {
		c.logger.Warn("AVIF编码器 检查失败", zap.Error(err))
	}

	// 检查 exiftool - 元数据迁移必需
	if err := c.checkExiftool(&tools); err != nil {
		c.logger.Warn("exiftool 检查失败", zap.Error(err))
	}

	// 统计检查结果
	c.logToolCheckSummary(&tools)

	c.logger.Info("✅ 工具依赖检查完成")
	return tools, nil
}

// logToolCheckSummary 记录工具检查统计
func (c *Checker) logToolCheckSummary(tools *types.ToolCheckResults) {
	totalTools := 0
	availableTools := 0

	// 统计所有工具
	if tools.HasFfmpeg {
		availableTools++
	}
	totalTools++

	if tools.HasCjxl {
		availableTools++
	}
	totalTools++

	if tools.HasAvifenc {
		availableTools++
	}
	totalTools++

	if tools.HasExiftool {
		availableTools++
	}
	totalTools++

	c.logger.Info("📊 工具检查统计",
		zap.Int("可用工具", availableTools),
		zap.Int("总工具数", totalTools),
		zap.Float64("可用率", float64(availableTools)/float64(totalTools)*100))

	// 给出安装建议
	c.suggestMissingTools(tools)
}

// suggestMissingTools 给出缺失工具的安装建议
func (c *Checker) suggestMissingTools(tools *types.ToolCheckResults) {
	missingTools := make([]string, 0)

	if !tools.HasFfmpeg {
		missingTools = append(missingTools, "FFmpeg: brew install ffmpeg")
	}
	if !tools.HasCjxl {
		missingTools = append(missingTools, "JPEG-XL: brew install jpeg-xl")
	}
	if !tools.HasAvifenc {
		missingTools = append(missingTools, "AVIF编码器: brew install libavif")
	}
	if !tools.HasExiftool {
		missingTools = append(missingTools, "exiftool: brew install exiftool")
	}

	if len(missingTools) > 0 {
		c.logger.Info("📝 缺失工具安装建议:")
		for _, tool := range missingTools {
			c.logger.Info(fmt.Sprintf("   %s", tool))
		}
	} else {
		c.logger.Info("🎉 所有工具均已安装")
	}
}

// checkHomebrew 检查 Homebrew
func (c *Checker) checkHomebrew() error {
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("Homebrew 未安装，无法自动安装依赖。请访问 https://brew.sh/ 安装")
	}
	c.logger.Info("✅ Homebrew 已安装")
	return nil
}

// checkJPEGXL 检查 JPEG-XL 工具
func (c *Checker) checkJPEGXL(tools *types.ToolCheckResults) error {
	if _, err := exec.LookPath("cjxl"); err == nil {
		tools.HasCjxl = true
		c.logger.Info("✅ cjxl 已找到")
		return nil
	}

	c.logger.Warn("⚠️  cjxl 未找到，建议安装: brew install jpeg-xl")
	return nil
}

// checkFFmpeg 检查 FFmpeg 工具 - 应用关键修复
func (c *Checker) checkFFmpeg(tools *types.ToolCheckResults) error {
	c.logger.Info("🔍 检查 FFmpeg 工具")

	// 动态检测FFmpeg和ffprobe路径 - 关键修复
	var ffmpegPath string
	var ffprobePath string

	// 首先尝试查找系统中的ffmpeg
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		ffmpegPath = path
		tools.HasFfmpeg = true
		c.logger.Info("✅ 找到 ffmpeg", zap.String("path", path))
	}

	// 尝试查找ffprobe（通常和ffmpeg在同一目录）
	if path, err := exec.LookPath("ffprobe"); err == nil {
		ffprobePath = path
		c.logger.Info("✅ 找到 ffprobe", zap.String("path", path))
	} else if ffmpegPath != "" {
		// 如果找不到ffprobe但找到了ffmpeg，尝试从ffmpeg目录推断ffprobe路径
		ffprobeCandidate := filepath.Join(filepath.Dir(ffmpegPath), "ffprobe")
		if _, err := os.Stat(ffprobeCandidate); err == nil {
			ffprobePath = ffprobeCandidate
			c.logger.Info("✅ 推断 ffprobe 路径", zap.String("path", ffprobeCandidate))
		}
	}

	// 检查内嵌版本
	c.checkEmbeddedFFmpeg(tools)

	// 如果仍未找到，使用默认命令名称
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
		c.logger.Warn("⚠️  未找到 ffmpeg 具体路径，使用默认命令名")
	}
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
		c.logger.Warn("⚠️  未找到 ffprobe 具体路径，使用默认命令名")
	}

	// 关键修复：正确映射路径
	tools.FfmpegDevPath = ffmpegPath
	tools.FfmpegStablePath = ffprobePath // 品质评估引擎使用这个作为 ffprobe 路径

	// 验证版本
	if err := c.validateFFmpegVersion(ffmpegPath); err != nil {
		c.logger.Warn("FFmpeg 版本验证失败", zap.Error(err))
	}

	// 检查编解码器支持
	c.checkFFmpegCodecs(ffmpegPath, tools)

	if !tools.HasFfmpeg {
		return fmt.Errorf("未找到可用的 FFmpeg 版本")
	}

	c.logger.Info("✅ FFmpeg 检查完成",
		zap.String("ffmpeg_path", tools.FfmpegDevPath),
		zap.String("ffprobe_path", tools.FfmpegStablePath))

	return nil
}

// checkEmbeddedFFmpeg 检查内嵌的 FFmpeg 版本
func (c *Checker) checkEmbeddedFFmpeg(tools *types.ToolCheckResults) {
	execDir := filepath.Dir(os.Args[0])

	// 检查内嵌开发版
	embeddedDev := filepath.Join(execDir, "bin", "ffmpeg-dev")
	if _, err := os.Stat(embeddedDev); err == nil {
		if c.validateFFmpegVersion(embeddedDev) == nil {
			tools.FfmpegDevPath = embeddedDev
			c.logger.Info("✅ 找到内嵌 FFmpeg 开发版", zap.String("path", embeddedDev))
		}
	}

	// 检查内嵌稳定版
	embeddedStable := filepath.Join(execDir, "bin", "ffmpeg-stable")
	if _, err := os.Stat(embeddedStable); err == nil {
		if c.validateFFmpegVersion(embeddedStable) == nil {
			tools.FfmpegStablePath = embeddedStable
			c.logger.Info("✅ 找到内嵌 FFmpeg 稳定版", zap.String("path", embeddedStable))
		}
	}
}

// validateFFmpegVersion 验证 FFmpeg 版本
func (c *Checker) validateFFmpegVersion(ffmpegPath string) error {
	out, err := exec.Command(ffmpegPath, "-version").Output()
	if err != nil {
		return fmt.Errorf("获取版本信息失败: %w", err)
	}

	re := regexp.MustCompile(`ffmpeg version (n?([0-9]+)(\.[0-9]+)*)`)
	matches := re.FindStringSubmatch(string(out))
	if len(matches) < 3 {
		return fmt.Errorf("无法解析版本信息")
	}

	versionStr := matches[1]
	parts := strings.Split(versionStr, ".")
	if len(parts) == 0 {
		return fmt.Errorf("版本格式错误")
	}

	majorVer, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("版本解析错误: %w", err)
	}

	if majorVer < 4 {
		return fmt.Errorf("版本过低 (需要 v4+, 找到 %s)", versionStr)
	}

	c.logger.Info("✅ FFmpeg 版本验证通过",
		zap.String("path", ffmpegPath),
		zap.String("version", versionStr))
	return nil
}

// checkFFmpegCodecs 检查 FFmpeg 编解码器支持
func (c *Checker) checkFFmpegCodecs(ffmpegPath string, tools *types.ToolCheckResults) {
	out, err := exec.Command(ffmpegPath, "-codecs").Output()
	if err != nil {
		c.logger.Warn("获取编解码器信息失败", zap.Error(err))
		return
	}

	codecInfo := string(out)

	// 检查AV1编解码器支持（FFmpeg 8.0 AVIF支持的核心）
	if strings.Contains(codecInfo, "av1") {
		// 进一步检查具体的AV1编码器
		if strings.Contains(codecInfo, "libaom-av1") {
			tools.HasLibaom = true
			c.logger.Info("✅ libaom-av1 支持 (AVIF基础编码)")
		}
		if strings.Contains(codecInfo, "libsvtav1") {
			tools.HasLibSvtAv1 = true
			c.logger.Info("✅ libsvtav1 支持 (AVIF高质量编码)")
		}
		if strings.Contains(codecInfo, "libdav1d") {
			tools.HasLibdav1d = true
			c.logger.Info("✅ libdav1d 支持 (AVIF解码)")
		}
	}

	// 检查JPEG-XL支持
	if strings.Contains(codecInfo, "libjxl") {
		tools.HasLibjxl = true
		c.logger.Info("✅ libjxl 支持 (JPEG-XL编解码)")
	}

	if strings.Contains(codecInfo, "videotoolbox") {
		tools.HasVToolbox = true
		c.logger.Info("✅ VideoToolbox 支持 (macOS 硬件加速)")
	}

	if strings.Contains(codecInfo, "libx264") {
		tools.HasLibx264 = true
		c.logger.Info("✅ libx264 支持")
	}

	if strings.Contains(codecInfo, "libx265") {
		tools.HasLibx265 = true
		c.logger.Info("✅ libx265 支持")
	}

	// 检查独立的avifenc工具
	c.checkAvifencTool(tools)
}

// checkExiftool 检查 exiftool
func (c *Checker) checkExiftool(tools *types.ToolCheckResults) error {
	if path, err := exec.LookPath("exiftool"); err == nil {
		tools.HasExiftool = true
		tools.ExiftoolPath = path
		c.logger.Info("✅ exiftool 已找到", zap.String("path", path))
		return nil
	}

	c.logger.Warn("⚠️  exiftool 未找到，建议安装: brew install exiftool")
	return nil
}

// checkAvifenc 检查 AVIF 编码器 - 独立检查函数
func (c *Checker) checkAvifenc(tools *types.ToolCheckResults) error {
	c.logger.Info("🔍 检查 AVIF 编码器")

	// 检查独立的 avifenc 工具
	if path, err := exec.LookPath("avifenc"); err == nil {
		tools.HasAvifenc = true
		tools.AvifencPath = path
		c.logger.Info("✅ avifenc 已找到", zap.String("path", path))

		// 验证 avifenc 版本
		if err := c.validateAvifencVersion(path); err != nil {
			c.logger.Warn("avifenc 版本验证失败", zap.Error(err))
		}
		return nil
	}

	c.logger.Warn("⚠️  avifenc 未找到，将使用 FFmpeg 作为 AVIF 编码器")
	c.logger.Info("📝 建议安装: brew install libavif")
	return nil
}

// checkAvifencTool 检查独立的avifenc工具
func (c *Checker) checkAvifencTool(tools *types.ToolCheckResults) {
	if path, err := exec.LookPath("avifenc"); err == nil {
		tools.HasAvifenc = true
		tools.AvifencPath = path
		c.logger.Info("✅ avifenc 已找到", zap.String("path", path))

		// 验证avifenc版本
		if err := c.validateAvifencVersion(path); err != nil {
			c.logger.Warn("avifenc 版本验证失败", zap.Error(err))
		}
	} else {
		c.logger.Warn("⚠️  avifenc 未找到，建议安装: brew install libavif")
	}
}

// validateAvifencVersion 验证avifenc版本
func (c *Checker) validateAvifencVersion(avifencPath string) error {
	out, err := exec.Command(avifencPath, "--version").Output()
	if err != nil {
		return fmt.Errorf("获取版本信息失败: %w", err)
	}

	versionInfo := string(out)
	c.logger.Info("✅ avifenc 版本验证通过",
		zap.String("path", avifencPath),
		zap.String("version_info", strings.TrimSpace(versionInfo)))
	return nil
}
