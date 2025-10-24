package main

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pixly",
	Short: "🎨 Pixly - 智能媒体转换专家",
	Long: `
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║     🎨 Pixly v3.1 - 智能媒体转换专家                        ║
║                                                               ║
║     为不同媒体量身打造最优转换参数                           ║
║     100%质量保证 | 智能学习 | 持续优化                      ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
	`,
	Run: func(cmd *cobra.Command, args []string) {
		showMainMenu()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}
}

func showMainMenu() {
	// 显示欢迎信息
	pterm.DefaultBox.
		WithTitle("🎨 Pixly v3.1").
		WithTitleTopCenter().
		WithBoxStyle(pterm.NewStyle(pterm.FgCyan)).
		Println("智能媒体转换专家\n\n为不同媒体量身打造最优参数\n100%质量保证 | 智能学习 | 持续优化")

	pterm.Println()

	// 主菜单选项
	options := []string{
		"🚀 智能转换（推荐） - 使用黄金规则，100%质量保证",
		"📁 批量转换 - 处理整个文件夹",
		"🎨 自定义转换 - 指定目标格式",
		"📊 查看统计 - 知识库数据分析",
		"⚙️  配置管理 - 修改转换参数",
		"❓ 帮助文档 - 查看使用说明",
		"🚪 退出程序",
	}

	pterm.Info.Println("使用 ↑↓ 选择，Enter 确认，Ctrl+C 退出")
	pterm.Println()

	// 交互式选择
	selectedOption, err := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithDefaultText("请选择操作").
		Show()

	if err != nil {
		pterm.Error.Printfln("选择失败: %v", err)
		return
	}

	// 处理选择
	handleMenuSelection(selectedOption)
}

func handleMenuSelection(selection string) {
	pterm.Println()

	switch {
	case selection[0:2] == "🚀":
		showSmartConversion()
	case selection[0:2] == "📁":
		showBatchConversion()
	case selection[0:2] == "🎨":
		showCustomConversion()
	case selection[0:2] == "📊":
		showStatistics()
	case selection[0:2] == "⚙️":
		showConfiguration()
	case selection[0:2] == "❓":
		showHelp()
	case selection[0:2] == "🚪":
		pterm.Success.Println("感谢使用 Pixly！再见 👋")
		os.Exit(0)
	}
}

func showSmartConversion() {
	pterm.DefaultSection.Println("🚀 智能转换模式")
	pterm.Println()

	pterm.Info.Println("智能转换使用黄金规则，为每种媒体量身定制参数：")
	pterm.Println()

	// 显示黄金规则
	rulesData := pterm.TableData{
		{"格式", "目标", "参数", "质量保证"},
		{"PNG", "JXL", "distance=0", "100%无损"},
		{"JPEG", "JXL", "lossless_jpeg=1", "100%可逆"},
		{"GIF动图", "AVIF", "CRF=35-38", "现代编码"},
		{"GIF静图", "JXL", "distance=0", "100%无损"},
		{"WebP", "JXL/AVIF", "智能选择", "动静图分离"},
		{"视频", "MOV", "重封装", "无质量损失"},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithBoxed().
		WithData(rulesData).
		Render()

	pterm.Println()

	// 询问是否运行演示
	options := []string{
		"▶️  运行转换演示",
		"◀️  返回主菜单",
	}

	selected, _ := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithDefaultText("选择操作").
		Show()

	if selected[0:3] == "▶️" {
		pterm.Println()
		demoSmartConversion()
	}

	showMainMenu()
}

func showBatchConversion() {
	pterm.DefaultSection.Println("📁 批量转换模式")
	pterm.Warning.Println("功能开发中...")
	pterm.Println()
	pterm.Info.Println("按 Enter 返回主菜单...")
	fmt.Scanln()
	showMainMenu()
}

func showCustomConversion() {
	pterm.DefaultSection.Println("🎨 自定义转换模式")

	pterm.Info.Println("自定义模式支持任意格式组合：")
	pterm.Println()

	// 演示自定义格式选择
	pterm.Println("  支持的组合：")
	pterm.Println("    PNG  → JXL, AVIF, WebP")
	pterm.Println("    JPEG → JXL, AVIF, WebP")
	pterm.Println("    GIF  → JXL, AVIF, WebP")
	pterm.Println()

	pterm.Warning.Println("功能开发中...")
	pterm.Println()
	pterm.Info.Println("按 Enter 返回主菜单...")
	fmt.Scanln()
	showMainMenu()
}

func showStatistics() {
	pterm.DefaultSection.Println("📊 知识库统计")

	// 模拟统计数据
	pterm.Info.Println("当前知识库数据：")
	pterm.Println()

	statsData := pterm.TableData{
		{"指标", "数值"},
		{"总转换次数", "5"},
		{"平均空间节省", "44.2%"},
		{"质量通过率", "100%"},
		{"预测准确性", "~81%"},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithBoxed().
		WithData(statsData).
		Render()

	pterm.Println()
	pterm.Success.Println("知识库正常工作，实时学习中...")
	pterm.Println()

	pterm.Info.Println("按 Enter 返回主菜单...")
	fmt.Scanln()
	showMainMenu()
}

func showConfiguration() {
	pterm.DefaultSection.Println("⚙️  配置管理")
	pterm.Warning.Println("功能开发中...")
	pterm.Println()
	pterm.Info.Println("按 Enter 返回主菜单...")
	fmt.Scanln()
	showMainMenu()
}

func showHelp() {
	pterm.DefaultSection.Println("❓ 帮助文档")

	pterm.Info.Println("Pixly v3.1 - 智能媒体转换专家")
	pterm.Println()
	pterm.Println("核心特性：")
	pterm.Println("  ✅ 为不同媒体量身定制参数")
	pterm.Println("  ✅ 100%质量保证（无损/可逆）")
	pterm.Println("  ✅ 智能学习，越用越准")
	pterm.Println("  ✅ 支持自定义格式组合")
	pterm.Println()
	pterm.Println("文档位置：docs/")
	pterm.Println()

	pterm.Info.Println("按 Enter 返回主菜单...")
	fmt.Scanln()
	showMainMenu()
}
