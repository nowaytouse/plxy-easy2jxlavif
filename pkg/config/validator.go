package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// Validator validates configuration
type Validator struct {
	config   *Config
	errors   []error
	warnings []string
}

// NewValidator creates a new configuration validator
func NewValidator(config *Config) *Validator {
	return &Validator{
		config:   config,
		errors:   []error{},
		warnings: []string{},
	}
}

// Validate performs full validation of the configuration
func (v *Validator) Validate() error {
	// 验证各个部分
	v.validateConcurrency()
	v.validateConversion()
	v.validateSecurity()
	v.validatePaths()
	v.validateTools()
	v.validateUI()
	v.validateLogging()

	// 如果有错误，返回组合错误
	if len(v.errors) > 0 {
		return fmt.Errorf("配置验证失败，发现 %d 个错误:\n%s",
			len(v.errors), v.formatErrors())
	}

	// 打印警告（如果有）
	if len(v.warnings) > 0 {
		fmt.Printf("⚠️  配置验证警告 (%d个):\n", len(v.warnings))
		for _, warning := range v.warnings {
			fmt.Printf("  ⚠️  %s\n", warning)
		}
		fmt.Println()
	}

	return nil
}

// validateConcurrency validates concurrency settings
func (v *Validator) validateConcurrency() {
	if v.config.Concurrency.ConversionWorkers < 1 {
		v.errors = append(v.errors, fmt.Errorf("conversion_workers 必须 >= 1"))
	}
	if v.config.Concurrency.ConversionWorkers > 128 {
		v.warnings = append(v.warnings, "conversion_workers > 128 可能导致资源耗尽")
	}

	if v.config.Concurrency.ScanWorkers < 1 {
		v.errors = append(v.errors, fmt.Errorf("scan_workers 必须 >= 1"))
	}

	if v.config.Concurrency.MemoryLimitMB < 512 {
		v.warnings = append(v.warnings, "memory_limit_mb < 512MB 可能不足")
	}
}

// validateConversion validates conversion settings
func (v *Validator) validateConversion() {
	validModes := map[string]bool{
		"auto+": true,
		"auto":  true,
		"smart": true,
		"batch": true,
	}

	if !validModes[v.config.Conversion.DefaultMode] {
		v.errors = append(v.errors, fmt.Errorf("无效的 default_mode: %s", v.config.Conversion.DefaultMode))
	}

	// 验证置信度阈值
	if v.config.Conversion.Predictor.ConfidenceThreshold < 0 ||
		v.config.Conversion.Predictor.ConfidenceThreshold > 1 {
		v.errors = append(v.errors, fmt.Errorf("confidence_threshold 必须在 0-1 之间"))
	}

	// 验证effort值
	if v.config.Conversion.Formats.PNG.Effort < 1 || v.config.Conversion.Formats.PNG.Effort > 9 {
		v.errors = append(v.errors, fmt.Errorf("PNG effort 必须在 1-9 之间"))
	}

	if v.config.Conversion.Formats.JPEG.Effort < 1 || v.config.Conversion.Formats.JPEG.Effort > 9 {
		v.errors = append(v.errors, fmt.Errorf("JPEG effort 必须在 1-9 之间"))
	}

	// 验证GIF设置
	if v.config.Conversion.Formats.GIF.AnimatedCRF < 0 || v.config.Conversion.Formats.GIF.AnimatedCRF > 63 {
		v.errors = append(v.errors, fmt.Errorf("GIF animated_crf 必须在 0-63 之间"))
	}

	if v.config.Conversion.Formats.GIF.AnimatedSpeed < 0 || v.config.Conversion.Formats.GIF.AnimatedSpeed > 10 {
		v.errors = append(v.errors, fmt.Errorf("GIF animated_speed 必须在 0-10 之间"))
	}
}

// validateSecurity validates security settings
func (v *Validator) validateSecurity() {
	if v.config.Security.MinFreeSpaceMB < 0 {
		v.errors = append(v.errors, fmt.Errorf("min_free_space_mb 必须 >= 0"))
	}

	if v.config.Security.MinFreeSpaceMB < 100 {
		v.warnings = append(v.warnings, "min_free_space_mb < 100MB 可能不足")
	}

	if v.config.Security.MaxFileSizeMB < 1 {
		v.errors = append(v.errors, fmt.Errorf("max_file_size_mb 必须 >= 1"))
	}

	// 验证禁止目录
	if len(v.config.Security.ForbiddenDirectories) == 0 && v.config.Security.EnablePathCheck {
		v.warnings = append(v.warnings, "启用了路径检查但未设置禁止目录")
	}
}

// validatePaths validates file paths
func (v *Validator) validatePaths() {
	// 验证日志路径
	if v.config.Logging.FilePath != "" {
		logDir := filepath.Dir(v.config.Logging.FilePath)
		if err := v.ensureDirectoryWritable(logDir); err != nil {
			v.errors = append(v.errors, fmt.Errorf("日志目录不可写: %s - %v", logDir, err))
		}
	}

	// 验证知识库路径
	if v.config.Knowledge.Enable && v.config.Knowledge.DBPath != "" {
		dbDir := filepath.Dir(v.config.Knowledge.DBPath)
		if err := v.ensureDirectoryWritable(dbDir); err != nil {
			v.errors = append(v.errors, fmt.Errorf("知识库目录不可写: %s - %v", dbDir, err))
		}
	}
}

// validateTools validates tool paths and availability
func (v *Validator) validateTools() {
	if !v.config.Tools.AutoDetect {
		// 如果禁用自动检测，检查手动指定的路径
		tools := map[string]string{
			"cjxl":     v.config.Tools.CJXLPath,
			"djxl":     v.config.Tools.DJXLPath,
			"avifenc":  v.config.Tools.AVIFEncPath,
			"avifdec":  v.config.Tools.AVIFDecPath,
			"ffmpeg":   v.config.Tools.FFmpegPath,
			"ffprobe":  v.config.Tools.FFprobePath,
			"exiftool": v.config.Tools.ExifToolPath,
		}

		for name, path := range tools {
			if path != "" {
				if _, err := os.Stat(path); os.IsNotExist(err) {
					v.warnings = append(v.warnings, fmt.Sprintf("工具路径不存在: %s - %s", name, path))
				}
			}
		}
	} else {
		// 自动检测模式，验证关键工具是否可用
		requiredTools := []string{"cjxl", "ffmpeg", "ffprobe"}
		for _, tool := range requiredTools {
			if _, err := exec.LookPath(tool); err != nil {
				v.warnings = append(v.warnings, fmt.Sprintf("未找到必需工具: %s", tool))
			}
		}
	}
}

// validateUI validates UI settings
func (v *Validator) validateUI() {
	validModes := map[string]bool{
		"interactive":     true,
		"non-interactive": true,
		"silent":          true,
	}

	if !validModes[v.config.UI.Mode] {
		v.errors = append(v.errors, fmt.Errorf("无效的 UI mode: %s", v.config.UI.Mode))
	}

	validThemes := map[string]bool{
		"dark":  true,
		"light": true,
		"auto":  true,
	}

	if !validThemes[v.config.UI.Theme] {
		v.errors = append(v.errors, fmt.Errorf("无效的 UI theme: %s", v.config.UI.Theme))
	}

	// 验证监控面板刷新间隔
	if v.config.UI.MonitorPanel.RefreshIntervalS < 1 {
		v.errors = append(v.errors, fmt.Errorf("monitor_panel.refresh_interval_s 必须 >= 1"))
	}

	if v.config.UI.MonitorPanel.RefreshIntervalS > 60 {
		v.warnings = append(v.warnings, "monitor_panel.refresh_interval_s > 60秒 刷新过慢")
	}
}

// validateLogging validates logging settings
func (v *Validator) validateLogging() {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLevels[v.config.Logging.Level] {
		v.errors = append(v.errors, fmt.Errorf("无效的日志级别: %s", v.config.Logging.Level))
	}

	validOutputs := map[string]bool{
		"console": true,
		"file":    true,
		"both":    true,
	}

	if !validOutputs[v.config.Logging.Output] {
		v.errors = append(v.errors, fmt.Errorf("无效的日志输出: %s", v.config.Logging.Output))
	}

	if v.config.Logging.MaxSizeMB < 1 {
		v.errors = append(v.errors, fmt.Errorf("logging.max_size_mb 必须 >= 1"))
	}
}

// ensureDirectoryWritable checks if a directory exists and is writable
func (v *Validator) ensureDirectoryWritable(dir string) error {
	// 尝试创建目录
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("无法创建目录: %w", err)
	}

	// 检查写权限
	testFile := filepath.Join(dir, ".pixly_write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("无写权限: %w", err)
	}
	os.Remove(testFile)

	return nil
}

// formatErrors formats multiple errors into a readable string
func (v *Validator) formatErrors() string {
	var sb strings.Builder
	for i, err := range v.errors {
		sb.WriteString(fmt.Sprintf("  %d. %v\n", i+1, err))
	}
	return sb.String()
}

// GetDiskSpace returns available disk space in MB
func GetDiskSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}

	// Available blocks * size per block / 1024 / 1024 = MB
	availableMB := uint64(stat.Bavail) * uint64(stat.Bsize) / 1024 / 1024
	return availableMB, nil
}

// ValidateDiskSpace checks if there is enough free disk space
func ValidateDiskSpace(path string, minMB uint64) error {
	available, err := GetDiskSpace(path)
	if err != nil {
		return fmt.Errorf("无法检查磁盘空间: %w", err)
	}

	if available < minMB {
		return fmt.Errorf("磁盘空间不足: 需要 %d MB, 可用 %d MB", minMB, available)
	}

	return nil
}
