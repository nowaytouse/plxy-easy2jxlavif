package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pixly/pkg/predictor"
	"pixly/pkg/ui"

	"github.com/pterm/pterm"
	"go.uber.org/zap"
)

func main() {
	// 创建UI配置（交互模式）
	config := ui.Interactive()

	// 初始化logger（根据模式，避免刷屏）
	logger, _ := ui.NewInteractiveLogger() // 仅显示INFO及以上
	defer logger.Sync()

	// 显示Banner
	ui.ShowBanner(config)

	// 欢迎动画
	animation := ui.NewAnimation(config)
	animation.ShowWelcomeAnimation()

	// 显示欢迎消息
	ui.ShowWelcomeMessage(config)

	// 主菜单
	for {
		options := []string{
			"🚀 智能转换模式（预测演示）",
			"🎨 完整转换功能（实际转换）",
			"📊 TESTPACK验证测试",
			"🎭 UI/UX特性展示",
			"🔍 知识库查询",
			"⚙️  配置管理",
			"👋 退出",
		}

		pterm.Info.Println("💡 操作提示：⬆️⬇️ 方向键选择 | ⏎ 回车确认 | 输入文字搜索")
		pterm.Println()

		selectedOption, _ := pterm.DefaultInteractiveSelect.
			WithOptions(options).
			WithDefaultText("请选择功能").
			Show()

		pterm.Println()

		switch selectedOption {
		case options[0]: // 智能转换（预测演示）
			runSmartConversion(config, logger, animation)

		case options[1]: // 完整转换功能
			runFullConversion(config, logger, animation)

		case options[2]: // TESTPACK测试
			runTestpackConversion(config, logger, animation)

		case options[3]: // UI/UX展示
			runUIUXDemo(config, animation)

		case options[4]: // 知识库查询
			runKnowledgeQuery(config, animation)

		case options[5]: // 配置管理
			runConfigManagement(config)

		case options[6]: // 退出
			ui.ShowGoodbye(config)
			return
		}

		pterm.Println()
	}
}

// runSmartConversion 智能转换模式
func runSmartConversion(config *ui.Config, logger *zap.Logger, animation *ui.Animation) {
	pterm.DefaultHeader.Println("🚀 智能转换模式")
	pterm.Println()

	// 安全检查器
	checker := ui.NewSafetyChecker(config)

	// 输入路径（使用bufio读取完整行，支持空格和特殊字符）
	pterm.Info.Println("📂 请输入要转换的目录路径：")
	pterm.Info.Println("💡 提示：支持拖拽文件夹到终端，或直接粘贴路径")
	fmt.Print("\n路径: ")

	reader := bufio.NewReader(os.Stdin)
	inputPath, _ := reader.ReadString('\n')
	inputPath = strings.TrimSpace(inputPath)

	// 移除可能的引号（macOS拖拽会自动加引号）
	inputPath = strings.Trim(inputPath, "'\"")

	// 处理shell转义字符（macOS拖拽会转义空格和特殊字符）
	inputPath = unescapeShellPath(inputPath)

	if inputPath == "" {
		pterm.Warning.Println("⚠️  路径为空，使用TESTPACK测试路径")
		inputPath = "/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!"
	}

	// 安全验证
	if err := checker.ValidateDirectory(inputPath); err != nil {
		pterm.Error.Printfln("安全检查失败: %v", err)
		return
	}

	pterm.Success.Printfln("✅ 路径验证通过: %s", inputPath)
	pterm.Println()

	// 扫描文件
	spinner := animation.ShowProcessingAnimation("扫描文件中")

	files, err := scanMediaFiles(inputPath)
	if err != nil {
		if spinner != nil {
			spinner.Fail("扫描失败")
		}
		pterm.Error.Printfln("扫描失败: %v", err)
		return
	}

	if spinner != nil {
		spinner.Success(fmt.Sprintf("找到 %d 个媒体文件", len(files)))
	}

	pterm.Println()

	// 文件数量检查
	if err := checker.CheckFileCount(len(files), 1000); err != nil {
		pterm.Warning.Println(err.Error())

		confirmed, _ := checker.ConfirmAction(
			fmt.Sprintf("是否继续转换 %d 个文件？", len(files)),
			30*time.Second,
		)

		if !confirmed {
			pterm.Info.Println("用户取消操作")
			return
		}
	}

	// 预测演示（前5个文件）
	pterm.DefaultSection.Println("📊 预测分析（前5个样本）")
	pterm.Println()

	featureExtractor := predictor.NewFeatureExtractor(logger, "ffprobe")
	mainPredictor := predictor.NewPredictor(logger, "ffprobe")

	sampleFiles := files
	if len(files) > 5 {
		sampleFiles = files[:5]
	}

	for i, file := range sampleFiles {
		prediction, err := mainPredictor.PredictOptimalParams(file)
		if err != nil {
			pterm.Warning.Printfln("⚠️  [%d] 预测失败: %s", i+1, filepath.Base(file))
			continue
		}

		// 提取特征用于显示
		features, _ := featureExtractor.ExtractFeatures(file)
		if features == nil {
			continue
		}

		pterm.Info.Printfln("[%d/%d] %s", i+1, len(sampleFiles), filepath.Base(file))
		pterm.Printfln("  格式: %s → %s", features.Format, prediction.Params.TargetFormat)
		pterm.Printfln("  预期节省: %.1f%%", prediction.ExpectedSaving*100)
		pterm.Printfln("  置信度: %.0f%%", prediction.Confidence*100)
		pterm.Println()
	}

	pterm.Success.Println("🎉 预测演示完成！")
	pterm.Info.Println("💡 提示：完整转换功能正在开发中...")
}

// runFullConversion 完整转换功能（实际转换）
func runFullConversion(config *ui.Config, logger *zap.Logger, animation *ui.Animation) {
	pterm.DefaultHeader.Println("🎨 完整转换功能")
	pterm.Println()

	// 安全检查器
	checker := ui.NewSafetyChecker(config)

	// 输入路径
	pterm.Info.Println("📂 请输入要转换的目录路径：")
	pterm.Info.Println("💡 提示：支持拖拽文件夹到终端，或直接粘贴路径")
	fmt.Print("\n路径: ")

	reader := bufio.NewReader(os.Stdin)
	inputPath, _ := reader.ReadString('\n')
	inputPath = strings.TrimSpace(inputPath)
	inputPath = strings.Trim(inputPath, "'\"")
	inputPath = unescapeShellPath(inputPath)

	if inputPath == "" {
		pterm.Warning.Println("⚠️  路径为空，使用TESTPACK测试路径")
		inputPath = "/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!"
	}

	// 安全验证
	if err := checker.ValidateDirectory(inputPath); err != nil {
		pterm.Error.Printfln("安全检查失败: %v", err)
		return
	}

	pterm.Success.Printfln("✅ 路径验证通过: %s", inputPath)
	pterm.Println()

	// 选择转换模式
	modeOptions := []string{
		"🔄 原地转换（替换原文件）",
		"📁 复制到新目录",
	}

	selectedMode, _ := pterm.DefaultInteractiveSelect.
		WithOptions(modeOptions).
		WithDefaultText("请选择转换模式").
		Show()

	pterm.Println()

	inPlace := (selectedMode == modeOptions[0])
	var outputDir string

	if !inPlace {
		pterm.Info.Println("📂 请输入输出目录路径：")
		fmt.Print("\n输出路径: ")
		outputPath, _ := reader.ReadString('\n')
		outputDir = strings.TrimSpace(outputPath)
		outputDir = strings.Trim(outputDir, "'\"")
		outputDir = unescapeShellPath(outputDir)

		if outputDir == "" {
			pterm.Warning.Println("⚠️  未指定输出目录，将在原目录生成新文件")
		}
	}

	// 最终确认
	confirmMsg := fmt.Sprintf("准备转换\n路径: %s\n模式: %s", inputPath, selectedMode)
	if !inPlace && outputDir != "" {
		confirmMsg += fmt.Sprintf("\n输出: %s", outputDir)
	}

	confirmed, err := checker.ConfirmAction(confirmMsg, 30*time.Second)
	if err != nil || !confirmed {
		pterm.Info.Println("❌ 用户取消操作")
		return
	}

	pterm.Println()

	// 创建转换引擎
	pterm.Info.Println("🔧 初始化转换引擎...")
	engine, err := NewConversionEngine(logger, config)
	if err != nil {
		pterm.Error.Printfln("❌ 引擎初始化失败: %v", err)
		return
	}
	defer engine.Close()

	pterm.Success.Println("✅ 引擎就绪")
	pterm.Println()

	// 执行转换
	ctx := context.Background()
	result, err := engine.ConvertDirectory(ctx, inputPath, outputDir, inPlace)

	if err != nil {
		pterm.Error.Printfln("❌ 转换过程出错: %v", err)
		if result != nil {
			engine.ShowResult(result)
		}
		return
	}

	// 显示结果
	engine.ShowResult(result)
}

// runTestpackConversion TESTPACK完整转换测试
func runTestpackConversion(config *ui.Config, logger *zap.Logger, animation *ui.Animation) {
	pterm.DefaultHeader.Println("📊 TESTPACK验证测试")
	pterm.Println()

	testpackPath := "/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!"

	// 检查路径
	if _, err := os.Stat(testpackPath); os.IsNotExist(err) {
		pterm.Error.Printfln("TESTPACK路径不存在: %s", testpackPath)
		return
	}

	pterm.Info.Printfln("TESTPACK路径: %s", testpackPath)
	pterm.Println()

	// 扫描文件
	spinner := animation.ShowProcessingAnimation("扫描TESTPACK文件")

	files, err := scanMediaFiles(testpackPath)
	if err != nil {
		if spinner != nil {
			spinner.Fail("扫描失败")
		}
		pterm.Error.Printfln("扫描失败: %v", err)
		return
	}

	if spinner != nil {
		spinner.Success(fmt.Sprintf("找到 %d 个文件", len(files)))
	}

	pterm.Println()

	// 按格式分类
	formatStats := make(map[string]int)
	for _, file := range files {
		ext := filepath.Ext(file)
		formatStats[ext]++
	}

	// 显示统计
	pterm.DefaultSection.Println("📊 文件统计")
	pterm.Println()

	tableData := pterm.TableData{
		{"格式", "数量", "占比"},
	}

	total := len(files)
	for ext, count := range formatStats {
		percentage := float64(count) / float64(total) * 100
		tableData = append(tableData, []string{
			ext,
			fmt.Sprintf("%d", count),
			fmt.Sprintf("%.1f%%", percentage),
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	pterm.Println()

	// 预测演示
	pterm.DefaultSection.Println("🧠 智能预测演示（随机10个样本）")
	pterm.Println()

	featureExtractor := predictor.NewFeatureExtractor(logger, "ffprobe")
	mainPredictor := predictor.NewPredictor(logger, "ffprobe")

	// 随机选择10个文件
	sampleSize := 10
	if len(files) < sampleSize {
		sampleSize = len(files)
	}

	successCount := 0
	totalSaving := 0.0

	for i := 0; i < sampleSize; i++ {
		file := files[i*len(files)/sampleSize]

		prediction, err := mainPredictor.PredictOptimalParams(file)
		if err != nil {
			pterm.Warning.Printfln("⚠️  [%d] 预测失败: %s", i+1, filepath.Base(file))
			continue
		}

		// 提取特征用于显示
		features, _ := featureExtractor.ExtractFeatures(file)
		if features == nil {
			continue
		}

		pterm.Info.Printfln("✅ [%d/%d] %s", i+1, sampleSize, filepath.Base(file))
		pterm.Printfln("  %s (%.1f MB) → %s",
			features.Format,
			float64(features.FileSize)/(1024*1024),
			prediction.Params.TargetFormat)
		pterm.Printfln("  预期节省: %.1f%% | 置信度: %.0f%%",
			prediction.ExpectedSaving*100,
			prediction.Confidence*100)

		if prediction.Params.LosslessJPEG {
			pterm.Println("  模式: lossless_jpeg=1 (100%可逆)")
		} else if prediction.Params.Distance == 0 {
			pterm.Println("  模式: distance=0 (无损)")
		}

		pterm.Println()

		successCount++
		totalSaving += prediction.ExpectedSaving
	}

	// 总结
	pterm.DefaultBox.WithTitle("📈 测试总结").WithTitleTopCenter().Println(
		fmt.Sprintf("总文件数: %d\n成功预测: %d/%d\n平均预期节省: %.1f%%",
			len(files),
			successCount,
			sampleSize,
			(totalSaving/float64(successCount))*100))

	pterm.Println()
	pterm.Success.Println("🎉 TESTPACK验证完成！")
}

// runUIUXDemo UI/UX特性展示
func runUIUXDemo(config *ui.Config, animation *ui.Animation) {
	pterm.DefaultHeader.Println("🎨 UI/UX特性展示")
	pterm.Println()

	// 1. 字符画展示
	pterm.DefaultSection.Println("1️⃣ 渐变字符画+材质")
	ui.ShowASCIIArt(config)

	// 2. 动画效果
	pterm.DefaultSection.Println("2️⃣ 动画效果")
	pterm.Println()

	animation.ShowLoadingAnimation("加载知识库", 800*time.Millisecond)
	animation.ShowSuccessEffect("加载完成")
	pterm.Println()

	// 3. 进度条演示
	pterm.DefaultSection.Println("3️⃣ 稳定进度条")
	pterm.Println()

	progressMgr := ui.NewProgressManager(config)
	bar, _ := ui.NewSafeProgressBar(progressMgr, "转换文件", 30)

	for i := 0; i < 30; i++ {
		bar.Increment()
		if i%5 == 0 {
			bar.SetMessage(fmt.Sprintf("处理文件 %d/30", i+1))
		}
		time.Sleep(50 * time.Millisecond)
	}

	bar.Finish()
	pterm.Success.Println("✅ 进度条演示完成")
	pterm.Println()

	// 4. 配色方案
	pterm.DefaultSection.Println("4️⃣ 配色方案")
	pterm.Println()

	scheme := ui.GetColorScheme("auto")
	pterm.Println("主色: " + pterm.NewStyle(scheme.Primary).Sprint("█████ Pixly"))
	pterm.Println("成功: " + pterm.NewStyle(scheme.Success).Sprint("█████ 转换成功"))
	pterm.Println("警告: " + pterm.NewStyle(scheme.Warning).Sprint("█████ 注意事项"))
	pterm.Println("强调: " + pterm.NewStyle(scheme.Accent).Sprint("█████ 重要信息"))
	pterm.Println()

	pterm.Success.Println("🎉 UI/UX演示完成！")
}

// runKnowledgeQuery 知识库查询
func runKnowledgeQuery(config *ui.Config, animation *ui.Animation) {
	pterm.DefaultHeader.Println("🔍 知识库查询")
	pterm.Println()

	animation.ShowLoadingAnimation("查询知识库", 500*time.Millisecond)

	pterm.Info.Println("知识库功能：")
	pterm.Println("  ✅ SQLite数据库")
	pterm.Println("  ✅ 自动记录转换历史")
	pterm.Println("  ✅ 预测准确性分析")
	pterm.Println("  ✅ 实时学习优化")
	pterm.Println()

	pterm.Info.Println("💡 提示：知识库在实际转换后自动生成")
	pterm.Info.Println("📊 当前状态：等待首次转换...")
}

// runConfigManagement 配置管理
func runConfigManagement(config *ui.Config) {
	pterm.DefaultHeader.Println("⚙️  配置管理")
	pterm.Println()

	configData := pterm.TableData{
		{"配置项", "当前值", "说明"},
		{"模式", getModeName(config.Mode), "交互/非交互"},
		{"动画", getBoolName(config.EnableAnimation), "是否启用动画"},
		{"颜色", getBoolName(config.EnableColor), "是否启用颜色"},
		{"进度条", getBoolName(config.EnableProgressBar), "是否显示进度条"},
		{"刷新率", fmt.Sprintf("%dms", config.ProgressRefreshRate), "进度条刷新间隔"},
		{"安全检查", getBoolName(config.SafetyChecks), "系统目录保护"},
		{"主题", config.Theme, "颜色主题"},
	}

	pterm.DefaultTable.WithHasHeader().WithData(configData).Render()
	pterm.Println()
}

// scanMediaFiles 扫描媒体文件
func scanMediaFiles(dir string) ([]string, error) {
	var files []string

	extensions := []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".mp4", ".mov", ".avi"}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		for _, validExt := range extensions {
			if ext == validExt {
				files = append(files, path)
				break
			}
		}

		return nil
	})

	return files, err
}

// unescapeShellPath 移除shell转义字符（处理macOS拖拽产生的转义）
func unescapeShellPath(path string) string {
	// 替换常见的shell转义
	replacements := map[string]string{
		`\ `: " ",  // 转义的空格
		`\!`: "!",  // 转义的感叹号
		`\(`: "(",  // 转义的左括号
		`\)`: ")",  // 转义的右括号
		`\[`: "[",  // 转义的左方括号
		`\]`: "]",  // 转义的右方括号
		`\{`: "{",  // 转义的左花括号
		`\}`: "}",  // 转义的右花括号
		`\'`: "'",  // 转义的单引号
		`\"`: "\"", // 转义的双引号
		`\$`: "$",  // 转义的美元符号
		`\&`: "&",  // 转义的和号
		`\*`: "*",  // 转义的星号
		`\;`: ";",  // 转义的分号
		`\|`: "|",  // 转义的管道符
		`\<`: "<",  // 转义的小于号
		`\>`: ">",  // 转义的大于号
		`\?`: "?",  // 转义的问号
		`\#`: "#",  // 转义的井号
		`\~`: "~",  // 转义的波浪号
		`\=`: "=",  // 转义的等号
	}

	result := path
	for escaped, unescaped := range replacements {
		result = strings.ReplaceAll(result, escaped, unescaped)
	}

	return result
}

// 辅助函数
func getModeName(mode ui.Mode) string {
	if mode == ui.ModeInteractive {
		return "交互模式"
	}
	return "非交互模式"
}

func getBoolName(b bool) string {
	if b {
		return "✅ 启用"
	}
	return "❌ 禁用"
}
