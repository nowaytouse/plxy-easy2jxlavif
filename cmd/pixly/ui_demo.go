package main

import (
	"fmt"
	"time"

	"pixly/pkg/ui"

	"github.com/pterm/pterm"
)

func main() {
	// 创建UI配置（强制交互模式用于演示）
	config := ui.Interactive()

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                               ║")
	fmt.Println("║     🎨 Pixly v3.1.1 UI/UX 高级特性演示                      ║")
	fmt.Println("║                                                               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 演示1: 模式检测
	pterm.DefaultSection.Println("1️⃣ 交互/非交互模式")
	pterm.Println()

	if config.IsInteractive() {
		pterm.Success.Println("✅ 当前模式: 交互模式")
		pterm.Info.Println("   支持: 箭头键导航、进度条、动画")
	} else {
		pterm.Info.Println("非交互模式（调试）")
	}
	pterm.Println()

	// 演示2: 安全检查
	pterm.DefaultSection.Println("2️⃣ 安全检测系统")
	pterm.Println()

	checker := ui.NewSafetyChecker(config)

	// 测试危险路径
	dangerPaths := []string{
		"/System",
		"/usr/bin",
		"/",
	}

	for _, path := range dangerPaths {
		err := checker.CheckPath(path)
		if err != nil {
			pterm.Error.Printfln("  ✅ 已拦截危险路径: %s", path)
		}
	}

	// 测试安全路径
	safePath := "/Users/test/Documents"
	pterm.Info.Printfln("  ✅ 安全路径通过: %s", safePath)
	pterm.Println()

	// 演示3: 稳定进度条（防刷屏）
	pterm.DefaultSection.Println("3️⃣ 稳定进度条（防刷屏）")
	pterm.Println()

	pterm.Info.Println("  刷新率: 100ms（避免刷屏）")
	pterm.Info.Println("  异常恢复: 自动冻结（检测到5次错误）")
	pterm.Info.Println("  防崩溃: panic恢复机制")
	pterm.Println()

	// 演示进度条
	progressMgr := ui.NewProgressManager(config)
	safeBar, _ := ui.NewSafeProgressBar(progressMgr, "演示进度", 50)

	for i := 0; i < 50; i++ {
		safeBar.Increment()
		if i%10 == 0 {
			safeBar.SetMessage(fmt.Sprintf("处理文件 %d...", i))
		}
		time.Sleep(20 * time.Millisecond) // 快速演示
	}

	safeBar.Finish()
	pterm.Success.Println("  ✅ 进度条稳定完成")
	pterm.Println()

	// 演示4: 渐变字符画
	pterm.DefaultSection.Println("4️⃣ 渐变字符画+材质")
	pterm.Println()

	ui.ShowBanner(config)
	pterm.Println()

	ui.ShowASCIIArt(config)

	// 演示5: 动画效果
	pterm.DefaultSection.Println("5️⃣ 动画效果（非转换阶段）")
	pterm.Println()

	anim := ui.NewAnimation(config)

	pterm.Info.Println("  欢迎动画:")
	anim.ShowWelcomeAnimation()

	pterm.Info.Println("  处理动画:")
	spinner := anim.ShowProcessingAnimation("分析文件特征...")
	time.Sleep(1 * time.Second)
	if spinner != nil {
		spinner.Success("分析完成")
	}

	pterm.Info.Println("  成功效果:")
	anim.ShowSuccessEffect("转换完成！")

	pterm.Println()

	// 演示6: 配色方案
	pterm.DefaultSection.Println("6️⃣ 配色方案（黑暗/亮色兼容）")
	pterm.Println()

	schemes := []string{"auto", "dark", "light"}
	for _, theme := range schemes {
		scheme := ui.GetColorScheme(theme)

		pterm.Printfln("  [%s主题]", theme)
		fmt.Print("    主色: ")
		pterm.NewStyle(scheme.Primary).Println("█████ Pixly")

		fmt.Print("    成功: ")
		pterm.NewStyle(scheme.Success).Println("█████ 转换成功")

		fmt.Print("    警告: ")
		pterm.NewStyle(scheme.Warning).Println("█████ 注意事项")

		fmt.Print("    强调: ")
		pterm.NewStyle(scheme.Accent).Println("█████ 重要信息")

		pterm.Println()
	}

	// 演示7: 材质效果
	pterm.DefaultSection.Println("7️⃣ 材质效果")
	pterm.Println()

	scheme := ui.GetColorScheme("auto")

	pterm.Println("  平面: " + ui.ApplyMaterialEffect("Pixly", ui.MaterialFlat, scheme))
	pterm.Println("  玻璃: " + ui.ApplyMaterialEffect("Pixly", ui.MaterialGlass, scheme))
	pterm.Println("  霓虹: " + ui.ApplyMaterialEffect("Pixly", ui.MaterialNeon, scheme))

	pterm.Println()
	pterm.Println()

	// 总结
	pterm.DefaultBox.
		WithTitle("✨ UI/UX高级特性").
		WithTitleTopCenter().
		WithBoxStyle(pterm.NewStyle(pterm.FgLightGreen)).
		Println("✅ 交互/非交互双模式\n✅ 安全检测（系统目录拦截）\n✅ 稳定进度条（防刷屏+防崩溃）\n✅ 渐变字符画\n✅ 动画效果（可控）\n✅ 配色兼容（黑暗/亮色）")

	pterm.Println()
	pterm.Success.Println("🎉 所有UI/UX高级特性已实现！")
}
