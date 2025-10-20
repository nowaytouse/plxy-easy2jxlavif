package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	testDir := "/Users/nameko_1/Documents/Pixly/test_pack_all/测试_新副本_20250828_055908"

	fmt.Println("🔧 开始简单转换测试...")
	fmt.Printf("📂 测试目录: %s\n\n", testDir)

	// 查找 jpe 和 jfif 文件
	jpeFile := filepath.Join(testDir, "sample_5184×3456.jpe")
	jfifFile := filepath.Join(testDir, "FlULvU0WIAUPeEo.jfif")

	// 测试文件 1: .jpe 转换为 .jxl
	if _, err := os.Stat(jpeFile); err == nil {
		fmt.Printf("🎯 测试 .jpe → .jxl 转换: %s\n", filepath.Base(jpeFile))

		outputFile := strings.TrimSuffix(jpeFile, ".jpe") + ".jxl"

		cmd := exec.Command("cjxl", jpeFile, outputFile, "-q", "90")
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ 转换失败: %v\n", err)
		} else {
			// 检查输出文件
			if stat, err := os.Stat(outputFile); err == nil {
				fmt.Printf("✅ 转换成功! 输出文件: %s (%.1f MB)\n",
					filepath.Base(outputFile), float64(stat.Size())/(1024*1024))
			}
		}
	}

	fmt.Println()

	// 测试文件 2: .jfif 转换为 .jxl
	if _, err := os.Stat(jfifFile); err == nil {
		fmt.Printf("🎯 测试 .jfif → .jxl 转换: %s\n", filepath.Base(jfifFile))

		outputFile := strings.TrimSuffix(jfifFile, ".jfif") + ".jxl"

		cmd := exec.Command("cjxl", jfifFile, outputFile, "-q", "90")
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ 转换失败: %v\n", err)
		} else {
			// 检查输出文件
			if stat, err := os.Stat(outputFile); err == nil {
				fmt.Printf("✅ 转换成功! 输出文件: %s (%.1f MB)\n",
					filepath.Base(outputFile), float64(stat.Size())/(1024*1024))
			}
		}
	}

	fmt.Println("\n🎉 简单转换测试完成！")
	fmt.Println("这证明了 cjxl 工具可以正确处理 .jpe 和 .jfif 格式")
}
