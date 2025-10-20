package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	AppVersion = "v22.1.0"
	AppName    = "Pixly"
)

func main() {
	// 检查是否为简洁模式（用于减少重复版本信息）
	quietMode := false
	for _, arg := range os.Args[1:] {
		if arg == "--quiet" {
			quietMode = true
			break
		}
	}

	if !quietMode {
		fmt.Printf("🚀 %s %s - 智能化媒体优化解决方案\n", AppName, AppVersion)
		fmt.Println("📱 统一启动器 - 支持GUI和命令行模式")
		fmt.Println()
	}

	// 检查启动参数
	if len(os.Args) > 1 {
		switch os.Args[1] {
		// GUI模式暂时禁用 - 专注CLI开发
		// case "--gui", "-g":
		// 	launchGUI()
		// 	return
		case "--cli", "-c":
			launchCLI()
			return
		case "--help", "-h":
			showHelp()
			return
		case "--version", "-v":
			showVersion()
			return
		default:
			// 默认启动CLI模式（GUI已禁用）
			fmt.Println("📟 启动CLI模式（当前专注CLI开发）")
			launchCLI()
		}
	} else {
		// 无参数时，直接启动CLI模式
		fmt.Println("📟 默认启动CLI模式")
		launchCLI()
	}
}

// launchGUI 启动GUI模式 - 暂时禁用，专注CLI开发
/*
func launchGUI() bool {
	fmt.Println("🖥️ 启动图形界面模式...")

	// 查找Python GUI脚本
	execDir, err := os.Executable()
	if err != nil {
		fmt.Printf("❌ 获取程序路径失败: %v\n", err)
		return false
	}

	baseDir := filepath.Dir(execDir)
	guiScript := filepath.Join(baseDir, "tools", "pixly_gui.py")

	// 如果在tools目录中没找到，尝试相对路径
	if _, err := os.Stat(guiScript); os.IsNotExist(err) {
		guiScript = filepath.Join(baseDir, "..", "tools", "pixly_gui.py")
	}

	// 再次检查
	if _, err := os.Stat(guiScript); os.IsNotExist(err) {
		fmt.Printf("❌ 找不到GUI脚本: %s\n", guiScript)
		return false
	}

	// 检查Python环境
	pythonCmd := getPythonCommand()
	if pythonCmd == "" {
		fmt.Println("❌ 未找到Python环境")
		return false
	}

	fmt.Printf("✅ 找到GUI脚本: %s\n", guiScript)
	fmt.Printf("🐍 使用Python: %s\n", pythonCmd)

	// 启动Python GUI
	cmd := exec.Command(pythonCmd, guiScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Printf("❌ GUI启动失败: %v\n", err)
		return false
	}

	return true
}
*/

func launchCLI() {
	fmt.Println("📟 正在启动命令行界面...")

	// 查找CLI程序
	execDir, err := os.Executable()
	if err != nil {
		fmt.Printf("❌ 获取程序路径失败: %v\n", err)
		return
	}

	baseDir := filepath.Dir(execDir)

	// 可能的CLI程序路径
	possibleCLI := []string{
		filepath.Join(baseDir, "pixly_cli"),
		filepath.Join(baseDir, "pixly_test"),
		filepath.Join(baseDir, "pixly"),
		filepath.Join(baseDir, "pixly_final"),
	}

	var cliProgram string
	for _, path := range possibleCLI {
		if _, err := os.Stat(path); err == nil {
			cliProgram = path
			break
		}
	}

	if cliProgram == "" {
		fmt.Println("❌ 找不到CLI程序")
		fmt.Println("💡 请确保已编译核心程序")
		return
	}

	fmt.Printf("🔗 启动程序: %s\n", filepath.Base(cliProgram))
	fmt.Println()

	// 启动CLI程序并设置环境变量标记
	cmd := exec.Command(cliProgram)
	cmd.Env = append(os.Environ(), "PIXLY_LAUNCHED_BY=launcher") // 标记由launcher启动
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Printf("❌ CLI启动失败: %v\n", err)
	}
}

func hasGUIEnvironment() bool {
	switch runtime.GOOS {
	case "darwin":
		// macOS - 检查是否有窗口系统
		return os.Getenv("TERM_PROGRAM") != "" || os.Getenv("SSH_CLIENT") == ""
	case "linux":
		// Linux - 检查DISPLAY变量
		return os.Getenv("DISPLAY") != ""
	case "windows":
		// Windows - 默认有GUI
		return true
	default:
		return false
	}
}

func getPythonCommand() string {
	// 按优先级检查Python命令
	pythonCmds := []string{"python3", "python", "py"}

	for _, cmd := range pythonCmds {
		if _, err := exec.LookPath(cmd); err == nil {
			// 验证Python版本
			if checkPythonVersion(cmd) {
				return cmd
			}
		}
	}

	return ""
}

func checkPythonVersion(pythonCmd string) bool {
	cmd := exec.Command(pythonCmd, "--version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	version := string(output)
	// 简单检查是否包含Python 3
	return len(version) > 0 && (len(version) >= 8) // 至少 "Python 3"
}

func showHelp() {
	fmt.Printf(`🚀 %s %s - 智能化媒体优化解决方案

用法:
  pixly-unified [选项]

选项:
  --cli, -c     启动命令行模式
  --help, -h    显示此帮助信息
  --version, -v 显示版本信息

启动模式:
  📟 CLI模式     - 命令行界面，适合自动化和批处理（当前默认）

示例:
  pixly-unified           # 默认CLI模式
  pixly-unified --cli     # 强制CLI模式

技术栈:
  - 核心引擎: Go %s
  - 转换工具: FFmpeg + libjxl + libavif

注意:
  - GUI模式已暂时禁用，专注CLI开发
  - 所有功能通过命令行界面提供

`, AppName, AppVersion, runtime.Version())
}

func showVersion() {
	fmt.Printf(`%s %s

编译信息:
  Go版本:    %s
  操作系统:  %s
  架构:      %s
  编译时间:  构建时确定

组件版本:
  核心引擎:  %s
  GUI界面:   %s
  
技术栈:
  - 智能双阶段分析架构
  - 三种专业处理模式
  - 现代化GUI+CLI双界面
  - 企业级安全机制
  - 高性能并发处理

`, AppName, AppVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH, AppVersion, AppVersion)
}
