package ui

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"pixly/pkg/core/config"
	"pixly/pkg/core/types"
	"pixly/pkg/security"

	"go.uber.org/zap"
)

// FlowManager 用户交互流程管理器
type FlowManager struct {
	logger          *zap.Logger
	uiManager       *Manager
	securityChecker *security.SecurityChecker
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewFlowManager 创建新的流程管理器
func NewFlowManager(logger *zap.Logger, uiManager *Manager, securityChecker *security.SecurityChecker) *FlowManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &FlowManager{
		logger:          logger,
		uiManager:       uiManager,
		securityChecker: securityChecker,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// FlowConfig 流程配置
type FlowConfig struct {
	EnableSecurityCheck   bool
	EnableCacheManagement bool
	EnableInstructions    bool
	AllowResumeSession    bool
	AutoSelectMode        string // 自动选择模式 ("", "auto+", "quality", "sticker")
}

// FlowResult 流程结果
type FlowResult struct {
	Success     bool
	Action      string // "convert", "cache", "exit", "error"
	Config      *config.Config
	TargetDir   string
	Mode        types.AppMode
	Error       error
	ElapsedTime time.Duration
	UserChoices map[string]interface{} // 记录用户选择
}

// StartInteractiveFlow 启动交互式流程
func (fm *FlowManager) StartInteractiveFlow(tools types.ToolCheckResults, flowConfig *FlowConfig) (*FlowResult, error) {
	startTime := time.Now()
	result := &FlowResult{
		UserChoices: make(map[string]interface{}),
	}

	fm.logger.Info("开始用户交互流程")

	// 设置默认流程配置
	if flowConfig == nil {
		flowConfig = &FlowConfig{
			EnableSecurityCheck:   true,
			EnableCacheManagement: false,
			EnableInstructions:    true,
			AllowResumeSession:    true,
		}
	}

	// 显示欢迎界面
	fm.uiManager.ShowWelcome()

	// 运行主交互会话
	sessionOpts := &SessionOptions{
		EnableBatchConversion: true,
		EnableCacheManagement: flowConfig.EnableCacheManagement,
		ShowInstructions:      flowConfig.EnableInstructions,
	}

	sessionResult, err := fm.uiManager.RunInteractiveSession(tools, sessionOpts)
	if err != nil {
		result.Error = err
		result.ElapsedTime = time.Since(startTime)
		return result, err
	}

	result.Action = sessionResult.Action
	result.UserChoices["main_action"] = sessionResult.Action

	// 如果用户选择退出
	if sessionResult.Action == "exit" {
		result.Success = true
		result.ElapsedTime = time.Since(startTime)
		return result, nil
	}

	// 如果用户选择转换
	if sessionResult.Action == "convert" {
		processResult, err := fm.handleConversionFlow(sessionResult.Config, flowConfig, tools)
		if err != nil {
			result.Error = err
			result.ElapsedTime = time.Since(startTime)
			return result, err
		}

		// 合并结果
		result.Success = processResult.Success
		result.Config = processResult.Config
		result.TargetDir = processResult.TargetDir
		result.Mode = processResult.Mode
		if processResult.Error != nil {
			result.Error = processResult.Error
		}

		// 合并用户选择记录
		for k, v := range processResult.UserChoices {
			result.UserChoices[k] = v
		}
	}

	result.ElapsedTime = time.Since(startTime)
	fm.logger.Info("用户交互流程完成",
		zap.Duration("elapsed", result.ElapsedTime),
		zap.String("action", result.Action),
		zap.Bool("success", result.Success))

	return result, nil
}

// handleConversionFlow 处理转换流程
func (fm *FlowManager) handleConversionFlow(cfg *config.Config, flowConfig *FlowConfig, tools types.ToolCheckResults) (*FlowResult, error) {
	result := &FlowResult{
		Config:      cfg,
		TargetDir:   cfg.TargetDir,
		UserChoices: make(map[string]interface{}),
	}

	// 记录用户选择的模式
	result.UserChoices["selected_mode"] = cfg.Mode

	// 解析模式
	switch cfg.Mode {
	case "auto+":
		result.Mode = types.ModeAutoPlus
	case "quality":
		result.Mode = types.ModeQuality
	case "sticker":
		result.Mode = types.ModeEmoji
	default:
		result.Mode = types.ModeAutoPlus
	}

	// 执行安全检查
	if flowConfig.EnableSecurityCheck {
		fm.uiManager.showInfo("🔒 正在执行安全检查...")

		securityResult, err := fm.securityChecker.PerformSecurityCheck(cfg.TargetDir)
		if err != nil {
			return result, fmt.Errorf("安全检查失败: %w", err)
		}

		result.UserChoices["security_check"] = map[string]interface{}{
			"passed":   securityResult.Passed,
			"warnings": len(securityResult.Warnings),
			"issues":   len(securityResult.Issues),
		}

		// 处理安全检查结果
		if !securityResult.Passed {
			fm.uiManager.showError("安全检查未通过：")
			for _, issue := range securityResult.Issues {
				fm.uiManager.showError(fmt.Sprintf("- %s: %s", issue.Type.String(), issue.Message))
				if issue.Suggestion != "" {
					fm.uiManager.showInfo(fmt.Sprintf("  建议: %s", issue.Suggestion))
				}
			}
			return result, fmt.Errorf("安全检查未通过，处理终止")
		}

		// 显示警告信息
		for _, warning := range securityResult.Warnings {
			fm.uiManager.showWarning(fmt.Sprintf("警告: %s", warning.Message))
		}

		fm.uiManager.showSuccess("✅ 安全检查通过")
	}

	// 验证配置
	if err := config.ValidateAndNormalize(cfg); err != nil {
		return result, fmt.Errorf("配置验证失败: %w", err)
	}

	// 显示配置摘要
	fm.showConfigurationSummary(cfg, result.Mode)

	result.Success = true
	return result, nil
}

// showConfigurationSummary 显示配置摘要
func (fm *FlowManager) showConfigurationSummary(cfg *config.Config, mode types.AppMode) {
	fm.uiManager.showInfo("📋 转换配置摘要")
	fmt.Printf("  🎯 处理模式: %s\n", mode.String())
	fmt.Printf("  📁 目标目录: %s\n", cfg.TargetDir)
	fmt.Printf("  ⚡ 并发任务: %d\n", cfg.ConcurrentJobs)
	fmt.Printf("  🔄 最大重试: %d\n", cfg.MaxRetries)
	fmt.Printf("  💾 启用备份: %t\n", cfg.EnableBackups)
	fmt.Printf("  🚀 硬件加速: %t\n", cfg.HwAccel)

	fm.uiManager.showSuccess("✅ 配置验证完成，准备开始处理")
}

// StartNonInteractiveFlow 启动非交互式流程
func (fm *FlowManager) StartNonInteractiveFlow(cfg *config.Config, tools types.ToolCheckResults) (*FlowResult, error) {
	startTime := time.Now()
	result := &FlowResult{
		Config:      cfg,
		TargetDir:   cfg.TargetDir,
		UserChoices: make(map[string]interface{}),
	}

	fm.logger.Info("开始非交互式流程", zap.String("target_dir", cfg.TargetDir))

	// 解析模式
	switch cfg.Mode {
	case "auto+":
		result.Mode = types.ModeAutoPlus
	case "quality":
		result.Mode = types.ModeQuality
	case "sticker":
		result.Mode = types.ModeEmoji
	default:
		result.Mode = types.ModeAutoPlus
	}

	// 验证配置
	if err := config.Validate(cfg); err != nil {
		result.Error = err
		result.ElapsedTime = time.Since(startTime)
		return result, fmt.Errorf("配置验证失败: %w", err)
	}

	// 执行安全检查
	fm.logger.Info("执行安全检查")
	securityResult, err := fm.securityChecker.PerformSecurityCheck(cfg.TargetDir)
	if err != nil {
		result.Error = err
		result.ElapsedTime = time.Since(startTime)
		return result, fmt.Errorf("安全检查失败: %w", err)
	}

	if !securityResult.Passed {
		result.Error = fmt.Errorf("安全检查未通过")
		result.ElapsedTime = time.Since(startTime)
		return result, result.Error
	}

	result.Success = true
	result.Action = "convert"
	result.ElapsedTime = time.Since(startTime)

	fm.logger.Info("非交互式流程完成",
		zap.Duration("elapsed", result.ElapsedTime),
		zap.String("mode", result.Mode.String()))

	return result, nil
}

// GetRecommendedConfig 获取推荐配置
func (fm *FlowManager) GetRecommendedConfig(targetDir string, scenario string) *config.Config {
	cfg := config.DefaultConfig() // 使用标准化的默认配置
	cfg.TargetDir = targetDir

	// 根据场景调整配置
	switch scenario {
	case "large_scale":
		cfg.ConcurrentJobs = min(16, runtime.NumCPU()) // 限制在合理范围内
		cfg.LogLevel = "warn"
	case "high_quality":
		cfg.Mode = "quality"
		cfg.ConcurrentJobs = max(1, runtime.NumCPU()/2) // 高质量模式使用较少并发
	case "quick_compress":
		cfg.Mode = "sticker"
		cfg.ConcurrentJobs = min(8, runtime.NumCPU()) // 中等并发数
	}

	// 确保配置合法
	config.NormalizeConfig(cfg)
	return cfg
}

// ShowSystemInfo 显示系统信息
func (fm *FlowManager) ShowSystemInfo(tools types.ToolCheckResults) {
	fm.uiManager.showInfo("🔧 系统工具状态")

	if tools.HasFfmpeg {
		fmt.Printf("  ✅ FFmpeg: 已找到\n")
		if tools.FfmpegDevPath != "" {
			fmt.Printf("    - 开发版: %s\n", tools.FfmpegDevPath)
		}
		if tools.FfmpegStablePath != "" {
			fmt.Printf("    - 稳定版: %s\n", tools.FfmpegStablePath)
		}
	} else {
		fmt.Printf("  ❌ FFmpeg: 未找到\n")
	}

	if tools.HasCjxl {
		fmt.Printf("  ✅ cjxl: 已找到\n")
	} else {
		fmt.Printf("  ❌ cjxl: 未找到\n")
	}

	if tools.HasExiftool {
		fmt.Printf("  ✅ exiftool: 已找到\n")
	} else {
		fmt.Printf("  ❌ exiftool: 未找到\n")
	}

	fmt.Println()
}

// Shutdown 关闭流程管理器
func (fm *FlowManager) Shutdown() error {
	if fm.cancel != nil {
		fm.cancel()
	}

	if fm.uiManager != nil {
		if err := fm.uiManager.Shutdown(); err != nil {
			fm.logger.Error("关闭UI管理器失败", zap.Error(err))
		}
	}

	fm.logger.Info("流程管理器已关闭")
	return nil
}

// GetUIManager 获取UI管理器
func (fm *FlowManager) GetUIManager() *Manager {
	return fm.uiManager
}

// IsInteractive 检查是否为交互模式
func (fm *FlowManager) IsInteractive() bool {
	return fm.uiManager != nil
}
