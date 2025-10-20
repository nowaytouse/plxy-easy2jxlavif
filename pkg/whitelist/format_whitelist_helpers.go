package whitelist

import (
	"fmt"
	"path/filepath"
	"strings"

	"pixly/pkg/core/types"

	"go.uber.org/zap"
)

// isSystemHiddenFile 检查是否是系统隐藏文件 - README 5.2节要求
func (fw *FormatWhitelist) isSystemHiddenFile(fileName string) bool {
	// README要求：以.开头的文件或系统定义的隐藏文件
	if strings.HasPrefix(fileName, ".") {
		return true
	}

	// 常见的系统文件
	systemFiles := []string{
		"Thumbs.db",   // Windows缩略图
		"Desktop.ini", // Windows桌面配置
		".DS_Store",   // macOS目录配置
		".localized",  // macOS本地化标记
		"__MACOSX",    // macOS压缩包文件
		"Icon\r",      // macOS图标文件
	}

	for _, systemFile := range systemFiles {
		if fileName == systemFile {
			return true
		}
	}

	return false
}

// isTargetFormat 检查是否是目标格式 - README 5.2节要求
func (fw *FormatWhitelist) isTargetFormat(ext string, mode types.AppMode) bool {
	if targetFormats, exists := fw.targetFormatsByMode[mode]; exists {
		return targetFormats[ext]
	}
	return false
}

// checkSkipRules 检查跳过规则
func (fw *FormatWhitelist) checkSkipRules(ext, filePath string) *SkipReason {
	// 检查特殊媒体类型
	if skipReason, exists := fw.skippedSpecialTypes[ext]; exists {
		return &skipReason
	}

	// 检查系统文件类型
	if skipReason, exists := fw.skippedSystemTypes[ext]; exists {
		return &skipReason
	}

	// 检查创作源文件
	if skipReason, exists := fw.skippedCreativeTypes[ext]; exists {
		return &skipReason
	}

	// 特殊路径检查（如果需要）
	if fw.isSpecialPath(filePath) {
		return &SkipReason{
			Category:    SkipSystemHidden,
			Reason:      "特殊路径",
			Description: "位于系统或隐藏目录中",
		}
	}

	return nil
}

// getFormatInfo 获取格式信息
func (fw *FormatWhitelist) getFormatInfo(ext string) *FormatInfo {
	// 查找图片格式
	if info, exists := fw.supportedImageFormats[ext]; exists {
		return &info
	}

	// 查找视频格式
	if info, exists := fw.supportedVideoFormats[ext]; exists {
		return &info
	}

	// 查找音频格式
	if info, exists := fw.supportedAudioFormats[ext]; exists {
		return &info
	}

	return nil
}

// isSpecialPath 检查是否是特殊路径
func (fw *FormatWhitelist) isSpecialPath(filePath string) bool {
	// 系统目录检查
	systemPaths := []string{
		"/System/",
		"/Library/",
		"/usr/",
		"/bin/",
		"/sbin/",
		"/var/",
		"/tmp/",
		"/private/",
		"/.Trash/",
		"/Applications/",
	}

	for _, systemPath := range systemPaths {
		if strings.HasPrefix(filePath, systemPath) {
			return true
		}
	}

	return false
}

// BatchCheckFiles 批量检查文件
func (fw *FormatWhitelist) BatchCheckFiles(filePaths []string, mode types.AppMode) ([]*CheckResult, error) {
	results := make([]*CheckResult, 0, len(filePaths))

	fw.logger.Info("开始批量文件格式检查",
		zap.Int("file_count", len(filePaths)),
		zap.String("mode", mode.String()))

	for _, filePath := range filePaths {
		result := fw.CheckFile(filePath, mode)
		results = append(results, result)
	}

	fw.logger.Info("批量文件格式检查完成",
		zap.Int("total_files", len(results)),
		zap.Int64("supported_files", fw.whitelistStats.SupportedFiles),
		zap.Int64("skipped_files", fw.whitelistStats.SkippedFiles))

	return results, nil
}

// FilterSupportedFiles 过滤出支持的文件
func (fw *FormatWhitelist) FilterSupportedFiles(filePaths []string, mode types.AppMode) ([]string, []string, error) {
	supportedFiles := make([]string, 0)
	skippedFiles := make([]string, 0)

	for _, filePath := range filePaths {
		result := fw.CheckFile(filePath, mode)

		if result.IsSupported && !result.ShouldSkip {
			supportedFiles = append(supportedFiles, filePath)
		} else {
			skippedFiles = append(skippedFiles, filePath)
		}
	}

	fw.logger.Info("文件过滤完成",
		zap.Int("total_files", len(filePaths)),
		zap.Int("supported_files", len(supportedFiles)),
		zap.Int("skipped_files", len(skippedFiles)))

	return supportedFiles, skippedFiles, nil
}

// GetSupportedExtensions 获取支持的扩展名列表
func (fw *FormatWhitelist) GetSupportedExtensions() []string {
	extensions := make([]string, 0)

	// 添加图片格式
	for ext, info := range fw.supportedImageFormats {
		if info.IsSupported {
			extensions = append(extensions, ext)
		}
	}

	// 添加视频格式
	for ext, info := range fw.supportedVideoFormats {
		if info.IsSupported {
			extensions = append(extensions, ext)
		}
	}

	return extensions
}

// GetFormatsByMediaType 按媒体类型获取格式
func (fw *FormatWhitelist) GetFormatsByMediaType(mediaType types.MediaType) []string {
	extensions := make([]string, 0)

	checkFormats := func(formats map[string]FormatInfo) {
		for ext, info := range formats {
			if info.IsSupported && info.MediaType == mediaType {
				extensions = append(extensions, ext)
			}
		}
	}

	checkFormats(fw.supportedImageFormats)
	checkFormats(fw.supportedVideoFormats)

	return extensions
}

// IsLivePhoto 检测是否是Live Photo - README 5.2节特殊处理
func (fw *FormatWhitelist) IsLivePhoto(filePath string) (bool, error) {
	// Live Photo检测逻辑
	// 1. 检查文件名模式
	// 2. 检查是否有配对的MOV文件
	// 3. 检查EXIF元数据中的ContentIdentifier标签

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	if ext != "heic" && ext != "jpg" && ext != "jpeg" {
		return false, nil // 只有HEIC和JPEG可能是Live Photo
	}

	// 检查配对的MOV文件
	baseName := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	movPath := baseName + ".MOV"

	if _, err := filepath.Glob(movPath); err == nil {
		// 找到配对的MOV文件，很可能是Live Photo
		fw.logger.Debug("检测到可能的Live Photo",
			zap.String("image_file", filePath),
			zap.String("video_file", movPath))
		return true, nil
	}

	return false, nil
}

// IsSpatialMedia 检测是否是空间图片/视频 - README 5.2节特殊处理
func (fw *FormatWhitelist) IsSpatialMedia(filePath string) (bool, error) {
	// 空间媒体检测逻辑
	// 这需要分析EXIF/元数据来检测VR/AR标记

	// 常见的空间媒体文件扩展名或标记
	spatialIndicators := []string{
		"_vr", "_360", "_spatial", "_ar",
	}

	fileName := strings.ToLower(filepath.Base(filePath))
	for _, indicator := range spatialIndicators {
		if strings.Contains(fileName, indicator) {
			fw.logger.Debug("检测到可能的空间媒体",
				zap.String("file_path", filePath),
				zap.String("indicator", indicator))
			return true, nil
		}
	}

	return false, nil
}

// GenerateSkipReport 生成跳过报告
func (fw *FormatWhitelist) GenerateSkipReport() string {
	var report strings.Builder

	report.WriteString("=== 格式白名单跳过统计报告 ===\n\n")

	// 总体统计
	report.WriteString(fmt.Sprintf("总检查文件数: %d\n", fw.whitelistStats.TotalFilesChecked))
	report.WriteString(fmt.Sprintf("支持的文件数: %d\n", fw.whitelistStats.SupportedFiles))
	report.WriteString(fmt.Sprintf("跳过的文件数: %d\n", fw.whitelistStats.SkippedFiles))
	report.WriteString(fmt.Sprintf("不支持文件数: %d\n\n", fw.whitelistStats.UnsupportedFiles))

	// 跳过原因统计
	report.WriteString("跳过原因分布:\n")
	for category, count := range fw.whitelistStats.SkipReasonCounts {
		if count > 0 {
			report.WriteString(fmt.Sprintf("  %s: %d 文件\n", category.String(), count))
		}
	}
	report.WriteString("\n")

	// 支持格式分布
	report.WriteString("支持的图片格式分布:\n")
	for ext, count := range fw.whitelistStats.ImageFormatCounts {
		if count > 0 {
			report.WriteString(fmt.Sprintf("  .%s: %d 文件\n", ext, count))
		}
	}
	report.WriteString("\n")

	report.WriteString("支持的视频格式分布:\n")
	for ext, count := range fw.whitelistStats.VideoFormatCounts {
		if count > 0 {
			report.WriteString(fmt.Sprintf("  .%s: %d 文件\n", ext, count))
		}
	}

	return report.String()
}

// GetStats 获取统计信息
func (fw *FormatWhitelist) GetStats() *WhitelistStats {
	return fw.whitelistStats
}

// ResetStats 重置统计信息
func (fw *FormatWhitelist) ResetStats() {
	fw.whitelistStats = &WhitelistStats{
		ImageFormatCounts: make(map[string]int64),
		VideoFormatCounts: make(map[string]int64),
		AudioFormatCounts: make(map[string]int64),
		SkipReasonCounts:  make(map[SkipCategory]int64),
		ModeFormatCounts:  make(map[types.AppMode]map[string]int64),
	}

	// 重新初始化模式统计
	for mode := range fw.targetFormatsByMode {
		fw.whitelistStats.ModeFormatCounts[mode] = make(map[string]int64)
	}
}

// AddCustomSkipRule 添加自定义跳过规则
func (fw *FormatWhitelist) AddCustomSkipRule(extension string, category SkipCategory, reason, description string) {
	skipReason := SkipReason{
		Category:    category,
		Reason:      reason,
		Description: description,
	}

	switch category {
	case SkipSpecialMedia:
		fw.skippedSpecialTypes[extension] = skipReason
	case SkipCreativeSource:
		fw.skippedCreativeTypes[extension] = skipReason
	default:
		fw.skippedSystemTypes[extension] = skipReason
	}

	fw.logger.Info("添加自定义跳过规则",
		zap.String("extension", extension),
		zap.String("category", category.String()),
		zap.String("reason", reason))
}

// AddCustomFormat 添加自定义格式支持
func (fw *FormatWhitelist) AddCustomFormat(extension, mimeType string, mediaType types.MediaType, isSupported bool, toolRequired, notes string) {
	formatInfo := FormatInfo{
		Extension:    extension,
		MimeType:     mimeType,
		MediaType:    mediaType,
		Description:  fmt.Sprintf("自定义格式: %s", extension),
		IsSupported:  isSupported,
		ToolRequired: toolRequired,
		Notes:        notes,
	}

	switch mediaType {
	case types.MediaTypeImage, types.MediaTypeAnimated:
		fw.supportedImageFormats[extension] = formatInfo
	case types.MediaTypeVideo:
		fw.supportedVideoFormats[extension] = formatInfo
	default:
		fw.supportedAudioFormats[extension] = formatInfo
	}

	fw.logger.Info("添加自定义格式支持",
		zap.String("extension", extension),
		zap.String("media_type", mediaType.String()),
		zap.Bool("is_supported", isSupported))
}
