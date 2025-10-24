package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SafetyChecker 安全检查器
type SafetyChecker struct {
	config *Config
}

// NewSafetyChecker 创建安全检查器
func NewSafetyChecker(config *Config) *SafetyChecker {
	return &SafetyChecker{
		config: config,
	}
}

// CheckPath 检查路径安全性
func (sc *SafetyChecker) CheckPath(path string) error {
	if !sc.config.SafetyChecks {
		return nil // 安全检查已禁用
	}

	// 检查路径是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("路径不存在: %s", path)
	}

	// 获取绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("无法获取绝对路径: %w", err)
	}

	// 检查是否是系统关键目录
	if sc.isSystemDirectory(absPath) {
		return fmt.Errorf("❌ 危险：系统目录不允许转换\n路径: %s\n为了安全，请选择用户目录", absPath)
	}

	// 检查是否是根目录
	if sc.isRootDirectory(absPath) {
		return fmt.Errorf("❌ 危险：根目录不允许转换\n请选择具体的子目录")
	}

	return nil
}

// isSystemDirectory 判断是否是系统目录
func (sc *SafetyChecker) isSystemDirectory(path string) bool {
	systemDirs := sc.getSystemDirectories()

	for _, sysDir := range systemDirs {
		if strings.HasPrefix(path, sysDir) {
			return true
		}
	}

	return false
}

// getSystemDirectories 获取系统关键目录列表
func (sc *SafetyChecker) getSystemDirectories() []string {
	switch runtime.GOOS {
	case "darwin": // macOS
		return []string{
			"/System",
			"/Library",
			"/Applications",
			"/bin",
			"/sbin",
			"/usr",
			"/var",
			"/private",
			"/etc",
			"/dev",
			"/Volumes/Macintosh HD/System",
		}
	case "linux":
		return []string{
			"/sys",
			"/proc",
			"/dev",
			"/boot",
			"/bin",
			"/sbin",
			"/usr/bin",
			"/usr/sbin",
			"/etc",
			"/lib",
			"/lib64",
		}
	case "windows":
		return []string{
			"C:\\Windows",
			"C:\\Program Files",
			"C:\\Program Files (x86)",
		}
	}

	return []string{}
}

// isRootDirectory 判断是否是根目录
func (sc *SafetyChecker) isRootDirectory(path string) bool {
	switch runtime.GOOS {
	case "darwin", "linux":
		return path == "/"
	case "windows":
		return len(path) == 3 && path[1] == ':' && path[2] == '\\'
	}
	return false
}

// ConfirmAction 确认用户操作（防止意外）
func (sc *SafetyChecker) ConfirmAction(message string, timeout time.Duration) (bool, error) {
	if !sc.config.IsInteractive() {
		return true, nil // 非交互模式默认确认
	}

	fmt.Printf("\n⚠️  %s\n", message)
	fmt.Print("   确认操作吗？(yes/no): ")

	// 使用channel实现超时
	result := make(chan string, 1)
	go func() {
		var input string
		fmt.Scanln(&input)
		result <- strings.ToLower(strings.TrimSpace(input))
	}()

	// 等待用户输入或超时
	select {
	case input := <-result:
		return input == "yes" || input == "y", nil
	case <-time.After(timeout):
		fmt.Println("\n   ⏱️  超时，操作取消")
		return false, fmt.Errorf("用户确认超时")
	}
}

// CheckFileCount 检查文件数量（防止意外大量转换）
func (sc *SafetyChecker) CheckFileCount(count int, threshold int) error {
	if !sc.config.SafetyChecks {
		return nil
	}

	if count > threshold {
		return fmt.Errorf("⚠️  文件数量过多（%d个），超过阈值（%d）\n建议分批处理或增加阈值", count, threshold)
	}

	return nil
}

// ValidateDirectory 验证目录（综合检查）
func (sc *SafetyChecker) ValidateDirectory(path string) error {
	// 路径安全性检查
	if err := sc.CheckPath(path); err != nil {
		return err
	}

	// 权限检查
	if !sc.hasWritePermission(path) {
		return fmt.Errorf("❌ 权限不足：无法写入目录 %s", path)
	}

	return nil
}

// hasWritePermission 检查写入权限
func (sc *SafetyChecker) hasWritePermission(path string) bool {
	// 尝试创建测试文件
	testFile := filepath.Join(path, ".pixly_permission_test")
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(testFile)
	return true
}
