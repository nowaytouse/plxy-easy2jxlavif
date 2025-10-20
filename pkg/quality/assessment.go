package quality

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"pixly/pkg/core/types"

	"go.uber.org/zap"
)

// QualityEngine 品质判断引擎
type QualityEngine struct {
	logger      *zap.Logger
	ffprobePath string
	ffmpegPath  string
	fastMode    bool // 快速模式，跳过深度分析
}

// QualityAssessment 品质评估结果
type QualityAssessment struct {
	FilePath        string                 `json:"file_path"`
	MediaType       types.MediaType        `json:"media_type"`
	QualityLevel    types.QualityLevel     `json:"quality_level"`
	Score           float64                `json:"score"`
	Width           int                    `json:"width"`
	Height          int                    `json:"height"`
	Duration        float64                `json:"duration,omitempty"`
	PixelDensity    float64                `json:"pixel_density"`
	JpegQuality     int                    `json:"jpeg_quality,omitempty"`
	BitRate         int64                  `json:"bit_rate,omitempty"`
	FrameRate       float64                `json:"frame_rate,omitempty"`
	IsCorrupted     bool                   `json:"is_corrupted"`
	Format          string                 `json:"format"`
	FileSize        int64                  `json:"file_size"`
	RecommendedMode types.AppMode          `json:"recommended_mode"`
	Confidence      float64                `json:"confidence"`
	AssessmentTime  time.Duration          `json:"assessment_time"`
	Details         map[string]interface{} `json:"details,omitempty"`
}

// NewQualityEngine 创建新的品质判断引擎
func NewQualityEngine(logger *zap.Logger, ffprobePath, ffmpegPath string, fastMode bool) *QualityEngine {
	return &QualityEngine{
		logger:      logger,
		ffprobePath: ffprobePath,
		ffmpegPath:  ffmpegPath,
		fastMode:    fastMode,
	}
}

// AssessFile 评估单个文件的品质
func (qe *QualityEngine) AssessFile(ctx context.Context, filePath string) (*QualityAssessment, error) {
	startTime := time.Now()

	assessment := &QualityAssessment{
		FilePath:   filePath,
		Details:    make(map[string]interface{}),
		Confidence: 0.0,
	}

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法获取文件信息: %w", err)
	}
	assessment.FileSize = fileInfo.Size()

	// 创建带超时的上下文 - 防止FFmpeg处理某些文件时永久卡住
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 使用 ffprobe 获取媒体信息
	mediaInfo, err := qe.getMediaInfo(ctxWithTimeout, filePath)
	if err != nil {
		qe.logger.Debug("ffprobe 分析失败", zap.String("file", filePath), zap.Error(err))
		assessment.IsCorrupted = true
		assessment.QualityLevel = types.QualityUnknown
		assessment.AssessmentTime = time.Since(startTime)
		return assessment, nil
	}

	// 解析媒体信息
	if err := qe.parseMediaInfo(assessment, mediaInfo); err != nil {
		qe.logger.Debug("解析媒体信息失败", zap.String("file", filePath), zap.Error(err))
		assessment.IsCorrupted = true
		assessment.QualityLevel = types.QualityUnknown
		assessment.AssessmentTime = time.Since(startTime)
		return assessment, nil
	}

	// 进行品质评估
	qe.assessQuality(assessment)

	// 生成模式推荐
	qe.recommendMode(assessment)

	assessment.AssessmentTime = time.Since(startTime)

	qe.logger.Debug("品质评估完成",
		zap.String("file", filepath.Base(filePath)),
		zap.String("quality", assessment.QualityLevel.String()),
		zap.Float64("score", assessment.Score),
		zap.String("recommended_mode", assessment.RecommendedMode.String()),
		zap.Duration("assessment_time", assessment.AssessmentTime),
	)

	return assessment, nil
}

// getMediaInfo 使用 ffprobe 获取媒体信息
func (qe *QualityEngine) getMediaInfo(ctx context.Context, filePath string) (map[string]interface{}, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	}

	cmd := exec.CommandContext(ctx, qe.ffprobePath, args...)

	// 为了防止程序卡住，设置命令执行超时
	output, err := cmd.Output()
	if err != nil {
		// 检查是否是超时错误
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("ffprobe 执行超时 (30秒) - 文件可能损坏或格式复杂: %s", filepath.Base(filePath))
		}
		return nil, fmt.Errorf("ffprobe 执行失败: %w", err)
	}

	// 简化的JSON解析 - 在实际实现中应使用 json.Unmarshal
	info := make(map[string]interface{})

	// 解析关键信息（简化版本）
	outputStr := string(output)

	// 提取宽度和高度
	if width := qe.extractNumber(outputStr, `"width":\s*(\d+)`); width > 0 {
		info["width"] = width
	}
	if height := qe.extractNumber(outputStr, `"height":\s*(\d+)`); height > 0 {
		info["height"] = height
	}

	// 提取时长
	if duration := qe.extractFloat(outputStr, `"duration":\s*"([0-9.]+)"`); duration > 0 {
		info["duration"] = duration
	}

	// 提取格式
	if format := qe.extractString(outputStr, `"format_name":\s*"([^"]+)"`); format != "" {
		info["format"] = format
	}

	// 提取比特率
	if bitRate := qe.extractNumber(outputStr, `"bit_rate":\s*"(\d+)"`); bitRate > 0 {
		info["bit_rate"] = bitRate
	}

	// 提取帧率
	if frameRate := qe.extractFloat(outputStr, `"avg_frame_rate":\s*"([0-9.]+)"`); frameRate > 0 {
		info["frame_rate"] = frameRate
	}

	// 提取编解码器
	if codec := qe.extractString(outputStr, `"codec_name":\s*"([^"]+)"`); codec != "" {
		info["codec"] = codec
	}

	return info, nil
}

// parseMediaInfo 解析媒体信息
func (qe *QualityEngine) parseMediaInfo(assessment *QualityAssessment, info map[string]interface{}) error {
	// 解析基本信息
	if width, ok := info["width"].(int); ok {
		assessment.Width = width
	}
	if height, ok := info["height"].(int); ok {
		assessment.Height = height
	}
	if duration, ok := info["duration"].(float64); ok {
		assessment.Duration = duration
	}
	if format, ok := info["format"].(string); ok {
		assessment.Format = format
	}
	if bitRate, ok := info["bit_rate"].(int64); ok {
		assessment.BitRate = bitRate
	}
	if frameRate, ok := info["frame_rate"].(float64); ok {
		assessment.FrameRate = frameRate
	}

	// 计算像素密度比
	if assessment.Width > 0 && assessment.Height > 0 {
		totalPixels := float64(assessment.Width * assessment.Height)
		fileSizeMB := float64(assessment.FileSize) / (1024 * 1024)

		if fileSizeMB > 0 {
			assessment.PixelDensity = totalPixels / (fileSizeMB * 1000000) // 像素/MB
		}
	}

	// 确定媒体类型
	assessment.MediaType = qe.determineMediaType(assessment)

	// 如果是JPEG，尝试获取品质信息
	if strings.Contains(strings.ToLower(assessment.Format), "jpeg") {
		assessment.JpegQuality = qe.estimateJpegQuality(assessment)
	}

	// 存储详细信息
	assessment.Details = info

	return nil
}

// determineMediaType 确定媒体类型
func (qe *QualityEngine) determineMediaType(assessment *QualityAssessment) types.MediaType {
	format := strings.ToLower(assessment.Format)

	// 视频格式
	if strings.Contains(format, "mp4") || strings.Contains(format, "mov") ||
		strings.Contains(format, "avi") || strings.Contains(format, "mkv") ||
		strings.Contains(format, "webm") {
		return types.MediaTypeVideo
	}

	// 动图格式
	if strings.Contains(format, "gif") ||
		(strings.Contains(format, "webp") && assessment.Duration > 0) ||
		strings.Contains(format, "apng") {
		return types.MediaTypeAnimated
	}

	// 静图格式
	return types.MediaTypeImage
}

// assessQuality 评估品质等级
func (qe *QualityEngine) assessQuality(assessment *QualityAssessment) {
	var score float64
	var confidence float64 = 0.5 // 基础置信度

	switch assessment.MediaType {
	case types.MediaTypeImage:
		score, confidence = qe.assessImageQuality(assessment)
	case types.MediaTypeAnimated:
		score, confidence = qe.assessAnimatedQuality(assessment)
	case types.MediaTypeVideo:
		score, confidence = qe.assessVideoQuality(assessment)
	default:
		score = 0.5
		confidence = 0.1
	}

	assessment.Score = score
	assessment.Confidence = confidence

	// 根据分数确定品质等级
	if score >= 0.9 {
		assessment.QualityLevel = types.QualityVeryHigh
	} else if score >= 0.75 {
		assessment.QualityLevel = types.QualityHigh
	} else if score >= 0.6 {
		assessment.QualityLevel = types.QualityMediumHigh
	} else if score >= 0.45 {
		assessment.QualityLevel = types.QualityMediumLow
	} else if score >= 0.3 {
		assessment.QualityLevel = types.QualityLow
	} else {
		assessment.QualityLevel = types.QualityVeryLow
	}
}

// assessImageQuality 评估静图品质 - 根据memory经验优化阈值
func (qe *QualityEngine) assessImageQuality(assessment *QualityAssessment) (float64, float64) {
	var score float64
	var confidence float64 = 0.7

	// 根据memory经验：调整文件大小阈值，降低低品质判断的敏感度
	fileSizeMB := float64(assessment.FileSize) / (1024 * 1024)
	format := strings.ToLower(assessment.Format)

	// memory经验：JPEG文件1MB+评为中高品质
	if strings.Contains(format, "jpeg") || strings.Contains(format, "jpg") {
		if fileSizeMB >= 1.0 {
			score += 0.4 // 提高到中高品质范围
			confidence = 0.85
		} else if fileSizeMB >= 0.5 {
			score += 0.3
		} else {
			score += 0.2
		}
	}
	// memory经验：PNG文件1MB+评为高品质
	if strings.Contains(format, "png") {
		if fileSizeMB >= 1.0 {
			score += 0.5 // 提高到高品质范围
			confidence = 0.9
		} else if fileSizeMB >= 0.5 {
			score += 0.4
		} else {
			score += 0.2
		}
	}
	// memory经验：WebP文件1MB+评为中高品质
	if strings.Contains(format, "webp") {
		if fileSizeMB >= 1.0 {
			score += 0.4
			confidence = 0.85
		} else if fileSizeMB >= 0.5 {
			score += 0.3
		} else {
			score += 0.2
		}
	}
	// memory经验：HEIF文件1MB+评为高品质
	if strings.Contains(format, "heif") || strings.Contains(format, "heic") {
		if fileSizeMB >= 1.0 {
			score += 0.5
			confidence = 0.9
		} else if fileSizeMB >= 0.5 {
			score += 0.4
		} else {
			score += 0.2
		}
	}

	// 基于像素密度的评估
	if assessment.PixelDensity > 2.5 {
		score += 0.2 // 降低像素密度权重
	} else if assessment.PixelDensity > 1.0 {
		score += 0.15
	} else if assessment.PixelDensity > 0.6 {
		score += 0.1
	} else {
		score += 0.05
	}

	// 基于JPEG品质的评估
	if assessment.JpegQuality > 0 {
		if assessment.JpegQuality >= 85 {
			score += 0.2
		} else if assessment.JpegQuality >= 70 {
			score += 0.15
		} else if assessment.JpegQuality >= 50 {
			score += 0.1
		}
		confidence = 0.85 // JPEG品质是可靠指标
	}

	// 基于分辨率的评估
	totalPixels := assessment.Width * assessment.Height
	if totalPixels >= 4000000 { // 4MP+
		score += 0.15
	} else if totalPixels >= 2000000 { // 2MP+
		score += 0.1
	} else if totalPixels >= 1000000 { // 1MP+
		score += 0.05
	}

	// memory经验：未知格式默认分数从0.3提高到0.5
	if score == 0 {
		score = 0.5 // 提高未知格式的默认分数
		confidence = 0.3
	}

	return min(score, 1.0), confidence
}

// assessAnimatedQuality 评估动图品质 - 根据memory经验优化阈值
func (qe *QualityEngine) assessAnimatedQuality(assessment *QualityAssessment) (float64, float64) {
	var score float64
	var confidence float64 = 0.6

	// memory经验：GIF文件2MB+评为中高品质
	fileSizeMB := float64(assessment.FileSize) / (1024 * 1024)
	format := strings.ToLower(assessment.Format)

	if strings.Contains(format, "gif") {
		if fileSizeMB >= 2.0 {
			score += 0.4 // 提高到中高品质范围
			confidence = 0.85
		} else if fileSizeMB >= 1.0 {
			score += 0.3
		} else {
			score += 0.2
		}
	}

	// 基于分辨率
	totalPixels := assessment.Width * assessment.Height
	if totalPixels >= 1000000 {
		score += 0.2 // 降低权重
	} else if totalPixels >= 500000 {
		score += 0.15
	} else {
		score += 0.1
	}

	// 基于帧率
	if assessment.FrameRate > 0 {
		if assessment.FrameRate >= 30 {
			score += 0.2
		} else if assessment.FrameRate >= 15 {
			score += 0.15
		} else {
			score += 0.1
		}
		confidence = 0.75
	}

	// 基于时长合理性
	if assessment.Duration > 0 && assessment.Duration < 30 {
		score += 0.15 // 合理的动图时长
	}

	// 基于比特率
	if assessment.BitRate > 0 {
		bitRateMbps := float64(assessment.BitRate) / 1000000
		if bitRateMbps > 5 {
			score += 0.15
		} else if bitRateMbps > 2 {
			score += 0.1
		}
	}

	// memory经验：未知格式默认分数从0.3提高到0.5
	if score == 0 {
		score = 0.5
		confidence = 0.3
	}

	return min(score, 1.0), confidence
}

// assessVideoQuality 评估视频品质
func (qe *QualityEngine) assessVideoQuality(assessment *QualityAssessment) (float64, float64) {
	var score float64
	var confidence float64 = 0.8

	// 基于分辨率
	totalPixels := assessment.Width * assessment.Height
	if totalPixels >= 8000000 { // 4K+
		score += 0.4
	} else if totalPixels >= 2000000 { // 1080p+
		score += 0.3
	} else if totalPixels >= 1000000 { // 720p+
		score += 0.2
	} else {
		score += 0.1
	}

	// 基于比特率
	if assessment.BitRate > 0 {
		bitRateMbps := float64(assessment.BitRate) / 1000000
		if bitRateMbps > 20 {
			score += 0.3
		} else if bitRateMbps > 10 {
			score += 0.2
		} else if bitRateMbps > 5 {
			score += 0.1
		}
	}

	// 基于帧率
	if assessment.FrameRate > 0 {
		if assessment.FrameRate >= 60 {
			score += 0.2
		} else if assessment.FrameRate >= 30 {
			score += 0.15
		} else if assessment.FrameRate >= 24 {
			score += 0.1
		}
	}

	// 基于时长合理性
	if assessment.Duration > 0 && assessment.Duration > 1 {
		score += 0.1 // 有效的视频时长
	}

	return min(score, 1.0), confidence
}

// recommendMode 推荐处理模式
func (qe *QualityEngine) recommendMode(assessment *QualityAssessment) {
	switch assessment.QualityLevel {
	case types.QualityVeryHigh, types.QualityHigh:
		assessment.RecommendedMode = types.ModeQuality // 高品质使用品质模式
	case types.QualityMediumHigh, types.QualityMediumLow:
		assessment.RecommendedMode = types.ModeAutoPlus // 中等品质使用自动模式+
	case types.QualityLow, types.QualityVeryLow:
		// 低品质需要用户决策，这里设为表情包模式作为默认建议
		assessment.RecommendedMode = types.ModeEmoji
	default:
		assessment.RecommendedMode = types.ModeAutoPlus
	}
}

// estimateJpegQuality 估算JPEG品质
func (qe *QualityEngine) estimateJpegQuality(assessment *QualityAssessment) int {
	// 简化的JPEG品质估算基于文件大小和分辨率
	if assessment.Width == 0 || assessment.Height == 0 || assessment.FileSize == 0 {
		return 0
	}

	totalPixels := float64(assessment.Width * assessment.Height)
	bytesPerPixel := float64(assessment.FileSize) / totalPixels

	// 基于每像素字节数估算品质
	if bytesPerPixel > 2.0 {
		return 95
	} else if bytesPerPixel > 1.5 {
		return 85
	} else if bytesPerPixel > 1.0 {
		return 75
	} else if bytesPerPixel > 0.5 {
		return 60
	} else if bytesPerPixel > 0.3 {
		return 45
	} else {
		return 30
	}
}

// BatchAssess 批量评估文件品质
func (qe *QualityEngine) BatchAssess(ctx context.Context, filePaths []string, callback func(*QualityAssessment)) error {
	for i, filePath := range filePaths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		assessment, err := qe.AssessFile(ctx, filePath)
		if err != nil {
			qe.logger.Error("品质评估失败",
				zap.String("file", filePath),
				zap.Error(err),
			)
			continue
		}

		if callback != nil {
			callback(assessment)
		}

		// 在快速模式下，每100个文件休息一下
		if qe.fastMode && i%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	return nil
}

// 辅助函数
func (qe *QualityEngine) extractNumber(text, pattern string) int {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			return num
		}
	}
	return 0
}

func (qe *QualityEngine) extractFloat(text, pattern string) float64 {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		if num, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return num
		}
	}
	return 0
}

func (qe *QualityEngine) extractString(text, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
