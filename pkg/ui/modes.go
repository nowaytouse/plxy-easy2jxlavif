package ui

import (
	"os"
)

// Mode UI模式
type Mode int

const (
	// ModeInteractive 交互模式（默认）
	ModeInteractive Mode = iota
	// ModeNonInteractive 非交互模式（调试用）
	ModeNonInteractive
)

// Config UI配置
type Config struct {
	Mode                Mode
	EnableAnimation     bool   // 是否启用动画
	EnableColor         bool   // 是否启用颜色
	EnableProgressBar   bool   // 是否启用进度条
	ProgressRefreshRate int    // 进度条刷新率（ms）
	SafetyChecks        bool   // 是否启用安全检查
	DebugMode           bool   // 调试模式
	Theme               string // 主题（auto/dark/light）
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Mode:                detectMode(),
		EnableAnimation:     true,
		EnableColor:         true,
		EnableProgressBar:   true,
		ProgressRefreshRate: 100, // 100ms刷新（避免刷屏）
		SafetyChecks:        true,
		DebugMode:           os.Getenv("PIXLY_DEBUG") == "true",
		Theme:               "auto",
	}
}

// detectMode 自动检测模式
func detectMode() Mode {
	// 检查是否有TTY（终端）
	if !isTerminal() {
		return ModeNonInteractive
	}

	// 检查环境变量
	if os.Getenv("PIXLY_NON_INTERACTIVE") == "true" {
		return ModeNonInteractive
	}

	// 检查是否在CI环境
	if os.Getenv("CI") == "true" {
		return ModeNonInteractive
	}

	return ModeInteractive
}

// isTerminal 检查是否是终端环境
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// NonInteractive 创建非交互模式配置
func NonInteractive() *Config {
	cfg := DefaultConfig()
	cfg.Mode = ModeNonInteractive
	cfg.EnableAnimation = false
	cfg.EnableProgressBar = false
	return cfg
}

// Interactive 创建交互模式配置
func Interactive() *Config {
	cfg := DefaultConfig()
	cfg.Mode = ModeInteractive
	return cfg
}

// IsInteractive 是否为交互模式
func (c *Config) IsInteractive() bool {
	return c.Mode == ModeInteractive
}

// ShouldShowAnimation 是否显示动画
func (c *Config) ShouldShowAnimation() bool {
	return c.IsInteractive() && c.EnableAnimation
}

// ShouldShowProgress 是否显示进度条
func (c *Config) ShouldShowProgress() bool {
	return c.EnableProgressBar
}
