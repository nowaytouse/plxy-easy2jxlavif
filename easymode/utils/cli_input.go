package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// PromptForDirectory 提示用户输入目录路径（交互模式）
// 如果提供的dir为空，会提示用户输入
// 返回处理后的目录路径（支持拖拽路径转义）
func PromptForDirectory(dir string) string {
	if dir != "" {
		return dir
	}
	
	// 交互模式：提示用户输入（支持拖拽）
	fmt.Println("📁 请拖入要处理的文件夹，然后按回车键：")
	fmt.Println("   （或直接输入路径）")
	fmt.Print("\n路径: ")
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("⚠️  读取输入失败: %v\n", err)
		return ""
	}
	
	// 清理输入并转义macOS拖拽路径
	path := strings.TrimSpace(input)
	path = unescapeShellPath(path)
	
	return path
}

// PromptForDirectoryWithMessage 使用自定义提示消息提示用户输入目录
func PromptForDirectoryWithMessage(dir string, message string) string {
	if dir != "" {
		return dir
	}
	
	fmt.Print(message)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	
	path := strings.TrimSpace(input)
	path = unescapeShellPath(path)
	return path
}

// PerformSafetyCheck 执行安全检查
func PerformSafetyCheck(targetPath string) error {
	// 检查是否为关键系统路径
	if isCriticalSystemPath(targetPath) {
		return fmt.Errorf("❌ 拒绝处理系统关键路径: %s", targetPath)
	}
	
	// 检查是否为敏感目录
	if isSensitiveDirectory(targetPath) {
		return fmt.Errorf("⚠️  警告: 这是敏感目录，建议谨慎处理: %s", targetPath)
	}
	
	// 检查磁盘空间
	free, total, err := getDiskSpace(targetPath)
	if err == nil {
		freeGB := float64(free) / (1024 * 1024 * 1024)
		totalGB := float64(total) / (1024 * 1024 * 1024)
		if freeGB < 1.0 {
			return fmt.Errorf("⚠️  磁盘空间不足: %.2f GB / %.2f GB", freeGB, totalGB)
		}
	}
	
	return nil
}

// unescapeShellPath 反转义macOS终端拖拽路径
func unescapeShellPath(path string) string {
	// macOS拖拽路径会转义空格和特殊字符
	path = strings.ReplaceAll(path, "\\ ", " ")
	path = strings.ReplaceAll(path, "\\(", "(")
	path = strings.ReplaceAll(path, "\\)", ")")
	path = strings.ReplaceAll(path, "\\&", "&")
	path = strings.ReplaceAll(path, "\\[", "[")
	path = strings.ReplaceAll(path, "\\]", "]")
	
	// 移除可能的引号
	path = strings.Trim(path, "\"'")
	
	return path
}

// isCriticalSystemPath 检查是否为系统关键路径
func isCriticalSystemPath(path string) bool {
	criticalPaths := []string{
		"/System",
		"/Library",
		"/private",
		"/usr",
		"/bin",
		"/sbin",
		"/etc",
		"/var",
		"/tmp",
		"/dev",
	}
	
	absPath, _ := filepath.Abs(path)
	for _, critical := range criticalPaths {
		if strings.HasPrefix(absPath, critical) {
			return true
		}
	}
	return false
}

// isSensitiveDirectory 检查是否为敏感目录
func isSensitiveDirectory(path string) bool {
	sensitiveDirs := []string{
		"/Applications",
		os.Getenv("HOME") + "/Desktop",
		os.Getenv("HOME") + "/Documents",
		os.Getenv("HOME") + "/Downloads",
	}
	
	absPath, _ := filepath.Abs(path)
	for _, sensitive := range sensitiveDirs {
		if absPath == sensitive {
			return true
		}
	}
	return false
}

// getDiskSpace 获取磁盘空间信息
func getDiskSpace(path string) (free, total uint64, err error) {
	var stat syscall.Statfs_t
	err = syscall.Statfs(path, &stat)
	if err != nil {
		return 0, 0, err
	}
	
	free = stat.Bavail * uint64(stat.Bsize)
	total = stat.Blocks * uint64(stat.Bsize)
	return free, total, nil
}

// ShowProgress 显示进度
func ShowProgress(current, total int, message string) {
	percent := float64(current) / float64(total) * 100
	fmt.Printf("\r⏳ %s: %d/%d (%.1f%%)", message, current, total, percent)
	if current == total {
		fmt.Println()
	}
}

// ShowBanner 显示工具横幅
func ShowBanner(toolName, version string) {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Printf("║   🎨 %-50s ║\n", toolName+" v"+version)
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}
