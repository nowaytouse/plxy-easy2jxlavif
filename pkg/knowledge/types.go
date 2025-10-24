package knowledge

import (
	"time"
)

// ConversionRecord 转换记录
// 用于记录每次转换的完整信息
type ConversionRecord struct {
	ID        int64
	CreatedAt time.Time

	// 文件信息
	FilePath       string
	FileName       string
	OriginalFormat string
	OriginalSize   int64

	// 文件特征
	Width            int
	Height           int
	HasAlpha         bool
	PixFmt           string
	IsAnimated       bool
	FrameCount       int
	EstimatedQuality int

	// 预测信息
	PredictorName        string
	PredictionRule       string
	PredictionConfidence float64
	PredictionTimeMs     int64

	// 预测参数
	PredictedFormat       string
	PredictedLossless     bool
	PredictedDistance     float64
	PredictedEffort       int
	PredictedLosslessJPEG bool
	PredictedCRF          int
	PredictedSpeed        int

	// 预测的空间节省
	PredictedSavingPercent float64
	PredictedOutputSize    int64

	// 实际转换结果
	ActualFormat           string
	ActualOutputSize       int64
	ActualConversionTimeMs int64

	// 实际空间节省
	ActualSavingPercent float64
	ActualSavingBytes   int64

	// 质量验证
	ValidationMethod string
	ValidationPassed bool
	PixelDiffPercent float64
	PSNRValue        float64
	SSIMValue        float64

	// 预测准确性
	PredictionErrorPercent float64
	WasExplored            bool

	// 用户反馈
	UserRating  int
	UserComment string

	// 元数据
	PixlyVersion string
	HostOS       string
}

// PredictionStats 预测准确性统计
type PredictionStats struct {
	ID             int64
	PredictorName  string
	PredictionRule string
	OriginalFormat string

	StatsFrom time.Time
	StatsTo   time.Time

	TotalConversions      int
	SuccessfulConversions int

	AvgPredictionErrorPercent    float64
	MedianPredictionErrorPercent float64
	StdPredictionErrorPercent    float64

	AvgPredictedSaving float64
	AvgActualSaving    float64

	PerfectQualityCount int
	GoodQualityCount    int

	AvgConversionTimeMs int64

	UpdatedAt time.Time
}

// AnomalyCase 异常案例
type AnomalyCase struct {
	ID                 int64
	ConversionRecordID int64
	AnomalyType        string
	AnomalySeverity    string
	Description        string
	DetectedAt         time.Time
	Resolved           bool
	ResolutionNote     string
}

// FormatCharacteristics 格式特征统计
type FormatCharacteristics struct {
	ID             int64
	OriginalFormat string
	PixFmt         string
	SizeRange      string

	SampleCount int

	BestTargetFormat string
	BestAvgSaving    float64
	BestSuccessRate  float64

	UpdatedAt time.Time
}

// RecordBuilder 转换记录构建器
// 方便从预测和实际结果构建记录
type RecordBuilder struct {
	record *ConversionRecord
}

// NewRecordBuilder 创建记录构建器
func NewRecordBuilder() *RecordBuilder {
	return &RecordBuilder{
		record: &ConversionRecord{
			CreatedAt: time.Now(),
		},
	}
}

// WithFileInfo 设置文件信息
func (rb *RecordBuilder) WithFileInfo(path, name, format string, size int64) *RecordBuilder {
	rb.record.FilePath = path
	rb.record.FileName = name
	rb.record.OriginalFormat = format
	rb.record.OriginalSize = size
	return rb
}

// FileFeatures 文件特征（避免循环依赖，在knowledge包中定义）
type FileFeatures struct {
	Width            int
	Height           int
	HasAlpha         bool
	PixFmt           string
	IsAnimated       bool
	FrameCount       int
	EstimatedQuality int
	Format           string
	FileSize         int64
}

// WithFeatures 设置文件特征
func (rb *RecordBuilder) WithFeatures(features *FileFeatures) *RecordBuilder {
	rb.record.Width = features.Width
	rb.record.Height = features.Height
	rb.record.HasAlpha = features.HasAlpha
	rb.record.PixFmt = features.PixFmt
	rb.record.IsAnimated = features.IsAnimated
	rb.record.FrameCount = features.FrameCount
	rb.record.EstimatedQuality = features.EstimatedQuality
	return rb
}

// Prediction 预测结果（避免循环依赖，在knowledge包中定义）
type Prediction struct {
	RuleName          string
	Confidence        float64
	PredictionTime    time.Duration
	Params            *ConversionParams
	ExpectedSaving    float64
	ExpectedSizeBytes int64
}

// ConversionParams 转换参数（避免循环依赖）
type ConversionParams struct {
	TargetFormat string
	Lossless     bool
	Distance     float64
	Effort       int
	LosslessJPEG bool
	CRF          int
	Speed        int
}

// WithPrediction 设置预测信息
func (rb *RecordBuilder) WithPrediction(prediction *Prediction, predictorName string) *RecordBuilder {
	rb.record.PredictorName = predictorName
	rb.record.PredictionRule = prediction.RuleName
	rb.record.PredictionConfidence = prediction.Confidence
	rb.record.PredictionTimeMs = prediction.PredictionTime.Milliseconds()

	// 安全处理Params字段
	if prediction.Params != nil {
		rb.record.PredictedFormat = prediction.Params.TargetFormat
		rb.record.PredictedLossless = prediction.Params.Lossless
		rb.record.PredictedDistance = prediction.Params.Distance
		rb.record.PredictedEffort = prediction.Params.Effort
		rb.record.PredictedLosslessJPEG = prediction.Params.LosslessJPEG
		rb.record.PredictedCRF = prediction.Params.CRF
		rb.record.PredictedSpeed = prediction.Params.Speed
	}

	rb.record.PredictedSavingPercent = prediction.ExpectedSaving
	rb.record.PredictedOutputSize = prediction.ExpectedSizeBytes

	return rb
}

// WithActualResult 设置实际转换结果
func (rb *RecordBuilder) WithActualResult(format string, outputSize int64, conversionTimeMs int64) *RecordBuilder {
	rb.record.ActualFormat = format
	rb.record.ActualOutputSize = outputSize
	rb.record.ActualConversionTimeMs = conversionTimeMs

	// 计算实际空间节省
	if rb.record.OriginalSize > 0 {
		rb.record.ActualSavingBytes = rb.record.OriginalSize - outputSize
		rb.record.ActualSavingPercent = float64(rb.record.ActualSavingBytes) / float64(rb.record.OriginalSize)
	}

	// 计算预测误差
	if rb.record.PredictedOutputSize > 0 {
		errorBytes := rb.record.ActualOutputSize - rb.record.PredictedOutputSize
		rb.record.PredictionErrorPercent = float64(errorBytes) / float64(rb.record.ActualOutputSize)
		if rb.record.PredictionErrorPercent < 0 {
			rb.record.PredictionErrorPercent = -rb.record.PredictionErrorPercent
		}
	}

	return rb
}

// WithValidation 设置质量验证结果
func (rb *RecordBuilder) WithValidation(method string, passed bool, pixelDiff, psnr, ssim float64) *RecordBuilder {
	rb.record.ValidationMethod = method
	rb.record.ValidationPassed = passed
	rb.record.PixelDiffPercent = pixelDiff
	rb.record.PSNRValue = psnr
	rb.record.SSIMValue = ssim
	return rb
}

// WithMetadata 设置元数据
func (rb *RecordBuilder) WithMetadata(version, os string) *RecordBuilder {
	rb.record.PixlyVersion = version
	rb.record.HostOS = os
	return rb
}

// Build 构建记录
func (rb *RecordBuilder) Build() *ConversionRecord {
	return rb.record
}
