package quality

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Analyzer is the main quality analyzer that delegates to specific analyzers
type Analyzer struct {
	imageAnalyzer *ImageAnalyzer
	// videoAnalyzer *VideoAnalyzer  // 未来添加
}

// NewAnalyzer creates a new quality analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		imageAnalyzer: NewImageAnalyzer(),
	}
}

// Analyze analyzes a file and returns quality metrics
func (a *Analyzer) Analyze(filePath string) (*QualityMetrics, error) {
	// 判断文件类型
	ext := strings.ToLower(filepath.Ext(filePath))
	
	// 图像格式
	imageExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".tiff": true,
		".tif":  true,
		".webp": true,
		".heic": true,
		".heif": true,
		".jxl":  true,
		".avif": true,
	}
	
	if imageExts[ext] {
		return a.imageAnalyzer.AnalyzeImage(filePath)
	}
	
	// 视频格式（未来实现）
	videoExts := map[string]bool{
		".mp4":  true,
		".avi":  true,
		".mkv":  true,
		".mov":  true,
		".flv":  true,
		".m4v":  true,
		".3gp":  true,
	}
	
	if videoExts[ext] {
		// 暂时使用基础分析
		return a.basicAnalysis(filePath)
	}
	
	return nil, fmt.Errorf("不支持的文件格式: %s", ext)
}

// basicAnalysis provides basic analysis for unsupported types
func (a *Analyzer) basicAnalysis(filePath string) (*QualityMetrics, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	
	return &QualityMetrics{
		FilePath:  filePath,
		FileSize:  fileInfo.Size(),
		Format:    strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), ".")),
		MediaType: "video",
		SizeClass: a.imageAnalyzer.classifySize(fileInfo.Size()),
	}, nil
}

// ClassifyQualityLevel classifies quality level based on bytes per pixel
func ClassifyQualityLevel(bpp float64, format string) string {
	analyzer := NewImageAnalyzer()
	return analyzer.classifyQuality(bpp, format)
}
