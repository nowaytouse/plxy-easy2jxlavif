package whitelist
package whitelist

import (
	"path/filepath"
	"strings"

	"pixly/pkg/core/types"

	"go.uber.org/zap"
)

// FormatWhitelist 格式白名单管理器
type FormatWhitelist struct {
	logger *zap.Logger
	
	// README要求：白名单模式识别可处理的媒体文件
	supportedImageFormats    map[string]FormatInfo
	supportedVideoFormats    map[string]FormatInfo
	supportedAudioFormats    map[string]FormatInfo
	
	// README要求：智能跳过文件类型
	skippedSpecialTypes      map[string]SkipReason
	skippedSystemTypes       map[string]SkipReason
	skippedCreativeTypes     map[string]SkipReason
	
	// 目标格式检查
	targetFormatsByMode      map[types.AppMode]map[string]bool
	
	// 统计信息
	whitelistStats          *WhitelistStats
}

// FormatInfo 格式信息
type FormatInfo struct {
	Extension    string              // 扩展名
	MimeType     string              // MIME类型
	MediaType    types.MediaType     // 媒体类型（静图/动图/视频）
	Description  string              // 格式描述
	IsSupported  bool                // 是否支持处理
	ToolRequired string              // 需要的工具（ffmpeg/cjxl/avifenc等）
	Notes        string              // 特殊说明
}

// SkipReason 跳过原因
type SkipReason struct {
	Category    SkipCategory // 跳过类别
	Reason      string       // 跳过原因
	Description string       // 详细描述
	IsRecursive bool         // 是否递归检查（如目录内的文件）
}

// SkipCategory 跳过类别
type SkipCategory int

const (
	SkipSpecialMedia    SkipCategory = iota // 特殊媒体类型
	SkipTargetFormat                        // 已是目标格式
	SkipNonMedia                           // 非媒体文件
	SkipCreativeSource                     // 创作源文件
	SkipSystemHidden                       // 系统/隐藏文件
	SkipUnsupported                        // 不支持的格式
	SkipCorrupted                          // 损坏文件
)

func (sc SkipCategory) String() string {
	switch sc {
	case SkipSpecialMedia:
		return "特殊媒体类型"
	case SkipTargetFormat:
		return "已是目标格式"
	case SkipNonMedia:
		return "非媒体文件"
	case SkipCreativeSource:
		return "创作源文件"
	case SkipSystemHidden:
		return "系统隐藏文件"
	case SkipUnsupported:
		return "不支持格式"
	case SkipCorrupted:
		return "损坏文件"
	default:
		return "未知"
	}
}

// WhitelistStats 白名单统计
type WhitelistStats struct {
	TotalFilesChecked    int64                     // 总检查文件数
	SupportedFiles       int64                     // 支持的文件数
	SkippedFiles         int64                     // 跳过的文件数
	UnsupportedFiles     int64                     // 不支持的文件数
	
	// 按格式分类统计
	ImageFormatCounts    map[string]int64          // 图片格式计数
	VideoFormatCounts    map[string]int64          // 视频格式计数
	AudioFormatCounts    map[string]int64          // 音频格式计数
	
	// 按跳过原因统计
	SkipReasonCounts     map[SkipCategory]int64    // 跳过原因计数
	
	// 按模式统计
	ModeFormatCounts     map[types.AppMode]map[string]int64 // 模式格式计数
}

// CheckResult 检查结果
type CheckResult struct {
	FilePath      string           // 文件路径
	IsSupported   bool             // 是否支持
	ShouldSkip    bool             // 是否应该跳过
	MediaType     types.MediaType  // 媒体类型
	FormatInfo    *FormatInfo      // 格式信息
	SkipReason    *SkipReason      // 跳过原因（如果跳过）
	TargetFormat  string           // 在指定模式下的目标格式
	Notes         []string         // 额外说明
}

// NewFormatWhitelist 创建格式白名单管理器
func NewFormatWhitelist(logger *zap.Logger) *FormatWhitelist {
	whitelist := &FormatWhitelist{
		logger:                logger,
		supportedImageFormats: make(map[string]FormatInfo),
		supportedVideoFormats: make(map[string]FormatInfo),
		supportedAudioFormats: make(map[string]FormatInfo),
		skippedSpecialTypes:   make(map[string]SkipReason),
		skippedSystemTypes:    make(map[string]SkipReason),
		skippedCreativeTypes:  make(map[string]SkipReason),
		targetFormatsByMode:   make(map[types.AppMode]map[string]bool),
		whitelistStats: &WhitelistStats{
			ImageFormatCounts: make(map[string]int64),
			VideoFormatCounts: make(map[string]int64),
			AudioFormatCounts: make(map[string]int64),
			SkipReasonCounts:  make(map[SkipCategory]int64),
			ModeFormatCounts:  make(map[types.AppMode]map[string]int64),
		},
	}
	
	// 初始化各种格式和跳过规则
	whitelist.initializeSupportedFormats()
	whitelist.initializeSkipRules()
	whitelist.initializeTargetFormats()
	
	logger.Info("格式白名单管理器初始化完成",
		zap.Int("supported_image_formats", len(whitelist.supportedImageFormats)),
		zap.Int("supported_video_formats", len(whitelist.supportedVideoFormats)),
		zap.Int("skip_rules", len(whitelist.skippedSpecialTypes)+len(whitelist.skippedSystemTypes)+len(whitelist.skippedCreativeTypes)))
	
	return whitelist
}

// initializeSupportedFormats 初始化支持的格式 - README 5.1节规定
func (fw *FormatWhitelist) initializeSupportedFormats() {
	// README 5.1: 图片格式支持
	fw.supportedImageFormats = map[string]FormatInfo{
		"jpg": {
			Extension:    "jpg",
			MimeType:     "image/jpeg",
			MediaType:    types.MediaTypeImage,
			Description:  "JPEG图像",
			IsSupported:  true,
			ToolRequired: "cjxl,ffmpeg",
			Notes:        "支持无损转JXL",
		},
		"jpeg": {
			Extension:    "jpeg",
			MimeType:     "image/jpeg",
			MediaType:    types.MediaTypeImage,
			Description:  "JPEG图像",
			IsSupported:  true,
			ToolRequired: "cjxl,ffmpeg",
			Notes:        "支持无损转JXL",
		},
		"png": {
			Extension:    "png",
			MimeType:     "image/png",
			MediaType:    types.MediaTypeImage,
			Description:  "PNG图像",
			IsSupported:  true,
			ToolRequired: "cjxl,ffmpeg",
			Notes:        "支持无损转换",
		},
		"tiff": {
			Extension:    "tiff",
			MimeType:     "image/tiff",
			MediaType:    types.MediaTypeImage,
			Description:  "TIFF图像",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "支持转换",
		},
		"tif": {
			Extension:    "tif",
			MimeType:     "image/tiff",
			MediaType:    types.MediaTypeImage,
			Description:  "TIFF图像",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "支持转换",
		},
		"webp": {
			Extension:    "webp",
			MimeType:     "image/webp",
			MediaType:    types.MediaTypeAnimated, // 可能是动图
			Description:  "WebP图像",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "需检测静图/动图",
		},
		"heif": {
			Extension:    "heif",
			MimeType:     "image/heif",
			MediaType:    types.MediaTypeImage,
			Description:  "HEIF图像",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "非Live Photo",
		},
		"heic": {
			Extension:    "heic",
			MimeType:     "image/heic",
			MediaType:    types.MediaTypeImage,
			Description:  "HEIC图像",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "非Live Photo",
		},
		"gif": {
			Extension:    "gif",
			MimeType:     "image/gif",
			MediaType:    types.MediaTypeAnimated,
			Description:  "GIF动图",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "动图处理",
		},
		"apng": {
			Extension:    "apng",
			MimeType:     "image/apng",
			MediaType:    types.MediaTypeAnimated,
			Description:  "动态PNG",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "动图处理",
		},
		"bmp": {
			Extension:    "bmp",
			MimeType:     "image/bmp",
			MediaType:    types.MediaTypeImage,
			Description:  "位图",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "基础格式",
		},
	}
	
	// README 5.1: 视频格式支持
	fw.supportedVideoFormats = map[string]FormatInfo{
		"mp4": {
			Extension:    "mp4",
			MimeType:     "video/mp4",
			MediaType:    types.MediaTypeVideo,
			Description:  "MP4视频",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "主流视频格式",
		},
		"webm": {
			Extension:    "webm",
			MimeType:     "video/webm",
			MediaType:    types.MediaTypeVideo,
			Description:  "WebM视频",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "Web视频格式",
		},
		"mov": {
			Extension:    "mov",
			MimeType:     "video/quicktime",
			MediaType:    types.MediaTypeVideo,
			Description:  "QuickTime视频",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "Apple标准格式",
		},
		"mkv": {
			Extension:    "mkv",
			MimeType:     "video/x-matroska",
			MediaType:    types.MediaTypeVideo,
			Description:  "Matroska视频",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "开源容器格式",
		},
		"avi": {
			Extension:    "avi",
			MimeType:     "video/x-msvideo",
			MediaType:    types.MediaTypeVideo,
			Description:  "AVI视频",
			IsSupported:  true,
			ToolRequired: "ffmpeg",
			Notes:        "传统视频格式",
		},
	}
	
	// 音频格式（虽然主要不处理，但需要识别）
	fw.supportedAudioFormats = map[string]FormatInfo{
		"mp3": {
			Extension:    "mp3",
			MimeType:     "audio/mpeg",
			MediaType:    types.MediaTypeUnknown,
			Description:  "MP3音频",
			IsSupported:  false,
			Notes:        "音频文件，不处理",
		},
		"wav": {
			Extension:    "wav",
			MimeType:     "audio/wav",
			MediaType:    types.MediaTypeUnknown,
			Description:  "WAV音频",
			IsSupported:  false,
			Notes:        "音频文件，不处理",
		},
		"flac": {
			Extension:    "flac",
			MimeType:     "audio/flac",
			MediaType:    types.MediaTypeUnknown,
			Description:  "FLAC音频",
			IsSupported:  false,
			Notes:        "音频文件，不处理",
		},
	}
}

// initializeSkipRules 初始化跳过规则 - README 5.2节规定
func (fw *FormatWhitelist) initializeSkipRules() {
	// README 5.2: 特殊媒体类型跳过
	fw.skippedSpecialTypes = map[string]SkipReason{
		"livephoto": {
			Category:    SkipSpecialMedia,
			Reason:      "Live Photo",
			Description: "iPhone Live Photo，包含音轨的图片文件",
		},
		"spatial": {
			Category:    SkipSpecialMedia,
			Reason:      "空间图片/视频",
			Description: "Spatial Images/Videos，VR/AR内容",
		},
		"audio_image": {
			Category:    SkipSpecialMedia,
			Reason:      "包含音轨的图片",
			Description: "包含音轨的图片文件",
		},
	}
	
	// README 5.2: 非媒体文件跳过
	fw.skippedSystemTypes = map[string]SkipReason{
		"psd": {
			Category:    SkipCreativeSource,
			Reason:      "创作源文件",
			Description: "Photoshop工程文件",
		},
		"pdf": {
			Category:    SkipNonMedia,
			Reason:      "文档文件",
			Description: "PDF文档",
		},
		"doc": {
			Category:    SkipNonMedia,
			Reason:      "文档文件",
			Description: "Word文档",
		},
		"py": {
			Category:    SkipNonMedia,
			Reason:      "代码文件",
			Description: "Python源码",
		},
		"zip": {
			Category:    SkipNonMedia,
			Reason:      "压缩文件",
			Description: "ZIP压缩包",
		},
		"rar": {
			Category:    SkipNonMedia,
			Reason:      "压缩文件",
			Description: "RAR压缩包",
		},
		"7z": {
			Category:    SkipNonMedia,
			Reason:      "压缩文件",
			Description: "7-Zip压缩包",
		},
	}
	
	// README 5.2: 创作源文件跳过
	fw.skippedCreativeTypes = map[string]SkipReason{
		"blend": {
			Category:    SkipCreativeSource,
			Reason:      "3D模型文件",
			Description: "Blender工程文件",
		},
		"max": {
			Category:    SkipCreativeSource,
			Reason:      "3D模型文件",
			Description: "3ds Max工程文件",
		},
		"maya": {
			Category:    SkipCreativeSource,
			Reason:      "3D模型文件",
			Description: "Maya工程文件",
		},
		"ai": {
			Category:    SkipCreativeSource,
			Reason:      "绘图工程文件",
			Description: "Adobe Illustrator文件",
		},
		"sketch": {
			Category:    SkipCreativeSource,
			Reason:      "绘图工程文件",
			Description: "Sketch设计文件",
		},
		"fig": {
			Category:    SkipCreativeSource,
			Reason:      "绘图工程文件",
			Description: "Figma设计文件",
		},
	}
}

// initializeTargetFormats 初始化目标格式 - README 5.2节规定
func (fw *FormatWhitelist) initializeTargetFormats() {
	// 三大处理模式的目标格式
	fw.targetFormatsByMode = map[types.AppMode]map[string]bool{
		types.ModeAutoPlus: {
			"jxl":  true, // 主要目标格式
			"avif": true, // 动图目标格式
			"mov":  true, // 视频目标格式
		},
		types.ModeQuality: {
			"jxl":  true, // 静图目标格式
			"avif": true, // 动图目标格式（无损）
			"mov":  true, // 视频目标格式
		},
		types.ModeEmoji: {
			"avif": true, // 表情包模式统一目标格式
		},
	}
	
	// 初始化模式统计
	for mode := range fw.targetFormatsByMode {
		fw.whitelistStats.ModeFormatCounts[mode] = make(map[string]int64)
	}
}

// CheckFile 检查单个文件
func (fw *FormatWhitelist) CheckFile(filePath string, mode types.AppMode) *CheckResult {
	fw.whitelistStats.TotalFilesChecked++
	
	result := &CheckResult{
		FilePath:    filePath,
		IsSupported: false,
		ShouldSkip:  false,
		MediaType:   types.MediaTypeUnknown,
		Notes:       make([]string, 0),
	}
	
	// 获取文件扩展名
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	fileName := filepath.Base(filePath)
	
	fw.logger.Debug("检查文件",
		zap.String("file_path", filePath),
		zap.String("extension", ext),
		zap.String("mode", mode.String()))
	
	// 1. 检查系统隐藏文件 - README 5.2节要求
	if fw.isSystemHiddenFile(fileName) {
		result.ShouldSkip = true
		result.SkipReason = &SkipReason{
			Category:    SkipSystemHidden,
			Reason:      "系统或隐藏文件",
			Description: "以.开头的文件或系统定义的隐藏文件",
		}
		fw.whitelistStats.SkippedFiles++
		fw.whitelistStats.SkipReasonCounts[SkipSystemHidden]++
		return result
	}
	
	// 2. 检查是否是目标格式 - README 5.2节要求：跳过已是目标格式的文件
	if fw.isTargetFormat(ext, mode) {
		result.ShouldSkip = true
		result.SkipReason = &SkipReason{
			Category:    SkipTargetFormat,
			Reason:      "已是目标格式",
			Description: fmt.Sprintf("文件已是%s模式的目标格式: %s", mode.String(), ext),
		}
		fw.whitelistStats.SkippedFiles++
		fw.whitelistStats.SkipReasonCounts[SkipTargetFormat]++
		return result
	}
	
	// 3. 检查跳过规则
	if skipReason := fw.checkSkipRules(ext, filePath); skipReason != nil {
		result.ShouldSkip = true
		result.SkipReason = skipReason
		fw.whitelistStats.SkippedFiles++
		fw.whitelistStats.SkipReasonCounts[skipReason.Category]++
		return result
	}
	
	// 4. 检查支持的格式
	if formatInfo := fw.getFormatInfo(ext); formatInfo != nil {
		result.IsSupported = formatInfo.IsSupported
		result.MediaType = formatInfo.MediaType
		result.FormatInfo = formatInfo
		
		if formatInfo.IsSupported {
			fw.whitelistStats.SupportedFiles++
			
			// 统计格式分布
			switch formatInfo.MediaType {
			case types.MediaTypeImage:
				fw.whitelistStats.ImageFormatCounts[ext]++
			case types.MediaTypeAnimated:
				fw.whitelistStats.ImageFormatCounts[ext]++ // 动图也算图片类
			case types.MediaTypeVideo:
				fw.whitelistStats.VideoFormatCounts[ext]++
			}
			
			// 统计模式格式分布
			if modeStats, exists := fw.whitelistStats.ModeFormatCounts[mode]; exists {
				modeStats[ext]++
			}
			
			result.Notes = append(result.Notes, fmt.Sprintf("支持格式: %s", formatInfo.Description))
		} else {
			result.ShouldSkip = true
			result.SkipReason = &SkipReason{
				Category:    SkipUnsupported,
				Reason:      "不支持的格式",
				Description: formatInfo.Notes,
			}
			fw.whitelistStats.UnsupportedFiles++
			fw.whitelistStats.SkipReasonCounts[SkipUnsupported]++
		}
	} else {
		// 未知格式，跳过
		result.ShouldSkip = true
		result.SkipReason = &SkipReason{
			Category:    SkipUnsupported,
			Reason:      "未知格式",
			Description: fmt.Sprintf("扩展名 .%s 不在支持列表中", ext),
		}
		fw.whitelistStats.UnsupportedFiles++
		fw.whitelistStats.SkipReasonCounts[SkipUnsupported]++
	}
	
	fw.logger.Debug("文件检查完成",
		zap.String("file_path", filePath),
		zap.Bool("is_supported", result.IsSupported),
		zap.Bool("should_skip", result.ShouldSkip),
		zap.String("media_type", result.MediaType.String()))
	
	return result
}