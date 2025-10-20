package extension
package extension

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pixly/pkg/core/types"

	"go.uber.org/zap"
)

// ExtensionCorrector 扩展名修正器
type ExtensionCorrector struct {
	logger          *zap.Logger
	
	// README要求：所有模式均集成文件扩展名自动修正功能
	enableCorrection     bool
	enableContainerFix   bool
	backupOriginalExt    bool
	
	// 格式映射表
	formatMappings       map[string]string          // MIME类型到扩展名映射
	containerMappings    map[string]ContainerConfig // 容器格式配置
	targetFormatMappings map[types.AppMode]map[string]string // 模式特定的目标格式
	
	// 统计信息
	correctionStats      *CorrectionStats
}

// ContainerConfig 容器格式配置
type ContainerConfig struct {
	FFmpegFormat    string   // FFmpeg格式参数 (如 -f avif, -f mov)
	ValidExtensions []string // 有效扩展名
	PreferredExt    string   // 首选扩展名
	RequiresSpecify bool     // 是否需要显式指定容器
}

// CorrectionStats 修正统计
type CorrectionStats struct {
	TotalFiles           int64            // 总文件数
	CorrectedExtensions  int64            // 修正扩展名数量
	CorrectedContainers  int64            // 修正容器格式数量
	SkippedFiles         int64            // 跳过文件数量
	ErrorFiles           int64            // 错误文件数量
	FormatDistribution   map[string]int64 // 格式分布统计
	ModeDistribution     map[types.AppMode]int64 // 模式分布统计
}

// CorrectionResult 修正结果
type CorrectionResult struct {
	OriginalPath      string           // 原始路径
	CorrectedPath     string           // 修正后路径
	OriginalExtension string           // 原始扩展名
	CorrectedExtension string          // 修正后扩展名
	ContainerFormat   string           // 容器格式
	WasCorrected      bool             // 是否进行了修正
	RequiresRename    bool             // 是否需要重命名文件
	FFmpegParams      []string         // 需要的FFmpeg参数
	ErrorMessage      string           // 错误信息
}

// NewExtensionCorrector 创建扩展名修正器
func NewExtensionCorrector(logger *zap.Logger) *ExtensionCorrector {
	corrector := &ExtensionCorrector{
		logger:            logger,
		enableCorrection:  true,
		enableContainerFix: true,
		backupOriginalExt: false, // 通过原子性文件操作已有备份机制
		
		formatMappings:       make(map[string]string),
		containerMappings:    make(map[string]ContainerConfig),
		targetFormatMappings: make(map[types.AppMode]map[string]string),
		
		correctionStats: &CorrectionStats{
			FormatDistribution: make(map[string]int64),
			ModeDistribution:   make(map[types.AppMode]int64),
		},
	}
	
	// 初始化格式映射
	corrector.initializeFormatMappings()
	corrector.initializeContainerMappings()
	corrector.initializeTargetFormatMappings()
	
	logger.Info("扩展名修正器初始化完成",
		zap.Bool("correction_enabled", corrector.enableCorrection),
		zap.Bool("container_fix_enabled", corrector.enableContainerFix))
	
	return corrector
}

// initializeFormatMappings 初始化格式映射 - README规定的格式支持
func (ec *ExtensionCorrector) initializeFormatMappings() {
	// MIME类型到扩展名映射
	ec.formatMappings = map[string]string{
		// 图片格式
		"image/jpeg":                    "jpg",
		"image/jpg":                     "jpg",
		"image/png":                     "png",
		"image/webp":                    "webp",
		"image/avif":                    "avif",
		"image/jxl":                     "jxl",
		"image/heif":                    "heif",
		"image/heic":                    "heic",
		"image/tiff":                    "tiff",
		"image/gif":                     "gif",
		"image/bmp":                     "bmp",
		"image/x-portable-anymap":       "pnm",
		"image/x-portable-pixmap":       "ppm",
		"image/x-portable-graymap":      "pgm",
		"image/x-portable-bitmap":       "pbm",
		
		// 视频格式
		"video/mp4":                     "mp4",
		"video/quicktime":               "mov",
		"video/x-msvideo":               "avi",
		"video/webm":                    "webm",
		"video/x-matroska":              "mkv",
		"video/x-ms-wmv":                "wmv",
		"video/x-flv":                   "flv",
		
		// 音频格式（虽然主要处理图像和视频）
		"audio/mpeg":                    "mp3",
		"audio/wav":                     "wav",
		"audio/x-flac":                  "flac",
		"audio/ogg":                     "ogg",
	}
}

// initializeContainerMappings 初始化容器映射 - README要求解决"Could not find tag for codec"错误
func (ec *ExtensionCorrector) initializeContainerMappings() {
	ec.containerMappings = map[string]ContainerConfig{
		"avif": {
			FFmpegFormat:    "avif",
			ValidExtensions: []string{"avif"},
			PreferredExt:    "avif",
			RequiresSpecify: true, // README要求：明确指定容器参数 (-f avif)
		},
		"jxl": {
			FFmpegFormat:    "image2",
			ValidExtensions: []string{"jxl"},
			PreferredExt:    "jxl",
			RequiresSpecify: false, // JPEG XL通常不需要显式容器指定
		},
		"mov": {
			FFmpegFormat:    "mov",
			ValidExtensions: []string{"mov", "qt"},
			PreferredExt:    "mov",
			RequiresSpecify: true, // README要求：明确指定容器参数 (-f mov)
		},
		"mp4": {
			FFmpegFormat:    "mp4",
			ValidExtensions: []string{"mp4", "m4v"},
			PreferredExt:    "mp4",
			RequiresSpecify: false, // MP4通常不需要显式指定
		},
		"webp": {
			FFmpegFormat:    "webp",
			ValidExtensions: []string{"webp"},
			PreferredExt:    "webp",
			RequiresSpecify: false,
		},
		"webm": {
			FFmpegFormat:    "webm",
			ValidExtensions: []string{"webm"},
			PreferredExt:    "webm",
			RequiresSpecify: false,
		},
	}
}

// initializeTargetFormatMappings 初始化目标格式映射 - README三大处理模式要求
func (ec *ExtensionCorrector) initializeTargetFormatMappings() {
	// 🤖 自动模式+ - 智能决策路由
	ec.targetFormatMappings[types.ModeAutoPlus] = map[string]string{
		// 极高/高品质 → 品质模式逻辑
		"jpg":  "jxl",  // JPEG → JXL (无损)
		"jpeg": "jxl",  // JPEG → JXL (无损)
		"png":  "jxl",  // PNG → JXL (无损)
		"webp": "jxl",  // WebP动图 → JXL (README要求：比AVIF更优压缩)
		"gif":  "avif", // GIF → AVIF (动图)
		"heif": "jxl",  // HEIF → JXL
		"heic": "jxl",  // HEIC → JXL
		"tiff": "jxl",  // TIFF → JXL
		
		// 视频 → MOV重包装
		"mp4":  "mov",
		"avi":  "mov",
		"webm": "mov",
		"mkv":  "mov",
	}
	
	// 🔥 品质模式 - 无损优先
	ec.targetFormatMappings[types.ModeQuality] = map[string]string{
		// README要求：静图 → JXL, 动图 → AVIF (无损), 视频 → MOV
		"jpg":  "jxl",
		"jpeg": "jxl",
		"png":  "jxl",
		"webp": "avif", // 动图处理
		"gif":  "avif", // 动图处理
		"heif": "jxl",
		"heic": "jxl",
		"tiff": "jxl",
		"bmp":  "jxl",
		
		// 视频格式
		"mp4":  "mov",
		"avi":  "mov",
		"webm": "mov",
		"mkv":  "mov",
	}
	
	// 🚀 表情包模式 - 极限压缩
	ec.targetFormatMappings[types.ModeEmoji] = map[string]string{
		// README要求：所有图片统一转换为AVIF，视频直接跳过
		"jpg":  "avif",
		"jpeg": "avif",
		"png":  "avif",
		"webp": "avif",
		"gif":  "avif",
		"heif": "avif",
		"heic": "avif",
		"tiff": "avif",
		"bmp":  "avif",
		
		// 视频文件在表情包模式下直接跳过（不映射）
	}
}

// CorrectExtension 修正文件扩展名
func (ec *ExtensionCorrector) CorrectExtension(filePath string, mode types.AppMode, mediaType types.MediaType, actualFormat string) (*CorrectionResult, error) {
	ec.logger.Debug("开始扩展名修正",
		zap.String("file_path", filePath),
		zap.String("mode", mode.String()),
		zap.String("media_type", mediaType.String()),
		zap.String("actual_format", actualFormat))
	
	result := &CorrectionResult{
		OriginalPath:       filePath,
		CorrectedPath:      filePath,
		OriginalExtension:  strings.ToLower(filepath.Ext(filePath)),
		CorrectedExtension: strings.ToLower(filepath.Ext(filePath)),
		WasCorrected:       false,
		RequiresRename:     false,
	}
	
	// 移除扩展名前的点
	if result.OriginalExtension != "" && result.OriginalExtension[0] == '.' {
		result.OriginalExtension = result.OriginalExtension[1:]
	}
	result.CorrectedExtension = result.OriginalExtension
	
	// 更新统计
	ec.correctionStats.TotalFiles++
	ec.correctionStats.ModeDistribution[mode]++
	
	// 1. 检查是否需要根据实际格式修正扩展名
	if actualFormat != "" {
		if correctedExt, shouldCorrect := ec.shouldCorrectForActualFormat(result.OriginalExtension, actualFormat); shouldCorrect {
			result.CorrectedExtension = correctedExt
			result.WasCorrected = true
			result.RequiresRename = true
			ec.correctionStats.CorrectedExtensions++
			
			ec.logger.Info("根据实际格式修正扩展名",
				zap.String("original_ext", result.OriginalExtension),
				zap.String("corrected_ext", result.CorrectedExtension),
				zap.String("actual_format", actualFormat))
		}
	}
	
	// 2. 检查目标格式映射 - README要求的模式特定修正
	if targetMappings, exists := ec.targetFormatMappings[mode]; exists {
		if targetExt, shouldMap := targetMappings[result.CorrectedExtension]; shouldMap {
			// 这是目标格式扩展名，用于后续转换
			result.CorrectedExtension = targetExt
			result.WasCorrected = true
			
			// 检查容器格式要求
			if containerConfig, hasContainer := ec.containerMappings[targetExt]; hasContainer {
				result.ContainerFormat = containerConfig.FFmpegFormat
				if containerConfig.RequiresSpecify {
					result.FFmpegParams = append(result.FFmpegParams, "-f", containerConfig.FFmpegFormat)
				}
				ec.correctionStats.CorrectedContainers++
			}
			
			ec.logger.Debug("应用目标格式映射",
				zap.String("mode", mode.String()),
				zap.String("target_ext", targetExt),
				zap.String("container_format", result.ContainerFormat))
		}
	}
	
	// 3. 生成修正后的完整路径
	if result.WasCorrected && result.RequiresRename {
		basePath := strings.TrimSuffix(filePath, filepath.Ext(filePath))
		result.CorrectedPath = basePath + "." + result.CorrectedExtension
	}
	
	// 4. 更新格式分布统计
	ec.correctionStats.FormatDistribution[result.CorrectedExtension]++
	
	ec.logger.Debug("扩展名修正完成",
		zap.String("original_path", result.OriginalPath),
		zap.String("corrected_path", result.CorrectedPath),
		zap.Bool("was_corrected", result.WasCorrected),
		zap.Strings("ffmpeg_params", result.FFmpegParams))
	
	return result, nil
}

// shouldCorrectForActualFormat 检查是否需要根据实际格式修正扩展名
func (ec *ExtensionCorrector) shouldCorrectForActualFormat(currentExt, actualFormat string) (string, bool) {
	// 标准化格式名称
	normalizedFormat := strings.ToLower(actualFormat)
	
	// 常见的格式不匹配情况
	formatCorrections := map[string]map[string]string{
		// 实际格式为JPEG，但扩展名错误
		"jpeg": {
			"png": "jpg",
			"gif": "jpg",
			"bmp": "jpg",
		},
		// 实际格式为PNG，但扩展名错误
		"png": {
			"jpg":  "png",
			"jpeg": "png",
			"gif":  "png",
		},
		// 实际格式为WebP，但扩展名错误
		"webp": {
			"jpg":  "webp",
			"jpeg": "webp",
			"png":  "webp",
		},
		// 实际格式为AVIF，但扩展名错误
		"avif": {
			"jpg":  "avif",
			"jpeg": "avif",
			"png":  "avif",
			"webp": "avif",
		},
	}
	
	if corrections, exists := formatCorrections[normalizedFormat]; exists {
		if correctExt, shouldCorrect := corrections[currentExt]; shouldCorrect {
			return correctExt, true
		}
	}
	
	return currentExt, false
}

// BatchCorrectExtensions 批量修正扩展名
func (ec *ExtensionCorrector) BatchCorrectExtensions(mediaFiles []*types.MediaInfo, mode types.AppMode) ([]*CorrectionResult, error) {
	results := make([]*CorrectionResult, 0, len(mediaFiles))
	
	ec.logger.Info("开始批量扩展名修正",
		zap.Int("file_count", len(mediaFiles)),
		zap.String("mode", mode.String()))
	
	for _, mediaInfo := range mediaFiles {
		result, err := ec.CorrectExtension(mediaInfo.Path, mode, mediaInfo.Type, mediaInfo.Format)
		if err != nil {
			ec.logger.Warn("扩展名修正失败",
				zap.String("file_path", mediaInfo.Path),
				zap.Error(err))
			
			result = &CorrectionResult{
				OriginalPath:  mediaInfo.Path,
				CorrectedPath: mediaInfo.Path,
				ErrorMessage:  err.Error(),
			}
			ec.correctionStats.ErrorFiles++
		}
		
		results = append(results, result)
	}
	
	ec.logger.Info("批量扩展名修正完成",
		zap.Int("total_files", len(results)),
		zap.Int64("corrected_extensions", ec.correctionStats.CorrectedExtensions),
		zap.Int64("corrected_containers", ec.correctionStats.CorrectedContainers))
	
	return results, nil
}

// ApplyFileRename 应用文件重命名（如果需要）
func (ec *ExtensionCorrector) ApplyFileRename(result *CorrectionResult) error {
	if !result.RequiresRename || result.OriginalPath == result.CorrectedPath {
		return nil // 无需重命名
	}
	
	// 检查目标文件是否已存在
	if _, err := os.Stat(result.CorrectedPath); err == nil {
		return fmt.Errorf("目标文件已存在: %s", result.CorrectedPath)
	}
	
	// 执行重命名
	if err := os.Rename(result.OriginalPath, result.CorrectedPath); err != nil {
		return fmt.Errorf("重命名失败: %w", err)
	}
	
	ec.logger.Info("文件重命名成功",
		zap.String("original", result.OriginalPath),
		zap.String("corrected", result.CorrectedPath))
	
	return nil
}

// GetStats 获取修正统计信息
func (ec *ExtensionCorrector) GetStats() *CorrectionStats {
	return ec.correctionStats
}

// ResetStats 重置统计信息
func (ec *ExtensionCorrector) ResetStats() {
	ec.correctionStats = &CorrectionStats{
		FormatDistribution: make(map[string]int64),
		ModeDistribution:   make(map[types.AppMode]int64),
	}
}

// IsTargetFormatForMode 检查指定扩展名是否是某模式的目标格式
func (ec *ExtensionCorrector) IsTargetFormatForMode(extension string, mode types.AppMode) bool {
	if targetMappings, exists := ec.targetFormatMappings[mode]; exists {
		for _, targetExt := range targetMappings {
			if targetExt == strings.ToLower(extension) {
				return true
			}
		}
	}
	return false
}

// GetTargetFormatForFile 获取文件在指定模式下的目标格式
func (ec *ExtensionCorrector) GetTargetFormatForFile(filePath string, mode types.AppMode) (string, bool) {
	currentExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	
	if targetMappings, exists := ec.targetFormatMappings[mode]; exists {
		if targetExt, hasMapping := targetMappings[currentExt]; hasMapping {
			return targetExt, true
		}
	}
	
	return "", false
}

// GenerateFFmpegContainerParams 生成FFmpeg容器参数
func (ec *ExtensionCorrector) GenerateFFmpegContainerParams(targetFormat string) []string {
	if containerConfig, exists := ec.containerMappings[targetFormat]; exists {
		if containerConfig.RequiresSpecify {
			return []string{"-f", containerConfig.FFmpegFormat}
		}
	}
	return nil
}