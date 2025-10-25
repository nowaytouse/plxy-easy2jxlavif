package quality

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ImageAnalyzer analyzes image quality
type ImageAnalyzer struct {
	ffprobePath string
}

// NewImageAnalyzer creates a new image analyzer
func NewImageAnalyzer() *ImageAnalyzer {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		ffprobePath = "ffprobe"
	}
	
	return &ImageAnalyzer{
		ffprobePath: ffprobePath,
	}
}

// AnalyzeImage analyzes an image file and returns quality metrics
func (ia *ImageAnalyzer) AnalyzeImage(filePath string) (*QualityMetrics, error) {
	startTime := time.Now()
	
	metrics := &QualityMetrics{
		FilePath:   filePath,
		MediaType:  "image",
		AnalyzedAt: startTime,
	}
	
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		metrics.AnalysisError = err.Error()
		return metrics, err
	}
	
	metrics.FileSize = fileInfo.Size()
	metrics.Format = strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	
	probeData, err := ia.ffprobeImage(filePath)
	if err != nil {
		metrics.AnalysisError = err.Error()
		return metrics, err
	}
	
	if len(probeData.Streams) > 0 {
		stream := probeData.Streams[0]
		metrics.Width = stream.Width
		metrics.Height = stream.Height
		metrics.PixelFormat = stream.PixFormat
		metrics.BitDepth = ia.estimateBitDepth(stream.PixFormat)
		metrics.ColorSpace = ia.detectColorSpace(stream)
		metrics.HasAlpha = strings.Contains(stream.PixFormat, "a") || strings.Contains(stream.PixFormat, "rgba")
	}
	
	if metrics.Width > 0 && metrics.Height > 0 {
		metrics.PixelCount = int64(metrics.Width) * int64(metrics.Height)
	}
	
	if metrics.PixelCount > 0 {
		metrics.BytesPerPixel = float64(metrics.FileSize) / float64(metrics.PixelCount)
	}
	
	metrics.QualityClass = ia.classifyQuality(metrics.BytesPerPixel, metrics.Format)
	metrics.EstimatedQuality = ia.estimateQuality(metrics)
	metrics.ContentType = ia.detectContentType(metrics)
	metrics.CompressionPotential = ia.assessCompressionPotential(metrics)
	metrics.IsAlreadyCompressed = metrics.BytesPerPixel < 0.5
	metrics.SizeClass = ia.classifySize(metrics.FileSize)
	metrics.AnalysisTime = time.Since(startTime)
	
	return metrics, nil
}

func (ia *ImageAnalyzer) ffprobeImage(filePath string) (*FFProbeData, error) {
	cmd := exec.Command(ia.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		filePath,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffprobe执行失败: %w", err)
	}
	
	var data FFProbeData
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("解析ffprobe输出失败: %w", err)
	}
	
	return &data, nil
}

func (ia *ImageAnalyzer) classifyQuality(bpp float64, format string) string {
	var highThreshold, mediumThreshold, lowThreshold float64
	
	switch format {
	case "jpg", "jpeg":
		highThreshold = 3.0
		mediumThreshold = 1.0
		lowThreshold = 0.2
	case "png":
		highThreshold = 4.0
		mediumThreshold = 2.0
		lowThreshold = 0.5
	default:
		highThreshold = 3.0
		mediumThreshold = 1.0
		lowThreshold = 0.3
	}
	
	if bpp >= highThreshold {
		return "极高"
	} else if bpp >= mediumThreshold {
		return "高"
	} else if bpp >= lowThreshold {
		return "中"
	} else if bpp >= 0.1 {
		return "低"
	}
	return "极低"
}

func (ia *ImageAnalyzer) estimateQuality(metrics *QualityMetrics) int {
	bpp := metrics.BytesPerPixel
	
	var quality float64
	switch metrics.Format {
	case "jpg", "jpeg":
		quality = (bpp / 3.0) * 100
	case "png":
		quality = (bpp / 5.0) * 100
	default:
		quality = (bpp / 4.0) * 100
	}
	
	if quality > 100 {
		quality = 100
	} else if quality < 0 {
		quality = 0
	}
	
	return int(quality)
}

func (ia *ImageAnalyzer) detectContentType(metrics *QualityMetrics) string {
	bpp := metrics.BytesPerPixel
	
	if metrics.Format == "jpg" || metrics.Format == "jpeg" {
		if bpp > 2.0 {
			return "photo"
		} else if bpp > 0.5 {
			return "mixed"
		} else {
			return "graphic"
		}
	}
	
	if metrics.Format == "png" {
		if bpp > 3.0 {
			return "screenshot"
		} else if bpp > 1.5 {
			return "photo"
		} else {
			return "graphic"
		}
	}
	
	return "mixed"
}

func (ia *ImageAnalyzer) assessCompressionPotential(metrics *QualityMetrics) float64 {
	bpp := metrics.BytesPerPixel
	
	switch metrics.Format {
	case "png":
		if bpp > 3.0 {
			return 0.8
		} else if bpp > 1.5 {
			return 0.6
		} else {
			return 0.3
		}
	case "jpg", "jpeg":
		if bpp > 2.0 {
			return 0.5
		} else if bpp > 0.8 {
			return 0.3
		} else {
			return 0.1
		}
	default:
		return 0.5
	}
}

func (ia *ImageAnalyzer) classifySize(fileSize int64) string {
	sizeMB := float64(fileSize) / (1024 * 1024)
	
	if sizeMB >= 50 {
		return "极大"
	} else if sizeMB >= 10 {
		return "大"
	} else if sizeMB >= 1 {
		return "中"
	} else if sizeMB >= 0.1 {
		return "小"
	}
	return "极小"
}

func (ia *ImageAnalyzer) estimateBitDepth(pixFmt string) int {
	if strings.Contains(pixFmt, "p10") || strings.Contains(pixFmt, "10le") {
		return 10
	} else if strings.Contains(pixFmt, "p12") {
		return 12
	} else if strings.Contains(pixFmt, "p16") || strings.Contains(pixFmt, "16le") {
		return 16
	}
	return 8
}

func (ia *ImageAnalyzer) detectColorSpace(stream FFProbeStream) string {
	if strings.Contains(stream.ColorSpace, "bt709") {
		return "BT.709"
	} else if strings.Contains(stream.ColorSpace, "bt2020") {
		return "BT.2020"
	}
	return "sRGB"
}

// FFProbeData represents ffprobe JSON output
type FFProbeData struct {
	Streams []FFProbeStream `json:"streams"`
	Format  FFProbeFormat   `json:"format"`
}

// FFProbeStream represents a stream from ffprobe
type FFProbeStream struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	PixFormat  string `json:"pix_fmt"`
	ColorSpace string `json:"color_space"`
	CodecName  string `json:"codec_name"`
	Duration   string `json:"duration"`
	BitRate    string `json:"bit_rate"`
	FrameRate  string `json:"r_frame_rate"`
}

// FFProbeFormat represents format info from ffprobe
type FFProbeFormat struct {
	Filename   string `json:"filename"`
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
	BitRate    string `json:"bit_rate"`
}
