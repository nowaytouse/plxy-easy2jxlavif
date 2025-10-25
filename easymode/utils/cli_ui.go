// utils/cli_ui.go - 归档工具统一CLI UI框架
// 提供简易拖入式交互界面 + 强大安全检查

package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// CLIConfig CLI配置
type CLIConfig struct {
	ToolName            string   // 工具名称
	ToolVersion         string   // 工具版本
	SupportedExts       []string // 支持的文件扩展名
	OutputFormat        string   // 输出格式
	DefaultWorkers      int      // 默认并发数
	AllowNonInteractive bool     // 允许非交互模式
}

// InteractiveMode 交互模式
type InteractiveMode struct {
	Config *CLIConfig
}

// NewInteractiveMode 创建交互模式
func NewInteractiveMode(config *CLIConfig) *InteractiveMode {
	return &InteractiveMode{Config: config}
}

// ShowBanner 显示工具横幅
func (im *InteractiveMode) ShowBanner() {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Printf("║                                                               ║\n")
	fmt.Printf("║   🎨 %s v%s                                      ║\n",
		padString(im.Config.ToolName, 20), padString(im.Config.ToolVersion, 10))
	fmt.Printf("║                                                               ║\n")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println("")
}

// PromptForDirectory 提示用户输入目录（支持拖拽）
func (im *InteractiveMode) PromptForDirectory() (string, error) {
	fmt.Println("📁 请拖入要处理的文件夹，然后按回车键：")
	fmt.Println("   （或直接输入路径）")
	fmt.Print("\n路径: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("读取输入失败: %v", err)
	}

	// 清理输入
	path := strings.TrimSpace(input)

	// macOS终端拖拽路径会转义特殊字符，需要反转义
	path = unescapeShellPath(path)

	// 验证路径
	if path == "" {
		return "", fmt.Errorf("路径不能为空")
	}

	return path, nil
}

// PerformSafetyCheck 执行安全检查
func (im *InteractiveMode) PerformSafetyCheck(targetPath string) error {
	fmt.Println("")
	fmt.Println("🔍 正在执行安全检查...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 1. 检查路径是否存在
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("无法解析路径: %v", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("路径不存在: %s", absPath)
		}
		return fmt.Errorf("无法访问路径: %v", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("路径不是文件夹: %s", absPath)
	}

	fmt.Printf("  ✅ 路径存在: %s\n", absPath)

	// 2. 检查是否为系统关键目录
	if isCriticalSystemPath(absPath) {
		return fmt.Errorf("禁止访问系统关键目录: %s", absPath)
	}

	fmt.Printf("  ✅ 路径安全: 非系统目录\n")

	// 3. 检查读写权限
	testFile := filepath.Join(absPath, ".pixly_permission_test")
	if file, err := os.Create(testFile); err != nil {
		return fmt.Errorf("目录没有写入权限: %v", err)
	} else {
		file.Close()
		os.Remove(testFile)
		fmt.Printf("  ✅ 权限验证: 可读可写\n")
	}

	// 4. 检查磁盘空间
	if freeSpace, totalSpace, err := getDiskSpace(absPath); err == nil {
		freeGB := float64(freeSpace) / 1024 / 1024 / 1024
		totalGB := float64(totalSpace) / 1024 / 1024 / 1024
		ratio := float64(freeSpace) / float64(totalSpace) * 100

		fmt.Printf("  💾 磁盘空间: %.1fGB / %.1fGB (%.1f%% 可用)\n", freeGB, totalGB, ratio)

		if ratio < 10 {
			return fmt.Errorf("磁盘空间不足（剩余%.1f%%），建议至少保留10%%", ratio)
		} else if ratio < 20 {
			fmt.Printf("  ⚠️  磁盘空间较少（剩余%.1f%%），建议谨慎处理\n", ratio)
		}
	}

	// 5. 检查是否为敏感目录
	if isSensitiveDirectory(absPath) {
		fmt.Printf("  ⚠️  敏感目录警告: %s\n", absPath)
		fmt.Print("\n  是否继续处理此目录？(输入 yes 确认): ")

		reader := bufio.NewReader(os.Stdin)
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(strings.ToLower(confirm))

		if confirm != "yes" && confirm != "y" {
			return fmt.Errorf("用户取消操作")
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ 安全检查通过！")
	fmt.Println("")

	return nil
}

// isCriticalSystemPath 检查是否为系统关键目录
func isCriticalSystemPath(path string) bool {
	criticalPaths := []string{
		"/System",
		"/Library/System",
		"/private",
		"/usr/bin",
		"/usr/sbin",
		"/bin",
		"/sbin",
		"/var/root",
		"/etc",
		"/dev",
		"/proc",
		"/Applications/Utilities",
		"/System/Library",
	}

	if runtime.GOOS == "darwin" {
		criticalPaths = append(criticalPaths, "/Volumes/Macintosh HD/System")
	}

	for _, critical := range criticalPaths {
		if strings.HasPrefix(path, critical) {
			return true
		}
	}

	return false
}

// isSensitiveDirectory 检查是否为敏感目录（需要确认）
func isSensitiveDirectory(path string) bool {
	sensitivePaths := []string{
		"/Applications",
		"/Library",
		"/usr",
		"/var",
	}

	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		sensitivePaths = append(sensitivePaths, homeDir)
	}

	for _, sensitive := range sensitivePaths {
		if path == sensitive {
			return true
		}
	}

	return false
}

// getDiskSpace 获取磁盘空间信息
func getDiskSpace(path string) (free, total uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}

	// 计算可用空间和总空间
	free = stat.Bavail * uint64(stat.Bsize)
	total = stat.Blocks * uint64(stat.Bsize)

	return free, total, nil
}

// unescapeShellPath 反转义Shell路径（macOS拖拽）
func unescapeShellPath(path string) string {
	// macOS终端拖拽会转义空格和特殊字符
	path = strings.ReplaceAll(path, "\\ ", " ")
	path = strings.ReplaceAll(path, "\\!", "!")
	path = strings.ReplaceAll(path, "\\(", "(")
	path = strings.ReplaceAll(path, "\\)", ")")
	path = strings.ReplaceAll(path, "\\[", "[")
	path = strings.ReplaceAll(path, "\\]", "]")
	path = strings.ReplaceAll(path, "\\&", "&")
	path = strings.ReplaceAll(path, "\\$", "$")

	// 移除可能的引号
	path = strings.Trim(path, "\"'")

	return path
}

// padString 填充字符串到指定长度
func padString(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	return s + strings.Repeat(" ", length-len(s))
}

// ShowProgress 显示简单进度
func ShowProgress(current, total int, message string) {
	percent := float64(current) / float64(total) * 100
	fmt.Printf("\r🎨 进度: [%d/%d] %.1f%% - %s", current, total, percent, message)
	if current == total {
		fmt.Println()
	}
}

// ShowSummary 显示最终总结
func ShowSummary(processed, succeeded, failed int) {
	fmt.Println("")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 处理完成")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  总计: %d 个文件\n", processed)
	fmt.Printf("  ✅ 成功: %d 个\n", succeeded)
	if failed > 0 {
		fmt.Printf("  ❌ 失败: %d 个\n", failed)
	}

	successRate := float64(succeeded) / float64(processed) * 100
	fmt.Printf("  📈 成功率: %.1f%%\n", successRate)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
