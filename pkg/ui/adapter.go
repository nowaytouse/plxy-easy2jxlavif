package ui

import (
	"context"
	"fmt"

	"pixly/pkg/core/config"
	"pixly/pkg/core/types"
	"pixly/pkg/security"
	"pixly/pkg/ui/progress"

	"go.uber.org/zap"
)

// Adapter UI适配器 - 提供与现有系统的向后兼容接口
type Adapter struct {
	logger      *zap.Logger
	flowManager *FlowManager
	uiManager   *Manager
}

// NewAdapter 创建新的UI适配器
func NewAdapter(logger *zap.Logger, securityChecker *security.SecurityChecker) *Adapter {
	uiManager := NewManager(logger, true) // 启用颜色
	flowManager := NewFlowManager(logger, uiManager, securityChecker)

	return &Adapter{
		logger:      logger,
		flowManager: flowManager,
		uiManager:   uiManager,
	}
}

// StartInteractiveSession 启动交互式会话（向后兼容）
func (a *Adapter) StartInteractiveSession(ctx context.Context, tools types.ToolCheckResults) (*config.Config, error) {
	a.logger.Info("启动交互式UI会话")

	// 配置流程
	flowConfig := &FlowConfig{
		EnableSecurityCheck:   true,
		EnableCacheManagement: false,
		EnableInstructions:    true,
		AllowResumeSession:    true,
	}

	// 启动交互式流程
	result, err := a.flowManager.StartInteractiveFlow(tools, flowConfig)
	if err != nil {
		return nil, fmt.Errorf("交互式流程失败: %w", err)
	}

	// 如果用户选择退出
	if result.Action == "exit" {
		return nil, nil // 返回nil表示用户退出
	}

	// 如果用户选择转换
	if result.Action == "convert" && result.Config != nil {
		a.logger.Info("交互式会话完成",
			zap.String("mode", result.Config.Mode),
			zap.String("target_dir", result.Config.TargetDir))
		return result.Config, nil
	}

	return nil, fmt.Errorf("未知的会话结果")
}

// GetUIManager 获取UI管理器
func (a *Adapter) GetUIManager() *Manager {
	return a.uiManager
}

// StartNonInteractiveSession 启动非交互式会话（向后兼容）
func (a *Adapter) StartNonInteractiveSession(ctx context.Context, cfg *config.Config, tools types.ToolCheckResults) error {
	a.logger.Info("启动非交互式UI会话")

	// 启动非交互式流程
	result, err := a.flowManager.StartNonInteractiveFlow(cfg, tools)
	if err != nil {
		return fmt.Errorf("非交互式流程失败: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("非交互式流程未成功完成")
	}

	a.logger.Info("非交互式会话完成",
		zap.String("mode", result.Mode.String()),
		zap.String("target_dir", result.TargetDir))

	return nil
}

// ShowWelcome 显示欢迎界面（向后兼容）
func (a *Adapter) ShowWelcome() {
	a.uiManager.ShowWelcome()
}

// ShowSystemInfo 显示系统信息（向后兼容）
func (a *Adapter) ShowSystemInfo(tools types.ToolCheckResults) {
	a.flowManager.ShowSystemInfo(tools)
}

// GetProgressManager 获取进度管理器（向后兼容）
func (a *Adapter) GetProgressManager() *progress.ProgressManager {
	return a.uiManager.GetProgressManager()
}

// ShowInfo 显示信息（向后兼容）
func (a *Adapter) ShowInfo(message string) {
	a.uiManager.showInfo(message)
}

// ShowSuccess 显示成功消息（向后兼容）
func (a *Adapter) ShowSuccess(message string) {
	a.uiManager.showSuccess(message)
}

// ShowWarning 显示警告（向后兼容）
func (a *Adapter) ShowWarning(message string) {
	a.uiManager.showWarning(message)
}

// ShowError 显示错误（向后兼容）
func (a *Adapter) ShowError(message string) {
	a.uiManager.showError(message)
}

// PauseProgress 暂停进度显示（向后兼容）
func (a *Adapter) PauseProgress() {
	a.uiManager.PauseProgress()
}

// ResumeProgress 恢复进度显示（向后兼容）
func (a *Adapter) ResumeProgress() {
	a.uiManager.ResumeProgress()
}

// Shutdown 关闭UI适配器
func (a *Adapter) Shutdown() error {
	a.logger.Info("关闭UI适配器")

	var lastErr error

	// 关闭流程管理器
	if err := a.flowManager.Shutdown(); err != nil {
		lastErr = err
		a.logger.Error("关闭流程管理器失败", zap.Error(err))
	}

	// 关闭UI管理器
	if err := a.uiManager.Shutdown(); err != nil {
		lastErr = err
		a.logger.Error("关闭UI管理器失败", zap.Error(err))
	}

	a.logger.Info("UI适配器已关闭")
	return lastErr
}

// GetRecommendedConfig 获取推荐配置（向后兼容）
func (a *Adapter) GetRecommendedConfig(targetDir string, scenario string) *config.Config {
	return a.flowManager.GetRecommendedConfig(targetDir, scenario)
}

// IsInteractive 检查是否为交互模式（向后兼容）
func (a *Adapter) IsInteractive() bool {
	return a.flowManager.IsInteractive()
}

// CreateDefaultConfig 创建默认配置（新增便利方法）
func (a *Adapter) CreateDefaultConfig(targetDir string) *config.Config {
	return &config.Config{
		TargetDir:           targetDir,
		Mode:                "auto+",
		ConcurrentJobs:      4,
		EnableBackups:       true,
		HwAccel:             true,
		MaxRetries:          2,
		LogLevel:            "info",
		SortOrder:           "size",
		CRF:                 28,
		StickerTargetFormat: "avif",
	}
}

// ValidateUserConfig 验证用户配置（新增便利方法）
func (a *Adapter) ValidateUserConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}

	if cfg.TargetDir == "" {
		return fmt.Errorf("目标目录不能为空")
	}

	// 使用新的标准化和验证函数
	return config.ValidateAndNormalize(cfg)
}

// 以下是全局便利函数，提供向后兼容性

var globalUIAdapter *Adapter

// SetGlobalUIAdapter 设置全局UI适配器
func SetGlobalUIAdapter(adapter *Adapter) {
	globalUIAdapter = adapter
}

// GetGlobalUIAdapter 获取全局UI适配器
func GetGlobalUIAdapter() *Adapter {
	return globalUIAdapter
}

// InitializeGlobalUI 初始化全局UI系统
func InitializeGlobalUI(logger *zap.Logger, securityChecker *security.SecurityChecker) *Adapter {
	adapter := NewAdapter(logger, securityChecker)
	SetGlobalUIAdapter(adapter)
	return adapter
}

// ShutdownGlobalUI 关闭全局UI系统
func ShutdownGlobalUI() error {
	if globalUIAdapter != nil {
		return globalUIAdapter.Shutdown()
	}
	return nil
}

// 全局便利函数（向后兼容旧代码）

// PauseAllProgressDisplays 暂停所有进度显示（全局函数）
func PauseAllProgressDisplays() {
	if globalUIAdapter != nil {
		globalUIAdapter.PauseProgress()
	}
}

// ResumeAllProgressDisplays 恢复所有进度显示（全局函数）
func ResumeAllProgressDisplays() {
	if globalUIAdapter != nil {
		globalUIAdapter.ResumeProgress()
	}
}

// ShowGlobalInfo 显示全局信息（全局函数）
func ShowGlobalInfo(message string) {
	if globalUIAdapter != nil {
		globalUIAdapter.ShowInfo(message)
	}
}

// ShowGlobalSuccess 显示全局成功消息（全局函数）
func ShowGlobalSuccess(message string) {
	if globalUIAdapter != nil {
		globalUIAdapter.ShowSuccess(message)
	}
}

// ShowGlobalWarning 显示全局警告（全局函数）
func ShowGlobalWarning(message string) {
	if globalUIAdapter != nil {
		globalUIAdapter.ShowWarning(message)
	}
}

// ShowGlobalError 显示全局错误（全局函数）
func ShowGlobalError(message string) {
	if globalUIAdapter != nil {
		globalUIAdapter.ShowError(message)
	}
}
