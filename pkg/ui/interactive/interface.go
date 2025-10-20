package interactive

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pixly/pkg/core/types"
	"pixly/pkg/ui/progress"

	"github.com/fatih/color"
	"go.uber.org/zap"
)

// GuardResult 交互保护器结果
type GuardResult struct {
	Value    string
	TimedOut bool
	Error    error
}

// InteractionGuard 简化版交互保护器
type InteractionGuard struct {
	logger          *zap.Logger
	userTimeout     time.Duration
	debugTimeout    time.Duration
	maxRetries      int
	enableDebugExit bool
}

// NewSimpleInteractionGuard 创建简化版交互保护器
func NewSimpleInteractionGuard(logger *zap.Logger, userTimeout time.Duration) *InteractionGuard {
	return &InteractionGuard{
		logger:          logger,
		userTimeout:     userTimeout,
		debugTimeout:    30 * time.Second,
		maxRetries:      3,
		enableDebugExit: true,
	}
}

// SafeChoiceWithCountdown 带倒计时的安全选择
func (ig *InteractionGuard) SafeChoiceWithCountdown(prompt string, validChoices []string, defaultChoice string, countdownSeconds int, operationName string) *GuardResult {
	timeout := time.Duration(countdownSeconds) * time.Second
	timeoutCh := time.After(timeout)
	responseCh := make(chan string, 1)

	// 启动输入goroutine
	go func() {
		fmt.Print(prompt)
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			ig.logger.Warn("用户输入错误", zap.Error(err))
			responseCh <- ""
			return
		}
		responseCh <- strings.TrimSpace(input)
	}()

	// 等待输入或超时
	select {
	case choice := <-responseCh:
		// 验证选择是否有效
		for _, valid := range validChoices {
			if choice == valid {
				return &GuardResult{Value: choice, TimedOut: false, Error: nil}
			}
		}
		// 无效选择，返回默认值
		if choice == "" {
			return &GuardResult{Value: defaultChoice, TimedOut: false, Error: nil}
		}
		return &GuardResult{Value: defaultChoice, TimedOut: false, Error: fmt.Errorf("无效选择: %s", choice)}
	case <-timeoutCh:
		return &GuardResult{Value: defaultChoice, TimedOut: true, Error: nil}
	}
}

// SafeChoice 安全选择（无倒计时）
func (ig *InteractionGuard) SafeChoice(prompt string, validChoices []string, defaultChoice string, operationName string) *GuardResult {
	return ig.SafeChoiceWithCountdown(prompt, validChoices, defaultChoice, int(ig.userTimeout.Seconds()), operationName)
}

// SafeInput 安全输入
func (ig *InteractionGuard) SafeInput(prompt string, operationName string) *GuardResult {
	timeoutCh := time.After(ig.userTimeout)
	responseCh := make(chan string, 1)

	go func() {
		fmt.Print(prompt)
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			ig.logger.Warn("用户输入错误", zap.Error(err))
			responseCh <- ""
			return
		}
		responseCh <- strings.TrimSpace(input)
	}()

	select {
	case input := <-responseCh:
		return &GuardResult{Value: input, TimedOut: false, Error: nil}
	case <-timeoutCh:
		return &GuardResult{Value: "", TimedOut: true, Error: fmt.Errorf("输入超时")}
	}
}

// Interface 用户交互界面
type Interface struct {
	logger   *zap.Logger
	reader   *bufio.Reader
	colorize bool
	// README要求：防卡死机制 - 集成InteractionGuard
	interactionGuard *InteractionGuard // 交互保护器
	userInputTimeout time.Duration     // 用户输入超时时间
	isDebugMode      bool              // 调试模式标志
}

// NewInterface 创建新的用户交互界面
func NewInterface(logger *zap.Logger, useColor bool) *Interface {
	// README要求：检测调试模式并设置相应的超时机制
	isDebug := os.Getenv("DEBUG_MODE") == "true" || os.Getenv("PIXLY_DEBUG") == "true"

	// README要求：调试模式下更短的超时时间
	userTimeout := 60 * time.Second // 普通模式60秒
	if isDebug {
		userTimeout = 30 * time.Second // 调试模式30秒
	}

	// 创建简化版InteractionGuard实例
	interactionGuard := NewSimpleInteractionGuard(logger, userTimeout)

	return &Interface{
		logger:           logger,
		reader:           bufio.NewReader(os.Stdin),
		colorize:         useColor,
		interactionGuard: interactionGuard,
		userInputTimeout: userTimeout,
		isDebugMode:      isDebug,
	}
}

// ShowWelcome 显示欢迎界面
func (ui *Interface) ShowWelcome() {
	// 清屏并显示精美的欢迎界面
	fmt.Print("\033[2J\033[H") // 清屏并回到顶部

	// 精美的ASCII艺术LOGO
	ui.showPixlyLogo()

	// 版本和描述信息
	fmt.Println()
	fmt.Println(ui.styleGradient("✨ 版本 22.0.0-MODULAR-REFACTORED - 智能化媒体优化解决方案 ✨"))
	fmt.Println()

	// 功能特性展示
	ui.showFeatures()

	// 装饰性分割线
	ui.showDivider("🎯 准备开始您的媒体优化之旅")
	fmt.Println()
}

// readInputWithTimeout 带超时的输入读取 - README要求的防卡死机制（使用InteractionGuard）
func (ui *Interface) readInputWithTimeout(prompt string) (string, error) {
	// 使用InteractionGuard进行安全输入
	result := ui.interactionGuard.SafeInput(prompt, "user_input")

	// 处理输入结果
	if result.TimedOut {
		if result.Error != nil {
			return "", fmt.Errorf("输入超时: %w", result.Error)
		}
		return "", fmt.Errorf("用户输入超时")
	}

	if result.Error != nil {
		return "", fmt.Errorf("输入错误: %w", result.Error)
	}

	return result.Value, nil
}

// GetTargetDirectory 获取目标目录 - 增强版，解决路径重复拼接bug
func (ui *Interface) GetTargetDirectory() (string, error) {
	progress.PauseAllProgress()
	defer progress.ResumeAllProgress()

	fmt.Println()
	ui.showDivider("📁 目录选择")
	fmt.Println()

	// 显示帮助信息
	if ui.colorize {
		helpBox := color.New(color.FgBlack, color.BgHiCyan, color.Bold).Sprint(" 📝 说明 ")
		helpText := color.New(color.FgHiCyan).Sprint("请指定要处理的媒体目录")
		fmt.Printf("  %s %s\n\n", helpBox, helpText)
	} else {
		fmt.Println("  📝 请指定要处理的媒体目录")
		fmt.Println()
	}

	// 显示操作方式
	ui.showInputMethods()

	fmt.Println()
	ui.showTip("支持含有中文、Emoji等特殊字符的路径名")
	fmt.Println()

	// 🔧 修复：增加重试机制，最多尝试3次
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		prompt := fmt.Sprintf("📋 目录路径 (尝试 %d/%d): ", attempt, maxAttempts)

		// README要求：使用超时机制防止卡死
		input, err := ui.readInputWithTimeout(ui.stylePrompt(prompt))
		if err != nil {
			if attempt == maxAttempts {
				return "", fmt.Errorf("输入超时，已达最大尝试次数: %w", err)
			}
			ui.ShowError(fmt.Sprintf("⚠️  输入超时，还可尝试 %d 次", maxAttempts-attempt))
			continue
		}

		// 🔧 修复：仔细清理路径，移除引号和多余空格
		path := strings.TrimSpace(input)
		// 移除可能的单引号和双引号包装
		path = strings.Trim(path, "'\"")
		// 再次清理空格
		path = strings.TrimSpace(path)

		if path == "" {
			if attempt == maxAttempts {
				return "", fmt.Errorf("路径不能为空，已达最大尝试次数")
			}
			ui.ShowError(fmt.Sprintf("⚠️  路径不能为空，还可尝试 %d 次", maxAttempts-attempt))
			continue
		}

		// 检查是否包含Unicode字符并显示友好信息
		if ui.containsNonASCII(path) {
			ui.ShowInfo("✅ 支持Unicode/Emoji字符，路径已正确处理")
		}

		// 🔧 修复：正确处理绝对路径，避免重复拼接
		var finalPath string
		if filepath.IsAbs(path) {
			// 已经是绝对路径，直接使用
			finalPath = filepath.Clean(path)
			ui.logger.Debug("使用绝对路径", zap.String("path", finalPath))
		} else {
			// 相对路径，转换为绝对路径
			absPath, err := filepath.Abs(path)
			if err != nil {
				if attempt == maxAttempts {
					return "", fmt.Errorf("无法解析相对路径: %w", err)
				}
				ui.ShowError(fmt.Sprintf("⚠️  无法解析路径，还可尝试 %d 次: %v", maxAttempts-attempt, err))
				continue
			}
			finalPath = absPath
			ui.logger.Debug("转换为绝对路径",
				zap.String("original", path),
				zap.String("absolute", finalPath))
		}

		// 检查目录是否存在和可访问
		stat, err := os.Stat(finalPath)
		if err != nil {
			if os.IsNotExist(err) {
				if attempt == maxAttempts {
					return "", fmt.Errorf("目录不存在: %s", finalPath)
				}
				ui.ShowError(fmt.Sprintf("⚠️  目录不存在: %s，还可尝试 %d 次", filepath.Base(finalPath), maxAttempts-attempt))
				continue
			} else if os.IsPermission(err) {
				if attempt == maxAttempts {
					return "", fmt.Errorf("没有访问权限: %s", finalPath)
				}
				ui.ShowError(fmt.Sprintf("⚠️  没有访问权限: %s，还可尝试 %d 次", filepath.Base(finalPath), maxAttempts-attempt))
				continue
			} else {
				if attempt == maxAttempts {
					return "", fmt.Errorf("无法访问目录: %s (%v)", finalPath, err)
				}
				ui.ShowError(fmt.Sprintf("⚠️  无法访问目录，还可尝试 %d 次: %v", maxAttempts-attempt, err))
				continue
			}
		}

		// 验证这确实是一个目录
		if !stat.IsDir() {
			if attempt == maxAttempts {
				return "", fmt.Errorf("指定路径不是目录: %s", finalPath)
			}
			ui.ShowError(fmt.Sprintf("⚠️  指定路径不是目录: %s，还可尝试 %d 次", filepath.Base(finalPath), maxAttempts-attempt))
			continue
		}

		// 成功！
		ui.logger.Info("用户选择目录",
			zap.String("path", finalPath),
			zap.String("display_name", filepath.Base(finalPath)),
			zap.Int("attempt", attempt))
		ui.ShowSuccess(fmt.Sprintf("🎉 已选择目录：%s", filepath.Base(finalPath)))
		ui.ShowInfo(fmt.Sprintf("📍 完整路径：%s", finalPath))
		return finalPath, nil
	}

	// 如果执行到这里，说明所有尝试都失败了
	return "", fmt.Errorf("已达最大尝试次数，无法获取有效的目录路径")
}

// SelectMode 选择处理模式
func (ui *Interface) SelectMode() (types.AppMode, error) {
	progress.PauseAllProgress()
	defer progress.ResumeAllProgress()

	fmt.Println()
	ui.showDivider("🎯 处理模式选择")
	fmt.Println()

	// 显示模式介绍
	ui.showModeOption("1", "🤖 自动模式+", "智能路由，覆盖大多数场景的最佳选择", []string{
		"高品质文件 → 无损压缩",
		"中等品质文件 → 平衡优化",
		"低品质文件 → 用户决策",
	}, color.FgHiGreen)

	ui.showModeOption("2", "🔥 品质模式", "所有文件强制无损压缩，最大保真度", []string{
		"静图 → JXL 无损",
		"动图 → AVIF 无损",
		"视频 → MOV 重包装",
	}, color.FgHiBlue)

	ui.showModeOption("3", "🚀 表情包模式", "极限压缩，适合网络分享", []string{
		"所有图片 → AVIF 压缩",
		"视频文件 → 跳过",
	}, color.FgHiYellow)

	fmt.Println()
	ui.showTip("建议新手选择 '自动模式+'，它会根据文件情况智能选择最优策略")
	fmt.Println()

	// README要求：使用InteractionGuard防止死循环，带重试限制
	validOptions := []string{"1", "2", "3"}
	defaultChoice := "1" // 默认选择自动模式+

	result := ui.interactionGuard.SafeChoice(
		ui.stylePrompt("⚡ 请选择 (1-3): "),
		validOptions,
		defaultChoice,
		"mode_selection",
	)

	// 处理输入结果
	if result.TimedOut || result.Error != nil {
		// 超时或错误情况下使用默认选择
		ui.logger.Info("模式选择超时或错误，自动选择自动模式+",
			zap.Bool("timed_out", result.TimedOut),
			zap.Error(result.Error))
		ui.ShowSuccess("🎉 已自动选择自动模式+ - 智能优化开始！")
		return types.ModeAutoPlus, nil
	}

	choice := result.Value
	switch choice {
	case "1":
		ui.logger.Info("用户选择自动模式+")
		ui.ShowSuccess("🎉 已选择自动模式+ - 智能优化开始！")
		return types.ModeAutoPlus, nil
	case "2":
		ui.logger.Info("用户选择品质模式")
		ui.ShowSuccess("🔥 已选择品质模式 - 最高品质保证！")
		return types.ModeQuality, nil
	case "3":
		ui.logger.Info("用户选择表情包模式")
		ui.ShowSuccess("🚀 已选择表情包模式 - 极限压缩开始！")
		return types.ModeEmoji, nil
	default:
		// 这种情况不应该发生，因为SafeChoice已经验证了输入
		ui.logger.Warn("意外的选择值", zap.String("choice", choice))
		ui.ShowSuccess("🎉 已自动选择自动模式+ - 智能优化开始！")
		return types.ModeAutoPlus, nil
	}
}

// HandleCorruptedFiles 处理损坏文件决策
func (ui *Interface) HandleCorruptedFiles(corruptedFiles []string) (string, error) {
	progress.PauseAllProgress()
	defer progress.ResumeAllProgress()

	fmt.Println()
	ui.showDivider("⚠️ 损坏文件检测")
	fmt.Println()

	// 显示警告信息
	warningText := fmt.Sprintf("检测到 %d 个可能损坏的文件，这些文件可能导致转换卡死。", len(corruptedFiles))
	if ui.colorize {
		warnBox := color.New(color.FgBlack, color.BgHiYellow, color.Bold).Sprint(" ⚠️  警告 ")
		warnText := color.New(color.FgHiYellow).Sprint(warningText)
		fmt.Printf("  %s %s\n\n", warnBox, warnText)
	} else {
		fmt.Printf("  ⚠️  %s\n\n", warningText)
	}

	// 显示部分损坏文件列表
	ui.showFileList("🖺 问题文件列表：", corruptedFiles, 5)

	fmt.Println()
	ui.showDivider("🎯 处理选项")
	fmt.Println()

	// 显示选项
	ui.showActionOption("1", "🔧 尝试修复", "尝试修复并继续处理", color.FgHiBlue)
	ui.showActionOption("2", "🖫️ 全部删除", "删除所有损坏文件", color.FgHiRed)
	ui.showActionOption("3", "⏹️ 终止任务", "停止本次转换", color.FgHiRed)
	ui.showActionOption("4", "⏭️ 忽略跳过", "跳过这些文件，继续处理其他文件 (推荐)", color.FgHiGreen)

	fmt.Println()
	ui.showTip("建议选择 '忽略跳过' 或 '尝试修复'，避免数据丢失")

	// README要求：倒计时+默认选择"忽略"
	countdownSeconds := 10
	defaultChoice := "4" // README要求的默认选择"忽略"

	// 使用InteractionGuard进行带倒计时的安全输入
	prompt := fmt.Sprintf("⚡ 请选择 (1-4) [%d秒后自动选择忽略]: ", countdownSeconds)
	result := ui.interactionGuard.SafeChoiceWithCountdown(
		ui.stylePrompt(prompt),
		[]string{"1", "2", "3", "4"},
		defaultChoice,
		countdownSeconds,
		"corrupted_files_decision",
	)

	// 处理输入结果
	var choice string
	if result.TimedOut {
		choice = defaultChoice
		fmt.Println()
		fmt.Println(ui.styleWarning("⏰ 超时，自动选择忽略损坏文件"))
	} else if result.Error != nil {
		choice = defaultChoice
		fmt.Println()
		fmt.Println(ui.styleWarning("❌ 输入错误，自动选择忽略损坏文件"))
	} else {
		choice = result.Value
	}

	switch choice {
	case "1":
		ui.logger.Info("用户选择尝试修复损坏文件")
		ui.ShowSuccess("🔧 正在尝试修复损坏文件...")
		return "repair", nil
	case "2":
		ui.logger.Info("用户选择删除损坏文件")
		ui.ShowError("🖫️ 注意：此操作不可逆，请确保已备份")
		return "delete", nil
	case "3":
		ui.logger.Info("用户选择终止任务")
		ui.ShowInfo("⏹️ 任务已终止")
		return "abort", nil
	case "4", "":
		ui.logger.Info("用户选择忽略损坏文件")
		ui.ShowSuccess("⏭️ 已忽略损坏文件，继续处理其他文件")
		return "ignore", nil
	default:
		// 无效选择，使用默认选项
		ui.logger.Warn("无效选择，使用默认选项忽略", zap.String("choice", choice))
		fmt.Println(ui.styleError("❌ 无效选择，自动选择忽略"))
		return "ignore", nil
	}
}

// HandleLowQualityFiles 处理低品质文件决策
func (ui *Interface) HandleLowQualityFiles(lowQualityFiles []string) (string, error) {
	progress.PauseAllProgress()
	defer progress.ResumeAllProgress()

	fmt.Println()
	ui.showDivider("🔍 极低品质文件检测")
	fmt.Println()

	// 显示警告信息
	warningText := fmt.Sprintf("检测到 %d 个极低品质文件，建议谨慎处理。", len(lowQualityFiles))
	if ui.colorize {
		warnBox := color.New(color.FgBlack, color.BgHiMagenta, color.Bold).Sprint(" 🔍 品质 ")
		warnText := color.New(color.FgHiMagenta).Sprint(warningText)
		fmt.Printf("  %s %s\n\n", warnBox, warnText)
	} else {
		fmt.Printf("  🔍 %s\n\n", warningText)
	}

	// 显示部分低品质文件列表
	ui.showFileList("📉 低品质文件列表：", lowQualityFiles, 5)

	fmt.Println()
	ui.showDivider("🎯 处理策略")
	fmt.Println()

	// 显示选项
	ui.showActionOption("1", "⏭️ 跳过忽略", "跳过这些文件，不进行处理 (推荐)", color.FgHiGreen)
	ui.showActionOption("2", "🖫️ 全部删除", "删除所有低品质文件", color.FgHiRed)
	ui.showActionOption("3", "🔧 强制转换", "使用平衡优化模式强制转换", color.FgHiYellow)
	ui.showActionOption("4", "🚀 表情包模式", "使用表情包模式处理", color.FgHiMagenta)

	fmt.Println()
	ui.showTip("低品质文件可能不适合进一步压缩，建议选择 '跳过忽略'")

	// 倒计时
	timeout := 5 * time.Second
	timeoutCh := time.After(timeout)
	responseCh := make(chan string, 1)

	go func() {
		input, _ := ui.readInputWithTimeout(ui.stylePrompt("⚡ 请选择 (1-4) [5秒后自动选择1]: "))
		responseCh <- input
	}()

	select {
	case choice := <-responseCh:
		switch choice {
		case "1", "":
			ui.logger.Info("用户选择跳过低品质文件")
			ui.ShowSuccess("⏭️ 已跳过低品质文件")
			return "skip", nil
		case "2":
			ui.logger.Info("用户选择删除低品质文件")
			ui.ShowError("🖫️ 注意：此操作不可逆，请确保已备份")
			return "delete", nil
		case "3":
			ui.logger.Info("用户选择强制转换低品质文件")
			ui.ShowSuccess("🔧 将使用平衡优化模式处理")
			return "force", nil
		case "4":
			ui.logger.Info("用户选择用表情包模式处理低品质文件")
			ui.ShowSuccess("🚀 将使用表情包模式处理")
			return "emoji", nil
		default:
			fmt.Println(ui.styleError("❌ 无效选择，自动选择跳过"))
			return "skip", nil
		}
	case <-timeoutCh:
		fmt.Println()
		fmt.Println(ui.styleWarning("⏰ 超时，自动选择跳过"))
		ui.logger.Info("低品质文件处理超时，自动选择跳过")
		return "skip", nil
	}
}

// ShowProcessingStart 显示处理开始信息
func (ui *Interface) ShowProcessingStart(mode types.AppMode, totalFiles int) {
	ui.showProcessingSummary(mode, totalFiles)
}

// ConfirmContinue 确认是否继续
func (ui *Interface) ConfirmContinue(message string) bool {
	progress.PauseAllProgress()
	defer progress.ResumeAllProgress()

	fmt.Println()
	fmt.Println(ui.styleWarning(message))
	fmt.Print(ui.stylePrompt("是否继续？(y/N): "))

	input, err := ui.reader.ReadString('\n')
	if err != nil {
		return false
	}

	response := strings.ToLower(strings.TrimSpace(input))
	return response == "y" || response == "yes"
}

// ShowMainMenu 显示主菜单
func (ui *Interface) ShowMainMenu() int {
	progress.PauseAllProgress()
	defer progress.ResumeAllProgress()

	fmt.Println()
	ui.showDivider("📋 主菜单 - 选择您的操作")
	fmt.Println()

	// 使用更美观的选项展示
	ui.showMenuOption("1", "🚀 开始处理", "选择目录并开始媒体转换之旅", color.FgHiGreen)
	ui.showMenuOption("2", "📦 缓存管理", "查看和管理JSON文件系统缓存", color.FgHiBlue)
	ui.showMenuOption("3", "💪 退出程序", "感谢使用 Pixly，期待下次相遇", color.FgHiRed)

	fmt.Println()
	ui.showTip("✨ 小贴士：支持拖拽文件夹到窗口，也可以直接输入路径 ✨")
	fmt.Println()

	// 防死循环机制：最多重试5次
	maxRetries := 5
	for retryCount := 0; retryCount < maxRetries; retryCount++ {
		// README要求：使用超时机制防止卡死
		input, err := ui.readInputWithTimeout(ui.stylePrompt("⚡ 请选择 (1-3): "))
		if err != nil {
			// 超时情况下默认选择退出
			ui.logger.Info("输入超时，自动选择退出")
			return 3
		}

		choice := strings.TrimSpace(input)

		switch choice {
		case "1":
			ui.ShowSuccess("🎉 开始媒体优化之旅！")
			return 1
		case "2":
			ui.ShowSuccess("📦 进入缓存管理系统")
			return 2
		case "3":
			ui.showGoodbye()
			return 3
		default:
			if retryCount < maxRetries-1 {
				fmt.Println(ui.styleError(fmt.Sprintf("❌ 无效选择 '%s'，请输入 1、2 或 3 (剩余重试次数: %d)", choice, maxRetries-retryCount-1)))
				time.Sleep(500 * time.Millisecond) // 短暂延迟避免刷屏
			} else {
				fmt.Println(ui.styleError("❌ 重试次数已用完，自动退出程序"))
				ui.logger.Warn("主菜单重试次数达到上限，自动退出")
				return 3
			}
		}
	}

	// 理论上不应该到达这里，但为了安全起见
	ui.logger.Warn("主菜单异常退出")
	return 3
}

// ShowEmbeddedFFmpegNote 显示嵌入式FFmpeg说明
func (ui *Interface) ShowEmbeddedFFmpegNote(note string) bool {
	if note == "" {
		return true
	}

	progress.PauseAllProgress()
	defer progress.ResumeAllProgress()

	fmt.Println()
	fmt.Println(ui.styleWarning("📦 嵌入式 FFmpeg 说明"))
	fmt.Println(ui.styleSubtle(note))
	fmt.Print(ui.stylePrompt("按 Enter 继续..."))

	ui.reader.ReadString('\n')
	return true
}

// ShowError 显示错误信息
func (ui *Interface) ShowError(message string) {
	fmt.Println()
	fmt.Println(ui.styleError("❌ " + message))
	fmt.Println()
}

// ShowSuccess 显示成功信息
func (ui *Interface) ShowSuccess(message string) {
	fmt.Println()
	fmt.Println(ui.styleSuccess("✅ " + message))
	fmt.Println()
}

// ShowInfo 显示信息
func (ui *Interface) ShowInfo(message string) {
	fmt.Println(ui.styleInfo("ℹ️  " + message))
}

// ShowWarning 显示警告信息
func (ui *Interface) ShowWarning(message string) {
	fmt.Println()
	fmt.Println(ui.styleWarning("⚠️  " + message))
	fmt.Println()
}

// 样式方法
func (ui *Interface) styleTitle(text string) string {
	if !ui.colorize {
		return text
	}
	return color.New(color.FgCyan, color.Bold).Sprint(text)
}

func (ui *Interface) styleBold(text string) string {
	if !ui.colorize {
		return text
	}
	return color.New(color.Bold).Sprint(text)
}

func (ui *Interface) styleSubtle(text string) string {
	if !ui.colorize {
		return text
	}
	return color.New(color.FgHiBlack).Sprint(text)
}

func (ui *Interface) stylePrompt(text string) string {
	if !ui.colorize {
		return text
	}
	return color.New(color.FgHiBlue).Sprint(text)
}

func (ui *Interface) styleOption(number, title, description string) string {
	if !ui.colorize {
		return fmt.Sprintf("%s. %s - %s", number, title, description)
	}

	numberStyle := color.New(color.FgHiGreen, color.Bold).Sprint(number)
	titleStyle := color.New(color.FgWhite, color.Bold).Sprint(title)
	descStyle := color.New(color.FgHiBlack).Sprint(description)

	return fmt.Sprintf("%s. %s - %s", numberStyle, titleStyle, descStyle)
}

func (ui *Interface) styleDetail(text string) string {
	if !ui.colorize {
		return text
	}
	return color.New(color.FgHiBlack).Sprint(text)
}

func (ui *Interface) styleError(text string) string {
	if !ui.colorize {
		return text
	}
	return color.New(color.FgRed, color.Bold).Sprint(text)
}

func (ui *Interface) styleSuccess(text string) string {
	if !ui.colorize {
		return text
	}
	return color.New(color.FgGreen, color.Bold).Sprint(text)
}

func (ui *Interface) styleWarning(text string) string {
	if !ui.colorize {
		return text
	}
	return color.New(color.FgYellow, color.Bold).Sprint(text)
}

func (ui *Interface) styleInfo(text string) string {
	if !ui.colorize {
		return text
	}
	return color.New(color.FgBlue).Sprint(text)
}

// showPixlyLogo 显示Pixly的ASCII艺术LOGO - 增强版彩色渐变效果
func (ui *Interface) showPixlyLogo() {
	logo := `
┌────────────────────────────────────────────────────────────────────────┐
│     ██████╗ ██╗██╗  ██╗██╗     ██╗   ██╗                 │
│     ██╔══██╗██║╚██╗██╔╝██║     ╚██╗ ██╔╝                 │
│     ██████╔╝██║ ╚███╔╝ ██║      ╚███╔╝                  │
│     ██╔═══╝ ██║  ╚██║  ██║       ╚██║                   │
│     ██║     ██║   ██║  ███████╗██║    🎨 媒体优化大师 │
│     ╚═╝     ╚═╝   ╚═╝  ╚══════╝╚═╝                   │
└────────────────────────────────────────────────────────────────────────┘`

	if ui.colorize {
		// 增强版彩色渲染LOGO - 多重渐变效果
		lines := strings.Split(logo, "\n")
		for i, line := range lines {
			switch {
			case i == 1: // 顶部边框 - 亮青色
				fmt.Println(color.New(color.FgHiCyan, color.Bold).Sprint(line))
			case i == 2: // 第一行LOGO - 紫色
				fmt.Println(color.New(color.FgHiMagenta, color.Bold).Sprint(line))
			case i == 3: // 第二行LOGO - 蓝紫色
				fmt.Println(color.New(color.FgMagenta, color.Bold).Sprint(line))
			case i == 4: // 第三行LOGO - 蓝色
				fmt.Println(color.New(color.FgHiBlue, color.Bold).Sprint(line))
			case i == 5: // 第四行LOGO - 青蓝色
				fmt.Println(color.New(color.FgBlue, color.Bold).Sprint(line))
			case i == 6: // 特殊行（带emoji）- 绿色高亮
				fmt.Println(color.New(color.FgHiGreen, color.Bold).Sprint(line))
			case i == 7: // 第六行LOGO - 青色
				fmt.Println(color.New(color.FgCyan, color.Bold).Sprint(line))
			case i == 8: // 底部边框 - 亮青色
				fmt.Println(color.New(color.FgHiCyan, color.Bold).Sprint(line))
			default:
				fmt.Println(color.New(color.FgWhite).Sprint(line))
			}
		}
	} else {
		fmt.Print(logo)
	}
}

// showFeatures 显示功能特性
func (ui *Interface) showFeatures() {
	features := []struct {
		icon        string
		title       string
		description string
		color       *color.Color
	}{
		{"🚀", "智能自动", "智能识别最优转换策略", color.New(color.FgHiGreen, color.Bold)},
		{"🏆", "无损品质", "保持原始质量的无损压缩", color.New(color.FgHiBlue, color.Bold)},
		{"⚡", "闪电处理", "高性能并发处理引擎", color.New(color.FgHiYellow, color.Bold)},
		{"🛡️", "安全可靠", "完整的备份与恢复机制", color.New(color.FgHiRed, color.Bold)},
	}

	fmt.Println(ui.styleGradient("✨ 核心特性 ✨"))
	fmt.Println()

	for _, feature := range features {
		if ui.colorize {
			fmt.Printf("   %s %s %s\n",
				feature.icon,
				feature.color.Sprint(feature.title),
				color.New(color.FgHiBlack).Sprint("- "+feature.description))
		} else {
			fmt.Printf("   %s %s - %s\n", feature.icon, feature.title, feature.description)
		}
	}
}

// showDivider 显示装饰性分割线
func (ui *Interface) showDivider(text string) {
	width := 72
	textLen := len([]rune(text)) // 正确处理unicode字符
	padding := (width - textLen) / 2

	if ui.colorize {
		line := strings.Repeat("─", padding) + " " + text + " " + strings.Repeat("─", padding)
		fmt.Println(color.New(color.FgHiCyan, color.Bold).Sprint(line))
	} else {
		line := strings.Repeat("-", padding) + " " + text + " " + strings.Repeat("-", padding)
		fmt.Println(line)
	}
}

// styleGradient 创建渐变效果文本
func (ui *Interface) styleGradient(text string) string {
	if !ui.colorize {
		return text
	}
	// 使用彩色组合创建渐变效果
	return color.New(color.FgHiMagenta, color.Bold, color.BgBlack).Sprint(text)
}

// styleHighlight 高亮显示
func (ui *Interface) styleHighlight(text string) string {
	if !ui.colorize {
		return text
	}
	return color.New(color.FgHiWhite, color.Bold, color.BgMagenta).Sprint(" " + text + " ")
}

// styleEmoji 带emoji的样式
func (ui *Interface) styleEmoji(emoji, text string) string {
	if !ui.colorize {
		return emoji + " " + text
	}
	return emoji + " " + color.New(color.FgHiWhite, color.Bold).Sprint(text)
}

// 辅助方法
func (ui *Interface) cleanPath(path string) string {
	// 清理路径中的特殊字符和引号
	path = strings.Trim(path, "\"'")
	path = strings.TrimSpace(path)

	// 处理可能的转义字符
	path = strings.ReplaceAll(path, "\\ ", " ")

	return path
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// showMenuOption 显示菜单选项
func (ui *Interface) showMenuOption(number, title, description string, colorAttr color.Attribute) {
	if !ui.colorize {
		fmt.Printf("  [%s] %s\n      %s\n\n", number, title, description)
		return
	}

	// 美化的选项展示
	numberBox := color.New(colorAttr, color.Bold, color.BgWhite).Sprintf(" %s ", number)
	titleStyled := color.New(colorAttr, color.Bold).Sprint(title)
	descStyled := color.New(color.FgHiBlack).Sprint(description)

	fmt.Printf("  %s %s\n      %s\n\n", numberBox, titleStyled, descStyled)
}

// showTip 显示小贴士
func (ui *Interface) showTip(text string) {
	if !ui.colorize {
		fmt.Printf("💡 %s\n", text)
		return
	}

	padding := "  "
	tipBox := color.New(color.FgBlack, color.BgHiYellow, color.Bold).Sprint(" 💡 TIP ")
	tipText := color.New(color.FgHiYellow).Sprint(text)

	fmt.Printf("%s%s %s\n", padding, tipBox, tipText)
}

// showGoodbye 显示告别界面
func (ui *Interface) showGoodbye() {
	fmt.Println()
	ui.showDivider("👋 告别")
	fmt.Println()

	goodbyeMsg := `
    ✨ 感谢使用 Pixly 媒体优化工具！✨
    
    🚀 希望我们的工具帮助您提升了媒体质量
    🎆 期待您的下次使用，祝您工作順利！
    
    💫 记得关注我们的更新哦~ 💫`

	if ui.colorize {
		lines := strings.Split(goodbyeMsg, "\n")
		for _, line := range lines {
			fmt.Println(color.New(color.FgHiCyan).Sprint(line))
		}
	} else {
		fmt.Print(goodbyeMsg)
	}

	fmt.Println()
	ui.showDivider("🌈 美好的一天")
	fmt.Println()
}

// showProcessingSummary 显示处理概要
func (ui *Interface) showProcessingSummary(mode types.AppMode, totalFiles int) {
	fmt.Println()
	ui.showDivider("🚀 处理概要")
	fmt.Println()

	if ui.colorize {
		modeColor := color.New(color.FgHiMagenta, color.Bold)
		countColor := color.New(color.FgHiGreen, color.Bold)

		fmt.Printf("  🎯 处理模式： %s\n", modeColor.Sprint(mode.String()))
		fmt.Printf("  📁 文件数量： %s 个\n", countColor.Sprint(totalFiles))
	} else {
		fmt.Printf("  模式: %s\n", mode.String())
		fmt.Printf("  文件数量: %d 个\n", totalFiles)
	}

	fmt.Println()
	ui.showTip("处理过程中请保持耐心，程序会智能优化您的媒体文件")
	fmt.Println()
}

// showModeOption 显示模式选项
func (ui *Interface) showModeOption(number, title, description string, details []string, colorAttr color.Attribute) {
	if !ui.colorize {
		fmt.Printf("[%s] %s\n    %s\n", number, title, description)
		for _, detail := range details {
			fmt.Printf("    • %s\n", detail)
		}
		fmt.Println()
		return
	}

	// 美化的模式选项展示
	numberBox := color.New(colorAttr, color.Bold, color.BgWhite).Sprintf(" %s ", number)
	titleStyled := color.New(colorAttr, color.Bold).Sprint(title)
	descStyled := color.New(color.FgHiBlack).Sprint(description)

	fmt.Printf("  %s %s\n", numberBox, titleStyled)
	fmt.Printf("      %s\n", descStyled)

	// 显示详细信息
	for _, detail := range details {
		detailStyled := color.New(color.FgHiBlack).Sprint("• " + detail)
		fmt.Printf("      %s\n", detailStyled)
	}
	fmt.Println()
}

// showFileList 显示文件列表
func (ui *Interface) showFileList(title string, files []string, maxCount int) {
	if len(files) == 0 {
		return
	}

	if ui.colorize {
		titleStyled := color.New(color.FgHiCyan, color.Bold).Sprint(title)
		fmt.Printf("  %s\n", titleStyled)
	} else {
		fmt.Printf("  %s\n", title)
	}

	showCount := min(len(files), maxCount)
	for i := 0; i < showCount; i++ {
		fileName := filepath.Base(files[i])
		if ui.colorize {
			fileStyled := color.New(color.FgHiBlack).Sprint("• " + fileName)
			fmt.Printf("     %s\n", fileStyled)
		} else {
			fmt.Printf("     • %s\n", fileName)
		}
	}

	if len(files) > showCount {
		remaining := len(files) - showCount
		if ui.colorize {
			remainingStyled := color.New(color.FgHiBlack).Sprintf("     ... 还有 %d 个文件", remaining)
			fmt.Println(remainingStyled)
		} else {
			fmt.Printf("     ... 还有 %d 个文件\n", remaining)
		}
	}
}

// showActionOption 显示动作选项
func (ui *Interface) showActionOption(number, title, description string, colorAttr color.Attribute) {
	if !ui.colorize {
		fmt.Printf("  [%s] %s - %s\n", number, title, description)
		return
	}

	numberBox := color.New(colorAttr, color.Bold, color.BgWhite).Sprintf(" %s ", number)
	titleStyled := color.New(colorAttr, color.Bold).Sprint(title)
	descStyled := color.New(color.FgHiBlack).Sprint(description)

	fmt.Printf("  %s %s - %s\n", numberBox, titleStyled, descStyled)
}

// showInputMethods 显示输入方式
func (ui *Interface) showInputMethods() {
	methods := []struct {
		icon        string
		method      string
		description string
	}{
		{"🗂️", "拖拽方式", "直接拖拽文件夹到窗口中"},
		{"⌨️", "输入方式", "手动输入完整的目录路径"},
		{"📎", "粘贴方式", "复制路径后直接粘贴"},
	}

	for _, method := range methods {
		if ui.colorize {
			iconStyled := method.icon
			methodStyled := color.New(color.FgHiGreen, color.Bold).Sprint(method.method)
			descStyled := color.New(color.FgHiBlack).Sprint(method.description)
			fmt.Printf("    %s %s - %s\n", iconStyled, methodStyled, descStyled)
		} else {
			fmt.Printf("    %s %s - %s\n", method.icon, method.method, method.description)
		}
	}
}

// containsNonASCII 检查字符串是否包含非ASCII字符（如中文、Emoji等）
func (ui *Interface) containsNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}
