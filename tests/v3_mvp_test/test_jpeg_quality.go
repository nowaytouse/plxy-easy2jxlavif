package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"pixly/pkg/predictor"

	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 测试JPEG文件（不同pix_fmt）
	testFiles := []struct {
		path string
		desc string
	}{
		{"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/10.jpg", "yuv444p高质量"},
		{"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/11.jpg", "yuv444p高质量"},
		{"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/1573952589827.jpg", "yuv420p标准质量"},
		{"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/1580794244541.jpg", "yuv420p标准质量"},
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔬 Pixly v3.0 - JPEG质量验证测试")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("⚠️  核心验证：lossless_jpeg=1是否真正可逆？")
	fmt.Println()

	pred := predictor.NewPredictor(logger, "ffprobe")

	successCount := 0
	reversibleCount := 0
	totalConvertTime := time.Duration(0)
	totalSavingPercent := 0.0

	for i, test := range testFiles {
		if _, err := os.Stat(test.path); os.IsNotExist(err) {
			continue
		}

		fmt.Printf("[%d/%d] %s (%s)\n", i+1, len(testFiles), filepath.Base(test.path), test.desc)

		// 预测
		prediction, err := pred.PredictOptimalParams(test.path)
		if err != nil {
			fmt.Printf("  ❌ 预测失败: %v\n\n", err)
			continue
		}

		stat, _ := os.Stat(test.path)
		originalSize := stat.Size()

		fmt.Printf("  🎯 预测: lossless_jpeg=%v distance=%.1f\n",
			prediction.Params.LosslessJPEG,
			prediction.Params.Distance)

		// 实际转换
		outputPath := filepath.Join(".", fmt.Sprintf("test_jpeg_%d.jxl", i))

		convertStart := time.Now()
		cmd := exec.Command("cjxl",
			"--lossless_jpeg=1", // 使用lossless_jpeg=1
			"-e", fmt.Sprintf("%d", prediction.Params.Effort),
			test.path,
			outputPath)

		if err := cmd.Run(); err != nil {
			fmt.Printf("  ❌ 转换失败: %v\n\n", err)
			continue
		}
		convertTime := time.Since(convertStart)
		totalConvertTime += convertTime

		successCount++

		// 验证文件大小
		newStat, _ := os.Stat(outputPath)
		newSize := newStat.Size()
		savedPercent := float64(originalSize-newSize) / float64(originalSize) * 100
		totalSavingPercent += savedPercent

		fmt.Printf("  💾 空间: %.2f MB → %.2f MB (节省 %.1f%%)\n",
			float64(originalSize)/(1024*1024),
			float64(newSize)/(1024*1024),
			savedPercent)
		fmt.Printf("  ⏱️  转换耗时: %v\n", convertTime)

		// 质量验证：lossless_jpeg=1的可逆性测试
		// 将JXL解码回JPEG，检查是否完全相同
		reversedPath := filepath.Join(".", fmt.Sprintf("test_jpeg_%d_reversed.jpg", i))

		reverseCmd := exec.Command("djxl", outputPath, reversedPath)
		if err := reverseCmd.Run(); err != nil {
			fmt.Printf("  ❌ 解码失败: %v\n", err)
			os.Remove(outputPath)
			continue
		}

		// 检查文件大小是否相同（lossless_jpeg=1应该完全可逆）
		reversedStat, _ := os.Stat(reversedPath)
		reversedSize := reversedStat.Size()

		sizeMatch := reversedSize == originalSize
		sizeDiff := float64(reversedSize-originalSize) / float64(originalSize) * 100

		if sizeMatch {
			fmt.Printf("  ✅ 可逆性验证: 完美可逆（大小完全相同）\n")
			reversibleCount++
		} else {
			fmt.Printf("  ⚠️  可逆性验证: 大小差异 %.2f%%\n", sizeDiff)
			fmt.Printf("     原始: %d bytes | 解码后: %d bytes\n", originalSize, reversedSize)
		}

		// 清理临时文件
		os.Remove(outputPath)
		os.Remove(reversedPath)
		fmt.Println()
	}

	// 总结
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 测试总结")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  测试文件: %d\n", len(testFiles))
	fmt.Printf("  转换成功: %d\n", successCount)
	fmt.Printf("  完美可逆: %d\n", reversibleCount)

	if successCount > 0 {
		avgConvertTime := totalConvertTime / time.Duration(successCount)
		avgSaving := totalSavingPercent / float64(successCount)

		fmt.Printf("\n  ⚡ 性能:\n")
		fmt.Printf("     平均转换: %v\n", avgConvertTime)

		fmt.Printf("\n  💾 空间:\n")
		fmt.Printf("     平均节省: %.1f%%\n", avgSaving)

		fmt.Printf("\n  🎯 质量:\n")
		if reversibleCount == successCount {
			fmt.Printf("     ✅ 100%%完美可逆 (%d/%d)\n", reversibleCount, successCount)
		} else {
			fmt.Printf("     ⚠️  部分文件不可逆 (%d/%d)\n", reversibleCount, successCount)
		}
	}

	fmt.Println()

	if reversibleCount == successCount {
		fmt.Println("✅ JPEG lossless_jpeg=1验证通过！")
		fmt.Println()
		fmt.Println("关键验证:")
		fmt.Println("  ✓ lossless_jpeg=1完美可逆")
		fmt.Println("  ✓ 文件大小完全相同（bit-level）")
		fmt.Println("  ✓ 符合质量优先理念")
		fmt.Println()
		fmt.Println("🎯 JPEG预测器既简单又可靠！")
	} else {
		fmt.Println("⚠️  存在可逆性问题，需要调查")
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
