package metamigrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// MetadataMigrator 元数据迁移器 - README要求的元数据完整性迁移
//
// 核心功能：
//   - 强制迁移EXIF、ICC、XMP等关键元数据
//   - 支持跨格式元数据转换和适配
//   - 提供元数据完整性验证和修复机制
//   - 实现色彩空间信息的准确迁移
//   - 支持补偿性元数据恢复策略
//
// 设计原则：
//   - 元数据完整性优先：优先保持原有元数据的完整性
//   - 格式兼容性：根据目标格式调整元数据结构
//   - 关键信息保护：确保关键元数据字段不丢失
//   - 可逆操作：为无色彩空间文件添加可逆的sRGB标签
//   - 渐进降级：完整迁移失败时尝试关键字段复制
type MetadataMigrator struct {
	logger           *zap.Logger
	exiftoolPath     string
	migrationMode    MigrationMode
	preserveFields   []string              // 优先保护的元数据字段
	formatMappings   map[string]FormatInfo // 格式特定的元数据映射
	colorSpaceConfig *ColorSpaceConfig     // 色彩空间配置
	validationLevel  ValidationLevel       // 验证级别
	migrationCache   map[string]*MigrationResult
}

// MigrationResult 迁移结果
type MigrationResult struct {
	SourcePath       string           `json:"source_path"`
	TargetPath       string           `json:"target_path"`
	SourceFormat     string           `json:"source_format"`
	TargetFormat     string           `json:"target_format"`
	MigratedFields   []MetadataField  `json:"migrated_fields"`
	LostFields       []MetadataField  `json:"lost_fields"`
	AddedFields      []MetadataField  `json:"added_fields"`
	ColorSpaceInfo   *ColorSpaceInfo  `json:"color_space_info"`
	ValidationStatus ValidationStatus `json:"validation_status"`
	MigrationTime    time.Duration    `json:"migration_time"`
	Success          bool             `json:"success"`
	ErrorMessage     string           `json:"error_message,omitempty"`
	Warnings         []string         `json:"warnings"`
}

// MetadataField 元数据字段
type MetadataField struct {
	Name        string      `json:"name"`
	Value       interface{} `json:"value"`
	Type        string      `json:"type"`
	Source      string      `json:"source"`      // exif, icc, xmp, iptc
	Critical    bool        `json:"critical"`    // 是否为关键字段
	Transformed bool        `json:"transformed"` // 是否经过转换
}

// ColorSpaceInfo 色彩空间信息
type ColorSpaceInfo struct {
	ProfileName        string  `json:"profile_name"`
	ColorSpace         string  `json:"color_space"`
	WhitePoint         string  `json:"white_point"`
	Primaries          string  `json:"primaries"`
	Gamma              float64 `json:"gamma"`
	ICCProfileEmbedded bool    `json:"icc_profile_embedded"`
	NeedsConversion    bool    `json:"needs_conversion"`
	AddedSRGB          bool    `json:"added_srgb"`
	ConversionMethod   string  `json:"conversion_method"`
}

// FormatInfo 格式信息
type FormatInfo struct {
	SupportedMetadata []string          `json:"supported_metadata"` // 支持的元数据类型
	RequiredFields    []string          `json:"required_fields"`    // 必需字段
	FieldMappings     map[string]string `json:"field_mappings"`     // 字段映射
	ColorSpaceSupport bool              `json:"color_space_support"`
}

// ColorSpaceConfig 色彩空间配置
type ColorSpaceConfig struct {
	DefaultProfile   string
	FallbackToSRGB   bool
	AllowConversion  bool
	RequiredAccuracy float64
}

// 枚举定义
type MigrationMode int
type ValidationLevel int
type ValidationStatus int

const (
	// 迁移模式
	MigrationComplete  MigrationMode = iota // 完整迁移
	MigrationEssential                      // 仅迁移关键元数据
	MigrationMinimal                        // 最小迁移
	MigrationCustom                         // 自定义迁移
)

const (
	// 验证级别
	ValidationStrict ValidationLevel = iota // 严格验证
	ValidationNormal                        // 正常验证
	ValidationBasic                         // 基础验证
	ValidationNone                          // 无验证
)

const (
	// 验证状态
	ValidationPassed  ValidationStatus = iota // 验证通过
	ValidationWarning                         // 有警告
	ValidationFailed                          // 验证失败
	ValidationSkipped                         // 跳过验证
)

// NewMetadataMigrator 创建元数据迁移器
func NewMetadataMigrator(logger *zap.Logger, exiftoolPath string) *MetadataMigrator {
	migrator := &MetadataMigrator{
		logger:          logger,
		exiftoolPath:    exiftoolPath,
		migrationMode:   MigrationComplete,
		validationLevel: ValidationNormal,
		migrationCache:  make(map[string]*MigrationResult),
	}

	// 初始化保护字段列表 - README要求的关键元数据
	migrator.preserveFields = []string{
		// EXIF核心字段
		"Make", "Model", "DateTime", "DateTimeOriginal", "DateTimeDigitized",
		"Orientation", "XResolution", "YResolution", "ColorSpace",
		"WhitePoint", "PrimaryChromaticities", "YCbCrCoefficients",
		"ExifVersion", "ComponentsConfiguration", "FlashpixVersion",

		// GPS信息
		"GPSLatitude", "GPSLongitude", "GPSAltitude", "GPSTimeStamp",
		"GPSLatitudeRef", "GPSLongitudeRef", "GPSAltitudeRef",

		// 相机参数
		"ExposureTime", "FNumber", "ExposureProgram", "ISOSpeedRatings",
		"ShutterSpeedValue", "ApertureValue", "FocalLength", "Flash",
		"MeteringMode", "LightSource", "FocalLengthIn35mmFilm",

		// ICC颜色管理
		"TransferFunction", "ReferenceBlackWhite",

		// XMP权限和版权
		"Copyright", "Artist", "Software", "ImageDescription",
		"XMPToolkit", "CreatorTool", "Rights",
	}

	migrator.initializeFormatMappings()
	migrator.initializeColorSpaceConfig()

	return migrator
}

// initializeFormatMappings 初始化格式映射
func (mm *MetadataMigrator) initializeFormatMappings() {
	mm.formatMappings = map[string]FormatInfo{
		"jpeg": {
			SupportedMetadata: []string{"exif", "icc", "iptc", "xmp"},
			RequiredFields:    []string{"Orientation", "ColorSpace"},
			FieldMappings:     map[string]string{},
			ColorSpaceSupport: true,
		},
		"jxl": {
			SupportedMetadata: []string{"exif", "icc", "xmp"},
			RequiredFields:    []string{"Orientation"},
			FieldMappings: map[string]string{
				"JPEG2000:ColorSpace": "ColorSpace",
			},
			ColorSpaceSupport: true,
		},
		"avif": {
			SupportedMetadata: []string{"exif", "icc"},
			RequiredFields:    []string{"Orientation"},
			FieldMappings:     map[string]string{},
			ColorSpaceSupport: true,
		},
		"webp": {
			SupportedMetadata: []string{"exif", "icc", "xmp"},
			RequiredFields:    []string{"Orientation"},
			FieldMappings:     map[string]string{},
			ColorSpaceSupport: true,
		},
		"png": {
			SupportedMetadata: []string{"exif", "icc", "text"},
			RequiredFields:    []string{},
			FieldMappings: map[string]string{
				"PNG:ColorType": "ColorSpace",
			},
			ColorSpaceSupport: true,
		},
		"heif": {
			SupportedMetadata: []string{"exif", "icc"},
			RequiredFields:    []string{"Orientation"},
			FieldMappings:     map[string]string{},
			ColorSpaceSupport: true,
		},
	}
}

// initializeColorSpaceConfig 初始化色彩空间配置
func (mm *MetadataMigrator) initializeColorSpaceConfig() {
	mm.colorSpaceConfig = &ColorSpaceConfig{
		DefaultProfile:   "sRGB",
		FallbackToSRGB:   true,
		AllowConversion:  true,
		RequiredAccuracy: 0.95,
	}
}

// MigrateMetadata 迁移元数据 - README核心功能
func (mm *MetadataMigrator) MigrateMetadata(ctx context.Context, sourcePath, targetPath string) (*MigrationResult, error) {
	startTime := time.Now()

	// 检查缓存
	cacheKey := fmt.Sprintf("%s->%s", sourcePath, targetPath)
	if cached, exists := mm.migrationCache[cacheKey]; exists {
		mm.logger.Debug("使用缓存的迁移结果",
			zap.String("source", filepath.Base(sourcePath)),
			zap.String("target", filepath.Base(targetPath)))
		return cached, nil
	}

	result := &MigrationResult{
		SourcePath:     sourcePath,
		TargetPath:     targetPath,
		SourceFormat:   strings.ToLower(strings.TrimPrefix(filepath.Ext(sourcePath), ".")),
		TargetFormat:   strings.ToLower(strings.TrimPrefix(filepath.Ext(targetPath), ".")),
		MigratedFields: make([]MetadataField, 0),
		LostFields:     make([]MetadataField, 0),
		AddedFields:    make([]MetadataField, 0),
		Warnings:       make([]string, 0),
	}

	mm.logger.Info("开始元数据迁移",
		zap.String("source_format", result.SourceFormat),
		zap.String("target_format", result.TargetFormat),
		zap.String("source", filepath.Base(sourcePath)),
		zap.String("target", filepath.Base(targetPath)))

	// 1. 提取源文件元数据
	sourceMetadata, err := mm.extractMetadata(ctx, sourcePath)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("提取源文件元数据失败: %v", err)
		result.Success = false
		return result, err
	}

	// 2. 分析色彩空间信息
	colorSpaceInfo, err := mm.analyzeColorSpace(ctx, sourcePath, sourceMetadata)
	if err != nil {
		mm.logger.Warn("色彩空间分析失败", zap.Error(err))
		result.Warnings = append(result.Warnings, fmt.Sprintf("色彩空间分析失败: %v", err))
	}
	result.ColorSpaceInfo = colorSpaceInfo

	// 3. 根据目标格式筛选和转换元数据
	migratedMetadata, err := mm.transformMetadataForTarget(result.TargetFormat, sourceMetadata)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("元数据转换失败: %v", err)
		result.Success = false
		return result, err
	}

	// 4. 写入目标文件元数据
	if err := mm.writeMetadata(ctx, targetPath, migratedMetadata, colorSpaceInfo); err != nil {
		// README要求：完整迁移失败时尝试关键字段复制
		mm.logger.Warn("完整元数据写入失败，尝试关键字段迁移", zap.Error(err))

		essentialMetadata := mm.extractEssentialMetadata(migratedMetadata)
		if essentialErr := mm.writeMetadata(ctx, targetPath, essentialMetadata, colorSpaceInfo); essentialErr != nil {
			result.ErrorMessage = fmt.Sprintf("关键元数据迁移也失败: %v", essentialErr)
			result.Success = false
			return result, fmt.Errorf("元数据迁移失败: %w", err)
		}

		result.Warnings = append(result.Warnings, "仅成功迁移关键元数据字段")
		result.MigratedFields = mm.convertToMetadataFields(essentialMetadata, true)
	} else {
		result.MigratedFields = mm.convertToMetadataFields(migratedMetadata, false)
	}

	// 5. 验证迁移结果
	if mm.validationLevel > ValidationNone {
		validationResult := mm.validateMigration(ctx, targetPath, sourceMetadata)
		result.ValidationStatus = validationResult.Status
		result.Warnings = append(result.Warnings, validationResult.Warnings...)
	}

	// 6. 处理色彩空间 - README要求：为无色彩空间文件添加可逆的sRGB标签
	if colorSpaceInfo != nil && colorSpaceInfo.NeedsConversion {
		if err := mm.handleColorSpaceConversion(ctx, targetPath, colorSpaceInfo); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("色彩空间处理失败: %v", err))
		}
	}

	result.MigrationTime = time.Since(startTime)
	result.Success = true

	// 缓存结果
	mm.migrationCache[cacheKey] = result

	mm.logger.Info("元数据迁移完成",
		zap.String("source", filepath.Base(sourcePath)),
		zap.String("target", filepath.Base(targetPath)),
		zap.Int("migrated_fields", len(result.MigratedFields)),
		zap.Int("warnings", len(result.Warnings)),
		zap.Duration("migration_time", result.MigrationTime),
		zap.Bool("success", result.Success))

	return result, nil
}

// extractMetadata 提取文件元数据
func (mm *MetadataMigrator) extractMetadata(ctx context.Context, filePath string) (map[string]interface{}, error) {
	if mm.exiftoolPath == "" {
		return nil, fmt.Errorf("exiftool路径未设置")
	}

	// 使用exiftool提取所有元数据
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	args := []string{
		"-json",
		"-all",                 // 提取所有元数据
		"-binary",              // 包含二进制数据
		"-coordFormat", "%.6f", // GPS坐标格式
		"-dateFormat", "%Y:%m:%d %H:%M:%S", // 日期格式标准化
		filePath,
	}

	cmd := exec.CommandContext(timeoutCtx, mm.exiftoolPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("exiftool执行失败: %w", err)
	}

	// 解析JSON输出
	var metadataArray []map[string]interface{}
	if err := json.Unmarshal(output, &metadataArray); err != nil {
		return nil, fmt.Errorf("元数据JSON解析失败: %w", err)
	}

	if len(metadataArray) == 0 {
		return make(map[string]interface{}), nil
	}

	metadata := metadataArray[0]

	mm.logger.Debug("提取到元数据字段",
		zap.String("file", filepath.Base(filePath)),
		zap.Int("field_count", len(metadata)))

	return metadata, nil
}

// analyzeColorSpace 分析色彩空间信息
func (mm *MetadataMigrator) analyzeColorSpace(ctx context.Context, filePath string, metadata map[string]interface{}) (*ColorSpaceInfo, error) {
	colorSpaceInfo := &ColorSpaceInfo{
		ConversionMethod: "exiftool",
	}

	// 从元数据中提取色彩空间信息
	if colorSpace, exists := metadata["ColorSpace"]; exists {
		if cs, ok := colorSpace.(string); ok {
			colorSpaceInfo.ColorSpace = cs
		} else if cs, ok := colorSpace.(float64); ok {
			colorSpaceInfo.ColorSpace = fmt.Sprintf("ColorSpace_%d", int(cs))
		}
	}

	// 提取ICC配置文件信息
	if iccProfile, exists := metadata["ICC_Profile"]; exists {
		colorSpaceInfo.ICCProfileEmbedded = true
		if profile, ok := iccProfile.(map[string]interface{}); ok {
			if profileDesc, exists := profile["Profile Description"]; exists {
				if desc, ok := profileDesc.(string); ok {
					colorSpaceInfo.ProfileName = desc
				}
			}
		}
	}

	// 提取白点信息
	if whitePoint, exists := metadata["WhitePoint"]; exists {
		if wp, ok := whitePoint.(string); ok {
			colorSpaceInfo.WhitePoint = wp
		}
	}

	// 提取主色度坐标
	if primaries, exists := metadata["PrimaryChromaticities"]; exists {
		if prim, ok := primaries.(string); ok {
			colorSpaceInfo.Primaries = prim
		}
	}

	// 提取Gamma值
	if gamma, exists := metadata["Gamma"]; exists {
		if g, ok := gamma.(float64); ok {
			colorSpaceInfo.Gamma = g
		} else if g, ok := gamma.(string); ok {
			if gVal, err := strconv.ParseFloat(g, 64); err == nil {
				colorSpaceInfo.Gamma = gVal
			}
		}
	}

	// README要求：判断是否需要添加sRGB标签
	if colorSpaceInfo.ColorSpace == "" || colorSpaceInfo.ColorSpace == "Uncalibrated" {
		colorSpaceInfo.NeedsConversion = true
		colorSpaceInfo.AddedSRGB = true
		mm.logger.Debug("检测到无色彩空间信息，将添加sRGB标签",
			zap.String("file", filepath.Base(filePath)))
	}

	return colorSpaceInfo, nil
}

// transformMetadataForTarget 根据目标格式转换元数据
func (mm *MetadataMigrator) transformMetadataForTarget(targetFormat string, metadata map[string]interface{}) (map[string]interface{}, error) {
	formatInfo, exists := mm.formatMappings[targetFormat]
	if !exists {
		return nil, fmt.Errorf("不支持的目标格式: %s", targetFormat)
	}

	transformedMetadata := make(map[string]interface{})

	// 根据迁移模式处理
	switch mm.migrationMode {
	case MigrationComplete:
		// 完整迁移：尝试迁移所有兼容字段
		for key, value := range metadata {
			if mm.isFieldSupported(key, formatInfo) {
				transformedKey := mm.transformFieldName(key, formatInfo)
				transformedMetadata[transformedKey] = value
			}
		}

	case MigrationEssential:
		// 仅迁移关键字段
		for _, field := range mm.preserveFields {
			if value, exists := metadata[field]; exists {
				if mm.isFieldSupported(field, formatInfo) {
					transformedKey := mm.transformFieldName(field, formatInfo)
					transformedMetadata[transformedKey] = value
				}
			}
		}

	case MigrationMinimal:
		// 最小迁移：仅必需字段
		for _, field := range formatInfo.RequiredFields {
			if value, exists := metadata[field]; exists {
				transformedMetadata[field] = value
			}
		}
	}

	mm.logger.Debug("元数据转换完成",
		zap.String("target_format", targetFormat),
		zap.Int("original_fields", len(metadata)),
		zap.Int("transformed_fields", len(transformedMetadata)))

	return transformedMetadata, nil
}

// writeMetadata 写入元数据到目标文件
func (mm *MetadataMigrator) writeMetadata(ctx context.Context, targetPath string, metadata map[string]interface{}, colorSpaceInfo *ColorSpaceInfo) error {
	if mm.exiftoolPath == "" {
		return fmt.Errorf("exiftool路径未设置")
	}

	if len(metadata) == 0 {
		mm.logger.Debug("无元数据需要写入", zap.String("target", filepath.Base(targetPath)))
		return nil
	}

	// 使用exiftool写入元数据
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	args := []string{
		"-overwrite_original", // 覆盖原文件
		"-preserve",           // 保留文件时间戳
	}

	// 添加元数据写入参数
	for key, value := range metadata {
		args = append(args, fmt.Sprintf("-%s=%v", key, value))
	}

	args = append(args, targetPath)

	cmd := exec.CommandContext(timeoutCtx, mm.exiftoolPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("写入元数据失败: %w (输出: %s)", err, string(output))
	}

	mm.logger.Debug("元数据写入完成",
		zap.String("target", filepath.Base(targetPath)),
		zap.Int("fields_written", len(metadata)))

	return nil
}

// handleColorSpaceConversion 处理色彩空间转换 - README要求的sRGB标签添加
func (mm *MetadataMigrator) handleColorSpaceConversion(ctx context.Context, targetPath string, colorSpaceInfo *ColorSpaceInfo) error {
	if !colorSpaceInfo.AddedSRGB {
		return nil
	}

	// README要求：为无色彩空间文件添加可逆的sRGB标签
	args := []string{
		"-overwrite_original",
		"-ColorSpace=1", // sRGB
		targetPath,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, mm.exiftoolPath, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("添加sRGB标签失败: %w", err)
	}

	mm.logger.Debug("成功添加sRGB色彩空间标签", zap.String("target", filepath.Base(targetPath)))
	return nil
}

// ValidationResult 验证结果
type ValidationResult struct {
	Status   ValidationStatus
	Warnings []string
	Details  map[string]interface{}
}

// validateMigration 验证迁移结果
func (mm *MetadataMigrator) validateMigration(ctx context.Context, targetPath string, originalMetadata map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Status:   ValidationPassed,
		Warnings: make([]string, 0),
		Details:  make(map[string]interface{}),
	}

	// 重新提取目标文件元数据进行对比
	targetMetadata, err := mm.extractMetadata(ctx, targetPath)
	if err != nil {
		result.Status = ValidationFailed
		result.Warnings = append(result.Warnings, fmt.Sprintf("验证时重新提取元数据失败: %v", err))
		return result
	}

	// 检查关键字段是否存在
	missingCritical := 0
	for _, field := range mm.preserveFields {
		if _, existsOriginal := originalMetadata[field]; existsOriginal {
			if _, existsTarget := targetMetadata[field]; !existsTarget {
				missingCritical++
				result.Warnings = append(result.Warnings, fmt.Sprintf("关键字段丢失: %s", field))
			}
		}
	}

	if missingCritical > 0 {
		result.Status = ValidationWarning
		result.Details["missing_critical_fields"] = missingCritical
	}

	result.Details["original_field_count"] = len(originalMetadata)
	result.Details["target_field_count"] = len(targetMetadata)

	return result
}

// 辅助方法
func (mm *MetadataMigrator) isFieldSupported(fieldName string, formatInfo FormatInfo) bool {
	// 检查字段是否在支持的元数据类型中
	fieldPrefix := strings.Split(fieldName, ":")[0]
	for _, supportedType := range formatInfo.SupportedMetadata {
		if strings.EqualFold(fieldPrefix, supportedType) || fieldPrefix == "" {
			return true
		}
	}
	return false
}

func (mm *MetadataMigrator) transformFieldName(fieldName string, formatInfo FormatInfo) string {
	if mappedName, exists := formatInfo.FieldMappings[fieldName]; exists {
		return mappedName
	}
	return fieldName
}

func (mm *MetadataMigrator) extractEssentialMetadata(metadata map[string]interface{}) map[string]interface{} {
	essential := make(map[string]interface{})

	for _, field := range mm.preserveFields {
		if value, exists := metadata[field]; exists {
			essential[field] = value
		}
	}

	return essential
}

func (mm *MetadataMigrator) convertToMetadataFields(metadata map[string]interface{}, essential bool) []MetadataField {
	fields := make([]MetadataField, 0, len(metadata))

	for key, value := range metadata {
		field := MetadataField{
			Name:     key,
			Value:    value,
			Type:     fmt.Sprintf("%T", value),
			Critical: mm.isEssentialField(key),
		}

		// 确定元数据源
		if strings.Contains(key, ":") {
			field.Source = strings.ToLower(strings.Split(key, ":")[0])
		} else {
			field.Source = "exif"
		}

		fields = append(fields, field)
	}

	return fields
}

func (mm *MetadataMigrator) isEssentialField(fieldName string) bool {
	for _, essential := range mm.preserveFields {
		if fieldName == essential {
			return true
		}
	}
	return false
}

// SetMigrationMode 设置迁移模式
func (mm *MetadataMigrator) SetMigrationMode(mode MigrationMode) {
	mm.migrationMode = mode
}

// SetValidationLevel 设置验证级别
func (mm *MetadataMigrator) SetValidationLevel(level ValidationLevel) {
	mm.validationLevel = level
}

// GetMigrationStats 获取迁移统计信息
func (mm *MetadataMigrator) GetMigrationStats() map[string]int {
	stats := make(map[string]int)
	stats["total_migrations"] = len(mm.migrationCache)

	successCount := 0
	for _, result := range mm.migrationCache {
		if result.Success {
			successCount++
		}
	}
	stats["successful_migrations"] = successCount
	stats["failed_migrations"] = len(mm.migrationCache) - successCount

	return stats
}

// ClearCache 清理缓存
func (mm *MetadataMigrator) ClearCache() {
	mm.migrationCache = make(map[string]*MigrationResult)
	mm.logger.Debug("元数据迁移缓存已清理")
}
