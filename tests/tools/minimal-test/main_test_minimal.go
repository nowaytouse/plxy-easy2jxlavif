package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const TestVersion = "TEST-1.0.0"

func main() {
	fmt.Printf("🧪 Pixly 测试版 v%s 启动\n", TestVersion)
	fmt.Println("=" + strings.Repeat("=", 50))

	// 显示测试菜单
	for {
		showTestMenu()
		choice := getInput("请选择 (1-4): ")

		switch choice {
		case "1":
			runBasicScan()
		case "2":
			runModeSelection()
		case "3":
			runCorruptedFileHandling()
		case "4":
			fmt.Println("👋 测试结束，谢谢使用！")
			return
		default:
			fmt.Println("❌ 无效选择，请重试")
		}
	}
}

func showTestMenu() {
	fmt.Println("\n📋 测试菜单")
	fmt.Println("1. 基础扫描测试")
	fmt.Println("2. 模式选择测试")
	fmt.Println("3. 损坏文件处理测试")
	fmt.Println("4. 退出程序")
	fmt.Println()
}

func getInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func runBasicScan() {
	fmt.Println("\n🔍 基础扫描测试")

	dirPath := getInput("请输入目录路径: ")
	if dirPath == "" {
		dirPath = "/Users/nameko_1/Documents/Pixly/test_pack_all/不同格式测试合集_测试运行"
		fmt.Printf("使用默认路径: %s\n", dirPath)
	}

	// 检查目录是否存在
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		fmt.Printf("❌ 目录不存在: %s\n", dirPath)
		return
	}

	fmt.Printf("✅ 开始扫描目录: %s\n", dirPath)

	var files []string
	var mediaFiles []string
	var problemFiles []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			files = append(files, path)

			ext := strings.ToLower(filepath.Ext(path))
			baseName := filepath.Base(path)

			// 检查是否为媒体文件
			if isMediaFile(ext) {
				mediaFiles = append(mediaFiles, path)

				// 模拟问题文件检测
				if strings.Contains(baseName, "corrupt") || info.Size() < 1024 {
					problemFiles = append(problemFiles, path)
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("❌ 扫描失败: %v\n", err)
		return
	}

	// 显示扫描结果
	fmt.Printf("\n📊 扫描完成\n")
	fmt.Printf("📁 总文件数: %d\n", len(files))
	fmt.Printf("🎬 媒体文件: %d\n", len(mediaFiles))
	fmt.Printf("⚠️ 问题文件: %d\n", len(problemFiles))

	if len(mediaFiles) > 0 {
		fmt.Println("\n🎬 发现的媒体文件:")
		for i, file := range mediaFiles {
			if i >= 5 {
				fmt.Printf("   ... 还有 %d 个文件\n", len(mediaFiles)-5)
				break
			}
			fmt.Printf("   • %s\n", filepath.Base(file))
		}
	}

	if len(problemFiles) > 0 {
		fmt.Println("\n⚠️ 发现的问题文件:")
		for _, file := range problemFiles {
			fmt.Printf("   • %s\n", filepath.Base(file))
		}
	}

	fmt.Println("\n✅ 扫描测试完成")
}

func runModeSelection() {
	fmt.Println("\n🎯 模式选择测试")
	fmt.Println("1. 自动模式+ (智能路由)")
	fmt.Println("2. 品质模式 (无损压缩)")
	fmt.Println("3. 表情包模式 (极限压缩)")

	choice := getInput("请选择处理模式 (1-3): ")

	var modeName string
	switch choice {
	case "1":
		modeName = "自动模式+"
	case "2":
		modeName = "品质模式"
	case "3":
		modeName = "表情包模式"
	default:
		fmt.Println("❌ 无效选择")
		return
	}

	fmt.Printf("✅ 已选择: %s\n", modeName)
	fmt.Printf("📝 模拟处理 3 个测试文件...\n")

	// 模拟处理过程
	testFiles := []string{"test1.jpg", "test2.png", "test3.mp4"}
	for i, file := range testFiles {
		fmt.Printf("🔄 处理 %s... ", file)
		time.Sleep(500 * time.Millisecond)
		fmt.Printf("✅ 完成 (%d/%d)\n", i+1, len(testFiles))
	}

	fmt.Println("✅ 模式选择测试完成")
}

func runCorruptedFileHandling() {
	fmt.Println("\n⚠️ 损坏文件处理测试")

	// 模拟发现损坏文件
	corruptedFiles := []string{"broken1.jpg", "corrupt_video.mp4", "damaged.png"}

	fmt.Printf("检测到 %d 个损坏文件:\n", len(corruptedFiles))
	for _, file := range corruptedFiles {
		fmt.Printf("   • %s\n", file)
	}

	fmt.Println("\n处理选项:")
	fmt.Println("1. 尝试修复")
	fmt.Println("2. 全部删除")
	fmt.Println("3. 终止任务")
	fmt.Println("4. 忽略跳过 (推荐)")

	// 倒计时选择
	fmt.Print("请选择 (1-4) [10秒后自动选择4]: ")

	type result struct {
		choice string
		err    error
	}

	resultCh := make(chan result, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		resultCh <- result{choice: strings.TrimSpace(input), err: err}
	}()

	// 倒计时
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	seconds := 10
	for {
		select {
		case res := <-resultCh:
			if res.err == nil && res.choice != "" {
				handleCorruptedChoice(res.choice)
				return
			}
			// 如果输入为空或有错误，继续等待或超时
		case <-timeout:
			fmt.Printf("\n⏰ 超时，自动选择忽略跳过\n")
			fmt.Println("✅ 已忽略损坏文件，继续处理其他文件")
			return
		case <-ticker.C:
			seconds--
			if seconds > 0 {
				fmt.Printf("\r请选择 (1-4) [%d秒后自动选择4]: ", seconds)
			}
		}
	}
}

func handleCorruptedChoice(choice string) {
	switch choice {
	case "1":
		fmt.Println("🔧 尝试修复损坏文件...")
		fmt.Println("✅ 修复完成")
	case "2":
		fmt.Println("🗑️ 删除所有损坏文件...")
		fmt.Println("✅ 删除完成")
	case "3":
		fmt.Println("⏹️ 任务已终止")
	case "4", "":
		fmt.Println("⏭️ 忽略损坏文件，继续处理其他文件")
		fmt.Println("✅ 已忽略损坏文件")
	default:
		fmt.Println("❌ 无效选择，自动选择忽略")
		fmt.Println("✅ 已忽略损坏文件")
	}
}

func isMediaFile(ext string) bool {
	mediaExts := []string{
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".heif", ".heic",
		".tiff", ".tif", ".bmp", ".avif", ".jxl",
		".mp4", ".mov", ".avi", ".mkv", ".webm", ".m4v",
	}

	for _, mediaExt := range mediaExts {
		if ext == mediaExt {
			return true
		}
	}
	return false
}
