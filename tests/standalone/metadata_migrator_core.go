package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// MetadataMigrator 元数据迁移器 - README要求的完善元数据迁移系统
type MetadataMigrator struct {
	logger           *zap.Logger
	exiftoolPath     string
	ffmpegPath       string
	enabledMigration bool
	backupDir        string
	fallbackMode     bool // 失败时复制关键字段模式
}

// MetadataResult 元数据迁移结果
type MetadataResult struct {
	SourcePath      string            `json:"source_path"`
	TargetPath      string            `json:"target_path"`
	Success         bool              `json:"success"`
	MigratedTags    []string          `json:"migrated_tags"`
	FailedTags      []string          `json:"failed_tags"`
	CriticalFields  map[string]string `json:"critical_fields"`
	ICCProfile      *ICCProfileInfo   `json:"icc_profile"`
	TimestampInfo   *TimestampInfo    `json:"timestamp_info"`
	MigrationMethod string            `json:"migration_method"`
	Error           string            `json:"error,omitempty"`
	ProcessedAt     time.Time         `json:"processed_at"`
}

// ICCProfileInfo ICC配置信息
type ICCProfileInfo struct {
	HasProfile    bool   `json:"has_profile"`
	ProfileSize   int64  `json:"profile_size"`
	ColorSpace    string `json:"color_space"`
	Description   string `json:"description"`
	Manufacturer  string `json:"manufacturer"`
	Model         string `json:"model"`
	Copyright     string `json:"copyright"`
	IsEmbedded    bool   `json:"is_embedded"`
	ProfilePath   string `json:"profile_path,omitempty"`
	FallbackAdded bool   `json:"fallback_added"` // 是否添加了fallback sRGB
}

// TimestampInfo 时间戳信息
type TimestampInfo struct {
	CreateTime    time.Time `json:"create_time"`
	ModifyTime    time.Time `json:"modify_time"`
	DateTimeOrig  time.Time `json:"datetime_orig,omitempty"`
	DateTimeDigit time.Time `json:"datetime_digit,omitempty"`
	GPSDateTime   time.Time `json:"gps_datetime,omitempty"`
	PreservedOrig bool      `json:"preserved_original"`
	PreservedMod  bool      `json:"preserved_modified"`
}

// CriticalMetadataFields 关键元数据字段
var CriticalMetadataFields = []string{
	// 基础时间信息
	"CreateDate", "ModifyDate", "DateTimeOriginal", "DateTimeDigitized",
	// 相机信息
	"Make", "Model", "Software", "Artist", "Copyright",
	// 拍摄参数
	"ISO", "ExposureTime", "FNumber", "FocalLength", "Flash", "WhiteBalance",
	// GPS信息
	"GPSLatitude", "GPSLongitude", "GPSAltitude", "GPSDateTime",
	// 图像参数
	"ImageWidth", "ImageHeight", "ColorSpace", "Orientation",
	// 镜头信息
	"LensModel", "LensMake", "LensSerialNumber",
}

// NewMetadataMigrator 创建元数据迁移器
func NewMetadataMigrator(logger *zap.Logger, exiftoolPath, ffmpegPath string) *MetadataMigrator {
	return &MetadataMigrator{
		logger:           logger,
		exiftoolPath:     exiftoolPath,
		ffmpegPath:       ffmpegPath,
		enabledMigration: true,
		backupDir:        "/tmp/metadata_backups",
		fallbackMode:     true,
	}
}

// PerformFullMigration 执行完整的元数据迁移 - README要求的强制迁移EXIF、ICC等
func (mm *MetadataMigrator) PerformFullMigration(ctx context.Context, sourcePath, targetPath string) (*MetadataResult, error) {
	result := &MetadataResult{
		SourcePath:      sourcePath,
		TargetPath:      targetPath,
		ProcessedAt:     time.Now(),
		CriticalFields:  make(map[string]string),
		MigrationMethod: "full_migration",
	}

	mm.logger.Info("开始完整元数据迁移",
		zap.String("source", filepath.Base(sourcePath)),
		zap.String("target", filepath.Base(targetPath)))

	// 第一步：提取源文件的所有元数据
	sourceMetadata, err := mm.extractAllMetadata(ctx, sourcePath)
	if err != nil {
		mm.logger.Warn("提取源文件元数据失败，尝试fallback模式", zap.Error(err))
		return mm.performFallbackMigration(ctx, sourcePath, targetPath)
	}

	// 第二步：处理ICC配置文件
	iccInfo, err := mm.handleICCProfile(ctx, sourcePath, targetPath, sourceMetadata)
	if err != nil {
		mm.logger.Warn("ICC配置处理失败", zap.Error(err))
	}
	result.ICCProfile = iccInfo

	// 第三步：处理时间戳信息
	timestampInfo, err := mm.handleTimestamps(sourcePath, targetPath, sourceMetadata)
	if err != nil {
		mm.logger.Warn("时间戳处理失败", zap.Error(err))
	}
	result.TimestampInfo = timestampInfo

	// 第四步：尝试完整元数据迁移
	success, migratedTags, failedTags := mm.performExiftoolMigration(ctx, sourcePath, targetPath, sourceMetadata)

	result.Success = success
	result.MigratedTags = migratedTags
	result.FailedTags = failedTags

	// 第五步：如果完整迁移失败，执行关键字段复制
	if !success && mm.fallbackMode {
		mm.logger.Info("完整迁移失败，执行关键字段复制模式")
		fallbackResult := mm.performCriticalFieldsCopy(ctx, targetPath, sourceMetadata)
		result.CriticalFields = fallbackResult
		result.MigrationMethod = "critical_fields_fallback"
	}

	// 第六步：验证迁移结果
	if err := mm.verifyMigration(ctx, targetPath, result); err != nil {
		mm.logger.Warn("元数据迁移验证失败", zap.Error(err))
		result.Error = err.Error()
	}

	mm.logger.Info("元数据迁移完成",
		zap.String("target", filepath.Base(targetPath)),
		zap.Bool("success", result.Success),
		zap.Int("migrated_tags", len(result.MigratedTags)),
		zap.Int("failed_tags", len(result.FailedTags)))

	return result, nil
}

// extractAllMetadata 提取所有元数据
func (mm *MetadataMigrator) extractAllMetadata(ctx context.Context, filePath string) (map[string]interface{}, error) {
	if mm.exiftoolPath == "" {
		return nil, fmt.Errorf("exiftool 路径未配置")
	}

	// 使用exiftool提取所有元数据为JSON格式
	args := []string{
		"-j",          // JSON输出
		"-all",        // 所有标签
		"-G",          // 显示组名
		"-duplicates", // 包含重复标签
		"-binary",     // 二进制数据
		filePath,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

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
		return nil, fmt.Errorf("未找到元数据")
	}

	return metadataArray[0], nil
}

// handleICCProfile 处理ICC配置文件 - README要求的ICC元数据强制迁移
func (mm *MetadataMigrator) handleICCProfile(ctx context.Context, sourcePath, targetPath string, metadata map[string]interface{}) (*ICCProfileInfo, error) {
	iccInfo := &ICCProfileInfo{}

	// 检查源文件是否有ICC配置
	if colorSpace, exists := metadata["ColorSpace"]; exists {
		iccInfo.ColorSpace = fmt.Sprintf("%v", colorSpace)
		iccInfo.HasProfile = true
	}

	if description, exists := metadata["ICCProfileDescription"]; exists {
		iccInfo.Description = fmt.Sprintf("%v", description)
		iccInfo.HasProfile = true
	}

	// 尝试提取ICC配置文件到临时文件
	if iccInfo.HasProfile {
		err := mm.extractICCProfile(ctx, sourcePath, targetPath)
		if err != nil {
			mm.logger.Warn("ICC配置提取失败", zap.Error(err))
			// 为无色彩空间信息的文件添加默认sRGB标签
			if err := mm.addDefaultSRGBProfile(ctx, targetPath); err == nil {
				iccInfo.FallbackAdded = true
				iccInfo.ColorSpace = "sRGB"
				mm.logger.Info("已添加默认sRGB配置", zap.String("target", filepath.Base(targetPath)))
			}
		} else {
			iccInfo.IsEmbedded = true
		}
	} else {
		// 没有ICC配置时，添加可逆的sRGB标签
		if err := mm.addDefaultSRGBProfile(ctx, targetPath); err == nil {
			iccInfo.FallbackAdded = true
			iccInfo.ColorSpace = "sRGB"
			mm.logger.Info("为无色彩空间文件添加sRGB配置", zap.String("target", filepath.Base(targetPath)))
		}
	}

	return iccInfo, nil
}
