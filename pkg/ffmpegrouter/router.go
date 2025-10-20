package ffmpegrouter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FFmpegRouter FFmpeg智能路由器 - README要求的版本管理和回退机制
type FFmpegRouter struct {
	logger          *zap.Logger
	versions        map[string]*FFmpegVersion
	versionMutex    sync.RWMutex
	defaultVersion  string
	fallbackVersion string
	config          *RouterConfig
	statistics      *RouterStatistics
}

// FFmpegVersion FFmpeg版本信息
type FFmpegVersion struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Path             string            `json:"path"`
	Version          string            `json:"version"`
	Type             VersionType       `json:"type"`
	Status           VersionStatus     `json:"status"`
	SupportedFormats map[string]bool   `json:"supported_formats"`
	HealthScore      int               `json:"health_score"`
	LastChecked      time.Time         `json:"last_checked"`
	FailureCount     int               `json:"failure_count"`
	SuccessCount     int               `json:"success_count"`
	Metadata         map[string]string `json:"metadata"`
}

// RouterConfig 路由配置
type RouterConfig struct {
	PreferSystemVersion    bool          `json:"prefer_system_version"`
	EnableEmbeddedFallback bool          `json:"enable_embedded_fallback"`
	EnableDevelopmentMode  bool          `json:"enable_development_mode"`
	HealthCheckInterval    time.Duration `json:"health_check_interval"`
	MaxFailureCount        int           `json:"max_failure_count"`
	SystemSearchPaths      []string      `json:"system_search_paths"`
	EmbeddedBasePath       string        `json:"embedded_base_path"`
}

// RouterStatistics 路由统计
type RouterStatistics struct {
	TotalRequests      int64            `json:"total_requests"`
	SuccessfulRequests int64            `json:"successful_requests"`
	FailedRequests     int64            `json:"failed_requests"`
	FallbackUsed       int64            `json:"fallback_used"`
	VersionUsage       map[string]int64 `json:"version_usage"`
	LastUpdated        time.Time        `json:"last_updated"`
}

// 枚举定义
type VersionType int
type VersionStatus int

const (
	VersionTypeSystem VersionType = iota
	VersionTypeEmbedded
	VersionTypeDevelopment
)

const (
	StatusUnknown VersionStatus = iota
	StatusAvailable
	StatusUnavailable
	StatusFailed
)

// NewFFmpegRouter 创建FFmpeg智能路由器
func NewFFmpegRouter(logger *zap.Logger, config *RouterConfig) (*FFmpegRouter, error) {
	if config == nil {
		config = &RouterConfig{
			PreferSystemVersion:    true,
			EnableEmbeddedFallback: true,
			EnableDevelopmentMode:  false,
			HealthCheckInterval:    5 * time.Minute,
			MaxFailureCount:        3,
			SystemSearchPaths:      getDefaultSearchPaths(),
			EmbeddedBasePath:       "./embedded/ffmpeg",
		}
	}

	router := &FFmpegRouter{
		logger:   logger,
		versions: make(map[string]*FFmpegVersion),
		config:   config,
		statistics: &RouterStatistics{
			VersionUsage: make(map[string]int64),
		},
	}

	// 发现和注册FFmpeg版本
	if err := router.discoverVersions(); err != nil {
		return nil, fmt.Errorf("发现FFmpeg版本失败: %w", err)
	}

	logger.Info("FFmpeg智能路由器初始化完成",
		zap.Int("discovered_versions", len(router.versions)),
		zap.String("default_version", router.defaultVersion),
		zap.String("fallback_version", router.fallbackVersion))

	return router, nil
}

// discoverVersions 发现FFmpeg版本 - README核心功能
func (router *FFmpegRouter) discoverVersions() error {
	router.logger.Info("开始发现FFmpeg版本")

	// 1. 搜索系统版本 - README要求：优先使用系统版本
	if router.config.PreferSystemVersion {
		router.discoverSystemVersions()
	}

	// 2. 注册内嵌版本 - README要求：内嵌版本回退机制
	if router.config.EnableEmbeddedFallback {
		router.registerEmbeddedVersions()
	}

	// 3. 注册开发版本 - README要求：开发版支持实验性格式
	if router.config.EnableDevelopmentMode {
		router.registerDevelopmentVersions()
	}

	// 4. 选择默认和回退版本
	router.selectDefaultVersions()

	if len(router.versions) == 0 {
		return fmt.Errorf("未发现任何可用的FFmpeg版本")
	}

	router.logger.Info("FFmpeg版本发现完成",
		zap.Int("total_versions", len(router.versions)),
		zap.String("default", router.defaultVersion),
		zap.String("fallback", router.fallbackVersion))

	return nil
}

// GetBestVersion 获取最佳版本 - README核心智能路由功能
func (router *FFmpegRouter) GetBestVersion(ctx context.Context, taskType string, inputFormat string, outputFormat string) (*FFmpegVersion, error) {
	router.versionMutex.RLock()
	defer router.versionMutex.RUnlock()

	router.logger.Debug("选择最佳FFmpeg版本",
		zap.String("task_type", taskType),
		zap.String("input_format", inputFormat),
		zap.String("output_format", outputFormat))

	// 1. 首先尝试默认版本
	if router.defaultVersion != "" {
		if version, exists := router.versions[router.defaultVersion]; exists && version.Status == StatusAvailable {
			if router.isVersionSuitable(version, inputFormat, outputFormat) {
				router.updateStatistics(version.ID)
				return version, nil
			}
		}
	}

	// 2. 遍历所有可用版本寻找最适合的
	var bestVersion *FFmpegVersion
	var bestScore int

	for _, version := range router.versions {
		if version.Status != StatusAvailable {
			continue
		}

		score := router.calculateVersionScore(version, taskType, inputFormat, outputFormat)
		if score > bestScore {
			bestScore = score
			bestVersion = version
		}
	}

	if bestVersion != nil {
		router.updateStatistics(bestVersion.ID)
		router.logger.Info("选择FFmpeg版本",
			zap.String("selected_version", bestVersion.ID),
			zap.String("version_name", bestVersion.Name),
			zap.Int("score", bestScore))
		return bestVersion, nil
	}

	// 3. 使用回退版本 - README要求：回退机制
	return router.getFallbackVersion()
}

// ExecuteCommand 执行FFmpeg命令 - README核心功能
func (router *FFmpegRouter) ExecuteCommand(ctx context.Context, taskType string, args []string, inputFormat string, outputFormat string) (*exec.Cmd, error) {
	// 获取最佳版本
	version, err := router.GetBestVersion(ctx, taskType, inputFormat, outputFormat)
	if err != nil {
		return nil, fmt.Errorf("选择FFmpeg版本失败: %w", err)
	}

	// 检查版本可用性
	if version.Status != StatusAvailable {
		router.logger.Warn("选择的版本不可用，尝试回退",
			zap.String("version", version.ID),
			zap.String("status", version.Status.String()))

		fallbackVersion, fallbackErr := router.getFallbackVersion()
		if fallbackErr != nil {
			return nil, fmt.Errorf("回退版本也不可用: %w", fallbackErr)
		}
		version = fallbackVersion
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, version.Path, args...)

	router.logger.Debug("执行FFmpeg命令",
		zap.String("version", version.ID),
		zap.String("path", version.Path),
		zap.Strings("args", args))

	return cmd, nil
}

// 版本发现实现
func (router *FFmpegRouter) discoverSystemVersions() {
	// 在系统PATH中搜索ffmpeg
	if systemPath, err := exec.LookPath("ffmpeg"); err == nil {
		if version := router.analyzeVersion("system_default", systemPath, VersionTypeSystem); version != nil {
			router.versions[version.ID] = version
			router.logger.Info("发现系统FFmpeg版本",
				zap.String("path", systemPath),
				zap.String("version", version.Version))
		}
	}

	// 在指定路径中搜索
	for _, searchPath := range router.config.SystemSearchPaths {
		ffmpegPath := filepath.Join(searchPath, "ffmpeg")
		if runtime.GOOS == "windows" {
			ffmpegPath += ".exe"
		}

		if _, err := os.Stat(ffmpegPath); err == nil {
			if version := router.analyzeVersion("system_"+filepath.Base(searchPath), ffmpegPath, VersionTypeSystem); version != nil {
				router.versions[version.ID] = version
				router.logger.Info("发现系统FFmpeg版本",
					zap.String("path", ffmpegPath))
			}
		}
	}
}

func (router *FFmpegRouter) registerEmbeddedVersions() {
	// 注册内嵌的稳定版本
	embeddedPath := filepath.Join(router.config.EmbeddedBasePath, runtime.GOOS, "ffmpeg")
	if runtime.GOOS == "windows" {
		embeddedPath += ".exe"
	}

	if _, err := os.Stat(embeddedPath); err == nil {
		if version := router.analyzeVersion("embedded_stable", embeddedPath, VersionTypeEmbedded); version != nil {
			router.versions[version.ID] = version
			router.logger.Info("注册内嵌FFmpeg版本",
				zap.String("path", embeddedPath))
		}
	}
}

func (router *FFmpegRouter) registerDevelopmentVersions() {
	// 注册开发版本（用于实验性格式）
	devPath := filepath.Join(router.config.EmbeddedBasePath, "dev", runtime.GOOS, "ffmpeg")
	if runtime.GOOS == "windows" {
		devPath += ".exe"
	}

	if _, err := os.Stat(devPath); err == nil {
		if version := router.analyzeVersion("embedded_dev", devPath, VersionTypeDevelopment); version != nil {
			version.Metadata["experimental"] = "true"
			router.versions[version.ID] = version
			router.logger.Info("注册开发FFmpeg版本",
				zap.String("path", devPath))
		}
	}
}

func (router *FFmpegRouter) analyzeVersion(id, path string, versionType VersionType) *FFmpegVersion {
	// 获取版本信息
	versionOutput := router.getVersionInfo(path)
	if versionOutput == "" {
		return nil
	}

	// 获取支持的格式
	formats := router.getSupportedFormats(path)

	version := &FFmpegVersion{
		ID:               id,
		Name:             fmt.Sprintf("FFmpeg %s", versionOutput),
		Path:             path,
		Version:          versionOutput,
		Type:             versionType,
		Status:           StatusAvailable,
		SupportedFormats: formats,
		HealthScore:      100,
		LastChecked:      time.Now(),
		FailureCount:     0,
		SuccessCount:     0,
		Metadata:         make(map[string]string),
	}

	return version
}

func (router *FFmpegRouter) getVersionInfo(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "-version")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// 解析版本号
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		firstLine := lines[0]
		parts := strings.Fields(firstLine)
		if len(parts) >= 3 {
			return parts[2]
		}
	}

	return "unknown"
}

func (router *FFmpegRouter) getSupportedFormats(path string) map[string]bool {
	formats := make(map[string]bool)

	// 获取编码器列表
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "-encoders")
	output, err := cmd.Output()
	if err != nil {
		// 如果无法获取编码器列表，假设支持常见格式
		commonFormats := []string{"mp4", "avi", "mov", "webm", "mkv", "jpeg", "png", "webp"}
		for _, format := range commonFormats {
			formats[format] = true
		}
		return formats
	}

	// 解析编码器支持
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "V.....") { // 视频编码器
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				codec := parts[1]
				formats[codec] = true
			}
		}
	}

	return formats
}

func (router *FFmpegRouter) selectDefaultVersions() {
	// README要求：优先选择系统版本作为默认版本
	for id, version := range router.versions {
		if version.Type == VersionTypeSystem && version.Status == StatusAvailable {
			router.defaultVersion = id
			break
		}
	}

	// 如果没有系统版本，选择内嵌版本
	if router.defaultVersion == "" {
		for id, version := range router.versions {
			if version.Type == VersionTypeEmbedded && version.Status == StatusAvailable {
				router.defaultVersion = id
				break
			}
		}
	}

	// README要求：选择内嵌版本作为回退版本
	for id, version := range router.versions {
		if version.Type == VersionTypeEmbedded && version.Status == StatusAvailable {
			router.fallbackVersion = id
			break
		}
	}
}

// 版本选择逻辑
func (router *FFmpegRouter) isVersionSuitable(version *FFmpegVersion, inputFormat, outputFormat string) bool {
	// 检查是否支持输入和输出格式
	if inputFormat != "" && !version.SupportedFormats[inputFormat] {
		return false
	}
	if outputFormat != "" && !version.SupportedFormats[outputFormat] {
		return false
	}
	return true
}

func (router *FFmpegRouter) calculateVersionScore(version *FFmpegVersion, taskType, inputFormat, outputFormat string) int {
	score := version.HealthScore

	// 类型偏好分数
	switch version.Type {
	case VersionTypeSystem:
		score += 30 // README要求：优先系统版本
	case VersionTypeEmbedded:
		score += 20
	case VersionTypeDevelopment:
		score += 10
	}

	// 成功率分数
	if version.SuccessCount > 0 {
		successRate := float64(version.SuccessCount) / float64(version.SuccessCount+version.FailureCount)
		score += int(successRate * 20)
	}

	// 格式支持分数
	if router.isVersionSuitable(version, inputFormat, outputFormat) {
		score += 15
	}

	// 失败惩罚
	score -= version.FailureCount * 5

	return score
}

func (router *FFmpegRouter) getFallbackVersion() (*FFmpegVersion, error) {
	if router.fallbackVersion != "" {
		if version, exists := router.versions[router.fallbackVersion]; exists && version.Status == StatusAvailable {
			router.statistics.FallbackUsed++
			router.logger.Info("使用回退FFmpeg版本", zap.String("version", version.ID))
			return version, nil
		}
	}
	return nil, fmt.Errorf("回退版本不可用")
}

// 统计和错误处理
func (router *FFmpegRouter) updateStatistics(versionID string) {
	router.statistics.TotalRequests++
	router.statistics.VersionUsage[versionID]++
	router.statistics.LastUpdated = time.Now()
}

func (router *FFmpegRouter) HandleCommandResult(versionID string, success bool, err error) {
	router.versionMutex.Lock()
	defer router.versionMutex.Unlock()

	version, exists := router.versions[versionID]
	if !exists {
		return
	}

	if success {
		version.SuccessCount++
		router.statistics.SuccessfulRequests++

		// 重置失败计数
		if version.FailureCount > 0 {
			version.FailureCount = 0
			if version.Status == StatusFailed {
				version.Status = StatusAvailable
				router.logger.Info("FFmpeg版本恢复可用", zap.String("version", versionID))
			}
		}
	} else {
		version.FailureCount++
		router.statistics.FailedRequests++

		// 如果失败次数过多，标记为失败状态
		if version.FailureCount >= router.config.MaxFailureCount {
			version.Status = StatusFailed
			router.logger.Warn("FFmpeg版本标记为失败",
				zap.String("version", versionID),
				zap.Int("failure_count", version.FailureCount),
				zap.Error(err))
		}
	}
}

// 工具方法
func getDefaultSearchPaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/usr/local/bin",
			"/opt/homebrew/bin",
			"/usr/bin",
		}
	case "linux":
		return []string{
			"/usr/bin",
			"/usr/local/bin",
			"/snap/bin",
		}
	case "windows":
		return []string{
			"C:\\ffmpeg\\bin",
			"C:\\Program Files\\ffmpeg\\bin",
		}
	default:
		return []string{"/usr/bin", "/usr/local/bin"}
	}
}

func (s VersionStatus) String() string {
	switch s {
	case StatusAvailable:
		return "available"
	case StatusUnavailable:
		return "unavailable"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// GetVersions 获取所有版本信息
func (router *FFmpegRouter) GetVersions() map[string]*FFmpegVersion {
	router.versionMutex.RLock()
	defer router.versionMutex.RUnlock()

	// 返回版本副本
	versions := make(map[string]*FFmpegVersion)
	for id, version := range router.versions {
		versionCopy := *version
		versions[id] = &versionCopy
	}

	return versions
}

// GetStatistics 获取路由统计信息
func (router *FFmpegRouter) GetStatistics() *RouterStatistics {
	return router.statistics
}

// RefreshVersions 刷新版本状态
func (router *FFmpegRouter) RefreshVersions() error {
	router.logger.Info("刷新FFmpeg版本状态")

	router.versionMutex.Lock()
	defer router.versionMutex.Unlock()

	for id, version := range router.versions {
		// 简单的健康检查
		if router.getVersionInfo(version.Path) != "" {
			version.Status = StatusAvailable
			version.LastChecked = time.Now()
		} else {
			version.Status = StatusUnavailable
		}

		router.logger.Debug("版本状态更新",
			zap.String("version", id),
			zap.String("status", version.Status.String()))
	}

	return nil
}
