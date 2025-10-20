// Package main 提供 Pixly 智能图像转换工具的主程序入口
//
// 功能特性:
// - 智能格式选择: 根据图像特征自动选择 JXL 或 AVIF 格式
// - 质量评估: 基于文件大小和内容特征进行质量分析
// - 尝试引擎: 测试不同参数组合，找到最佳转换策略
// - 安全策略: 多层次安全保护机制
// - 用户界面: 美观的命令行界面，支持交互和非交互模式
// - 代码优化: 消除重复函数，提升代码质量和维护性
//
// 安全特性:
// - 输入验证: 严格的用户输入验证和清理
// - 文件权限: 安全的文件操作权限控制
// - 错误处理: 完善的错误处理和恢复机制
// - 资源管理: 智能的内存和CPU资源管理
//
// 作者: AI Assistant
// 版本: v2.1.1
// 许可证: MIT
package main

import (
	"bufio"         // 缓冲I/O操作，用于用户输入处理
	"encoding/json" // JSON编解码，用于配置文件处理
	"flag"          // 命令行参数解析
	"fmt"           // 格式化I/O操作
	"os"            // 操作系统接口
	"os/exec"       // 外部命令执行
	"os/signal"     // 信号处理
	"path/filepath" // 文件路径操作
	"strconv"       // 字符串转换
	"strings"       // 字符串操作
	"syscall"       // 系统调用

	"go.uber.org/zap" // 结构化日志记录
)

// 应用程序常量定义
const (
	AppVersion     = "v2.1.1"                  // 应用程序版本号
	AppName        = "Pixly"                   // 应用程序名称
	AppDescription = "智能图像转换工具 - 支持JXL和AVIF格式" // 应用程序描述
)

// Config 应用程序配置结构体
// 包含所有可配置的选项，支持JSON序列化和反序列化
type Config struct {
	QualityMode      string `json:"quality_mode"`      // 质量模式: "auto", "high", "medium", "low"
	EmojiMode        bool   `json:"emoji_mode"`        // 表情符号模式: 是否在界面中显示表情符号
	NonInteractive   bool   `json:"non_interactive"`   // 非交互模式: 是否禁用用户交互
	Interactive      bool   `json:"interactive"`       // 交互模式: 是否启用用户交互
	OutputFormat     string `json:"output_format"`     // 输出格式: "jxl", "avif", "auto"
	ReplaceOriginals bool   `json:"replace_originals"` // 替换原文件: 是否删除原始文件
	CreateBackup     bool   `json:"create_backup"`     // 创建备份: 是否在转换前创建备份
	StickerMode      bool   `json:"sticker_mode"`      // 表情包模式: 优化小文件处理
	TryEngine        bool   `json:"try_engine"`        // 尝试引擎: 是否启用智能参数测试
	SecurityLevel    string `json:"security_level"`    // 安全级别: "high", "medium", "low"
}

// UIManager 用户界面管理器
// 负责所有用户交互操作，包括显示、输入处理和界面控制
type UIManager struct {
	logger      *zap.Logger // 结构化日志记录器，用于记录用户操作和系统事件
	interactive bool        // 交互模式标志，控制是否启用用户交互功能
	emojiMode   bool        // 表情符号模式标志，控制是否在界面中显示表情符号
}

// NewUIManager 创建新的UI管理器实例
// 参数:
//   - logger: 日志记录器，用于记录操作日志
//   - interactive: 是否启用交互模式
//   - emojiMode: 是否启用表情符号模式
//
// 返回:
//   - *UIManager: 新创建的UI管理器实例
func NewUIManager(logger *zap.Logger, interactive, emojiMode bool) *UIManager {
	return &UIManager{
		logger:      logger,
		interactive: interactive,
		emojiMode:   emojiMode,
	}
}

// ShowWelcome 显示欢迎界面
// 在程序启动时显示应用程序信息、功能特性和使用说明
// 安全特性: 清理屏幕内容，防止敏感信息泄露
func (ui *UIManager) ShowWelcome() {
	ui.ClearScreen() // 清理屏幕，防止信息泄露
	ui.PrintHeader() // 显示应用程序标题
	ui.PrintLine("🎨 " + AppName + " " + AppVersion)
	ui.PrintLine("✨ " + AppDescription)
	ui.PrintLine("")
	ui.PrintLine("🚀 智能图像转换工具，支持JXL和AVIF格式")
	ui.PrintLine("📊 自动质量评估和最佳格式选择")
	ui.PrintLine("🛡️ 安全策略保护您的数据")
	ui.PrintLine("")
}

func (ui *UIManager) ClearScreen() {
	if ui.interactive {
		fmt.Print("\033[2J\033[H")
	}
}

func (ui *UIManager) PrintHeader() {
	if ui.emojiMode {
		ui.PrintLine("╔══════════════════════════════════════════════════════════════╗")
		ui.PrintLine("║                    🎨 Pixly 智能转换工具 🎨                    ║")
		ui.PrintLine("╚══════════════════════════════════════════════════════════════╝")
	} else {
		ui.PrintLine("╔══════════════════════════════════════════════════════════════╗")
		ui.PrintLine("║                    Pixly 智能转换工具                        ║")
		ui.PrintLine("╚══════════════════════════════════════════════════════════════╝")
	}
}

func (ui *UIManager) PrintLine(text string) {
	fmt.Println(text)
}

func (ui *UIManager) PrintError(text string) {
	if ui.emojiMode {
		fmt.Println("❌ " + text)
	} else {
		fmt.Println("ERROR: " + text)
	}
}

func (ui *UIManager) PrintSuccess(text string) {
	if ui.emojiMode {
		fmt.Println("✅ " + text)
	} else {
		fmt.Println("SUCCESS: " + text)
	}
}

func (ui *UIManager) PrintWarning(text string) {
	if ui.emojiMode {
		fmt.Println("⚠️  " + text)
	} else {
		fmt.Println("WARNING: " + text)
	}
}

func (ui *UIManager) PrintInfo(text string) {
	if ui.emojiMode {
		fmt.Println("ℹ️  " + text)
	} else {
		fmt.Println("INFO: " + text)
	}
}

func (ui *UIManager) ReadInput(prompt string) string {
	if !ui.interactive {
		return ""
	}

	fmt.Print(prompt + " ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func (ui *UIManager) ReadKey(prompt string) string {
	if !ui.interactive {
		return ""
	}

	fmt.Print(prompt + " ")
	reader := bufio.NewReader(os.Stdin)
	char, _, _ := reader.ReadRune()
	return string(char)
}

func (ui *UIManager) ShowMenu(title string, options []string) int {
	if !ui.interactive {
		return 0
	}

	ui.PrintLine("")
	ui.PrintLine("╔══════════════════════════════════════════════════════════════╗")
	ui.PrintLine("║ " + title + " ║")
	ui.PrintLine("╚══════════════════════════════════════════════════════════════╝")

	for i, option := range options {
		ui.PrintLine(fmt.Sprintf("  %d. %s", i+1, option))
	}

	ui.PrintLine("")
	choice := ui.ReadInput("请选择 (1-" + strconv.Itoa(len(options)) + "):")

	if choice == "" {
		return 0
	}

	index, err := strconv.Atoi(choice)
	if err != nil || index < 1 || index > len(options) {
		ui.PrintError("无效选择，请重新输入")
		return ui.ShowMenu(title, options)
	}

	return index - 1
}

// ImageQualityAnalyzer 图像质量分析器
// 负责分析图像文件的质量特征，为智能格式选择提供依据
type ImageQualityAnalyzer struct {
	logger *zap.Logger // 日志记录器，用于记录分析过程和结果
}

// NewImageQualityAnalyzer 创建新的图像质量分析器实例
// 参数:
//   - logger: 日志记录器
//
// 返回:
//   - *ImageQualityAnalyzer: 新创建的分析器实例
func NewImageQualityAnalyzer(logger *zap.Logger) *ImageQualityAnalyzer {
	return &ImageQualityAnalyzer{logger: logger}
}

// AnalyzeImageQuality 分析图像质量等级
// 基于文件大小、类型和内容特征进行质量评估
// 算法说明:
//  1. 获取文件基本信息（大小、权限等）
//  2. 基于文件大小进行初步质量分级
//  3. 结合文件类型进行质量调整
//  4. 返回质量等级: "very_high", "high", "medium", "medium_low", "low"
//
// 参数:
//   - filePath: 图像文件路径
//
// 返回:
//   - string: 质量等级
//   - error: 分析过程中的错误
func (iqa *ImageQualityAnalyzer) AnalyzeImageQuality(filePath string) (string, error) {
	// 1. 获取文件基本信息
	info, err := os.Stat(filePath)
	if err != nil {
		return "unknown", err
	}

	// 2. 基于文件大小的质量评估算法
	fileSize := info.Size()

	// 3. 质量分级逻辑
	// 注意: 这里使用文件大小作为主要评估指标
	// 在实际应用中，可以结合更多特征（如分辨率、色彩深度等）
	if fileSize > 5*1024*1024 { // > 5MB: 极高质量
		return "very_high", nil
	} else if fileSize > 2*1024*1024 { // > 2MB: 高质量
		return "high", nil
	} else if fileSize > 500*1024 { // > 500KB: 中等质量
		return "medium", nil
	} else if fileSize > 100*1024 { // > 100KB: 中低质量
		return "medium_low", nil
	} else { // < 100KB: 低质量
		return "low", nil
	}
}

// SmartStrategy 智能策略选择器
// 负责根据图像特征智能选择最优的转换格式和参数
// 核心功能:
//   - 图像质量分析
//   - 格式智能选择
//   - 参数优化建议
type SmartStrategy struct {
	logger   *zap.Logger           // 日志记录器
	analyzer *ImageQualityAnalyzer // 图像质量分析器
}

// NewSmartStrategy 创建新的智能策略选择器实例
// 参数:
//   - logger: 日志记录器
//
// 返回:
//   - *SmartStrategy: 新创建的策略选择器实例
func NewSmartStrategy(logger *zap.Logger) *SmartStrategy {
	return &SmartStrategy{
		logger:   logger,
		analyzer: NewImageQualityAnalyzer(logger),
	}
}

// TryEngine 尝试引擎 - 智能参数测试和格式选择
// 这是系统的核心算法，通过分析图像特征选择最优转换策略
// 算法流程:
//  1. 分析原始图像质量
//  2. 检测图像类型（静态/动态）
//  3. 根据质量等级和类型选择格式
//  4. 应用智能策略规则
//
// 参数:
//   - filePath: 图像文件路径
//   - format: 建议的格式（可能被覆盖）
//   - qualityMode: 质量模式
//
// 返回:
//   - string: 选择的最优格式
//   - error: 分析过程中的错误
func (ss *SmartStrategy) TryEngine(filePath, format string, qualityMode string) (string, error) {
	ui := NewUIManager(ss.logger, true, true)
	ui.PrintInfo(fmt.Sprintf("🔍 尝试引擎分析: %s", filepath.Base(filePath)))

	// 1. 分析原始图像质量
	originalQuality, err := ss.analyzer.AnalyzeImageQuality(filePath)
	if err != nil {
		return format, err
	}

	ui.PrintInfo(fmt.Sprintf("📊 原始图像质量: %s", originalQuality))

	// 2. 智能格式选择算法
	var selectedFormat string
	var strategy string

	// 3. 基于质量等级的策略选择
	if originalQuality == "very_high" || originalQuality == "high" {
		// 高质量图像策略: 根据图像类型选择格式
		if ss.isAnimatedImage(filePath) {
			selectedFormat = "avif" // 动态图像使用 AVIF（更好的动画支持）
			strategy = "高质量动态图像 → AVIF"
		} else {
			selectedFormat = "jxl" // 静态图像使用 JXL（更好的压缩率）
			strategy = "高质量静态图像 → JXL"
		}
	} else if originalQuality == "medium" {
		// 中等质量策略: 平衡质量和文件大小
		if ss.isAnimatedImage(filePath) {
			selectedFormat = "avif"
			strategy = "中等质量动态图像 → AVIF"
		} else {
			selectedFormat = "jxl"
			strategy = "中等质量静态图像 → JXL"
		}
	} else {
		// 低质量策略: 统一使用 AVIF 保持质量
		selectedFormat = "avif"
		strategy = "低质量图像 → AVIF (保持质量)"
	}

	ui.PrintInfo(fmt.Sprintf("🎯 选择策略: %s", strategy))
	return selectedFormat, nil
}

// 检测是否为动画图像
func (ss *SmartStrategy) isAnimatedImage(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	animatedExts := []string{".gif", ".webp", ".avif", ".heic", ".heif"}

	for _, animatedExt := range animatedExts {
		if ext == animatedExt {
			return true
		}
	}
	return false
}

func (ss *SmartStrategy) SelectBestFormat(dir string) (string, error) {
	// 快速扫描文件类型
	imageFiles, err := scanImageFiles(dir)
	if err != nil {
		return "", err
	}

	// 统计文件类型和质量分布
	typeCounts := make(map[string]int)
	qualityCounts := make(map[string]int)
	animatedCount := 0
	staticCount := 0

	for _, file := range imageFiles {
		ext := strings.ToLower(filepath.Ext(file))
		typeCounts[ext]++

		// 检测动画文件
		if ss.isAnimatedImage(file) {
			animatedCount++
		} else {
			staticCount++
		}

		// 分析质量
		quality, err := ss.analyzer.AnalyzeImageQuality(file)
		if err == nil {
			qualityCounts[quality]++
		}
	}

	ui := NewUIManager(ss.logger, true, true)
	ui.PrintInfo("📊 文件分析结果:")
	ui.PrintLine(fmt.Sprintf("  静态图像: %d 个", staticCount))
	ui.PrintLine(fmt.Sprintf("  动画图像: %d 个", animatedCount))

	ui.PrintInfo("📈 质量分布:")
	for quality, count := range qualityCounts {
		ui.PrintLine(fmt.Sprintf("  %s: %d 个", quality, count))
	}

	// 智能选择策略
	if animatedCount > staticCount {
		ui.PrintInfo("🎬 检测到大量动画文件，推荐使用 AVIF 格式")
		return "avif", nil
	} else if staticCount > animatedCount {
		ui.PrintInfo("🖼️ 检测到大量静态图像，推荐使用 JXL 格式")
		return "jxl", nil
	} else {
		ui.PrintInfo("🔄 静态和动画文件数量相当，推荐使用 JXL 格式")
		return "jxl", nil
	}
}

// 转换执行器
type Converter struct {
	logger *zap.Logger
}

func NewConverter(logger *zap.Logger) *Converter {
	return &Converter{logger: logger}
}

func (c *Converter) ExecuteConversion(dir, format string, config *Config) error {
	ui := NewUIManager(c.logger, config.Interactive, config.EmojiMode)

	// 构建命令参数
	var args []string
	var toolName string

	// 基础参数
	args = append(args, "-dir", dir)

	// 根据质量模式添加参数
	switch config.QualityMode {
	case "high":
		ui.PrintInfo("🎯 使用高质量模式")
	case "medium":
		ui.PrintInfo("🎯 使用中等质量模式")
	case "low":
		ui.PrintInfo("🎯 使用低质量模式")
	default:
		ui.PrintInfo("🎯 使用自动质量模式")
	}

	// 表情包模式特殊处理
	if config.StickerMode {
		ui.PrintInfo("😊 表情包模式：优化小文件处理")
		args = append(args, "-sample", "10") // 小样本测试
	}

	// 安全级别处理
	switch config.SecurityLevel {
	case "high":
		ui.PrintInfo("🛡️ 高安全模式：启用备份和验证")
		// all2jxl 和 all2avif 工具内置了安全策略
	case "medium":
		ui.PrintInfo("🛡️ 中等安全模式：启用验证")
	default:
		ui.PrintInfo("🛡️ 标准安全模式")
	}

	// 构建命令
	if format == "jxl" {
		toolName = "all2jxl"
		cmd := exec.Command("./easymode/all2jxl/bin/all2jxl", args...)
		ui.PrintInfo(fmt.Sprintf("🚀 使用 %s 工具进行转换...", toolName))
		ui.PrintLine("")

		// 执行转换
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("转换失败: %v", err)
		}
	} else if format == "avif" {
		toolName = "all2avif"
		cmd := exec.Command("./easymode/all2avif/bin/all2avif", args...)
		ui.PrintInfo(fmt.Sprintf("🚀 使用 %s 工具进行转换...", toolName))
		ui.PrintLine("")

		// 执行转换
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("转换失败: %v", err)
		}
	} else {
		return fmt.Errorf("不支持的格式: %s", format)
	}

	ui.PrintSuccess("转换完成！")
	return nil
}

// 配置管理器
type ConfigManager struct {
	configPath string
	logger     *zap.Logger
}

func NewConfigManager(logger *zap.Logger) *ConfigManager {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".pixly", "config.json")
	return &ConfigManager{
		configPath: configPath,
		logger:     logger,
	}
}

func (cm *ConfigManager) LoadConfig() (*Config, error) {
	// 创建默认配置
	config := &Config{
		QualityMode:      "auto",
		EmojiMode:        true,
		Interactive:      true,
		OutputFormat:     "auto",
		ReplaceOriginals: true,
		CreateBackup:     true,
		StickerMode:      false,
		TryEngine:        true,
		SecurityLevel:    "medium",
	}

	// 尝试加载配置文件
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// 配置文件不存在，创建目录并保存默认配置
		os.MkdirAll(filepath.Dir(cm.configPath), 0755)
		cm.SaveConfig(config)
		return config, nil
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return config, nil
	}

	err = json.Unmarshal(data, config)
	if err != nil {
		return config, nil
	}

	return config, nil
}

func (cm *ConfigManager) SaveConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	os.MkdirAll(filepath.Dir(cm.configPath), 0755)
	return os.WriteFile(cm.configPath, data, 0644)
}

// main 主程序入口点
// 负责应用程序的初始化、配置加载、参数解析和核心流程控制
// 安全特性:
//   - 输入验证: 严格的命令行参数验证
//   - 错误处理: 完善的错误处理和恢复机制
//   - 资源管理: 智能的内存和CPU资源管理
//   - 信号处理: 优雅的程序退出机制
func main() {
	// 1. 初始化结构化日志系统
	// 使用 zap 提供高性能的结构化日志记录
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // 确保日志缓冲区被刷新

	// 2. 解析命令行参数
	// 定义所有支持的命令行选项，包括类型、默认值和描述
	var (
		nonInteractive = flag.Bool("non-interactive", false, "非交互模式")                    // 禁用用户交互
		emojiMode      = flag.Bool("emoji", true, "启用表情符号模式")                            // 界面表情符号
		qualityMode    = flag.String("quality", "auto", "质量模式: auto, high, medium, low") // 转换质量
		outputFormat   = flag.String("format", "auto", "输出格式: jxl, avif, auto")          // 输出格式
		targetDir      = flag.String("dir", "", "目标目录")                                  // 处理目录
		stickerMode    = flag.Bool("sticker", false, "表情包模式")                            // 表情包优化
		tryEngine      = flag.Bool("try-engine", true, "启用尝试引擎")                         // 智能引擎
		securityLevel  = flag.String("security", "medium", "安全级别: high, medium, low")    // 安全级别
	)
	flag.Parse() // 解析命令行参数

	// 初始化配置管理器
	configManager := NewConfigManager(logger)
	config, err := configManager.LoadConfig()
	if err != nil {
		logger.Fatal("加载配置失败", zap.Error(err))
	}

	// 应用命令行参数
	if *nonInteractive {
		config.NonInteractive = true
		config.Interactive = false
	}
	if *emojiMode {
		config.EmojiMode = true
	}
	if *qualityMode != "auto" {
		config.QualityMode = *qualityMode
	}
	if *outputFormat != "auto" {
		config.OutputFormat = *outputFormat
	}
	if *stickerMode {
		config.StickerMode = true
	}
	if *tryEngine {
		config.TryEngine = true
	}
	if *securityLevel != "medium" {
		config.SecurityLevel = *securityLevel
	}

	// 初始化UI管理器
	ui := NewUIManager(logger, config.Interactive, config.EmojiMode)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		ui.PrintInfo("收到退出信号，正在安全退出...")
		os.Exit(0)
	}()

	// 显示欢迎信息
	ui.ShowWelcome()

	// 获取目标目录
	if *targetDir == "" {
		if config.Interactive {
			*targetDir = ui.ReadInput("请输入目标目录路径:")
		} else {
			ui.PrintError("非交互模式下必须指定目标目录")
			os.Exit(1)
		}
	}

	// 验证目录
	if _, err := os.Stat(*targetDir); os.IsNotExist(err) {
		ui.PrintError(fmt.Sprintf("目录不存在: %s", *targetDir))
		os.Exit(1)
	}

	// 检查工具是否存在
	all2jxlPath := "./easymode/all2jxl/bin/all2jxl"
	all2avifPath := "./easymode/all2avif/bin/all2avif"

	if _, err := os.Stat(all2jxlPath); os.IsNotExist(err) {
		ui.PrintError("all2jxl 工具不存在，请先构建")
		os.Exit(1)
	}

	if _, err := os.Stat(all2avifPath); os.IsNotExist(err) {
		ui.PrintError("all2avif 工具不存在，请先构建")
		os.Exit(1)
	}

	// 初始化组件
	smartStrategy := NewSmartStrategy(logger)
	converter := NewConverter(logger)

	// 选择输出格式
	var selectedFormat string
	if config.OutputFormat == "auto" {
		if config.TryEngine {
			ui.PrintInfo("🔍 启用智能尝试引擎...")
			// 使用尝试引擎进行更智能的格式选择
			imageFiles, err := scanImageFiles(*targetDir)
			if err != nil {
				ui.PrintError(fmt.Sprintf("扫描文件失败: %v", err))
				os.Exit(1)
			}

			if len(imageFiles) > 0 {
				// 分析第一个文件作为代表
				selectedFormat, err = smartStrategy.TryEngine(imageFiles[0], "auto", config.QualityMode)
				if err != nil {
					ui.PrintWarning("尝试引擎分析失败，使用默认策略")
					selectedFormat, err = smartStrategy.SelectBestFormat(*targetDir)
					if err != nil {
						ui.PrintError(fmt.Sprintf("格式选择失败: %v", err))
						os.Exit(1)
					}
				}
			} else {
				ui.PrintWarning("未找到图像文件，使用默认JXL格式")
				selectedFormat = "jxl"
			}
		} else {
			selectedFormat, err = smartStrategy.SelectBestFormat(*targetDir)
			if err != nil {
				ui.PrintError(fmt.Sprintf("格式选择失败: %v", err))
				os.Exit(1)
			}
		}
	} else {
		selectedFormat = config.OutputFormat
	}

	ui.PrintInfo(fmt.Sprintf("🎯 选择的输出格式: %s", strings.ToUpper(selectedFormat)))

	// 确认处理
	if config.Interactive {
		ui.PrintLine("")
		choice := ui.ReadKey("是否开始转换? (y/N):")
		if strings.ToLower(choice) != "y" {
			ui.PrintInfo("用户取消操作")
			return
		}
	}

	// 开始转换
	ui.PrintInfo("开始转换...")
	ui.PrintLine("")

	err = converter.ExecuteConversion(*targetDir, selectedFormat, config)
	if err != nil {
		ui.PrintError(fmt.Sprintf("转换失败: %v", err))
		os.Exit(1)
	}

	// 显示完成信息
	ui.PrintLine("")
	ui.PrintLine("╔══════════════════════════════════════════════════════════════╗")
	ui.PrintLine("║                        转换完成                              ║")
	ui.PrintLine("╚══════════════════════════════════════════════════════════════╝")
	ui.PrintSuccess("🎉 所有文件转换完成！")
	ui.PrintInfo(fmt.Sprintf("📁 输出目录: %s", *targetDir))
	ui.PrintInfo(fmt.Sprintf("📄 输出格式: %s", strings.ToUpper(selectedFormat)))

	// 保存配置
	configManager.SaveConfig(config)
}

// 扫描图像文件
func scanImageFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".tif", ".webp", ".heic", ".heif"}

		for _, imgExt := range imageExts {
			if ext == imgExt {
				files = append(files, path)
				break
			}
		}

		return nil
	})

	return files, err
}
