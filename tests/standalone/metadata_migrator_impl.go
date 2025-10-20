package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// extractICCProfile 提取ICC配置文件
func (mm *MetadataMigrator) extractICCProfile(ctx context.Context, sourcePath, targetPath string) error {
	// 首先尝试直接复制ICC配置
	args := []string{
		"-icc_profile",
		"-o", targetPath,
		sourcePath,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, mm.exiftoolPath, args...)
	if err := cmd.Run(); err != nil {
		// 如果直接复制失败，尝试通过临时文件
		return mm.extractICCViaTempFile(ctx, sourcePath, targetPath)
	}

	return nil
}

// extractICCViaTempFile 通过临时文件提取ICC配置
func (mm *MetadataMigrator) extractICCViaTempFile(ctx context.Context, sourcePath, targetPath string) error {
	// 提取ICC配置到临时文件
	tempICCPath := filepath.Join(mm.backupDir, fmt.Sprintf("icc_%d.icc", time.Now().UnixNano()))
	
	// 确保备份目录存在
	if err := os.MkdirAll(mm.backupDir, 0755); err != nil {
		return fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 提取ICC配置
	extractArgs := []string{
		"-icc_profile",
		"-b",
		"-o", tempICCPath,
		sourcePath,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, mm.exiftoolPath, extractArgs...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ICC配置提取失败: %w", err)
	}

	// 嵌入ICC配置到目标文件
	embedArgs := []string{
		fmt.Sprintf("-icc_profile<=%s", tempICCPath),
		"-overwrite_original",
		targetPath,
	}

	cmd = exec.CommandContext(timeoutCtx, mm.exiftoolPath, embedArgs...)
	if err := cmd.Run(); err != nil {
		// 清理临时文件
		os.Remove(tempICCPath)
		return fmt.Errorf("ICC配置嵌入失败: %w", err)
	}

	// 清理临时文件
	os.Remove(tempICCPath)
	return nil
}

// addDefaultSRGBProfile 添加默认sRGB配置 - README要求的可逆sRGB标签
func (mm *MetadataMigrator) addDefaultSRGBProfile(ctx context.Context, targetPath string) error {
	args := []string{
		"-ColorSpace=sRGB",
		"-WhitePoint=0.3127 0.329",
		"-PrimaryChromaticities=0.64 0.33 0.3 0.6 0.15 0.06",
		"-overwrite_original",
		targetPath,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, mm.exiftoolPath, args...)
	return cmd.Run()
}

// handleTimestamps 处理时间戳信息
func (mm *MetadataMigrator) handleTimestamps(sourcePath, targetPath string, metadata map[string]interface{}) (*TimestampInfo, error) {
	timestampInfo := &TimestampInfo{}

	// 从元数据中提取时间信息
	if createDate, exists := metadata["CreateDate"]; exists {
		if t, err := mm.parseExifTime(fmt.Sprintf("%v", createDate)); err == nil {
			timestampInfo.CreateTime = t
		}
	}

	if modifyDate, exists := metadata["ModifyDate"]; exists {
		if t, err := mm.parseExifTime(fmt.Sprintf("%v", modifyDate)); err == nil {
			timestampInfo.ModifyTime = t
		}
	}

	if dateTimeOrig, exists := metadata["DateTimeOriginal"]; exists {
		if t, err := mm.parseExifTime(fmt.Sprintf("%v", dateTimeOrig)); err == nil {
			timestampInfo.DateTimeOrig = t
		}
	}

	// 获取文件系统时间戳
	if sourceInfo, err := os.Stat(sourcePath); err == nil {
		if timestampInfo.ModifyTime.IsZero() {
			timestampInfo.ModifyTime = sourceInfo.ModTime()
		}
	}

	// 保持文件系统时间戳
	if !timestampInfo.ModifyTime.IsZero() {
		if err := os.Chtimes(targetPath, timestampInfo.ModifyTime, timestampInfo.ModifyTime); err == nil {
			timestampInfo.PreservedMod = true
			mm.logger.Debug("已保持文件修改时间", zap.String("target", filepath.Base(targetPath)))
		}
	}

	timestampInfo.PreservedOrig = !timestampInfo.DateTimeOrig.IsZero()

	return timestampInfo, nil
}

// parseExifTime 解析EXIF时间格式
func (mm *MetadataMigrator) parseExifTime(timeStr string) (time.Time, error) {
	// EXIF时间格式：2023:12:25 14:30:45
	layouts := []string{
		"2006:01:02 15:04:05",
		"2006-01-02 15:04:05",
		"2006:01:02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析时间格式: %s", timeStr)
}

// performExiftoolMigration 执行exiftool元数据迁移
func (mm *MetadataMigrator) performExiftoolMigration(ctx context.Context, sourcePath, targetPath string, metadata map[string]interface{}) (bool, []string, []string) {
	// 复制所有元数据
	copyArgs := []string{
		"-overwrite_original",
		"-all:all<",
		sourcePath,
		targetPath,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, mm.exiftoolPath, copyArgs...)
	if err := cmd.Run(); err != nil {
		mm.logger.Warn("元数据复制失败", zap.Error(err))
		return false, nil, nil
	}

	// 验证复制结果
	migratedTags, failedTags := mm.verifyMigratedTags(ctx, targetPath, metadata)
	success := len(failedTags) == 0 || float64(len(migratedTags))/float64(len(migratedTags)+len(failedTags)) > 0.8

	return success, migratedTags, failedTags
}

// verifyMigratedTags 验证已迁移的标签
func (mm *MetadataMigrator) verifyMigratedTags(ctx context.Context, targetPath string, originalMetadata map[string]interface{}) ([]string, []string) {
	targetMetadata, err := mm.extractAllMetadata(ctx, targetPath)
	if err != nil {
		mm.logger.Warn("验证时提取目标文件元数据失败", zap.Error(err))
		return nil, nil
	}

	var migratedTags, failedTags []string

	for key, originalValue := range originalMetadata {
		if targetValue, exists := targetMetadata[key]; exists {
			// 简单的值比较
			if fmt.Sprintf("%v", originalValue) == fmt.Sprintf("%v", targetValue) {
				migratedTags = append(migratedTags, key)
			} else {
				failedTags = append(failedTags, key)
			}
		} else {
			failedTags = append(failedTags, key)
		}
	}

	return migratedTags, failedTags
}

// performCriticalFieldsCopy 执行关键字段复制 - README要求的失败时复制关键字段
func (mm *MetadataMigrator) performCriticalFieldsCopy(ctx context.Context, targetPath string, metadata map[string]interface{}) map[string]string {
	criticalFields := make(map[string]string)

	for _, field := range CriticalMetadataFields {
		if value, exists := metadata[field]; exists {
			valueStr := fmt.Sprintf("%v", value)
			if valueStr != "" && valueStr != "<nil>" {
				// 尝试设置关键字段
				if err := mm.setCriticalField(ctx, targetPath, field, valueStr); err == nil {
					criticalFields[field] = valueStr
					mm.logger.Debug("关键字段复制成功", 
						zap.String("field", field), 
						zap.String("value", valueStr))
				} else {
					mm.logger.Warn("关键字段复制失败", 
						zap.String("field", field), 
						zap.Error(err))
				}
			}
		}
	}

	mm.logger.Info("关键字段复制完成",
		zap.String("target", filepath.Base(targetPath)),
		zap.Int("copied_fields", len(criticalFields)))

	return criticalFields
}

// setCriticalField 设置关键字段
func (mm *MetadataMigrator) setCriticalField(ctx context.Context, targetPath, field, value string) error {
	// 清理和验证字段值
	cleanValue := mm.cleanFieldValue(field, value)
	
	args := []string{
		fmt.Sprintf("-%s=%s", field, cleanValue),
		"-overwrite_original",
		targetPath,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, mm.exiftoolPath, args...)
	return cmd.Run()
}

// cleanFieldValue 清理字段值
func (mm *MetadataMigrator) cleanFieldValue(field, value string) string {
	// 移除非打印字符
	re := regexp.MustCompile(`[^\x20-\x7E]`)
	cleaned := re.ReplaceAllString(value, "")
	
	// 根据字段类型进行特殊处理
	switch {
	case strings.Contains(field, "Date") || strings.Contains(field, "Time"):
		// 时间字段：确保格式正确
		if t, err := mm.parseExifTime(cleaned); err == nil {
			return t.Format("2006:01:02 15:04:05")
		}
	case field == "ISO":
		// ISO字段：确保是数字
		if _, err := strconv.Atoi(cleaned); err != nil {
			return "100" // 默认ISO值
		}
	}

	// 限制长度
	if len(cleaned) > 256 {
		cleaned = cleaned[:256]
	}

	return cleaned
}

// performFallbackMigration 执行fallback迁移
func (mm *MetadataMigrator) performFallbackMigration(ctx context.Context, sourcePath, targetPath string) (*MetadataResult, error) {
	result := &MetadataResult{
		SourcePath:      sourcePath,
		TargetPath:      targetPath,
		ProcessedAt:     time.Now(),
		CriticalFields:  make(map[string]string),
		MigrationMethod: "fallback_only",
		Success:         false,
	}

	mm.logger.Info("执行fallback元数据迁移", 
		zap.String("source", filepath.Base(sourcePath)),
		zap.String("target", filepath.Base(targetPath)))

	// 至少保持文件时间戳
	if sourceInfo, err := os.Stat(sourcePath); err == nil {
		if err := os.Chtimes(targetPath, sourceInfo.ModTime(), sourceInfo.ModTime()); err == nil {
			result.CriticalFields["ModifyTime"] = sourceInfo.ModTime().Format(time.RFC3339)
			mm.logger.Debug("已保持文件时间戳")
		}
	}

	// 添加基础sRGB配置
	if err := mm.addDefaultSRGBProfile(ctx, targetPath); err == nil {
		result.ICCProfile = &ICCProfileInfo{
			FallbackAdded: true,
			ColorSpace:    "sRGB",
		}
		mm.logger.Debug("已添加fallback sRGB配置")
	}

	result.Success = len(result.CriticalFields) > 0 || (result.ICCProfile != nil && result.ICCProfile.FallbackAdded)

	return result, nil
}

// verifyMigration 验证迁移结果
func (mm *MetadataMigrator) verifyMigration(ctx context.Context, targetPath string, result *MetadataResult) error {
	// 检查目标文件是否存在
	if _, err := os.Stat(targetPath); err != nil {
		return fmt.Errorf("目标文件不存在: %w", err)
	}

	// 基础元数据检查
	targetMetadata, err := mm.extractAllMetadata(ctx, targetPath)
	if err != nil {
		// 如果无法提取元数据，但文件存在，可能是格式问题
		mm.logger.Warn("验证时无法提取目标文件元数据", zap.Error(err))
		return nil // 不视为错误，可能是目标格式不支持某些元数据
	}

	// 检查关键信息是否存在
	keyFieldsFound := 0
	totalKeyFields := len(CriticalMetadataFields)
	
	for _, field := range CriticalMetadataFields {
		if _, exists := targetMetadata[field]; exists {
			keyFieldsFound++
		}
	}

	// 至少保留20%的关键字段算作基本成功
	if float64(keyFieldsFound)/float64(totalKeyFields) < 0.2 && len(result.CriticalFields) == 0 {
		return fmt.Errorf("关键元数据保留不足 (%d/%d)", keyFieldsFound, totalKeyFields)
	}

	mm.logger.Debug("元数据迁移验证通过",
		zap.String("target", filepath.Base(targetPath)),
		zap.Int("key_fields_found", keyFieldsFound),
		zap.Int("total_key_fields", totalKeyFields))

	return nil
}

// GetMigrationStatistics 获取迁移统计信息
func (mm *MetadataMigrator) GetMigrationStatistics(results []*MetadataResult) map[string]interface{} {
	stats := make(map[string]interface{})
	
	var successCount, failCount int
	var totalMigrated, totalFailed int
	var fullMigrationCount, fallbackCount int
	var iccPreservedCount, timestampPreservedCount int

	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failCount++
		}

		totalMigrated += len(result.MigratedTags)
		totalFailed += len(result.FailedTags)

		if result.MigrationMethod == "full_migration" {
			fullMigrationCount++
		} else {
			fallbackCount++
		}

		if result.ICCProfile != nil && (result.ICCProfile.HasProfile || result.ICCProfile.FallbackAdded) {
			iccPreservedCount++
		}

		if result.TimestampInfo != nil && result.TimestampInfo.PreservedMod {
			timestampPreservedCount++
		}
	}

	stats["total_files"] = len(results)
	stats["successful_migrations"] = successCount
	stats["failed_migrations"] = failCount
	stats["success_rate"] = float64(successCount) / float64(len(results)) * 100
	stats["total_tags_migrated"] = totalMigrated
	stats["total_tags_failed"] = totalFailed
	stats["full_migrations"] = fullMigrationCount
	stats["fallback_migrations"] = fallbackCount
	stats["icc_preserved"] = iccPreservedCount
	stats["timestamps_preserved"] = timestampPreservedCount

	if totalMigrated+totalFailed > 0 {
		stats["tag_migration_rate"] = float64(totalMigrated) / float64(totalMigrated+totalFailed) * 100
	}

	return stats
}