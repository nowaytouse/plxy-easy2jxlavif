package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// PostProcessingValidator 处理完成后验证器
type PostProcessingValidator struct {
	logger *zap.Logger
}

// ValidationResult 验证结果
type ValidationResult struct {
	TotalFiles          int                    `json:"total_files"`
	ProcessedFiles      int                    `json:"processed_files"`
	SkippedFiles        int                    `json:"skipped_files"`
	FailedFiles         int                    `json:"failed_files"`
	UnprocessedFiles    []UnprocessedFile      `json:"unprocessed_files"`
	SizeValidation      SizeValidationResult   `json:"size_validation"`
	ExifValidation      ExifValidationResult   `json:"exif_validation"`
	FormatValidation    FormatValidationResult `json:"format_validation"`
	ProcessingTime      time.Duration          `json:"processing_time"`
	ValidationTimestamp time.Time              `json:"validation_timestamp"`
}

// UnprocessedFile 未处理的文件信息
type UnprocessedFile struct {
	FilePath      string `json:"file_path"`
	Reason        string `json:"reason"`
	FileType      string `json:"file_type"`
	FileSize      int64  `json:"file_size"`
	IsSupported   bool   `json:"is_supported"`
	ShouldProcess bool   `json:"should_process"`
}

// SizeValidationResult 大小验证结果
type SizeValidationResult struct {
	OriginalTotalSize  int64   `json:"original_total_size"`
	ProcessedTotalSize int64   `json:"processed_total_size"`
	SizeReduction      int64   `json:"size_reduction"`
	CompressionRatio   float64 `json:"compression_ratio"`
	SizeValidationPass bool    `json:"size_validation_pass"`
}

// ExifValidationResult EXIF验证结果
type ExifValidationResult struct {
	FilesWithExif        int     `json:"files_with_exif"`
	ExifPreserved        int     `json:"exif_preserved"`
	ExifLost             int     `json:"exif_lost"`
	ExifPreservationRate float64 `json:"exif_preservation_rate"`
	ExifValidationPass   bool    `json:"exif_validation_pass"`
}

// FormatValidationResult 格式验证结果
type FormatValidationResult struct {
	TargetFormatFiles    int            `json:"target_format_files"`
	SourceFormatFiles    int            `json:"source_format_files"`
	FormatDistribution   map[string]int `json:"format_distribution"`
	FormatValidationPass bool           `json:"format_validation_pass"`
}

// NewPostProcessingValidator 创建新的验证器
func NewPostProcessingValidator(logger *zap.Logger) *PostProcessingValidator {
	return &PostProcessingValidator{
		logger: logger,
	}
}

// ValidateDirectory 验证目录处理结果
func (v *PostProcessingValidator) ValidateDirectory(workDir string, targetFormat string, expectedProcessed int) (*ValidationResult, error) {
	v.logger.Info("开始验证目录处理结果",
		zap.String("work_dir", workDir),
		zap.String("target_format", targetFormat),
		zap.Int("expected_processed", expectedProcessed))

	startTime := time.Now()
	result := &ValidationResult{
		ValidationTimestamp: time.Now(),
	}

	// 扫描所有文件
	allFiles, err := v.scanAllFiles(workDir)
	if err != nil {
		return nil, fmt.Errorf("扫描文件失败: %w", err)
	}

	result.TotalFiles = len(allFiles)
	v.logger.Info("扫描完成", zap.Int("total_files", result.TotalFiles))

	// 分析文件类型和状态
	processedFiles, skippedFiles, failedFiles, unprocessedFiles := v.analyzeFiles(allFiles, targetFormat)
	result.ProcessedFiles = len(processedFiles)
	result.SkippedFiles = len(skippedFiles)
	result.FailedFiles = len(failedFiles)
	result.UnprocessedFiles = unprocessedFiles

	// 大小验证
	sizeResult, err := v.validateSizes(processedFiles, targetFormat)
	if err != nil {
		v.logger.Warn("大小验证失败", zap.Error(err))
	} else {
		result.SizeValidation = *sizeResult
	}

	// EXIF验证
	exifResult, err := v.validateExif(processedFiles, targetFormat)
	if err != nil {
		v.logger.Warn("EXIF验证失败", zap.Error(err))
	} else {
		result.ExifValidation = *exifResult
	}

	// 格式验证
	formatResult, err := v.validateFormats(allFiles, targetFormat)
	if err != nil {
		v.logger.Warn("格式验证失败", zap.Error(err))
	} else {
		result.FormatValidation = *formatResult
	}

	result.ProcessingTime = time.Since(startTime)

	// 生成验证报告
	v.generateValidationReport(result, workDir)

	return result, nil
}

// scanAllFiles 扫描目录中的所有文件
func (v *PostProcessingValidator) scanAllFiles(workDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			v.logger.Warn("扫描文件时出错", zap.String("path", path), zap.Error(err))
			return nil
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

// analyzeFiles 分析文件状态
func (v *PostProcessingValidator) analyzeFiles(allFiles []string, targetFormat string) ([]string, []string, []string, []UnprocessedFile) {
	var processedFiles, skippedFiles, failedFiles []string
	var unprocessedFiles []UnprocessedFile

	for _, filePath := range allFiles {
		fileName := filepath.Base(filePath)
		ext := strings.ToLower(filepath.Ext(filePath))

		// 跳过系统文件
		if v.isSystemFile(fileName) {
			continue
		}

		// 检查是否是目标格式
		if strings.TrimPrefix(ext, ".") == strings.TrimPrefix(targetFormat, ".") {
			processedFiles = append(processedFiles, filePath)
			continue
		}

		// 检查是否是支持的源格式
		if v.isSupportedSourceFormat(ext) {
			// 检查是否有对应的目标文件
			targetPath := v.getTargetPath(filePath, targetFormat)
			if _, err := os.Stat(targetPath); err == nil {
				// 目标文件存在，说明已处理
				processedFiles = append(processedFiles, filePath)
			} else {
				// 目标文件不存在，说明未处理
				unprocessedFile := UnprocessedFile{
					FilePath:      filePath,
					Reason:        "目标文件不存在",
					FileType:      ext,
					IsSupported:   true,
					ShouldProcess: true,
				}
				if stat, err := os.Stat(filePath); err == nil {
					unprocessedFile.FileSize = stat.Size()
				}
				unprocessedFiles = append(unprocessedFiles, unprocessedFile)
			}
		} else {
			// 不支持的格式
			unprocessedFile := UnprocessedFile{
				FilePath:      filePath,
				Reason:        "不支持的源格式",
				FileType:      ext,
				IsSupported:   false,
				ShouldProcess: false,
			}
			if stat, err := os.Stat(filePath); err == nil {
				unprocessedFile.FileSize = stat.Size()
			}
			unprocessedFiles = append(unprocessedFiles, unprocessedFile)
		}
	}

	return processedFiles, skippedFiles, failedFiles, unprocessedFiles
}

// validateSizes 验证文件大小
func (v *PostProcessingValidator) validateSizes(processedFiles []string, targetFormat string) (*SizeValidationResult, error) {
	result := &SizeValidationResult{}

	for _, filePath := range processedFiles {
		// 获取原始文件大小
		if stat, err := os.Stat(filePath); err == nil {
			result.OriginalTotalSize += stat.Size()
		}

		// 获取目标文件大小
		targetPath := v.getTargetPath(filePath, targetFormat)
		if stat, err := os.Stat(targetPath); err == nil {
			result.ProcessedTotalSize += stat.Size()
		}
	}

	result.SizeReduction = result.OriginalTotalSize - result.ProcessedTotalSize
	if result.OriginalTotalSize > 0 {
		result.CompressionRatio = float64(result.ProcessedTotalSize) / float64(result.OriginalTotalSize) * 100
	}
	result.SizeValidationPass = result.SizeReduction > 0

	return result, nil
}

// validateExif 验证EXIF数据
func (v *PostProcessingValidator) validateExif(processedFiles []string, targetFormat string) (*ExifValidationResult, error) {
	result := &ExifValidationResult{}

	// 这里需要实现EXIF检查逻辑
	// 由于EXIF检查比较复杂，这里提供基本框架
	for _, filePath := range processedFiles {
		// 检查原始文件是否有EXIF
		if v.hasExifData(filePath) {
			result.FilesWithExif++

			// 检查目标文件是否保留了EXIF
			targetPath := v.getTargetPath(filePath, targetFormat)
			if v.hasExifData(targetPath) {
				result.ExifPreserved++
			} else {
				result.ExifLost++
			}
		}
	}

	if result.FilesWithExif > 0 {
		result.ExifPreservationRate = float64(result.ExifPreserved) / float64(result.FilesWithExif) * 100
	}
	result.ExifValidationPass = result.ExifPreservationRate >= 80.0 // 80%以上保留率认为通过

	return result, nil
}

// validateFormats 验证格式分布
func (v *PostProcessingValidator) validateFormats(allFiles []string, targetFormat string) (*FormatValidationResult, error) {
	result := &FormatValidationResult{
		FormatDistribution: make(map[string]int),
	}

	for _, filePath := range allFiles {
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext == "" {
			ext = "unknown"
		}
		result.FormatDistribution[ext]++

		if strings.TrimPrefix(ext, ".") == strings.TrimPrefix(targetFormat, ".") {
			result.TargetFormatFiles++
		} else if v.isSupportedSourceFormat(ext) {
			result.SourceFormatFiles++
		}
	}

	result.FormatValidationPass = result.TargetFormatFiles > 0

	return result, nil
}

// generateValidationReport 生成验证报告
func (v *PostProcessingValidator) generateValidationReport(result *ValidationResult, workDir string) {
	reportPath := filepath.Join(workDir, "validation_report.json")

	// 生成JSON报告
	reportData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		v.logger.Error("生成验证报告失败", zap.Error(err))
		return
	}

	if err := os.WriteFile(reportPath, reportData, 0644); err != nil {
		v.logger.Error("写入验证报告失败", zap.Error(err))
		return
	}

	// 生成用户友好的文本报告
	textReportPath := filepath.Join(workDir, "validation_report.txt")
	v.generateTextReport(result, textReportPath)

	v.logger.Info("验证报告已生成",
		zap.String("json_report", reportPath),
		zap.String("text_report", textReportPath))
}

// generateTextReport 生成文本格式的验证报告
func (v *PostProcessingValidator) generateTextReport(result *ValidationResult, reportPath string) {
	var report strings.Builder

	report.WriteString("=== 处理结果验证报告 ===\n")
	report.WriteString(fmt.Sprintf("验证时间: %s\n", result.ValidationTimestamp.Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("处理耗时: %v\n\n", result.ProcessingTime))

	// 文件统计
	report.WriteString("📊 文件统计:\n")
	report.WriteString(fmt.Sprintf("  总文件数: %d\n", result.TotalFiles))
	report.WriteString(fmt.Sprintf("  已处理: %d\n", result.ProcessedFiles))
	report.WriteString(fmt.Sprintf("  跳过: %d\n", result.SkippedFiles))
	report.WriteString(fmt.Sprintf("  失败: %d\n", result.FailedFiles))
	report.WriteString(fmt.Sprintf("  未处理: %d\n\n", len(result.UnprocessedFiles)))

	// 大小验证
	report.WriteString("💾 大小验证:\n")
	report.WriteString(fmt.Sprintf("  原始总大小: %.2f MB\n", float64(result.SizeValidation.OriginalTotalSize)/(1024*1024)))
	report.WriteString(fmt.Sprintf("  处理后大小: %.2f MB\n", float64(result.SizeValidation.ProcessedTotalSize)/(1024*1024)))
	report.WriteString(fmt.Sprintf("  节省空间: %.2f MB\n", float64(result.SizeValidation.SizeReduction)/(1024*1024)))
	report.WriteString(fmt.Sprintf("  压缩率: %.1f%%\n", result.SizeValidation.CompressionRatio))
	report.WriteString(fmt.Sprintf("  验证结果: %s\n\n", v.boolToStatus(result.SizeValidation.SizeValidationPass)))

	// EXIF验证
	report.WriteString("📷 EXIF验证:\n")
	report.WriteString(fmt.Sprintf("  有EXIF文件: %d\n", result.ExifValidation.FilesWithExif))
	report.WriteString(fmt.Sprintf("  EXIF保留: %d\n", result.ExifValidation.ExifPreserved))
	report.WriteString(fmt.Sprintf("  EXIF丢失: %d\n", result.ExifValidation.ExifLost))
	report.WriteString(fmt.Sprintf("  保留率: %.1f%%\n", result.ExifValidation.ExifPreservationRate))
	report.WriteString(fmt.Sprintf("  验证结果: %s\n\n", v.boolToStatus(result.ExifValidation.ExifValidationPass)))

	// 格式验证
	report.WriteString("🎨 格式验证:\n")
	report.WriteString(fmt.Sprintf("  目标格式文件: %d\n", result.FormatValidation.TargetFormatFiles))
	report.WriteString(fmt.Sprintf("  源格式文件: %d\n", result.FormatValidation.SourceFormatFiles))
	report.WriteString(fmt.Sprintf("  验证结果: %s\n\n", v.boolToStatus(result.FormatValidation.FormatValidationPass)))

	// 未处理文件详情
	if len(result.UnprocessedFiles) > 0 {
		report.WriteString("⚠️  未处理文件详情:\n")
		for i, file := range result.UnprocessedFiles {
			if i >= 10 { // 只显示前10个
				report.WriteString(fmt.Sprintf("  ... 还有 %d 个文件未显示\n", len(result.UnprocessedFiles)-10))
				break
			}
			report.WriteString(fmt.Sprintf("  %s - %s (%s)\n",
				filepath.Base(file.FilePath), file.Reason, file.FileType))
		}
		report.WriteString("\n")
	}

	// 总体评估
	report.WriteString("🎯 总体评估:\n")
	overallPass := result.SizeValidation.SizeValidationPass &&
		result.ExifValidation.ExifValidationPass &&
		result.FormatValidation.FormatValidationPass &&
		len(result.UnprocessedFiles) == 0

	report.WriteString(fmt.Sprintf("  整体验证: %s\n", v.boolToStatus(overallPass)))
	if !overallPass {
		report.WriteString("  建议: 检查未处理文件和验证失败的项目\n")
	}

	// 写入文件
	if err := os.WriteFile(reportPath, []byte(report.String()), 0644); err != nil {
		v.logger.Error("写入文本报告失败", zap.Error(err))
	}
}

// 辅助方法
func (v *PostProcessingValidator) isSystemFile(fileName string) bool {
	systemFiles := []string{".DS_Store", "Thumbs.db", "Desktop.ini", ".localized"}
	for _, sysFile := range systemFiles {
		if fileName == sysFile {
			return true
		}
	}
	return strings.HasPrefix(fileName, ".")
}

func (v *PostProcessingValidator) isSupportedSourceFormat(ext string) bool {
	supportedFormats := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff", ".tif", ".heic", ".heif", ".avif"}
	for _, format := range supportedFormats {
		if ext == format {
			return true
		}
	}
	return false
}

func (v *PostProcessingValidator) getTargetPath(sourcePath, targetFormat string) string {
	ext := filepath.Ext(sourcePath)
	return strings.TrimSuffix(sourcePath, ext) + "." + strings.TrimPrefix(targetFormat, ".")
}

func (v *PostProcessingValidator) hasExifData(filePath string) bool {
	// 简化的EXIF检查，实际实现需要更复杂的逻辑
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// 读取文件头检查EXIF标记
	header := make([]byte, 1024)
	if _, err := file.Read(header); err != nil {
		return false
	}

	// 检查JPEG EXIF标记
	if strings.Contains(string(header), "Exif") {
		return true
	}

	return false
}

func (v *PostProcessingValidator) boolToStatus(pass bool) string {
	if pass {
		return "✅ 通过"
	}
	return "❌ 失败"
}
