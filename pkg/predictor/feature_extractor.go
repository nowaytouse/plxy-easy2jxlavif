package predictor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// FeatureExtractor 特征提取器
// 从媒体文件中提取用于预测的特征信息
type FeatureExtractor struct {
	logger      *zap.Logger
	ffprobePath string
}

// NewFeatureExtractor 创建特征提取器
func NewFeatureExtractor(logger *zap.Logger, ffprobePath string) *FeatureExtractor {
	return &FeatureExtractor{
		logger:      logger,
		ffprobePath: ffprobePath,
	}
}

// ExtractFeatures 提取文件特征
// 快速提取，目标时间 <0.1秒
func (fe *FeatureExtractor) ExtractFeatures(filePath string) (*FileFeatures, error) {
	startTime := time.Now()

	features := &FileFeatures{
		FilePath: filePath,
	}

	// 1. 获取文件基本信息
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}
	features.FileSize = stat.Size()

	// 2. 获取文件格式
	ext := strings.ToLower(filepath.Ext(filePath))
	features.Format = strings.TrimPrefix(ext, ".")

	// 3. 使用FFprobe提取详细信息（关键步骤）
	if err := fe.extractFFprobeInfo(filePath, features); err != nil {
		// FFprobe失败时使用fallback
		fe.logger.Warn("FFprobe提取失败，使用fallback",
			zap.String("file", filepath.Base(filePath)),
			zap.Error(err))
		fe.applyFallback(features)
	}

	// 4. 计算派生特征
	fe.calculateDerivedFeatures(features)

	extractTime := time.Since(startTime)
	fe.logger.Debug("特征提取完成",
		zap.String("file", filepath.Base(filePath)),
		zap.String("format", features.Format),
		zap.Int("width", features.Width),
		zap.Int("height", features.Height),
		zap.Duration("time", extractTime))

	return features, nil
}

// FFprobeOutput FFprobe输出结构
type FFprobeOutput struct {
	Streams []struct {
		Width            int    `json:"width"`
		Height           int    `json:"height"`
		PixFmt           string `json:"pix_fmt"`
		CodecName        string `json:"codec_name"`
		CodecType        string `json:"codec_type"`
		BitsPerRawSample string `json:"bits_per_raw_sample"`
		ColorSpace       string `json:"color_space"`
		NbFrames         string `json:"nb_frames"`
		RFrameRate       string `json:"r_frame_rate"`
		AvgFrameRate     string `json:"avg_frame_rate"`
	} `json:"streams"`
	Format struct {
		Size     string `json:"size"`
		BitRate  string `json:"bit_rate"`
		Duration string `json:"duration"`
	} `json:"format"`
}

// extractFFprobeInfo 使用FFprobe提取详细信息
func (fe *FeatureExtractor) extractFFprobeInfo(filePath string, features *FileFeatures) error {
	// 构建FFprobe命令
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		filePath,
	}

	cmd := exec.Command(fe.ffprobePath, args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("FFprobe执行失败: %w", err)
	}

	// 解析JSON输出
	var probeData FFprobeOutput
	if err := json.Unmarshal(output, &probeData); err != nil {
		return fmt.Errorf("解析FFprobe输出失败: %w", err)
	}

	// 检查是否有视频流
	if len(probeData.Streams) == 0 {
		return fmt.Errorf("未找到媒体流")
	}

	// 提取第一个视频/图像流的信息
	stream := probeData.Streams[0]

	// 基本尺寸
	features.Width = stream.Width
	features.Height = stream.Height

	// 像素格式（关键特征）
	features.PixFmt = stream.PixFmt

	// 检测透明度
	features.HasAlpha = fe.detectAlpha(stream.PixFmt)

	// 色彩空间
	features.ColorSpace = stream.ColorSpace
	if features.ColorSpace == "" {
		features.ColorSpace = fe.inferColorSpace(stream.PixFmt)
	}

	// 位深度
	features.BitDepth = fe.parseBitDepth(stream.BitsPerRawSample, stream.PixFmt)

	// 检测动图
	features.IsAnimated = fe.detectAnimation(stream.NbFrames, stream.CodecName)
	if features.IsAnimated {
		features.FrameCount = fe.parseFrameCount(stream.NbFrames)
		features.FrameRate = fe.parseFrameRate(stream.AvgFrameRate)
	} else {
		features.FrameCount = 1
	}

	// 针对JPEG的质量估算
	if features.Format == "jpg" || features.Format == "jpeg" {
		features.EstimatedQuality = fe.estimateJPEGQuality(stream.PixFmt)
	}

	return nil
}

// detectAlpha 检测是否有透明通道
func (fe *FeatureExtractor) detectAlpha(pixFmt string) bool {
	alphaFormats := []string{
		"rgba", "argb", "bgra", "abgr",
		"yuva", "gbra", "rgba64",
		"rgba64be", "rgba64le",
	}

	pixFmtLower := strings.ToLower(pixFmt)
	for _, format := range alphaFormats {
		if strings.Contains(pixFmtLower, format) {
			return true
		}
	}
	return false
}

// inferColorSpace 根据pix_fmt推断色彩空间
func (fe *FeatureExtractor) inferColorSpace(pixFmt string) string {
	pixFmtLower := strings.ToLower(pixFmt)

	if strings.Contains(pixFmtLower, "rgba") {
		return "rgba"
	}
	if strings.Contains(pixFmtLower, "rgb") {
		return "rgb"
	}
	if strings.Contains(pixFmtLower, "gray") {
		return "grayscale"
	}
	if strings.Contains(pixFmtLower, "yuv") {
		return "yuv"
	}
	return "rgb" // 默认
}

// parseBitDepth 解析位深度
func (fe *FeatureExtractor) parseBitDepth(bitsPerRawSample string, pixFmt string) int {
	// 优先使用bits_per_raw_sample
	if bitsPerRawSample == "16" {
		return 16
	}
	if bitsPerRawSample == "32" {
		return 32
	}

	// 根据pix_fmt推断
	pixFmtLower := strings.ToLower(pixFmt)
	if strings.Contains(pixFmtLower, "64") {
		return 16 // rgba64 = 16 bits per channel
	}
	if strings.Contains(pixFmtLower, "48") {
		return 16 // rgb48 = 16 bits per channel
	}
	if strings.Contains(pixFmtLower, "16") {
		return 16
	}

	return 8 // 默认
}

// detectAnimation 检测是否为动图
func (fe *FeatureExtractor) detectAnimation(nbFrames string, codecName string) bool {
	// 检查帧数
	if nbFrames != "" && nbFrames != "1" && nbFrames != "0" {
		return true
	}

	// 某些编解码器天然是动图
	animatedCodecs := []string{"gif", "apng", "webp"} // webp可能是动图
	codecLower := strings.ToLower(codecName)
	for _, codec := range animatedCodecs {
		if codec == codecLower {
			return true // 可能是动图，需要进一步检查
		}
	}

	return false
}

// parseFrameCount 解析帧数
func (fe *FeatureExtractor) parseFrameCount(nbFrames string) int {
	var count int
	fmt.Sscanf(nbFrames, "%d", &count)
	if count <= 0 {
		count = 1
	}
	return count
}

// parseFrameRate 解析帧率
func (fe *FeatureExtractor) parseFrameRate(frameRate string) float64 {
	// 帧率格式通常是 "25/1" 或 "30000/1001"
	var num, den float64
	n, _ := fmt.Sscanf(frameRate, "%f/%f", &num, &den)
	if n == 2 && den > 0 {
		return num / den
	}
	return 0
}

// estimateJPEGQuality 根据pix_fmt估算JPEG质量
// 基于PIXLY最初版本的质量分析体系
func (fe *FeatureExtractor) estimateJPEGQuality(pixFmt string) int {
	switch pixFmt {
	case "yuv444p", "yuvj444p":
		return 98 // 4:4:4采样，接近无损
	case "yuv422p", "yuvj422p":
		return 80 // 4:2:2采样，高质量
	case "yuv420p", "yuvj420p":
		return 65 // 4:2:0采样，标准质量
	default:
		return 50 // 未知格式
	}
}

// calculateDerivedFeatures 计算派生特征
func (fe *FeatureExtractor) calculateDerivedFeatures(features *FileFeatures) {
	// 计算字节/像素
	pixelCount := features.Width * features.Height
	if pixelCount > 0 {
		features.BytesPerPixel = float64(features.FileSize) / float64(pixelCount)
	}

	// 估算复杂度（简化版）
	// 高BytesPerPixel通常意味着更复杂或更高质量
	if features.BytesPerPixel > 3.0 {
		features.Complexity = 0.9 // 高复杂度
	} else if features.BytesPerPixel > 1.0 {
		features.Complexity = 0.7
	} else if features.BytesPerPixel > 0.5 {
		features.Complexity = 0.5
	} else {
		features.Complexity = 0.3 // 低复杂度（可能已高度压缩）
	}

	// 针对PNG的调整
	if features.Format == "png" {
		// PNG的BytesPerPixel如果很小，说明已经高度压缩，但转JXL仍有巨大空间
		if features.BytesPerPixel < 0.5 {
			features.Compression = 0.9 // 已高度压缩
		} else {
			features.Compression = 0.3 // 压缩率较低
		}
	}
}

// applyFallback 应用fallback（当FFprobe失败时）
func (fe *FeatureExtractor) applyFallback(features *FileFeatures) {
	// 提供合理的默认值
	features.Width = 1920
	features.Height = 1080
	features.BitDepth = 8

	// 根据格式设置默认值
	switch features.Format {
	case "png":
		features.HasAlpha = true // 保守估计
		features.ColorSpace = "rgba"
		features.PixFmt = "rgba"
	case "jpg", "jpeg":
		features.HasAlpha = false
		features.ColorSpace = "rgb"
		features.PixFmt = "yuv420p"
		features.EstimatedQuality = 65
	case "gif":
		features.HasAlpha = false
		features.ColorSpace = "rgb"
		features.IsAnimated = true // GIF通常是动图
	}

	// 计算派生特征
	fe.calculateDerivedFeatures(features)
}
