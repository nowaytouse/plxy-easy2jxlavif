package formatsupport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// addFormat 添加格式支持
func (fsm *FormatSupportManager) addFormat(formatInfo *FormatInfo) {
	formatInfo.LastUpdated = time.Now()

	// 添加到主映射
	key := strings.ToLower(formatInfo.Name)
	fsm.supportedFormats[key] = formatInfo

	// 为每个扩展名创建映射
	for _, ext := range formatInfo.Extensions {
		extKey := strings.ToLower(ext)
		fsm.supportedFormats[extKey] = formatInfo
	}

	// 添加到类别映射
	if fsm.formatCategories[formatInfo.Category] == nil {
		fsm.formatCategories[formatInfo.Category] = make([]string, 0)
	}
	fsm.formatCategories[formatInfo.Category] = append(fsm.formatCategories[formatInfo.Category], key)
}

// detectFormatByExtension 通过文件扩展名检测格式
func (fsm *FormatSupportManager) detectFormatByExtension(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	// 查找扩展名对应的格式
	if formatInfo, exists := fsm.supportedFormats[ext]; exists {
		return strings.ToLower(formatInfo.Name)
	}

	return ""
}

// detectFormatByContent 通过文件内容检测格式
func (fsm *FormatSupportManager) detectFormatByContent(ctx context.Context, filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		fsm.logger.Debug("无法打开文件进行内容检测", zap.String("file", filepath.Base(filePath)), zap.Error(err))
		return ""
	}
	defer file.Close()

	// 读取文件头部分
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil || n == 0 {
		return ""
	}

	buffer = buffer[:n]

	// 检测常见格式的魔数
	magicNumbers := map[string]string{
		"\xFF\xD8\xFF":                 "jpeg",
		"\x89PNG\r\n\x1A\n":            "png",
		"RIFF":                         "webp", // 需要进一步检查WEBP
		"\x00\x00\x00\x20ftypavif":     "avif",
		"\x00\x00\x00\x0CJXLb\x1A\x1A": "jxl",
		"\x00\x00\x00\x18ftypheic":     "heic",
		"\x00\x00\x00\x20ftypMSNV":     "mp4",
		"\x1A\x45\xDF\xA3":             "webm",
	}

	bufferStr := string(buffer)

	for magic, format := range magicNumbers {
		if strings.HasPrefix(bufferStr, magic) {
			// 对于RIFF格式，需要进一步检查是否为WebP
			if format == "webp" && len(buffer) >= 12 {
				if strings.Contains(string(buffer[8:12]), "WEBP") {
					return "webp"
				}
				continue
			}
			return format
		}
	}

	fsm.logger.Debug("未能通过内容检测到格式", zap.String("file", filepath.Base(filePath)))
	return ""
}

// handleUnsupportedFormat 处理不支持的格式
func (fsm *FormatSupportManager) handleUnsupportedFormat(filePath, detectedFormat string) (*FormatSupportResult, error) {
	fsm.stats.FailedDetections++

	action := fsm.config.DefaultAction
	recommendations := []string{}

	// 生成建议
	if detectedFormat != "" {
		recommendations = append(recommendations, fmt.Sprintf("检测到格式: %s（不支持）", detectedFormat))
	} else {
		recommendations = append(recommendations, "无法识别文件格式")
	}

	// 根据配置确定动作
	switch fsm.config.DefaultAction {
	case ActionSkip:
		recommendations = append(recommendations, "建议跳过此文件")
		fsm.stats.SkippedFiles++
	case ActionConvert:
		recommendations = append(recommendations, "建议转换为支持的格式")
		action = ActionConvert
	case ActionPrompt:
		recommendations = append(recommendations, "请手动确认处理方式")
	}

	result := &FormatSupportResult{
		FilePath:        filePath,
		DetectedFormat:  detectedFormat,
		FormatInfo:      nil,
		Supported:       false,
		Status:          StatusUnsupported,
		Action:          action,
		Recommendations: recommendations,
		ConversionPaths: []ConversionPath{},
		DetectionTime:   0,
	}

	fsm.logger.Info("检测到不支持的格式",
		zap.String("file", filepath.Base(filePath)),
		zap.String("detected_format", detectedFormat),
		zap.String("action", action.String()))

	return result, nil
}

// evaluateSupportStatus 评估支持状态
func (fsm *FormatSupportManager) evaluateSupportStatus(formatInfo *FormatInfo) SupportStatus {
	// 检查实验性格式
	if formatInfo.Experimental && !fsm.config.AllowExperimental {
		return StatusExperimental
	}

	// 根据支持级别确定状态
	switch formatInfo.SupportLevel {
	case SupportFull:
		return StatusSupported
	case SupportPartial, SupportBasic:
		return StatusSupported // 部分支持也认为是支持的
	case SupportExperimental:
		if fsm.config.AllowExperimental {
			return StatusExperimental
		}
		return StatusUnsupported
	case SupportLegacy:
		return StatusDeprecated
	case SupportNone:
		return StatusUnsupported
	default:
		return StatusUnknown
	}
}

// generateRecommendations 生成处理建议
func (fsm *FormatSupportManager) generateRecommendations(formatInfo *FormatInfo, filePath string) []string {
	recommendations := []string{}

	// 基于格式信息生成建议
	recommendations = append(recommendations, fmt.Sprintf("格式: %s (%s)", formatInfo.Name, formatInfo.RecommendedUse))

	// 支持级别建议
	switch formatInfo.SupportLevel {
	case SupportFull:
		recommendations = append(recommendations, "完全支持，推荐使用")
	case SupportPartial:
		recommendations = append(recommendations, "部分支持，可能有功能限制")
	case SupportBasic:
		recommendations = append(recommendations, "基础支持，建议转换为更好的格式")
	case SupportExperimental:
		recommendations = append(recommendations, "实验性支持，请谨慎使用")
	case SupportLegacy:
		recommendations = append(recommendations, "传统格式，建议更新到现代格式")
	}

	// 处理提示
	if formatInfo.ProcessingHints != nil {
		if len(formatInfo.ProcessingHints.BestPractices) > 0 {
			recommendations = append(recommendations, "最佳实践: "+strings.Join(formatInfo.ProcessingHints.BestPractices, ", "))
		}

		if len(formatInfo.ProcessingHints.PerformanceNotes) > 0 {
			recommendations = append(recommendations, "性能提示: "+strings.Join(formatInfo.ProcessingHints.PerformanceNotes, ", "))
		}
	}

	// 限制说明
	if len(formatInfo.Limitations) > 0 {
		recommendations = append(recommendations, "注意限制: "+strings.Join(formatInfo.Limitations, ", "))
	}

	return recommendations
}

// findConversionPaths 查找转换路径
func (fsm *FormatSupportManager) findConversionPaths(sourceFormat string) []ConversionPath {
	paths := []ConversionPath{}

	sourceInfo, exists := fsm.supportedFormats[sourceFormat]
	if !exists {
		return paths
	}

	// 获取推荐的转换目标
	for _, target := range sourceInfo.ConversionTargets {
		targetInfo, targetExists := fsm.supportedFormats[target]
		if !targetExists {
			continue
		}

		// 评估转换质量影响
		qualityImpact := fsm.evaluateQualityImpact(sourceInfo, targetInfo)

		// 估算转换时间
		estimatedTime := fsm.estimateConversionTime(sourceFormat, target)

		path := ConversionPath{
			TargetFormat:    target,
			Method:          "ffmpeg",
			QualityImpact:   qualityImpact,
			PerformanceHint: fsm.getPerformanceHint(sourceFormat, target),
			RequiredTools:   []string{"ffmpeg"},
			EstimatedTime:   estimatedTime,
			Confidence:      fsm.calculateConversionConfidence(sourceFormat, target),
		}

		paths = append(paths, path)
	}

	// 如果没有直接转换路径，查找通用转换
	if len(paths) == 0 {
		paths = fsm.findFallbackConversions(sourceFormat)
	}

	return paths
}

// evaluateQualityImpact 评估质量影响
func (fsm *FormatSupportManager) evaluateQualityImpact(source, target *FormatInfo) QualityImpact {
	// 无损到无损：无影响
	if source.Quality == QualityLossless && target.Quality == QualityLossless {
		return ImpactNone
	}

	// 无损到有损：中等影响
	if source.Quality == QualityLossless && target.Quality != QualityLossless {
		return ImpactModerate
	}

	// 有损到无损：无额外影响（但不能恢复质量）
	if source.Quality != QualityLossless && target.Quality == QualityLossless {
		return ImpactMinimal
	}

	// 有损到有损：根据质量等级判断
	qualityOrder := map[QualitySupport]int{
		QualityHighLossy:   3,
		QualityMediumLossy: 2,
		QualityLowLossy:    1,
		QualityVariable:    2, // 可变质量假设为中等
	}

	sourceLevel := qualityOrder[source.Quality]
	targetLevel := qualityOrder[target.Quality]

	if targetLevel >= sourceLevel {
		return ImpactMinimal
	} else if targetLevel == sourceLevel-1 {
		return ImpactModerate
	} else {
		return ImpactHigh
	}
}

// estimateConversionTime 估算转换时间
func (fsm *FormatSupportManager) estimateConversionTime(source, target string) time.Duration {
	// 基于格式复杂度的简单估算
	conversionTimes := map[string]map[string]time.Duration{
		"jpeg": {
			"webp": 2 * time.Second,
			"avif": 5 * time.Second,
			"jxl":  3 * time.Second,
			"png":  1 * time.Second,
		},
		"png": {
			"jpeg": 1 * time.Second,
			"webp": 3 * time.Second,
			"avif": 6 * time.Second,
			"jxl":  4 * time.Second,
		},
		"webp": {
			"jpeg": 2 * time.Second,
			"png":  3 * time.Second,
			"avif": 4 * time.Second,
			"jxl":  3 * time.Second,
		},
	}

	if sourceMap, exists := conversionTimes[source]; exists {
		if time, exists := sourceMap[target]; exists {
			return time
		}
	}

	// 默认估算时间
	return 3 * time.Second
}

// getPerformanceHint 获取性能提示
func (fsm *FormatSupportManager) getPerformanceHint(source, target string) string {
	hints := map[string]map[string]string{
		"jpeg": {
			"webp": "快速转换，轻微质量提升",
			"avif": "较慢转换，显著质量提升",
			"jxl":  "中等转换，质量提升明显",
		},
		"png": {
			"jpeg": "快速转换，文件大小显著减小",
			"webp": "中等转换，保持透明度",
			"avif": "较慢转换，最佳压缩比",
		},
	}

	if sourceMap, exists := hints[source]; exists {
		if hint, exists := sourceMap[target]; exists {
			return hint
		}
	}

	return "标准转换"
}

// calculateConversionConfidence 计算转换信心度
func (fsm *FormatSupportManager) calculateConversionConfidence(source, target string) float64 {
	// 基于格式支持度和转换历史的信心度
	confidence := 0.8 // 基础信心度

	sourceInfo := fsm.supportedFormats[source]
	targetInfo := fsm.supportedFormats[target]

	if sourceInfo != nil && sourceInfo.SupportLevel == SupportFull {
		confidence += 0.1
	}

	if targetInfo != nil && targetInfo.SupportLevel == SupportFull {
		confidence += 0.1
	}

	// 限制在0-1范围内
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// findFallbackConversions 查找后备转换方案
func (fsm *FormatSupportManager) findFallbackConversions(sourceFormat string) []ConversionPath {
	// 通用后备转换：转换为JPEG（最通用的格式）
	fallbackPaths := []ConversionPath{
		{
			TargetFormat:    "jpeg",
			Method:          "ffmpeg",
			QualityImpact:   ImpactModerate,
			PerformanceHint: "通用格式转换",
			RequiredTools:   []string{"ffmpeg"},
			EstimatedTime:   5 * time.Second,
			Confidence:      0.7,
		},
	}

	return fallbackPaths
}

// determineAction 确定处理动作
func (fsm *FormatSupportManager) determineAction(status SupportStatus, formatInfo *FormatInfo) DefaultAction {
	switch status {
	case StatusSupported:
		return ActionAllow
	case StatusDeprecated:
		if fsm.config.EnableConversionHints && len(formatInfo.ConversionTargets) > 0 {
			return ActionConvert
		}
		return ActionAllow
	case StatusExperimental:
		if fsm.config.AllowExperimental {
			return ActionAllow
		}
		return ActionSkip
	case StatusUnsupported:
		return fsm.config.DefaultAction
	default:
		return ActionSkip
	}
}

// 缓存相关方法
func (fsm *FormatSupportManager) getCachedResult(filePath string) *CachedFormatResult {
	fsm.mutex.RLock()
	defer fsm.mutex.RUnlock()

	cached, exists := fsm.formatCache[filePath]
	if !exists {
		return nil
	}

	// 检查是否过期
	if time.Now().After(cached.ExpiresAt) {
		delete(fsm.formatCache, filePath)
		return nil
	}

	return cached
}

func (fsm *FormatSupportManager) cacheResult(filePath string, result *FormatSupportResult) {
	fsm.mutex.Lock()
	defer fsm.mutex.Unlock()

	cached := &CachedFormatResult{
		DetectedFormat:  result.DetectedFormat,
		SupportStatus:   result.Status,
		Recommendations: result.Recommendations,
		ConversionPaths: result.ConversionPaths,
		CacheTime:       time.Now(),
		ExpiresAt:       time.Now().Add(fsm.config.CacheTimeout),
		Metadata:        make(map[string]interface{}),
	}

	fsm.formatCache[filePath] = cached

	// 限制缓存大小
	if len(fsm.formatCache) > 1000 {
		fsm.cleanupCache()
	}
}

func (fsm *FormatSupportManager) cleanupCache() {
	// 删除最旧的10%缓存条目
	deleteCount := len(fsm.formatCache) / 10
	deletedCount := 0

	for filePath, cached := range fsm.formatCache {
		if deletedCount >= deleteCount {
			break
		}

		// 删除过期的或最旧的条目
		if time.Now().After(cached.ExpiresAt) || deletedCount < deleteCount {
			delete(fsm.formatCache, filePath)
			deletedCount++
		}
	}
}

func (fsm *FormatSupportManager) convertCachedResult(cached *CachedFormatResult) *FormatSupportResult {
	return &FormatSupportResult{
		DetectedFormat:  cached.DetectedFormat,
		Status:          cached.SupportStatus,
		Recommendations: cached.Recommendations,
		ConversionPaths: cached.ConversionPaths,
		CacheHit:        true,
	}
}

// initializeConversionMatrix 初始化转换矩阵
func (fsm *FormatSupportManager) initializeConversionMatrix() {
	fsm.conversionMatrix = map[string][]string{
		"jpeg": {"webp", "avif", "jxl", "png"},
		"png":  {"jpeg", "webp", "avif", "jxl"},
		"webp": {"jpeg", "png", "avif", "jxl"},
		"avif": {"jpeg", "png", "webp", "jxl"},
		"jxl":  {"jpeg", "png", "webp", "avif"},
		"heic": {"jpeg", "png", "webp", "avif"},
		"mp4":  {"webm", "mov", "avi"},
		"webm": {"mp4", "mov", "avi"},
	}
}

// initializeStatistics 初始化统计信息
func (fsm *FormatSupportManager) initializeStatistics() {
	fsm.stats.FormatsByCategory = make(map[FormatCategory]int)
	fsm.stats.FormatsBySupportLevel = make(map[SupportLevel]int)
	fsm.stats.PopularFormats = make(map[string]int)
	fsm.stats.ConversionMatrix = make(map[string]map[string]int)
	fsm.stats.PerformanceMetrics = &PerformanceMetrics{}
}

// updateStatistics 更新统计信息
func (fsm *FormatSupportManager) updateStatistics(result *FormatSupportResult) {
	fsm.mutex.Lock()
	defer fsm.mutex.Unlock()

	// 更新检测统计
	if result.Supported {
		fsm.stats.SuccessfulDetections++
	} else {
		fsm.stats.FailedDetections++
	}

	// 更新格式统计
	if result.DetectedFormat != "" {
		fsm.stats.PopularFormats[result.DetectedFormat]++
	}

	// 更新转换建议统计
	if len(result.ConversionPaths) > 0 {
		fsm.stats.ConversionsRecommended++
	}

	// 更新性能指标
	if result.DetectionTime > 0 {
		metrics := fsm.stats.PerformanceMetrics
		metrics.TotalDetectionTime += result.DetectionTime

		if metrics.FastestDetection == 0 || result.DetectionTime < metrics.FastestDetection {
			metrics.FastestDetection = result.DetectionTime
		}

		if result.DetectionTime > metrics.SlowestDetection {
			metrics.SlowestDetection = result.DetectionTime
		}

		detectionCount := fsm.stats.SuccessfulDetections + fsm.stats.FailedDetections
		if detectionCount > 0 {
			metrics.AverageDetectionTime = metrics.TotalDetectionTime / time.Duration(detectionCount)
		}
	}

	// 更新缓存命中率
	if result.CacheHit {
		totalAttempts := fsm.stats.DetectionAttempts
		if totalAttempts > 0 {
			cacheHits := float64(totalAttempts - fsm.stats.SuccessfulDetections - fsm.stats.FailedDetections)
			fsm.stats.PerformanceMetrics.CacheHitRate = cacheHits / float64(totalAttempts)
		}
	}
}

// 字符串方法
func (fc FormatCategory) String() string {
	switch fc {
	case CategoryImage:
		return "image"
	case CategoryVideo:
		return "video"
	case CategoryAudio:
		return "audio"
	case CategoryRaw:
		return "raw"
	case CategoryVector:
		return "vector"
	case CategoryDocument:
		return "document"
	case CategoryArchive:
		return "archive"
	case CategoryOther:
		return "other"
	default:
		return "unknown"
	}
}

func (sl SupportLevel) String() string {
	switch sl {
	case SupportFull:
		return "full"
	case SupportPartial:
		return "partial"
	case SupportBasic:
		return "basic"
	case SupportExperimental:
		return "experimental"
	case SupportLegacy:
		return "legacy"
	case SupportNone:
		return "none"
	default:
		return "unknown"
	}
}

func (ss SupportStatus) String() string {
	switch ss {
	case StatusSupported:
		return "supported"
	case StatusUnsupported:
		return "unsupported"
	case StatusDeprecated:
		return "deprecated"
	case StatusExperimental:
		return "experimental"
	case StatusConvertible:
		return "convertible"
	case StatusUnknown:
		return "unknown"
	default:
		return "undefined"
	}
}

func (da DefaultAction) String() string {
	switch da {
	case ActionAllow:
		return "allow"
	case ActionSkip:
		return "skip"
	case ActionConvert:
		return "convert"
	case ActionPrompt:
		return "prompt"
	default:
		return "unknown"
	}
}

// GetSupportedFormats 获取支持的格式列表
func (fsm *FormatSupportManager) GetSupportedFormats() map[string]*FormatInfo {
	fsm.mutex.RLock()
	defer fsm.mutex.RUnlock()

	// 返回副本
	formats := make(map[string]*FormatInfo)
	for key, info := range fsm.supportedFormats {
		infoCopy := *info
		formats[key] = &infoCopy
	}

	return formats
}

// GetFormatStatistics 获取格式统计信息
func (fsm *FormatSupportManager) GetFormatStatistics() *FormatStatistics {
	fsm.mutex.RLock()
	defer fsm.mutex.RUnlock()

	// 返回副本
	stats := *fsm.stats
	return &stats
}

// IsFormatSupported 检查格式是否支持
func (fsm *FormatSupportManager) IsFormatSupported(format string) bool {
	fsm.mutex.RLock()
	defer fsm.mutex.RUnlock()

	formatInfo, exists := fsm.supportedFormats[strings.ToLower(format)]
	if !exists {
		return false
	}

	return formatInfo.SupportLevel != SupportNone
}

// Enable 启用格式支持检查
func (fsm *FormatSupportManager) Enable() {
	fsm.enabled = true
	fsm.logger.Info("格式支持检查已启用")
}

// Disable 禁用格式支持检查
func (fsm *FormatSupportManager) Disable() {
	fsm.enabled = false
	fsm.logger.Info("格式支持检查已禁用")
}
