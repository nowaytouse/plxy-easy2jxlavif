package predictor

import "time"

// FileFeatures 文件特征信息
// 用于智能预测最优转换参数
type FileFeatures struct {
	// 基本信息
	FilePath string
	Format   string // "png", "jpeg", "gif", etc.
	FileSize int64
	Width    int
	Height   int

	// 格式特征
	HasAlpha   bool
	ColorSpace string // "rgb", "rgba", "grayscale"
	BitDepth   int    // 8, 16, 32
	PixFmt     string // FFprobe的pix_fmt字段

	// 质量特征（主要用于JPEG）
	EstimatedQuality int     // 0-100
	NoiseLevel       float64 // 0-1
	Compression      float64 // 当前压缩率

	// 内容特征
	IsAnimated bool // 是否动图
	FrameCount int  // 帧数
	FrameRate  float64

	// 派生特征
	BytesPerPixel float64 // 文件大小/像素数
	Complexity    float64 // 图像复杂度估算 (0-1)
}

// ConversionParams 转换参数
// 由预测器生成的最优转换参数
type ConversionParams struct {
	TargetFormat string // "jxl", "avif", "mov"

	// JXL参数
	Lossless     bool
	Distance     float64 // 0=无损, 1-15=有损
	Effort       int     // 1-9，压缩力度
	LosslessJPEG bool    // JPEG无损重包装

	// AVIF参数
	CRF   int // 0-63，质量参数
	Speed int // 0-10，编码速度

	// MOV参数（视频）
	Repackage bool // 仅重封装
	CopyCodec bool // 复制编码

	// 通用参数
	Quality       int  // 通用质量参数（0-100）
	Threads       int  // 线程数
	PreserveAlpha bool // 保留透明度
}

// Prediction 预测结果
// 包含预测的参数和置信度
type Prediction struct {
	Params     *ConversionParams
	Confidence float64 // 0-1，预测置信度
	Method     string  // "lookup_table", "rule_based", "regression"
	RuleName   string  // 使用的规则名称（如果是规则预测）

	// 预测的空间节省
	ExpectedSaving    float64 // 预期节省百分比 (0-1)
	ExpectedSizeBytes int64   // 预期文件大小

	// 辅助信息
	ShouldExplore         bool               // 是否需要探索
	ExplorationCandidates []ConversionParams // 探索候选参数

	PredictionTime time.Duration
}

// PredictionRule 预测规则
type PredictionRule struct {
	Name       string
	Condition  func(*FileFeatures) bool
	Prediction *ConversionParams
	Confidence float64
	Priority   int // 规则优先级，数字越大优先级越高
}

// ExplorationResult 探索结果
type ExplorationResult struct {
	BestParams   *ConversionParams
	BestSize     int64
	TestedParams []ConversionParams
	TestResults  map[string]int64 // 参数描述 -> 文件大小
	ExploreTime  time.Duration
}
