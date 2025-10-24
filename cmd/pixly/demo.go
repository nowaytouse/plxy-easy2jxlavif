package main

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
)

// 演示智能转换流程
func demoSmartConversion() {
	// 标题
	pterm.DefaultHeader.
		WithBackgroundStyle(pterm.NewStyle(pterm.BgCyan)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		Println("智能转换演示")

	pterm.Println()

	// 阶段1: 扫描文件
	pterm.DefaultSection.Println("📂 阶段1: 扫描文件")
	pterm.Println()

	scanSpinner, _ := pterm.DefaultSpinner.Start("扫描文件中...")
	time.Sleep(1 * time.Second)
	scanSpinner.Success("扫描完成！发现 245 个文件")

	pterm.Println()

	// 显示格式分布
	formatData := [][]string{
		{"格式", "数量", "大小", "目标格式"},
		{"PNG", "45", "123.5 MB", "JXL"},
		{"JPEG", "180", "456.2 MB", "JXL"},
		{"GIF", "20", "89.7 MB", "AVIF"},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderStyle(pterm.NewStyle(pterm.FgLightCyan)).
		WithData(pterm.TableData(formatData)).
		Render()

	pterm.Println()

	// 阶段2: 预测分析
	pterm.DefaultSection.Println("🔬 阶段2: 预测分析")
	pterm.Println()

	predictSpinner, _ := pterm.DefaultSpinner.Start("分析文件特征...")
	time.Sleep(800 * time.Millisecond)
	predictSpinner.Success("预测完成！")

	pterm.Println()

	// 预测结果
	pterm.Info.Printfln("预期空间节省: %.1f%%", 52.3)
	pterm.Info.Printfln("预期输出大小: %.1f MB", 332.1)
	pterm.Info.Printfln("质量保证: 100%% (无损/可逆)")

	pterm.Println()

	// 阶段3: 转换处理
	pterm.DefaultSection.Println("🔄 阶段3: 转换处理")
	pterm.Println()

	// 创建多进度条
	multi := pterm.DefaultMultiPrinter
	pb1, _ := pterm.DefaultProgressbar.WithTotal(245).WithTitle("总进度").Start()
	pb2, _ := pterm.DefaultProgressbar.WithTotal(100).WithTitle("当前文件").Start()

	multi.NewWriter()

	// 模拟转换
	for i := 0; i < 245; i++ {
		pb1.Increment()

		// 模拟当前文件进度
		for j := 0; j < 100; j += 20 {
			pb2.Add(20)
			time.Sleep(2 * time.Millisecond)
		}
		pb2.Current = 0

		if i%60 == 0 {
			time.Sleep(10 * time.Millisecond) // 稍微慢一点以便观察
		}
	}

	pb1.Stop()
	pb2.Stop()

	pterm.Println()

	// 阶段4: 结果展示
	pterm.DefaultSection.Println("✅ 阶段4: 转换完成")
	pterm.Println()

	pterm.Success.Println("转换成功！")
	pterm.Println()

	resultData := [][]string{
		{"指标", "数值"},
		{"成功转换", "242/245 (98.8%)"},
		{"原始大小", "669.4 MB"},
		{"转换后", "298.7 MB"},
		{"节省空间", "370.7 MB (55.4%)"},
		{"转换耗时", "8分32秒"},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithBoxed().
		WithHeaderStyle(pterm.NewStyle(pterm.FgLightGreen)).
		WithData(pterm.TableData(resultData)).
		Render()

	pterm.Println()

	// 知识库更新
	pterm.Info.Println("✨ 知识库已自动更新：+242条记录")
	pterm.Info.Println("📈 预测准确性将持续提升")

	pterm.Println()
	pterm.Info.Println("按 Enter 返回主菜单...")
	fmt.Scanln()
}
