package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
)

// Validator 验证器接口
type Validator interface {
	// ValidateDirectory 验证目录处理结果
	ValidateDirectory(workDir string, targetFormat string, expectedProcessed int) (*ValidationResult, error)

	// ValidateFile 验证单个文件
	ValidateFile(filePath string, targetFormat string) (*FileValidationResult, error)

	// GenerateReport 生成验证报告
	GenerateReport(result *ValidationResult, outputPath string) error
}

// FileValidationResult 单文件验证结果
type FileValidationResult struct {
	FilePath         string    `json:"file_path"`
	IsValid          bool      `json:"is_valid"`
	HasTargetFile    bool      `json:"has_target_file"`
	SizeReduction    int64     `json:"size_reduction"`
	CompressionRatio float64   `json:"compression_ratio"`
	ExifPreserved    bool      `json:"exif_preserved"`
	ValidationTime   time.Time `json:"validation_time"`
	Issues           []string  `json:"issues"`
}

// ValidationConfig 验证配置
type ValidationConfig struct {
	EnableSizeValidation    bool    `json:"enable_size_validation"`
	EnableExifValidation    bool    `json:"enable_exif_validation"`
	EnableFormatValidation  bool    `json:"enable_format_validation"`
	MinCompressionRatio     float64 `json:"min_compression_ratio"`
	MinExifPreservationRate float64 `json:"min_exif_preservation_rate"`
	MaxUnprocessedFiles     int     `json:"max_unprocessed_files"`
}

// DefaultValidationConfig 默认验证配置
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		EnableSizeValidation:    true,
		EnableExifValidation:    true,
		EnableFormatValidation:  true,
		MinCompressionRatio:     0.1, // 至少10%压缩
		MinExifPreservationRate: 0.8, // 至少80%EXIF保留
		MaxUnprocessedFiles:     0,   // 不允许有未处理文件
	}
}

// NewValidator 创建验证器
func NewValidator(logger *zap.Logger, config *ValidationConfig) Validator {
	if config == nil {
		config = DefaultValidationConfig()
	}

	return &PostProcessingValidator{
		logger: logger,
	}
}

// ValidateFile 验证单个文件
func (v *PostProcessingValidator) ValidateFile(filePath string, targetFormat string) (*FileValidationResult, error) {
	result := &FileValidationResult{
		FilePath:       filePath,
		ValidationTime: time.Now(),
	}

	// 检查目标文件是否存在
	targetPath := v.getTargetPath(filePath, targetFormat)
	if _, err := os.Stat(targetPath); err == nil {
		result.HasTargetFile = true
		result.IsValid = true
	} else {
		result.IsValid = false
		result.Issues = append(result.Issues, "目标文件不存在")
	}

	return result, nil
}

// GenerateReport 生成验证报告
func (v *PostProcessingValidator) GenerateReport(result *ValidationResult, outputPath string) error {
	// 生成JSON报告
	reportData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("生成验证报告失败: %w", err)
	}

	if err := os.WriteFile(outputPath, reportData, 0644); err != nil {
		return fmt.Errorf("写入验证报告失败: %w", err)
	}

	return nil
}

// ValidateWithContext 带上下文的验证
func ValidateWithContext(ctx context.Context, validator Validator, workDir string, targetFormat string, expectedProcessed int) (*ValidationResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return validator.ValidateDirectory(workDir, targetFormat, expectedProcessed)
	}
}
