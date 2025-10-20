package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pixly/pkg/core/types"

	"go.uber.org/zap"
)

// FileMorphologyClassifier 文件形态分类器 - README要求的精确文件识别和形态区分
//
// 核心功能：
//   - 超越文件扩展名，基于真实内容进行分析
//   - 精确区分"静图"、"动图"、"视频"三种核心形态
//   - 处理容器格式的多形态问题（GIF、APNG、WebP、HEIC等）
//   - 使用ffprobe作为核心分析工具
//   - 识别特殊类型（Live Photo、空间图片等）
//
// 性能特点：
//   - 智能缓存机制，避免重复分析
//   - 快速预判 + 深度验证的双阶段架构
//   - 超时控制，防止ffprobe卡死
//   - 批量处理优化
type FileMorphologyClassifier struct {
	logger         *zap.Logger
	ffprobePath    string
	exiftoolPath   string
	fastMode       bool                         // 快速模式：跳过ffprobe深度分析
	cacheEnabled   bool                         // 启用分析结果缓存
	cache          map[string]*MorphologyResult // 内存缓存
	timeoutSeconds int                          // ffprobe超时时间（秒）
}

// MorphologyResult 形态分析结果
type MorphologyResult struct {
	FilePath       string                 `json:"file_path"`
	TrueFormat     string                 `json:"true_format"`     // 真实格式（基于内容）
	MediaType      types.MediaType        `json:"media_type"`      // 媒体类型
	FrameCount     int                    `json:"frame_count"`     // 帧数
	Duration       float64                `json:"duration"`        // 时长（秒）
	Width          int                    `json:"width"`           // 宽度
	Height         int                    `json:"height"`          // 高度
	CodecName      string                 `json:"codec_name"`      // 编解码器名称
	IsAnimated     bool                   `json:"is_animated"`     // 是否为动画
	IsLivePhoto    bool                   `json:"is_live_photo"`   // 是否为Live Photo
	IsSpatial      bool                   `json:"is_spatial"`      // 是否为空间图片/视频
	HasAudio       bool                   `json:"has_audio"`       // 是否包含音轨
	Confidence     float64                `json:"confidence"`      // 分类置信度
	AnalysisMethod string                 `json:"analysis_method"` // 分析方法："extension", "ffprobe", "exiftool"
	AnalysisTime   time.Duration          `json:"analysis_time"`   // 分析耗时
	Details        map[string]interface{} `json:"details"`         // 详细信息
	Warnings       []string               `json:"warnings"`        // 警告信息
}

// FFProbeOutput ffprobe输出结构
type FFProbeOutput struct {
	Streams []FFProbeStream `json:"streams"`
	Format  FFProbeFormat   `json:"format"`
}

// FFProbeStream ffprobe流信息
type FFProbeStream struct {
	Index        int            `json:"index"`
	CodecName    string         `json:"codec_name"`
	CodecType    string         `json:"codec_type"`
	Width        int            `json:"width"`
	Height       int            `json:"height"`
	Duration     string         `json:"duration"`
	NbFrames     string         `json:"nb_frames"`
	AvgFrameRate string         `json:"avg_frame_rate"`
	RFrameRate   string         `json:"r_frame_rate"`
	Disposition  map[string]int `json:"disposition"`
}

// FFProbeFormat ffprobe格式信息
type FFProbeFormat struct {
	Filename   string            `json:"filename"`
	FormatName string            `json:"format_name"`
	Duration   string            `json:"duration"`
	Size       string            `json:"size"`
	BitRate    string            `json:"bit_rate"`
	Tags       map[string]string `json:"tags"`
}

// NewFileMorphologyClassifier 创建文件形态分类器
func NewFileMorphologyClassifier(logger *zap.Logger, ffprobePath, exiftoolPath string) *FileMorphologyClassifier {
	return &FileMorphologyClassifier{
		logger:         logger,
		ffprobePath:    ffprobePath,
		exiftoolPath:   exiftoolPath,
		fastMode:       false, // 默认启用深度分析
		cacheEnabled:   true,
		cache:          make(map[string]*MorphologyResult),
		timeoutSeconds: 30, // 30秒超时
	}
}

// ClassifyFile 分类单个文件 - README要求的精确形态识别
func (fmc *FileMorphologyClassifier) ClassifyFile(ctx context.Context, filePath string) (*MorphologyResult, error) {
	startTime := time.Now()

	// 检查缓存
	if fmc.cacheEnabled {
		if cached, exists := fmc.cache[filePath]; exists {
			fmc.logger.Debug("使用缓存结果", zap.String("file", filepath.Base(filePath)))
			return cached, nil
		}
	}

	result := &MorphologyResult{
		FilePath:   filePath,
		Details:    make(map[string]interface{}),
		Warnings:   make([]string, 0),
		Confidence: 0.0,
	}

	// 阶段1：基于扩展名的快速预判
	fmc.performExtensionBasedClassification(result)

	// 阶段2：ffprobe深度分析（README核心要求）
	if !fmc.fastMode && fmc.ffprobePath != "" {
		if err := fmc.performFFProbeAnalysis(ctx, result); err != nil {
			fmc.logger.Warn("ffprobe分析失败，使用扩展名结果",
				zap.String("file", filepath.Base(filePath)),
				zap.Error(err))
			result.Warnings = append(result.Warnings, fmt.Sprintf("ffprobe分析失败: %v", err))
			result.AnalysisMethod = "extension_fallback"
		} else {
			result.AnalysisMethod = "ffprobe"
			result.Confidence = 0.95 // ffprobe分析的高置信度
		}
	} else {
		result.AnalysisMethod = "extension"
		result.Confidence = 0.7 // 扩展名分析的中等置信度
	}

	// 阶段3：特殊类型检测（Live Photo、空间媒体等）
	if fmc.exiftoolPath != "" {
		fmc.detectSpecialTypes(ctx, result)
	}

	// 阶段4：最终形态确定
	fmc.determineMediaType(result)

	result.AnalysisTime = time.Since(startTime)

	// 缓存结果
	if fmc.cacheEnabled {
		fmc.cache[filePath] = result
	}

	fmc.logger.Debug("文件形态分类完成",
		zap.String("file", filepath.Base(filePath)),
		zap.String("media_type", result.MediaType.String()),
		zap.String("true_format", result.TrueFormat),
		zap.Bool("is_animated", result.IsAnimated),
		zap.Float64("confidence", result.Confidence),
		zap.Duration("analysis_time", result.AnalysisTime))

	return result, nil
}

// performExtensionBasedClassification 基于扩展名的快速分类
func (fmc *FileMorphologyClassifier) performExtensionBasedClassification(result *MorphologyResult) {
	ext := strings.ToLower(filepath.Ext(result.FilePath))
	baseName := strings.ToLower(filepath.Base(result.FilePath))

	// 设置初始格式信息
	result.TrueFormat = strings.TrimPrefix(ext, ".")

	switch ext {
	case ".jpg", ".jpeg":
		result.MediaType = types.MediaTypeImage
		result.IsAnimated = false
		result.Confidence = 0.9 // JPEG高置信度为静图

	case ".png":
		result.MediaType = types.MediaTypeImage
		result.IsAnimated = false
		// PNG通常是静图，但APNG需要深度检测
		if strings.Contains(baseName, "apng") {
			result.Confidence = 0.5 // 需要进一步验证
		} else {
			result.Confidence = 0.85
		}

	case ".gif":
		// GIF可能是静图也可能是动图，需要深度分析
		result.MediaType = types.MediaTypeAnimated
		result.IsAnimated = true
		result.Confidence = 0.6 // 需要ffprobe验证

	case ".webp":
		// WebP是最需要深度分析的格式：可能是静图、动图
		result.MediaType = types.MediaTypeImage // 默认静图
		result.IsAnimated = false
		result.Confidence = 0.4 // 强烈需要深度验证

	case ".heif", ".heic":
		result.MediaType = types.MediaTypeImage
		result.IsAnimated = false
		// HEIC可能包含Live Photo或序列
		result.Confidence = 0.6 // 需要exiftool检测特殊类型

	case ".avif":
		result.MediaType = types.MediaTypeImage
		result.IsAnimated = false
		result.Confidence = 0.8

	case ".jxl":
		result.MediaType = types.MediaTypeImage
		result.IsAnimated = false
		result.Confidence = 0.8

	case ".mp4", ".mov", ".avi", ".mkv", ".webm", ".m4v":
		result.MediaType = types.MediaTypeVideo
		result.IsAnimated = true
		result.Confidence = 0.9

	case ".tiff", ".tif":
		result.MediaType = types.MediaTypeImage
		result.IsAnimated = false
		result.Confidence = 0.85

	case ".bmp":
		result.MediaType = types.MediaTypeImage
		result.IsAnimated = false
		result.Confidence = 0.95 // BMP几乎总是静图

	default:
		result.MediaType = types.MediaTypeUnknown
		result.Confidence = 0.1
		result.Warnings = append(result.Warnings, "未知文件扩展名: "+ext)
	}
}

// performFFProbeAnalysis 执行ffprobe深度分析 - README核心功能
func (fmc *FileMorphologyClassifier) performFFProbeAnalysis(ctx context.Context, result *MorphologyResult) error {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(fmc.timeoutSeconds)*time.Second)
	defer cancel()

	// 构建ffprobe命令
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		result.FilePath,
	}

	cmd := exec.CommandContext(timeoutCtx, fmc.ffprobePath, args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ffprobe执行失败: %w", err)
	}

	// 解析JSON输出
	var ffprobeOutput FFProbeOutput
	if err := json.Unmarshal(output, &ffprobeOutput); err != nil {
		return fmt.Errorf("ffprobe输出解析失败: %w", err)
	}

	// 分析流信息
	return fmc.analyzeFFProbeStreams(result, &ffprobeOutput)
}

// analyzeFFProbeStreams 分析ffprobe流信息 - README要求的精确形态区分
func (fmc *FileMorphologyClassifier) analyzeFFProbeStreams(result *MorphologyResult, output *FFProbeOutput) error {
	videoStreams := 0
	audioStreams := 0
	var primaryVideoStream *FFProbeStream

	// 分析所有流
	for i, stream := range output.Streams {
		switch stream.CodecType {
		case "video":
			videoStreams++
			if primaryVideoStream == nil {
				primaryVideoStream = &output.Streams[i]
			}
		case "audio":
			audioStreams++
			result.HasAudio = true
		}
	}

	// 根据流信息确定形态
	if videoStreams == 0 {
		// 没有视频流，可能是纯音频或损坏文件
		result.MediaType = types.MediaTypeUnknown
		result.Warnings = append(result.Warnings, "未检测到视频流")
		return nil
	}

	// 分析主视频流
	if primaryVideoStream != nil {
		result.Width = primaryVideoStream.Width
		result.Height = primaryVideoStream.Height
		result.CodecName = primaryVideoStream.CodecName

		// 获取帧数 - README核心指标：nb_frames > 1 判定为动图/视频
		frameCount := 0
		if primaryVideoStream.NbFrames != "" {
			if count, err := strconv.Atoi(primaryVideoStream.NbFrames); err == nil {
				frameCount = count
			}
		}
		result.FrameCount = frameCount

		// 获取时长
		duration := 0.0
		if output.Format.Duration != "" {
			if d, err := strconv.ParseFloat(output.Format.Duration, 64); err == nil {
				duration = d
			}
		}
		result.Duration = duration

		// README要求的核心判断逻辑：nb_frames > 1 即为动图或视频
		if frameCount > 1 || duration > 0 {
			result.IsAnimated = true

			// 区分动图和视频
			if fmc.isVideoFormat(result.TrueFormat) || audioStreams > 0 || duration > 60 {
				result.MediaType = types.MediaTypeVideo
			} else {
				result.MediaType = types.MediaTypeAnimated
			}
		} else {
			// 单帧 = 静图
			result.IsAnimated = false
			result.MediaType = types.MediaTypeImage
		}

		// 更新真实格式
		if output.Format.FormatName != "" {
			result.TrueFormat = fmc.parseFormatName(output.Format.FormatName)
		}
	}

	// 存储详细信息
	result.Details["video_streams"] = videoStreams
	result.Details["audio_streams"] = audioStreams
	result.Details["primary_codec"] = result.CodecName
	result.Details["ffprobe_format"] = output.Format.FormatName

	return nil
}

// detectSpecialTypes 检测特殊类型 - Live Photo、空间媒体等
func (fmc *FileMorphologyClassifier) detectSpecialTypes(ctx context.Context, result *MorphologyResult) {
	if fmc.exiftoolPath == "" {
		return
	}

	// 使用exiftool检测特殊标签
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	args := []string{
		"-json",
		"-ContentIdentifier",
		"-MediaGroupUUID",
		"-SpatialOvercaptureIdentifier",
		result.FilePath,
	}

	cmd := exec.CommandContext(timeoutCtx, fmc.exiftoolPath, args...)
	output, err := cmd.Output()
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("exiftool检测失败: %v", err))
		return
	}

	// 简化的特殊类型检测
	outputStr := string(output)

	// Live Photo检测
	if strings.Contains(outputStr, "ContentIdentifier") ||
		strings.Contains(outputStr, "MediaGroupUUID") {
		result.IsLivePhoto = true
		result.Warnings = append(result.Warnings, "检测到Live Photo")
	}

	// 空间媒体检测
	if strings.Contains(outputStr, "SpatialOvercaptureIdentifier") ||
		strings.Contains(outputStr, "spatial") {
		result.IsSpatial = true
		result.Warnings = append(result.Warnings, "检测到空间媒体")
	}

	result.Details["exiftool_analysis"] = "completed"
}

// determineMediaType 最终形态确定 - README要求的智能跳过机制
func (fmc *FileMorphologyClassifier) determineMediaType(result *MorphologyResult) {
	// README要求的智能跳过规则
	if result.IsLivePhoto {
		result.MediaType = types.MediaTypeUnknown // 跳过Live Photo
		result.Warnings = append(result.Warnings, "Live Photo - 自动跳过")
		return
	}

	if result.IsSpatial {
		result.MediaType = types.MediaTypeUnknown // 跳过空间媒体
		result.Warnings = append(result.Warnings, "空间媒体 - 自动跳过")
		return
	}

	if result.HasAudio && result.MediaType == types.MediaTypeImage {
		result.MediaType = types.MediaTypeUnknown // 跳过包含音轨的图片
		result.Warnings = append(result.Warnings, "包含音轨的图片文件 - 自动跳过")
		return
	}

	// 提升置信度（经过完整分析）
	if result.AnalysisMethod == "ffprobe" {
		result.Confidence = 0.95
	}
}

// BatchClassifyFiles 批量分类文件 - 性能优化版本
func (fmc *FileMorphologyClassifier) BatchClassifyFiles(ctx context.Context, filePaths []string) (map[string]*MorphologyResult, error) {
	results := make(map[string]*MorphologyResult)

	fmc.logger.Info("开始批量文件形态分类", zap.Int("total_files", len(filePaths)))

	for i, filePath := range filePaths {
		result, err := fmc.ClassifyFile(ctx, filePath)
		if err != nil {
			fmc.logger.Warn("文件分类失败",
				zap.String("file", filepath.Base(filePath)),
				zap.Error(err))
			// 创建错误结果
			result = &MorphologyResult{
				FilePath:   filePath,
				MediaType:  types.MediaTypeUnknown,
				Confidence: 0.0,
				Warnings:   []string{fmt.Sprintf("分类失败: %v", err)},
			}
		}

		results[filePath] = result

		// 进度日志
		if (i+1)%100 == 0 {
			fmc.logger.Info("批量分类进度",
				zap.Int("processed", i+1),
				zap.Int("total", len(filePaths)))
		}
	}

	fmc.logger.Info("批量文件形态分类完成",
		zap.Int("total_files", len(filePaths)),
		zap.Int("cache_hits", len(fmc.cache)))

	return results, nil
}

// 辅助方法
func (fmc *FileMorphologyClassifier) isVideoFormat(format string) bool {
	videoFormats := []string{"mp4", "mov", "avi", "mkv", "webm", "m4v", "flv", "wmv"}
	format = strings.ToLower(format)
	for _, vf := range videoFormats {
		if strings.Contains(format, vf) {
			return true
		}
	}
	return false
}

func (fmc *FileMorphologyClassifier) parseFormatName(formatName string) string {
	// 解析ffprobe的format_name，提取主要格式
	parts := strings.Split(strings.ToLower(formatName), ",")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return formatName
}

// ClearCache 清理缓存
func (fmc *FileMorphologyClassifier) ClearCache() {
	fmc.cache = make(map[string]*MorphologyResult)
	fmc.logger.Debug("文件形态分类缓存已清理")
}

// GetCacheStats 获取缓存统计
func (fmc *FileMorphologyClassifier) GetCacheStats() map[string]int {
	stats := make(map[string]int)

	for _, result := range fmc.cache {
		stats[result.MediaType.String()]++
	}

	stats["total_cached"] = len(fmc.cache)
	return stats
}
