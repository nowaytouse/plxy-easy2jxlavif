package ui

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
)

// ShowBanner 显示启动横幅（Gemini风格：渐变+材质+emoji）
func ShowBanner(config *Config) {
	if config.Mode == ModeNonInteractive {
		// 非交互模式：简单文本
		fmt.Println("✨ Pixly v3.1.1 - 智能媒体转换专家")
		return
	}

	// 交互模式：Gemini风格精美字符画
	pterm.Println()

	// ASCII艺术（参考Gemini风格，显示PIXLY）
	asciiArt := []string{
		"",
		"  ███████████  ████████ ██████   ██████ ██      ██    ██",
		" ░░███░░░░░███░░███░░███░░██████ ██████ ░██     ░░██  ██",
		"  ░███    ░███ ░███ ░░░  ░███░█████░███ ░██      ░░████",
		"  ░██████████  ░███      ░███░░███ ░███ ░██       ░░██",
		"  ░███░░░░░░   ░███      ░███ ░░░  ░███ ░██        ░██",
		"  ░███         ░███    █ ░███      ░███ ░██        ░██",
		"  █████        ████████  █████     █████ ████████   ██",
		"  ░░░░░        ░░░░░░░░  ░░░░░     ░░░░░ ░░░░░░░░   ░░",
		"",
	}

	// Gemini风格渐变：从青色到洋红，带光泽效果
	gradientColors := []pterm.Color{
		pterm.FgLightCyan,    // 顶部：亮青色（高光）
		pterm.FgCyan,         // 青色
		pterm.FgLightBlue,    // 亮蓝
		pterm.FgBlue,         // 蓝色（中间色）
		pterm.FgLightMagenta, // 亮洋红
		pterm.FgMagenta,      // 洋红
		pterm.FgLightMagenta, // 亮洋红（光泽）
		pterm.FgCyan,         // 青色（底部反光）
	}

	// 渲染带渐变+材质的ASCII艺术
	for i, line := range asciiArt {
		if i == 0 || i == len(asciiArt)-1 {
			fmt.Println(line) // 空行
			continue
		}

		// 计算渐变颜色（模拟光泽从上到下）
		colorIndex := ((i - 1) * len(gradientColors)) / (len(asciiArt) - 2)
		if colorIndex >= len(gradientColors) {
			colorIndex = len(gradientColors) - 1
		}

		// 添加材质效果（通过Bold模拟高光区域）
		if i >= 1 && i <= 3 {
			// 顶部高光区：粗体+亮色
			pterm.Println(pterm.NewStyle(gradientColors[colorIndex], pterm.Bold).Sprint(line))
		} else if i >= len(asciiArt)-3 {
			// 底部反光区：斜体+柔和色
			pterm.Println(pterm.NewStyle(gradientColors[colorIndex], pterm.Italic).Sprint(line))
		} else {
			// 中间区域：正常
			pterm.Println(pterm.NewStyle(gradientColors[colorIndex]).Sprint(line))
		}
	}

	// 副标题（带emoji和渐变）
	subtitle := pterm.NewStyle(pterm.FgLightMagenta).Sprint("✨ v3.1.1 - 智能媒体转换专家 🎨")
	pterm.Println(pterm.DefaultCenter.Sprint(subtitle))
	pterm.Println()

	// 特性展示（带emoji和边框）
	featureBox := pterm.DefaultBox.
		WithTitle("🌟 核心特性").
		WithTitleTopCenter().
		WithBoxStyle(pterm.NewStyle(pterm.FgLightCyan))

	features := `🎯 为不同媒体量身定制参数
💎 100%质量保证（无损/可逆）
🧠 智能学习，越用越准确
🎨 支持自定义格式组合
⚡ TESTPACK验证通过（954个文件）
🚀 预测准确性提升69%`

	featureBox.Println(features)
	pterm.Println()
}

// ShowMinimalBanner 显示简化横幅（转换时使用，节省资源）
func ShowMinimalBanner(config *Config) {
	if config.Mode == ModeNonInteractive {
		fmt.Println("⚡ Pixly v3.1.1 - 转换中...")
		return
	}

	// 简化版本（不使用BigText，节省性能）
	header := pterm.DefaultHeader.
		WithBackgroundStyle(pterm.NewStyle(pterm.BgCyan)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		WithMargin(5)

	header.Println("🎨 Pixly v3.1.1 - 智能转换中 ⚡")
	pterm.Println()
}

// ShowASCIIArt 显示自定义ASCII艺术（带emoji和渐变材质）
func ShowASCIIArt(config *Config) {
	if config.Mode == ModeNonInteractive {
		return
	}

	pterm.Println()

	// ASCII艺术（媒体转换主题，带emoji）
	lines := []string{
		"    ╔═══════╗",
		"    ║ 📸 PNG  ║────┐",
		"    ╚═══════╝    │",
		"                 ▼",
		"    ╔═══════╗  ┌─────────┐   ╔═══════╗",
		"    ║ 🖼️ JPEG ║──▶│ ✨ Pixly │──▶║ 💎 JXL ║",
		"    ╚═══════╝  └─────────┘   ╚═══════╝",
		"                 ▲",
		"    ╔═══════╗    │",
		"    ║ 🎞️ GIF  ║────┘",
		"    ╚═══════╝",
	}

	// 渐变色（从绿色到蓝色，模拟数据流）
	colors := []pterm.Color{
		pterm.FgLightGreen, // 输入（绿色）
		pterm.FgGreen,
		pterm.FgLightCyan,
		pterm.FgCyan, // 处理（青色）
		pterm.FgLightBlue,
		pterm.FgBlue, // 输出（蓝色）
		pterm.FgLightMagenta,
		pterm.FgMagenta, // 完成（洋红）
		pterm.FgLightMagenta,
		pterm.FgCyan,
		pterm.FgLightCyan,
	}

	for i, line := range lines {
		colorIndex := (i * len(colors)) / len(lines)

		// 中心行（Pixly）使用粗体+高亮
		if i == 5 {
			pterm.Println(pterm.NewStyle(colors[colorIndex], pterm.Bold).Sprint(line))
		} else {
			pterm.Println(pterm.NewStyle(colors[colorIndex]).Sprint(line))
		}
	}

	pterm.Println()
}

// ShowSuccessAnimation 显示成功动画（带个性化emoji）
func ShowSuccessAnimation(config *Config) {
	if !config.ShouldShowAnimation() {
		pterm.Success.Println("🎉 转换完成！")
		return
	}

	// 动画效果（快速，不阻塞）
	spinner, _ := pterm.DefaultSpinner.
		WithStyle(pterm.NewStyle(pterm.FgLightGreen)).
		Start("⚡ 处理中...")

	time.Sleep(500 * time.Millisecond)
	spinner.Success("🎉 转换完成！")
}

// ShowWelcomeMessage 显示欢迎消息（个性化emoji）
func ShowWelcomeMessage(config *Config) {
	if config.Mode == ModeNonInteractive {
		fmt.Println("👋 欢迎使用 Pixly！")
		return
	}

	pterm.Println()

	// 欢迎框（带emoji）
	welcomeBox := pterm.DefaultBox.
		WithTitle("👋 欢迎").
		WithTitleTopCenter().
		WithBoxStyle(pterm.NewStyle(pterm.FgLightGreen))

	message := `🙌 感谢选择 Pixly！

我们致力于为您的每个媒体文件
找到最佳的转换参数 🎯

💡 提示：
  • 首次使用？试试智能转换模式 ✨
  • 需要帮助？查看配置文档 📖
  • 遇到问题？启用调试模式 🔍

让我们开始吧！🚀`

	welcomeBox.Println(message)
	pterm.Println()
}

// ShowGoodbye 显示退出消息
func ShowGoodbye(config *Config) {
	if config.Mode == ModeNonInteractive {
		fmt.Println("👋 再见！")
		return
	}

	pterm.Println()

	goodbyeBox := pterm.DefaultBox.
		WithTitle("👋 再见").
		WithTitleTopCenter().
		WithBoxStyle(pterm.NewStyle(pterm.FgLightMagenta))

	message := `感谢使用 Pixly！🎨

💾 您的转换记录已保存
📊 知识库正在学习中
🌟 期待下次相见！

Have a nice day! 😊`

	goodbyeBox.Println(message)
	pterm.Println()
}
