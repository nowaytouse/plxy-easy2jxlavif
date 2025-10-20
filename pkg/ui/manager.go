package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"pixly/pkg/core/config"
	"pixly/pkg/core/types"
	"pixly/pkg/ui/interactive"
	"pixly/pkg/ui/progress"

	"github.com/fatih/color"
	"go.uber.org/zap"
)

// Manager UI管理器 - 统一管理所有用户界面逻辑
type Manager struct {
	logger        *zap.Logger
	userInterface *interactive.Interface
	progress      *progress.ProgressManager
	reader        *bufio.Reader
	colorize      bool
	debugMode     bool
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewManager 创建新的UI管理器
func NewManager(logger *zap.Logger, enableColor bool) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		logger:        logger,
		userInterface: interactive.NewInterface(logger, enableColor),
		progress:      progress.NewProgressManager(logger),
		reader:        bufio.NewReader(os.Stdin),
		colorize:      enableColor,
		debugMode:     os.Getenv("DEBUG_MODE") == "true" || os.Getenv("PIXLY_DEBUG") == "true",
		ctx:           ctx,
		cancel:        cancel,
	}
}

// SessionOptions 会话选项
type SessionOptions struct {
	EnableBatchConversion bool
	EnableCacheManagement bool
	ShowInstructions      bool
}

// SessionResult 会话结果
type SessionResult struct {
	Action        string // "convert", "cache", "exit", "error"
	Config        *config.Config
	SelectedPaths []string
	Error         error
}

// ShowWelcome 显示欢迎界面
func (m *Manager) ShowWelcome() {
	m.userInterface.ShowWelcome()
}

// RunInteractiveSession 运行交互式会话
func (m *Manager) RunInteractiveSession(tools types.ToolCheckResults, opts *SessionOptions) (*SessionResult, error) {
	if opts == nil {
		opts = &SessionOptions{
			EnableBatchConversion: true,
			EnableCacheManagement: true,
			ShowInstructions:      true,
		}
	}

	noInputCount := 0
	for {
		// 暂停所有进度显示
		m.progress.PauseAll()

		// 显示主菜单
		choice := m.showMainMenu(opts)

		// 恢复进度显示
		m.progress.ResumeAll()

		// 处理空输入
		if choice == "" {
			noInputCount++
			if noInputCount >= 3 {
				m.showError("❌ 3次未输入，程序将自动退出")
				return &SessionResult{Action: "exit"}, nil
			}
			m.showWarning("⚠️ 未检测到输入，请重新选择")
			continue
		}

		// 重置计数器
		noInputCount = 0

		// 处理用户选择
		switch choice {
		case "1":
			// 批量转换
			result, err := m.handleBatchConversion(tools)
			if err != nil {
				m.showError(fmt.Sprintf("转换配置失败: %v", err))
				continue
			}
			return result, nil

		case "2":
			// 缓存管理
			if opts.EnableCacheManagement {
				m.handleCacheManagement()
				continue
			}
			m.showError("❌ 缓存管理功能未启用")

		case "3":
			// 显示说明
			if opts.ShowInstructions {
				m.showEmbeddedFFmpegInstructions()
				continue
			}
			// 退出程序
			m.showGoodbye()
			return &SessionResult{Action: "exit"}, nil

		case "4":
			// 退出程序
			m.showGoodbye()
			return &SessionResult{Action: "exit"}, nil

		default:
			m.showError("❌ 无效选择，请输入有效的选项")
		}
	}
}

// showMainMenu 显示主菜单
func (m *Manager) showMainMenu(opts *SessionOptions) string {
	fmt.Println()
	fmt.Println(m.styleBold("🚀 Pixly 媒体转换工具"))

	// 动态菜单项
	menuItems := []string{"1. 🔄 转换核心（直接进入转换流程）"}

	if opts.EnableCacheManagement {
		menuItems = append(menuItems, "2. 📦 缓存管理（查看和管理JSON文件系统缓存）")
	}

	if opts.ShowInstructions {
		menuItems = append(menuItems, fmt.Sprintf("%d. ℹ️ 查看嵌入式FFmpeg使用说明", len(menuItems)+1))
	}

	menuItems = append(menuItems, fmt.Sprintf("%d. 🚪 退出程序", len(menuItems)+1))

	// 显示菜单项
	for _, item := range menuItems {
		fmt.Println(item)
	}

	fmt.Print("\n请选择操作: ")

	input, _ := m.reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// handleBatchConversion 处理批量转换
func (m *Manager) handleBatchConversion(tools types.ToolCheckResults) (*SessionResult, error) {
	// 获取目标目录
	targetDir, err := m.userInterface.GetTargetDirectory()
	if err != nil {
		return nil, fmt.Errorf("获取目标目录失败: %w", err)
	}

	// 选择处理模式
	mode, err := m.userInterface.SelectMode()
	if err != nil {
		return nil, fmt.Errorf("选择模式失败: %w", err)
	}

	// 创建配置
	cfg := config.DefaultConfig() // 使用标准化的默认配置
	cfg.Mode = mode.String()
	cfg.TargetDir = targetDir

	// 确保配置完整性
	config.NormalizeConfig(cfg)

	return &SessionResult{
		Action:        "convert",
		Config:        cfg,
		SelectedPaths: []string{targetDir},
	}, nil
}

// handleCacheManagement 处理缓存管理
func (m *Manager) handleCacheManagement() {
	m.showInfo("📦 缓存管理功能")
	m.showInfo("暂未实现，将在后续版本中提供")
}

// showEmbeddedFFmpegInstructions 显示FFmpeg说明
func (m *Manager) showEmbeddedFFmpegInstructions() {
	fmt.Println()
	fmt.Println(m.styleBold("📚 嵌入式FFmpeg使用说明"))
	fmt.Println()

	instructions := `
🔧 FFmpeg 版本要求：
  • 开发版：v8.0+ (推荐用于新功能)
  • 稳定版：v7.11+ (用于稳定处理)

📦 内嵌版本检测：
  • 程序会自动检测内嵌的FFmpeg版本
  • 优先使用内嵌版本，确保兼容性

⚙️ 手动安装：
  brew install ffmpeg
  
🎯 建议配置：
  • 同时安装开发版和稳定版
  • 开发版用于新格式支持
  • 稳定版用于可靠的批量处理

💡 故障排除：
  • 如果检测失败，请检查PATH环境变量
  • 确保FFmpeg可以在终端中直接调用
  • 重新安装Homebrew版本：brew reinstall ffmpeg
`

	fmt.Println(instructions)
	fmt.Println()
	m.showPrompt("按回车键返回主菜单...")
	m.reader.ReadString('\n')
}

// 工具方法

// showInfo 显示信息
func (m *Manager) showInfo(message string) {
	if m.colorize {
		fmt.Println(color.New(color.FgHiCyan).Sprint("ℹ️ " + message))
	} else {
		fmt.Println("ℹ️ " + message)
	}
}

// showSuccess 显示成功消息
func (m *Manager) showSuccess(message string) {
	if m.colorize {
		fmt.Println(color.New(color.FgHiGreen).Sprint("✅ " + message))
	} else {
		fmt.Println("✅ " + message)
	}
}

// showWarning 显示警告
func (m *Manager) showWarning(message string) {
	if m.colorize {
		fmt.Println(color.New(color.FgHiYellow).Sprint("⚠️ " + message))
	} else {
		fmt.Println("⚠️ " + message)
	}
}

// showError 显示错误
func (m *Manager) showError(message string) {
	if m.colorize {
		fmt.Println(color.New(color.FgHiRed).Sprint("❌ " + message))
	} else {
		fmt.Println("❌ " + message)
	}
}

// showGoodbye 显示告别界面
func (m *Manager) showGoodbye() {
	fmt.Println()
	if m.colorize {
		fmt.Println(color.New(color.FgHiGreen).Sprint("👋 感谢使用 Pixly，再见！"))
	} else {
		fmt.Println("👋 感谢使用 Pixly，再见！")
	}
}

// showPrompt 显示提示
func (m *Manager) showPrompt(prompt string) {
	if m.colorize {
		fmt.Print(color.New(color.FgHiCyan).Sprint(prompt))
	} else {
		fmt.Print(prompt)
	}
}

// styleBold 粗体样式
func (m *Manager) styleBold(text string) string {
	if m.colorize {
		return color.New(color.Bold).Sprint(text)
	}
	return text
}

// GetProgressManager 获取进度管理器
func (m *Manager) GetProgressManager() *progress.ProgressManager {
	return m.progress
}

// GetInterface 获取交互界面
func (m *Manager) GetInterface() *interactive.Interface {
	return m.userInterface
}

// Shutdown 关闭UI管理器
func (m *Manager) Shutdown() error {
	if m.cancel != nil {
		m.cancel()
	}

	if m.progress != nil {
		m.progress.Stop()
	}

	m.logger.Info("UI管理器已关闭")
	return nil
}

// EnableDebugMode 启用调试模式
func (m *Manager) EnableDebugMode() {
	m.debugMode = true
	m.logger.Info("UI管理器已启用调试模式")
}

// GetDebugMode 获取调试模式状态
func (m *Manager) GetDebugMode() bool {
	return m.debugMode
}

// PauseProgress 暂停进度显示
func (m *Manager) PauseProgress() {
	if m.progress != nil {
		m.progress.PauseAll()
	}
}

// ResumeProgress 恢复进度显示
func (m *Manager) ResumeProgress() {
	if m.progress != nil {
		m.progress.ResumeAll()
	}
}
