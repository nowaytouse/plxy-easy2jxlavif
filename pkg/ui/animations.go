package ui

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
)

// Animation 动画效果
type Animation struct {
	config   *Config
	disabled bool // 转换阶段禁用动画
}

// NewAnimation 创建动画管理器
func NewAnimation(config *Config) *Animation {
	return &Animation{
		config:   config,
		disabled: false,
	}
}

// DisableForPerformance 为性能禁用动画（转换阶段）
func (a *Animation) DisableForPerformance() {
	a.disabled = true
}

// Enable 重新启用动画
func (a *Animation) Enable() {
	a.disabled = false
}

// ShouldAnimate 是否应该播放动画
func (a *Animation) ShouldAnimate() bool {
	return a.config.ShouldShowAnimation() && !a.disabled
}

// ShowWelcomeAnimation 显示欢迎动画（启动时，带emoji）
func (a *Animation) ShowWelcomeAnimation() {
	if !a.ShouldAnimate() {
		return
	}

	// 淡入效果（模拟）
	spinner, _ := pterm.DefaultSpinner.
		WithSequence("⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏").
		WithStyle(pterm.NewStyle(pterm.FgLightCyan)).
		Start("✨ 初始化 Pixly...")

	time.Sleep(800 * time.Millisecond)
	spinner.Success("🚀 就绪！")
}

// ShowProcessingAnimation 显示处理动画（轻量级，带emoji）
func (a *Animation) ShowProcessingAnimation(message string) *pterm.SpinnerPrinter {
	if !a.config.ShouldShowProgress() {
		fmt.Println("⚙️ " + message)
		return nil
	}

	// 使用简单spinner（转换时）
	spinner, _ := pterm.DefaultSpinner.
		WithShowTimer(true).
		WithRemoveWhenDone(true).
		Start("⚙️ " + message)

	return spinner
}

// ShowSuccessEffect 显示成功效果（带emoji）
func (a *Animation) ShowSuccessEffect(message string) {
	if !a.ShouldAnimate() {
		pterm.Success.Println("✅ " + message)
		return
	}

	// 快速成功动画
	spinner, _ := pterm.DefaultSpinner.
		WithStyle(pterm.NewStyle(pterm.FgLightGreen)).
		Start("⏳ " + message)

	time.Sleep(300 * time.Millisecond)
	spinner.Success("✅ " + message)
}

// ShowLoadingAnimation 显示加载动画（知识库查询等，带emoji）
func (a *Animation) ShowLoadingAnimation(message string, duration time.Duration) {
	if !a.ShouldAnimate() {
		fmt.Println("🔍 " + message)
		return
	}

	spinner, _ := pterm.DefaultSpinner.
		WithSequence("◐", "◓", "◑", "◒").
		WithStyle(pterm.NewStyle(pterm.FgLightBlue)).
		Start("🔍 " + message)

	time.Sleep(duration)
	spinner.Stop()
}

// ShowPulseEffect 显示脉冲效果（重要信息）
func (a *Animation) ShowPulseEffect(text string) {
	if !a.ShouldAnimate() {
		fmt.Println(text)
		return
	}

	// 脉冲效果（颜色变化）
	colors := []pterm.Color{
		pterm.FgLightCyan,
		pterm.FgCyan,
		pterm.FgLightCyan,
	}

	for _, color := range colors {
		fmt.Print("\r" + pterm.NewStyle(color).Sprint(text))
		time.Sleep(200 * time.Millisecond)
	}
	fmt.Println()
}

// TypewriterEffect 打字机效果（欢迎消息等）
func (a *Animation) TypewriterEffect(text string, speed time.Duration) {
	if !a.ShouldAnimate() {
		fmt.Println(text)
		return
	}

	for _, char := range text {
		fmt.Print(string(char))
		time.Sleep(speed)
	}
	fmt.Println()
}
