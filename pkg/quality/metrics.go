package quality

import "time"

// QualityMetrics represents comprehensive quality metrics for a media file
type QualityMetrics struct {
	// 基础信息
	FilePath    string
	FileSize    int64
	Format      string
	MediaType   string // "image", "video", "animation"
	
	// 图像特征
	Width       int
	Height      int
	PixelCount  int64
	HasAlpha    bool
	PixelFormat string    // rgba, rgb, yuv420p, etc.
	BitDepth    int       // 8, 10, 12, 16
	ColorSpace  string    // sRGB, AdobeRGB, etc.
	
	// 质量评估
	BytesPerPixel    float64  // 文件大小/像素数
	EstimatedQuality int      // 0-100
	ComplexityScore  float64  // 复杂度（0-1）
	NoiseLevel       float64  // 噪声水平（0-1）
	ContentType      string   // photo, graphic, screenshot, mixed
	
	// 压缩评估
	CompressionPotential float64  // 0-1，越高压缩潜力越大
	IsAlreadyCompressed  bool     // 是否已经高度压缩
	CompressionRatio     float64  // 当前压缩比估算
	
	// 分类
	QualityClass string  // 极高/高/中/低/极低
	SizeClass    string  // 极大/大/中/小/极小
	
	// 视频特定
	Duration      float64  // 视频时长（秒）
	FrameRate     float64  // 帧率
	Bitrate       int64    // 比特率
	Codec         string   // 编解码器
	Container     string   // 容器格式
	
	// 分析元数据
	AnalyzedAt    time.Time
	AnalysisTime  time.Duration
	AnalysisError string
}

// QualityDistribution represents quality distribution statistics
type QualityDistribution struct {
	ExtremelyHigh int
	High          int
	Medium        int
	Low           int
	ExtremelyLow  int
	Total         int
}

// CompressionStats represents compression effectiveness statistics
type CompressionStats struct {
	Format         string
	FileCount      int
	AvgSaving      float64
	BestSaving     float64
	WorstSaving    float64
	AvgBPP         float64  // Average Bytes Per Pixel
	TotalBefore    int64
	TotalAfter     int64
}

// QualityReport represents a complete quality analysis report
type QualityReport struct {
	SessionID           string
	StartTime           time.Time
	EndTime             time.Time
	Duration            time.Duration
	TotalFiles          int
	
	// 质量分布
	QualityDistribution QualityDistribution
	
	// 平均指标
	AvgBytesPerPixel    struct {
		Before float64
		After  float64
	}
	
	// 压缩效果
	CompressionEffectiveness map[string]CompressionStats
	
	// 格式分布
	FormatDistribution struct {
		Source map[string]int
		Target map[string]int
	}
	
	// 内容类型分布
	ContentTypeDistribution map[string]int
	
	// 质量类别分布
	QualityClassDistribution map[string]int
}
